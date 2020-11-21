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
	// actions may require page object that are not yet processed
	// so, we make two passes: a first pass to fill the map indirect ref -> page object
	// and a second pass to do the real processing
	r.allocatesPages(pagesDict)

	root, err := r.resolvePageTree(pagesDict)
	if err != nil {
		return model.PageTree{}, err
	}
	return *root, err
}

// delay error handling to the second pass
func (r resolver) allocatesPages(pages pdfcpu.Object) {
	ref, isRef := pages.(pdfcpu.IndirectRef)
	pagesDict, _ := r.resolve(pages).(pdfcpu.Dict)
	name, _ := r.resolveName(pagesDict["Type"])
	switch name {
	case "Pages": // recursion
		kids, _ := r.resolveArray(pagesDict["Kids"])
		for _, kid := range kids {
			r.allocatesPages(kid)
		}
	case "Page": // allocate a page object and store it
		if isRef {
			r.pages[ref] = new(model.PageObject)
		}
	}
}

func (r resolver) resolveStream(content pdfcpu.Object) (*model.Stream, error) {
	content = r.resolve(content)
	if content == nil {
		return nil, nil
	}
	stream, ok := content.(pdfcpu.StreamDict)
	if !ok {
		return nil, errType("Content stream", content)
	}
	var out model.Stream
	// length will be deduced from the content
	out.Content = stream.Raw
	filters := r.resolve(stream.Dict["Filter"])
	if filterName, isName := filters.(pdfcpu.Name); isName {
		filters = pdfcpu.Array{filterName}
	}
	ar, _ := filters.(pdfcpu.Array)
	for _, name := range ar {
		if filterName, isName := r.resolveName(name); isName {
			out.Filter = []model.Filter{{Name: model.Name(filterName)}}
		}
	}
	decode := r.resolve(stream.Dict["DecodeParms"])
	switch decode := decode.(type) {
	case pdfcpu.Array: // one dict param per filter
		if len(decode) != len(out.Filter) {
			return nil, fmt.Errorf("unexpected length for DecodeParms array: %d", len(decode))
		}
		for i, parms := range decode {
			out.Filter[i].DecodeParams = r.processDecodeParms(parms)
		}
	case pdfcpu.Dict: // one filter and one dict param
		if len(out.Filter) != 1 {
			return nil, errType("DecodeParms", decode)
		}
		out.Filter[0].DecodeParams = r.processDecodeParms(decode)
	}

	return &out, nil
}

// `page` has been previously allocated and must be filled
func (r resolver) resolvePageObject(node pdfcpu.Dict, page *model.PageObject) error {
	resources, err := r.resolveOneResourceDict(node["Resources"])
	if err != nil {
		return err
	}
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
			ct, err := r.resolveStream(v)
			if err != nil {
				return err
			}
			if ct != nil {
				page.Contents = append(page.Contents, model.ContentStream{Stream: *ct})
			}
		}
	case pdfcpu.StreamDict:
		ct, err := r.resolveStream(contents)
		if err != nil {
			return err
		}
		if ct != nil {
			page.Contents = append(page.Contents, model.ContentStream{Stream: *ct})
		}
	}

	annots, _ := r.resolveArray(node["Annots"])
	for _, annot := range annots {
		an, err := r.resolveAnnotation(annot)
		if err != nil {
			return err
		}
		page.Annots = append(page.Annots, an)
	}
	if st, ok := r.resolveInt(node["StructParents"]); ok {
		page.StructParents = model.Int(st)
	}
	if tabs, ok := r.resolveName(node["Tabs"]); ok {
		page.Tabs = tabs
	}
	return nil
}

