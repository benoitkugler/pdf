package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) processPages(entry pdfcpu.Object) (model.PageTree, error) {
	pages := r.resolve(entry)
	pagesDict, isDict := pages.(pdfcpu.Dict)
	if !isDict {
		return model.PageTree{}, errType("Pages", pages)
	}
	root, err := r.resolvePageTree(pagesDict, nil)
	if err != nil {
		return model.PageTree{}, err
	}
	return *root, err
}

func (r resolver) processContentStream(content pdfcpu.Object) (*model.ContentStream, error) {
	content = r.resolve(content)
	if content == nil {
		return nil, nil
	}
	stream, ok := content.(pdfcpu.StreamDict)
	if !ok {
		return nil, errType("Content stream", content)
	}
	var out model.ContentStream
	// length will be deduced from the content
	out.Content = stream.Raw

	filters := r.resolve(stream.Dict["Filter"])
	if filterName, isName := filters.(pdfcpu.Name); isName {
		filters = pdfcpu.Array{filterName}
	}
	ar, _ := filters.(pdfcpu.Array)
	for _, name := range ar {
		if filterName, isName := r.resolveName(name); isName {
			if f := model.NewFilter(string(filterName)); f != "" {
				out.Filter = []model.Filter{f}
			}
		}
	}
	decode := r.resolve(stream.Dict["DecodeParms"])
	switch decode := decode.(type) {
	case pdfcpu.Array: // one dict param per filter
		if len(decode) != len(out.Filter) {
			return nil, fmt.Errorf("unexpected length for DecodeParms array: %d", len(decode))
		}
		for _, parms := range decode {
			parmsModel := r.processDecodeParms(parms)
			out.DecodeParms = append(out.DecodeParms, parmsModel)
		}
	case pdfcpu.Dict: // one filter and one dict param
		if len(out.Filter) != 1 {
			return nil, errType("DecodeParms", decode)
		}
		out.DecodeParms = append(out.DecodeParms, r.processDecodeParms(decode))
	}

	return &out, nil
}

func (r *resolver) resolvePageObject(node pdfcpu.Dict, parent *model.PageTree) (*model.PageObject, error) {
	resources, err := r.resolveOneResourceDict(node["Resources"])
	if err != nil {
		return nil, err
	}
	var page model.PageObject
	page.Parent = parent
	page.Resources = resources
	page.MediaBox = r.rectangleFromArray(node["MediaBox"])
	page.CropBox = r.rectangleFromArray(node["CropBox"])
	page.BleedBox = r.rectangleFromArray(node["BleedBox"])
	page.TrimBox = r.rectangleFromArray(node["TrimBox"])
	page.ArtBox = r.rectangleFromArray(node["ArtBox"])
	if rot, ok := r.resolveInt(node["Rotate"]); ok {
		page.Rotate = model.NewRotation(rot)
	}

	// one content stream won't probably be referenced twice:
	// dont bother tracking the refs
	contents := r.resolve(node["Contents"])
	switch contents := contents.(type) {
	case pdfcpu.Array: // array of streams
		page.Contents = make([]model.ContentStream, len(contents))
		for _, v := range contents {
			cts, err := r.processContentStream(v)
			if err != nil {
				return nil, err
			}
			if cts != nil {
				page.Contents = append(page.Contents, *cts)
			}
		}
	case pdfcpu.StreamDict:
		ct, err := r.processContentStream(contents)
		if err != nil {
			return nil, err
		}
		if ct != nil {
			page.Contents = append(page.Contents, *ct)
		}
	}

	annots, _ := r.resolve(node["Annots"]).(pdfcpu.Array)
	for _, annot := range annots {
		an, err := r.resolveAnnotation(annot)
		if err != nil {
			return nil, err
		}
		page.Annots = append(page.Annots, an)
	}
	return &page, nil
}

func (r *resolver) resolveAnnotation(annot pdfcpu.Object) (*model.Annotation, error) {
	annotRef, isRef := annot.(pdfcpu.IndirectRef)
	if annotModel := r.annotations[annotRef]; isRef && annotModel != nil {
		return annotModel, nil
	}
	annot = r.resolve(annot)
	annotDict, isDict := annot.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Annotation", annot)
	}
	annotModel, err := r.resolveAnnotationFields(annotDict)
	if err != nil {
		return nil, err
	}
	if isRef { // write the annotation back into the cache
		r.annotations[annotRef] = &annotModel
	}
	return &annotModel, nil
}

