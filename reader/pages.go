package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) processPages(catalog pdfcpu.Dict) (model.PageTree, error) {
	pages := r.resolve(catalog["Pages"])
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

func (r resolver) resolvePageObject(node pdfcpu.Dict, parent *model.PageTree) (*model.PageObject, error) {
	resources, err := r.resolveResources(node["Resources"])
	if err != nil {
		return nil, err
	}
	var page model.PageObject
	page.Parent = parent
	page.Resources = resources
	if ar, isArray := node["MediaBox"].(pdfcpu.Array); isArray {
		page.MediaBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["CropBox"].(pdfcpu.Array); isArray {
		page.CropBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["BleedBox"].(pdfcpu.Array); isArray {
		page.BleedBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["TrimBox"].(pdfcpu.Array); isArray {
		page.TrimBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["ArtBox"].(pdfcpu.Array); isArray {
		page.ArtBox = rectangleFromArray(ar)
	}
	if rot := node.IntEntry("Rotate"); rot != nil {
		page.Rotate = model.NewRotation(*rot)
	}

	annots := node.ArrayEntry("Annots")
	for _, annot := range annots {
		annotRef, isRef := annot.(pdfcpu.IndirectRef)
		if annotModel := r.annotations[annotRef]; isRef && annotModel != nil {
			page.Annots = append(page.Annots, annotModel)
			continue
		}
		annot = r.resolve(annot)
		annotDict, isDict := annot.(pdfcpu.Dict)
		if !isDict {
			return nil, errType("Annotation", annot)
		}
		var annotModel model.Annotation
		if rect := rectangleFromArray(annotDict.ArrayEntry("Rect")); rect != nil {
			annotModel.Rect = *rect
		}

		contents, _ := annotDict["Contents"].(pdfcpu.StringLiteral)
		annotModel.Contents = decodeStringLit(contents)

		if f := annotDict.IntEntry("F"); f != nil {
			annotModel.F = *f
		}
		if name := annotDict.NameEntry("Name"); name != nil {
			annotModel.AS = model.Name(*name)
		}

		annotModel.AP, err = r.resolveAppearanceDict(annotDict["AP"])
		if err != nil {
			return nil, err
		}

		annotModel.Subtype, err = r.resolveAnnotationSubType(annotDict)
		if err != nil {
			return nil, err
		}

		if isRef { // write the annotation back into the cache
			r.annotations[annotRef] = &annotModel
		}
		page.Annots = append(page.Annots, &annotModel)
	}
	return &page, nil
}

// node, possibly root
func (r resolver) resolvePageTree(node pdfcpu.Dict, parent *model.PageTree) (*model.PageTree, error) {
	resources, err := r.resolveResources(node["Resources"])
	if err != nil {
		return nil, err
	}
	var page model.PageTree
	page.Parent = parent
	page.Resources = resources
	kids, _ := r.resolve(node["Kids"]).(pdfcpu.Array)
	for _, node := range kids {
		// each page should be distinct, don't bother
		// tracking the refs
		node = r.resolve(node)
		nodeDict, ok := node.(pdfcpu.Dict)
		if !ok {
			return nil, errType("PageNode", node)
		}
		kid, err := r.processPageNode(nodeDict, &page)
		if err != nil {
			return nil, err
		}
		page.Kids = append(page.Kids, kid)
	}
	return &page, nil
}

func (r resolver) processPageNode(node pdfcpu.Dict, parent *model.PageTree) (model.PageNode, error) {
	switch node["Type"] {
	case pdfcpu.Name("Pages"):
		return r.resolvePageTree(node, parent)
	case pdfcpu.Name("Page"):
		return r.resolvePageObject(node, parent)
	default:
		return nil, fmt.Errorf("unexpected value for Type field of page node: %s", node["Type"])
	}
}

// TODO:
func (r resolver) resolveAnnotationSubType(annot pdfcpu.Dict) (model.AnnotationType, error) {
	var err error
	switch annot["Subtype"] {
	case pdfcpu.Name("Link"):
		return model.LinkAnnotation{}, nil
	case pdfcpu.Name("FileAttachment"):
		var an model.FileAttachmentAnnotation
		title, _ := annot["T"].(pdfcpu.StringLiteral)
		an.T = decodeStringLit(title)
		an.FS, err = r.resolveFileSpec(annot["FS"])
		return an, err
	case pdfcpu.Name("Widget"):
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

	var file model.FileSpec
	fsObj := r.resolve(fs)
	fsDict, isDict := fsObj.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("FileSpec", fsObj)
	}
	uf, _ := fsDict["UF"].(pdfcpu.StringLiteral)
	desc, _ := fsDict["Desc"].(pdfcpu.StringLiteral)
	file.UF = decodeStringLit(uf)
	file.Desc = decodeStringLit(desc)

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
	if size := paramsDict.IntEntry("Size"); size != nil {
		paramsModel.Size = *size
	}

	checkSum, _ := paramsDict["CheckSum"].(pdfcpu.StringLiteral)
	paramsModel.CheckSum = decodeStringLit(checkSum)

	if cd := paramsDict.StringLiteralEntry("CreationDate"); cd != nil {
		paramsModel.CreationDate, _ = pdfcpu.DateTime(cd.Value())
	}
	if md := paramsDict.StringLiteralEntry("ModDate"); md != nil {
		paramsModel.ModDate, _ = pdfcpu.DateTime(md.Value())
	}

	var (
		out model.EmbeddedFileStream
		err error
	)
	out.Params = paramsModel
	out.Content = stream.Raw
	out.StreamDict, err = r.processStreamDict(stream.Dict)
	if err != nil {
		return nil, err
	}
	if isFileRef { // write back to the cache
		r.fileContents[fileEntryRef] = &out
	}
	return &out, err
}

func (r resolver) processDecodeParms(parms pdfcpu.Object) map[model.Name]int {
	parmsModel := make(map[model.Name]int)
	parmsDict, _ := r.resolve(parms).(pdfcpu.Dict)
	for paramName, paramVal := range parmsDict {
		var intVal int
		switch val := paramVal.(type) {
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

func (r resolver) processStreamDict(dict pdfcpu.Dict) (model.StreamDict, error) {
	var out model.StreamDict
	filters := r.resolve(dict["Filters"])
	if filterName, isName := filters.(pdfcpu.Name); isName {
		filters = pdfcpu.Array{filterName}
	}
	ar, _ := filters.(pdfcpu.Array)
	for _, name := range ar {
		if filterName, isName := name.(pdfcpu.Name); isName {
			if f := model.NewFilter(string(filterName)); f != "" {
				out.Filters = []model.Filter{f}
			}
		}
	}
	decode := r.resolve(dict["DecodeParms"])
	switch decode := decode.(type) {
	case pdfcpu.Array: // one dict param per filter
		if len(decode) != len(out.Filters) {
			return out, fmt.Errorf("unexpected length for DecodeParms array: %d", len(decode))
		}
		for _, parms := range decode {
			parmsModel := r.processDecodeParms(parms)
			out.DecodeParms = append(out.DecodeParms, parmsModel)
		}
	case pdfcpu.Dict: // one filter and one dict param
		if len(out.Filters) != 1 {
			return out, errType("DecodeParms", decode)
		}
		out.DecodeParms = append(out.DecodeParms, r.processDecodeParms(decode))
	}
	return out, nil
}