func (r resolver) resolveAnnotation(annot pdfcpu.Object) (*model.AnnotationDict, error) {
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

func (r resolver) resolveAnnotationFields(annotDict pdfcpu.Dict) (model.AnnotationDict, error) {
	var (
		annotModel model.AnnotationDict
		err        error
	)
	if rect := r.rectangleFromArray(annotDict["Rect"]); rect != nil {
		annotModel.Rect = *rect
	}

	contents, _ := isString(r.resolve(annotDict["Contents"]))
	annotModel.Contents = decodeTextString(contents)

	nm, _ := isString(r.resolve(annotDict["NM"]))
	annotModel.NM = decodeTextString(nm)

	if m, ok := isString(r.resolve(annotDict["M"])); ok {
		if mt, ok := pdfcpu.DateTime(m); ok {
			annotModel.M = mt
		}
	}

	if f, ok := r.resolveInt(annotDict["F"]); ok {
		annotModel.F = model.AnnotationFlag(f)
	}
	if name, ok := r.resolveName(annotDict["Name"]); ok {
		annotModel.AS = name
	}
	border, _ := r.resolveArray(annotDict["Border"])
	var bo model.Border
	if len(border) >= 3 {
		bo.HCornerRadius, _ = r.resolveNumber(border[0])
		bo.VCornerRadius, _ = r.resolveNumber(border[1])
		bo.BorderWidth, _ = r.resolveNumber(border[2])
		if len(border) == 4 {
			dash, _ := r.resolveArray(border[4])
			bo.DashArray = r.processFloatArray(dash)
		}
		annotModel.Border = &bo
	}

	color, _ := r.resolveArray(annotDict["C"])
	switch len(color) {
	case 0, 1, 3, 4: // accepted color values
		annotModel.C = r.processFloatArray(color)
	}

	annotModel.AP, err = r.resolveAppearanceDict(annotDict["AP"])
	if err != nil {
		return annotModel, err
	}
	if st, ok := r.resolveInt(annotDict["StructParent"]); ok {
		annotModel.StructParent = model.Int(st)
	}

	annotModel.Subtype, err = r.resolveAnnotationSubType(annotDict)
	if err != nil {
		return annotModel, err
	}

	return annotModel, nil
}

// node, possibly root
func (r resolver) resolvePageTree(node pdfcpu.Dict) (*model.PageTree, error) {
	var page model.PageTree
	resources, err := r.resolveOneResourceDict(node["Resources"])
	if err != nil {
		return nil, err
	}
	page.Resources = resources
	kids, _ := r.resolveArray(node["Kids"])
	for _, node := range kids {
		kid, err := r.processPageNode(node)
		if err != nil {
			return nil, err
		}
		page.Kids = append(page.Kids, kid)
	}
	return &page, nil
}

func (r resolver) processPageNode(node pdfcpu.Object) (model.PageNode, error) {
	// track the refs to page object, needed by destinations
	ref, isRef := node.(pdfcpu.IndirectRef)
	node = r.resolve(node)
	nodeDict, ok := node.(pdfcpu.Dict)
	if !ok {
		return nil, errType("PageNode", node)
	}
	name, _ := r.resolveName(nodeDict["Type"])
	switch name {
	case "Pages":
		return r.resolvePageTree(nodeDict)
	case "Page":
		var page *model.PageObject
		if isRef {
			page = r.pages[ref]
		} else { // should not happen
			page = new(model.PageObject)
		}
		err := r.resolvePageObject(nodeDict, page)
		return page, err
	default:
		return nil, fmt.Errorf("unexpected value for Type field of page node: %s", nodeDict["Type"])
	}
}

func (r resolver) resolveDestinationLocation(dest pdfcpu.Array) (model.DestinationLocation, error) {
	name, _ := r.resolveName(dest[1])
	switch name {
	case "Fit", "FitB":
		return model.DestinationLocationFit(name), nil
	case "FitH", "FitV", "FitBH", "FitBV":
		if len(dest) < 3 {
			return nil, errType("Destination Fit", dest)
		}
		loc := model.DestinationLocationFitDim{}
		loc.Name = name
		if left, ok := r.resolveNumber(dest[2]); ok {
			loc.Dim = model.Float(left)
		}
		return loc, nil
	case "XYZ":
		if len(dest) < 5 {
			return nil, errType("Destination XYZ", dest)
		}
		loc := model.DestinationLocationXYZ{}
		if left, ok := r.resolveNumber(dest[2]); ok {
			loc.Left = model.Float(left)
		}
		if top, ok := r.resolveNumber(dest[3]); ok {
			loc.Top = model.Float(top)
		}
		loc.Zoom, _ = r.resolveNumber(dest[4])
		return loc, nil
	case "FitR":
		if len(dest) < 6 {
			return nil, errType("Destination FitR", dest)
		}
		loc := model.DestinationLocationFitR{}
		loc.Left, _ = r.resolveNumber(dest[2])
		loc.Bottom, _ = r.resolveNumber(dest[3])
		loc.Right, _ = r.resolveNumber(dest[4])
		loc.Top, _ = r.resolveNumber(dest[5])
		return loc, nil
	default:
		return nil, fmt.Errorf("in Destination, got unsupported mode %s", dest[1])
	}
}

func (r resolver) resolveExplicitDestination(dest pdfcpu.Array) (model.DestinationExplicit, error) {
	if len(dest) < 2 {
		return nil, errType("Destination", dest)
	}
	var err error
	if pageRef, isRef := dest[0].(pdfcpu.IndirectRef); isRef { // page is intern
		var out model.DestinationExplicitIntern
		out.Location, err = r.resolveDestinationLocation(dest)
		if err != nil {
			return nil, err
		}
		out.Page = r.pages[pageRef]
		return out, nil
	} else { // page is extern
		var out model.DestinationExplicitExtern
		out.Page, _ = r.resolveInt(dest[0])
		out.Location, err = r.resolveDestinationLocation(dest)
		return out, err
	}
}

func (r resolver) processDestination(dest pdfcpu.Object) (model.Destination, error) {
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

// TODO: more annotation subtypes
func (r resolver) resolveAnnotationSubType(annot pdfcpu.Dict) (model.Annotation, error) {
	var err error
	name, _ := r.resolveName(annot["Subtype"])
	switch name {
	case "Link":
		var an model.AnnotationLink
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
		var an model.AnnotationFileAttachment
		title, _ := isString(r.resolve(annot["T"]))
		an.T = decodeTextString(title)
		an.FS, err = r.resolveFileSpec(annot["FS"])
		return an, err
	case "Widget":
		// TODO:
		return model.AnnotationWidget{}, nil
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
	cs, err := r.resolveStream(stream)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, errors.New("missing file content stream")
	}
	out.Stream = *cs
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