func (r *resolver) resolveAnnotationFields(annotDict pdfcpu.Dict) (model.Annotation, error) {
	var (
		annotModel model.Annotation
		err        error
	)
	if rect := r.rectangleFromArray(annotDict["Rect"]); rect != nil {
		annotModel.Rect = *rect
	}

	contents, _ := isString(r.resolve(annotDict["Contents"]))
	annotModel.Contents = decodeTextString(contents)

	if f, ok := r.resolveInt(annotDict["F"]); ok {
		annotModel.F = f
	}
	if name, ok := r.resolveName(annotDict["Name"]); ok {
		annotModel.AS = name
	}
	border, _ := r.resolve(annotDict["Border"]).(pdfcpu.Array)
	var bo model.Border
	if len(border) >= 3 {
		bo.HCornerRadius, _ = isNumber(r.resolve(border[0]))
		bo.VCornerRadius, _ = isNumber(r.resolve(border[1]))
		bo.BorderWidth, _ = isNumber(r.resolve(border[2]))
		if len(border) == 4 {
			dash, _ := r.resolve(border[4]).(pdfcpu.Array)
			bo.DashArray = r.processFloatArray(dash)
		}
		annotModel.Border = &bo
	}
	annotModel.AP, err = r.resolveAppearanceDict(annotDict["AP"])
	if err != nil {
		return annotModel, err
	}

	annotModel.Subtype, err = r.resolveAnnotationSubType(annotDict)
	if err != nil {
		return annotModel, err
	}

	return annotModel, nil
}

// node, possibly root
func (r *resolver) resolvePageTree(node pdfcpu.Dict, parent *model.PageTree) (*model.PageTree, error) {
	var page model.PageTree
	resources, err := r.resolveOneResourceDict(node["Resources"])
	if err != nil {
		return nil, err
	}
	page.Parent = parent
	page.Resources = resources
	kids, _ := r.resolve(node["Kids"]).(pdfcpu.Array)
	for _, node := range kids {
		// track the refs to page object, needed by destinations
		ref, isRef := node.(pdfcpu.IndirectRef)
		node = r.resolve(node)
		nodeDict, ok := node.(pdfcpu.Dict)
		if !ok {
			return nil, errType("PageNode", node)
		}
		kid, err := r.processPageNode(nodeDict, &page)
		if err != nil {
			return nil, err
		}
		if pagePtr, ok := kid.(*model.PageObject); ok && isRef {
			r.pages[ref] = pagePtr
		}
		page.Kids = append(page.Kids, kid)
	}
	return &page, nil
}

func (r *resolver) processPageNode(node pdfcpu.Dict, parent *model.PageTree) (model.PageNode, error) {
	name, _ := r.resolveName(node["Type"])
	switch name {
	case "Pages":
		return r.resolvePageTree(node, parent)
	case "Page":
		return r.resolvePageObject(node, parent)
	default:
		return nil, fmt.Errorf("unexpected value for Type field of page node: %s", node["Type"])
	}
}

// TODO: support more destination
func (r *resolver) resolveExplicitDestination(dest pdfcpu.Array) (*model.ExplicitDestination, error) {
	if len(dest) != 5 {
		return nil, nil
	}
	out := new(model.ExplicitDestination)
	pageRef, isRef := dest[0].(pdfcpu.IndirectRef)
	if !isRef {
		return nil, errType("Dest.Page", dest[0])
	}
	if name, _ := r.resolveName(dest[1]); name != "XYZ" {
		return nil, fmt.Errorf("expected /XYZ in Destination, got unsupported %s", dest[1])
	}
	if left, ok := isNumber(r.resolve(dest[2])); ok {
		out.Left = &left
	}
	if top, ok := isNumber(r.resolve(dest[3])); ok {
		out.Top = &top
	}
	out.Zoom, _ = isNumber(r.resolve(dest[4]))
	// store the incomplete destination to process later on
	r.destinationsToComplete = append(r.destinationsToComplete,
		incompleteDest{ref: pageRef, destination: out})
	return out, nil
}

// TODO: support more destination format
func (r *resolver) processDestination(dest pdfcpu.Object) (model.Destination, error) {
	dest = r.resolve(dest)
	switch dest := dest.(type) {
	case pdfcpu.Name:
		return model.DestinationName(dest), nil
	case pdfcpu.StringLiteral, pdfcpu.HexLiteral:
		d, _ := isString(dest)
		return model.DestinationString(decodeTextString(d)), nil
	case pdfcpu.Array:
		return r.resolveExplicitDestination(dest)
	default:
		return nil, errType("Destination", dest)
	}
}

// TODO: support more
func (r *resolver) processAction(action pdfcpu.Dict) (model.Action, error) {
	name, _ := r.resolveName(action["S"])
	switch name {
	case "URI":
		uri, _ := isString(r.resolve(action["URI"]))
		return model.URIAction(uri), nil
	case "GoTo":
		dest, err := r.processDestination(action["D"])
		if err != nil {
			return nil, err
		}
		return model.GoToAction{D: dest}, nil
	default:
		return nil, nil
	}
}

// TODO: more annotation subtypes
func (r *resolver) resolveAnnotationSubType(annot pdfcpu.Dict) (model.AnnotationType, error) {
	var err error
	name, _ := r.resolveName(annot["Subtype"])
	switch name {
	case "Link":
		var an model.LinkAnnotation
		aDict, isDict := r.resolve(annot["A"]).(pdfcpu.Dict)
		if isDict {
			an.A, err = r.processAction(aDict)
			return an, err
		}
		an.Dest, err = r.processDestination(annot["Dest"])
		if err != nil {
			return an, err
		}
		return an, nil
	case "FileAttachment":
		var an model.FileAttachmentAnnotation
		title, _ := isString(r.resolve(annot["T"]))
		an.T = decodeTextString(title)
		an.FS, err = r.resolveFileSpec(annot["FS"])
		return an, err
	case "Widget":
		// TODO:
		return model.WidgetAnnotation{}, nil
	default:
		return nil, nil
	}
}

func (r resolver) resolveFileSpec(fs pdfcpu.Object) (*model.FileSpec, error) {
	fsRef, isFsRef := fs.(pdfcpu.IndirectRef)
	if fileSpec := r.fileSpecs[fsRef]; isFsRef && fileSpec != nil {
		return fileSpec, nil
	}
	fs = r.resolve(fs)
	var file model.FileSpec
	if fileString, ok := isString(fs); ok { // File Specification String
		file.UF = fileString
	} else { // File Specification Dictionary
		fsDict, isDict := fs.(pdfcpu.Dict)
		if !isDict {
			return nil, errType("FileSpec", fs)
		}
		uf, _ := isString(r.resolve(fsDict["UF"]))
		desc, _ := isString(r.resolve(fsDict["Desc"]))
		file.UF = decodeTextString(uf)
		file.Desc = decodeTextString(desc)

		ef := r.resolve(fsDict["EF"])
		efDict, isDict := ef.(pdfcpu.Dict)
		if !isDict {
			return nil, errType("EF Dict", ef)
		}
		fileEntry := efDict["UF"]
		for _, alt := range [...]string{"F", "DOS", "Mac", "Unix"} {
			if fileEntry != nil {
				break
			}
			fileEntry = efDict[alt]
		}
		var err error
		file.EF, err = r.resolveFileContent(fileEntry)
		if err != nil {
			return nil, err
		}
	}
	if isFsRef { // write back to the cache
		r.fileSpecs[fsRef] = &file
	}
	return &file, nil
}

func (r resolver) resolveFileContent(fileEntry pdfcpu.Object) (*model.EmbeddedFileStream, error) {
	fileEntryRef, isFileRef := fileEntry.(pdfcpu.IndirectRef)
	if emb := r.fileContents[fileEntryRef]; isFileRef && emb != nil {
		return emb, nil
	}
	fileEntry = r.resolve(fileEntry)
	stream, isStream := fileEntry.(pdfcpu.StreamDict)
	if !isStream {
		return nil, errType("Stream Dict", fileEntry)
	}

	params := r.resolve(stream.Dict["Params"])
	paramsDict, isDict := params.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("FileStream.Params", params)
	}
	var paramsModel model.EmbeddedFileParams
	if size, ok := r.resolveInt(paramsDict["Size"]); ok {
		paramsModel.Size = size
	}

	checkSum, _ := isString(r.resolve(paramsDict["CheckSum"]))
	paramsModel.CheckSum = checkSum

	if cd, ok := isString(r.resolve(paramsDict["CreationDate"])); ok {
		paramsModel.CreationDate, _ = pdfcpu.DateTime(cd)
	}
	if md, ok := isString(r.resolve(paramsDict["ModDate"])); ok {
		paramsModel.ModDate, _ = pdfcpu.DateTime(md)
	}

	var (
		out model.EmbeddedFileStream
		err error
	)
	out.Params = paramsModel
	cs, err := r.processContentStream(stream)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, errors.New("missing file content stream")
	}
	out.ContentStream = *cs
	if isFileRef { // write back to the cache
		r.fileContents[fileEntryRef] = &out
	}
	return &out, err
}

func (r resolver) processDecodeParms(parms pdfcpu.Object) map[model.Name]int {
	parmsDict, _ := r.resolve(parms).(pdfcpu.Dict)
	parmsModel := make(map[model.Name]int)
	for paramName, paramVal := range parmsDict {
		var intVal int
		switch val := r.resolve(paramVal).(type) {
		case pdfcpu.Boolean:
			if val {
				intVal = 1
			} else {
				intVal = 0
			}
		case pdfcpu.Integer:
			intVal = val.Value()
		default:
			continue
		}
		parmsModel[model.Name(paramName)] = intVal
	}
	return parmsModel
}
