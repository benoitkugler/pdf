package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
	"github.com/benoitkugler/pdf/reader/parser"
)

func (r resolver) processPages(entry model.Object) (model.PageTree, error) {
	pages := r.resolve(entry)
	pagesDict, isDict := pages.(model.ObjDict)
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
func (r resolver) allocatesPages(pages model.Object) {
	ref, isRef := pages.(model.ObjIndirectRef)
	pagesDict, _ := r.resolve(pages).(model.ObjDict)
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

// return false instead of an error for null values
func (r resolver) resolveStream(content model.Object) (model.Stream, bool, error) {
	var out model.Stream
	content = r.resolve(content)
	if (content == nil || content == model.ObjNull{}) {
		return out, false, nil
	}
	stream, ok := content.(model.ObjStream)
	if !ok {
		return out, false, errType("Content stream", content)
	}
	// length will be deduced from the content
	out.Content = stream.Content

	var err error
	out.Filter, err = parser.ParseFilters(stream.Args["Filter"], stream.Args["DecodeParms"], func(o parser.Object) (parser.Object, error) {
		return r.resolve(o), nil
	})
	if err != nil {
		return out, false, err
	}

	return out, true, nil
}

// `page` has been previously allocated and must be filled
func (r resolver) resolvePageObject(node model.ObjDict, page *model.PageObject) error {
	if node["Resources"] != nil {
		resources, err := r.resolveOneResourceDict(node["Resources"])
		if err != nil {
			return err
		}
		page.Resources = &resources
	}
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
	case model.ObjArray: // array of streams
		page.Contents = make([]model.ContentStream, 0, len(contents))
		for _, v := range contents {
			ct, ok, err := r.resolveStream(v)
			if err != nil {
				return err
			}
			if ok { // invalid content stream are just ignored
				page.Contents = append(page.Contents, model.ContentStream{Stream: ct})
			}
		}
	case model.ObjStream:
		ct, ok, err := r.resolveStream(contents)
		if err != nil {
			return err
		}
		if ok {
			page.Contents = append(page.Contents, model.ContentStream{Stream: ct})
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
		page.StructParents = model.ObjInt(st)
	}
	if tabs, ok := r.resolveName(node["Tabs"]); ok {
		page.Tabs = tabs
	}
	return nil
}

func (r resolver) resolveAnnotation(annot model.Object) (*model.AnnotationDict, error) {
	annotRef, isRef := annot.(model.ObjIndirectRef)
	if annotModel := r.annotations[annotRef]; isRef && annotModel != nil {
		return annotModel, nil
	}
	var out model.AnnotationDict
	if isRef {
		// annotation may have action which refer back to them
		// to avoid loop we begin by register the new pointer
		// which will be update soon
		r.annotations[annotRef] = &out
	}
	annot = r.resolve(annot)
	annotDict, isDict := annot.(model.ObjDict)
	if !isDict {
		return nil, errType("Annotation", annot)
	}
	err := r.resolveAnnotationFields(annotDict, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r resolver) resolveAnnotationFields(annotDict model.ObjDict, out *model.AnnotationDict) error {
	var err error
	out.BaseAnnotation, err = r.resolveBaseAnnotation(annotDict)
	if err != nil {
		return err
	}

	out.Subtype, err = r.resolveAnnotationSubType(annotDict)

	return err
}

func (r resolver) resolveBaseAnnotation(annotDict model.ObjDict) (out model.BaseAnnotation, err error) {
	if rect := r.rectangleFromArray(annotDict["Rect"]); rect != nil {
		out.Rect = *rect
	}

	contents, _ := file.IsString(r.resolve(annotDict["Contents"]))
	out.Contents = DecodeTextString(contents)

	nm, _ := file.IsString(r.resolve(annotDict["NM"]))
	out.NM = DecodeTextString(nm)

	if m, ok := file.IsString(r.resolve(annotDict["M"])); ok {
		if mt, ok := DateTime(m); ok {
			out.M = mt
		}
	}

	out.AP, err = r.resolveAppearanceDict(annotDict["AP"])
	if err != nil {
		return out, err
	}
	if name, ok := r.resolveName(annotDict["AS"]); ok {
		out.AS = name
	}
	if f, ok := r.resolveInt(annotDict["F"]); ok {
		out.F = model.AnnotationFlag(f)
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
		out.Border = &bo
	}

	color, _ := r.resolveArray(annotDict["C"])
	switch len(color) {
	case 0, 1, 3, 4: // accepted color lengths
		out.C = r.processFloatArray(color)
	}

	if st, ok := r.resolveInt(annotDict["StructParent"]); ok {
		out.StructParent = model.ObjInt(st)
	}
	return out, nil
}

func (r resolver) resolveBorderStyle(o model.Object) *model.BorderStyle {
	dict, _ := r.resolve(o).(model.ObjDict)
	if dict == nil {
		return nil
	}
	var out model.BorderStyle
	if w, ok := r.resolveNumber(dict["W"]); ok {
		out.W = model.ObjFloat(w)
	}
	out.S, _ = r.resolveName(dict["S"])
	d, _ := r.resolveArray(dict["D"])
	if d != nil {
		out.D = r.processFloatArray(d)
	}
	return &out
}

// node, possibly root
func (r resolver) resolvePageTree(node model.ObjDict) (*model.PageTree, error) {
	var page model.PageTree
	if node["Resources"] != nil { // else, inherited
		resources, err := r.resolveOneResourceDict(node["Resources"])
		if err != nil {
			return nil, err
		}
		page.Resources = &resources
	}
	page.MediaBox = r.rectangleFromArray(node["MediaBox"])

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

func (r resolver) processPageNode(node model.Object) (model.PageNode, error) {
	// track the refs to page object, needed by destinations
	ref, isRef := node.(model.ObjIndirectRef)
	node = r.resolve(node)
	nodeDict, ok := node.(model.ObjDict)
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

func (r resolver) resolveDestinationLocation(dest model.ObjArray) (model.DestinationLocation, error) {
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
			loc.Dim = model.ObjFloat(left)
		}
		return loc, nil
	case "XYZ":
		if len(dest) < 5 {
			return nil, errType("Destination XYZ", dest)
		}
		loc := model.DestinationLocationXYZ{}
		if left, ok := r.resolveNumber(dest[2]); ok {
			loc.Left = model.ObjFloat(left)
		}
		if top, ok := r.resolveNumber(dest[3]); ok {
			loc.Top = model.ObjFloat(top)
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

func (r resolver) resolveExplicitDestination(dest model.ObjArray) (model.DestinationExplicit, error) {
	if len(dest) < 2 {
		return nil, errType("Destination", dest)
	}
	var err error
	if pageRef, isRef := dest[0].(model.ObjIndirectRef); isRef { // page is intern
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

func (r resolver) processDestination(dest model.Object) (model.Destination, error) {
	dest = r.resolve(dest)
	switch dest := dest.(type) {
	case model.ObjName:
		return model.DestinationName(dest), nil
	case model.ObjStringLiteral, model.ObjHexLiteral:
		d, _ := file.IsString(dest)
		return model.DestinationString(DecodeTextString(d)), nil
	case model.ObjArray:
		return r.resolveExplicitDestination(dest)
	default:
		return nil, errType("Destination", dest)
	}
}

// TODO: more annotation subtypes
func (r resolver) resolveAnnotationSubType(annot model.ObjDict) (model.Annotation, error) {
	var err error
	name, _ := r.resolveName(annot["Subtype"])
	switch name {
	case "Text":
		var an model.AnnotationText
		an.AnnotationMarkup, err = r.resolveAnnotationMarkup(annot)
		if err != nil {
			return nil, err
		}
		an.Open, _ = r.resolveBool(annot["Open"])
		an.Name, _ = r.resolveName(annot["Name"])
		if st, ok := file.IsString(r.resolve(annot["State"])); ok {
			an.State = DecodeTextString(st)
		}
		if st, ok := file.IsString(r.resolve(annot["StateModel"])); ok {
			an.StateModel = DecodeTextString(st)
		}
		return an, nil
	case "Link":
		var an model.AnnotationLink
		if aDict, isDict := r.resolve(annot["A"]).(model.ObjDict); isDict {
			an.A, err = r.processAction(aDict)
			if err != nil {
				return nil, err
			}
		} else if dest := r.resolve(annot["Dest"]); dest != nil {
			an.Dest, err = r.processDestination(dest)
			if err != nil {
				return nil, err
			}
		}
		h, _ := r.resolveName(annot["H"])
		an.H = model.Highlighting(h)
		an.PA, err = r.processAction(annot["PA"])
		if err != nil {
			return nil, err
		}
		qp, _ := r.resolveArray(annot["QuadPoints"])
		an.QuadPoints = r.processFloatArray(qp)
		an.BS = r.resolveBorderStyle(annot["BS"])
		return an, nil
	case "FileAttachment":
		var an model.AnnotationFileAttachment
		title, _ := file.IsString(r.resolve(annot["T"]))
		an.T = DecodeTextString(title)
		an.FS, err = r.resolveFileSpec(annot["FS"])
		return an, err
	case "Widget":
		var an model.AnnotationWidget
		h, _ := r.resolveName(annot["H"])
		an.H = model.Highlighting(h)
		an.MK, err = r.resolveAnnotationMK(annot["MK"])
		if err != nil {
			return nil, err
		}
		an.A, err = r.processAction(annot["A"])
		if err != nil {
			return nil, err
		}
		an.BS = r.resolveBorderStyle(annot["BS"])
		an.AA, err = r.resolveAnnotationAA(annot["AA"])
		if err != nil {
			return nil, err
		}

		return an, nil
	case "Screen":
		var an model.AnnotationScreen
		title, _ := file.IsString(r.resolve(annot["T"]))
		an.T = DecodeTextString(title)
		an.MK, err = r.resolveAnnotationMK(annot["MK"])
		if err != nil {
			return nil, err
		}
		an.A, err = r.processAction(annot["A"])
		if err != nil {
			return nil, err
		}
		an.AA, err = r.resolveAnnotationAA(annot["AA"])
		if err != nil {
			return nil, err
		}
		if ref, isRef := annot["P"].(model.ObjIndirectRef); isRef {
			an.P = r.pages[ref]
		}
		return an, nil
	case "": // a form field may come here
		return nil, nil
	default:
		fmt.Println("TODO annot :", name, annot)
		return nil, nil
	}
}

func (r resolver) resolveAnnotationMarkup(annot model.ObjDict) (out model.AnnotationMarkup, err error) {
	t, _ := file.IsString(r.resolve(annot["T"]))
	out.T = DecodeTextString(t)
	out.Popup, err = r.resolveAnnotationPopup(annot)
	if err != nil {
		return out, err
	}
	if ca, ok := r.resolveNumber(annot["CA"]); ok {
		out.CA = model.ObjFloat(ca)
	}
	out.RC = r.textOrStream(annot["RC"])

	cd, _ := file.IsString(r.resolve(annot["CreationDate"]))
	out.CreationDate, _ = DateTime(cd)

	subj, _ := file.IsString(r.resolve(annot["Subj"]))
	out.Subj = DecodeTextString(subj)

	out.IT, _ = r.resolveName(annot["IT"])
	return out, nil
}

func (r resolver) resolveAnnotationPopup(o model.Object) (*model.AnnotationPopup, error) {
	o = r.resolve(o)
	if o == nil {
		return nil, nil
	}
	dict, ok := o.(model.ObjDict)
	if !ok {
		return nil, errType("Popup Annotation", o)
	}
	var (
		out model.AnnotationPopup
		err error
	)
	out.BaseAnnotation, err = r.resolveBaseAnnotation(dict)
	if err != nil {
		return nil, err
	}
	out.Open, _ = r.resolveBool(dict["Open"])
	return &out, nil
}

func (r resolver) resolveAnnotationAA(o model.Object) (out model.AnnotationAdditionalActions, err error) {
	dict, _ := r.resolve(o).(model.ObjDict)
	if dict == nil {
		return out, nil
	}
	out.E, err = r.processAction(dict["E"])
	if err != nil {
		return out, err
	}
	out.X, err = r.processAction(dict["X"])
	if err != nil {
		return out, err
	}
	out.D, err = r.processAction(dict["D"])
	if err != nil {
		return out, err
	}
	out.U, err = r.processAction(dict["U"])
	if err != nil {
		return out, err
	}
	out.Fo, err = r.processAction(dict["Fo"])
	if err != nil {
		return out, err
	}
	out.Bl, err = r.processAction(dict["Bl"])
	if err != nil {
		return out, err
	}
	out.PO, err = r.processAction(dict["PO"])
	if err != nil {
		return out, err
	}
	out.PC, err = r.processAction(dict["PC"])
	if err != nil {
		return out, err
	}
	out.PV, err = r.processAction(dict["PV"])
	if err != nil {
		return out, err
	}
	out.PI, err = r.processAction(dict["PI"])
	if err != nil {
		return out, err
	}
	return out, nil
}

func (r resolver) resolveAnnotationMK(o model.Object) (*model.AppearanceCharacteristics, error) {
	dict, _ := r.resolve(o).(model.ObjDict)
	if dict == nil {
		return nil, nil
	}
	var out model.AppearanceCharacteristics
	rt, _ := r.resolveInt(dict["R"])
	out.R = model.NewRotation(rt)

	bc, _ := r.resolveArray(dict["BC"])
	out.BC = r.processFloatArray(bc)

	bg, _ := r.resolveArray(dict["BG"])
	out.BG = r.processFloatArray(bg)

	ts, _ := file.IsString(r.resolve(dict["CA"]))
	out.CA = DecodeTextString(ts)
	ts, _ = file.IsString(r.resolve(dict["RC"]))
	out.RC = DecodeTextString(ts)
	ts, _ = file.IsString(r.resolve(dict["AC"]))
	out.AC = DecodeTextString(ts)

	var err error
	if of := dict["I"]; r.resolve(of) != nil {
		out.I, err = r.resolveOneXObjectForm(of)
		if err != nil {
			return nil, err
		}
	}
	if of := dict["RI"]; r.resolve(of) != nil {
		out.RI, err = r.resolveOneXObjectForm(of)
		if err != nil {
			return nil, err
		}
	}
	if of := dict["IX"]; r.resolve(of) != nil {
		out.IX, err = r.resolveOneXObjectForm(of)
		if err != nil {
			return nil, err
		}
	}
	out.IF = r.resolveIconFit(dict["IF"])
	if tp, ok := r.resolveInt(dict["TP"]); ok {
		out.TP = uint8(tp)
	}
	return &out, nil
}

func (r resolver) resolveIconFit(o model.Object) *model.IconFit {
	dict, ok := r.resolve(o).(model.ObjDict)
	if !ok {
		return nil
	}
	var out model.IconFit
	out.SW, _ = r.resolveName(dict["SW"])
	out.S, _ = r.resolveName(dict["S"])
	if a, ok := r.resolveArray(dict["A"]); ok && len(a) == 2 {
		a := r.processFloatArray(a)
		out.A = &[2]Fl{a[0], a[1]}
	}
	if fb, ok := r.resolveBool(dict["FB"]); ok {
		out.FB = fb
	}
	return &out
}

func (r resolver) resolveFileSpec(fs model.Object) (*model.FileSpec, error) {
	fsRef, isFsRef := fs.(model.ObjIndirectRef)
	if fileSpec := r.fileSpecs[fsRef]; isFsRef && fileSpec != nil {
		return fileSpec, nil
	}
	fs = r.resolve(fs)

	var fileSpec model.FileSpec
	if fileString, ok := file.IsString(fs); ok { // File Specification String
		fileSpec.UF = fileString
	} else { // File Specification Dictionary
		fsDict, isDict := fs.(model.ObjDict)
		if !isDict {
			return nil, errType("FileSpec", fs)
		}

		// we give the priority to UF, and default to F
		uf, _ := file.IsString(r.resolve(fsDict["UF"]))
		fileSpec.UF = DecodeTextString(uf)
		if fileSpec.UF == "" {
			fileSpec.UF, _ = file.IsString(r.resolve(fsDict["F"]))
		}

		desc, _ := file.IsString(r.resolve(fsDict["Desc"]))
		fileSpec.Desc = DecodeTextString(desc)

		ef := r.resolve(fsDict["EF"])
		efDict, isDict := ef.(model.ObjDict)
		if !isDict {
			return nil, errType("EF Dict", ef)
		}
		fileEntry := efDict["UF"]
		for _, alt := range [...]model.Name{"F", "DOS", "Mac", "Unix"} {
			if fileEntry != nil {
				break
			}
			fileEntry = efDict[alt]
		}
		var err error
		fileSpec.EF, err = r.resolveFileContent(fileEntry)
		if err != nil {
			return nil, err
		}
	}
	if isFsRef { // write back to the cache
		r.fileSpecs[fsRef] = &fileSpec
	}
	return &fileSpec, nil
}

func (r resolver) resolveFileContent(fileEntry model.Object) (*model.EmbeddedFileStream, error) {
	fileEntryRef, isFileRef := fileEntry.(model.ObjIndirectRef)
	if emb := r.fileContents[fileEntryRef]; isFileRef && emb != nil {
		return emb, nil
	}
	fileEntry = r.resolve(fileEntry)
	stream, isStream := fileEntry.(model.ObjStream)
	if !isStream {
		return nil, errType("Stream Dict", fileEntry)
	}

	var (
		out model.EmbeddedFileStream
		err error
	)
	paramsDict, _ := r.resolve(stream.Args["Params"]).(model.ObjDict) // optional
	if size, ok := r.resolveInt(paramsDict["Size"]); ok {
		out.Params.Size = size
	}

	checkSum, _ := file.IsString(r.resolve(paramsDict["CheckSum"]))
	out.Params.CheckSum = checkSum

	if cd, ok := file.IsString(r.resolve(paramsDict["CreationDate"])); ok {
		out.Params.CreationDate, _ = DateTime(cd)
	}
	if md, ok := file.IsString(r.resolve(paramsDict["ModDate"])); ok {
		out.Params.ModDate, _ = DateTime(md)
	}

	cs, ok, err := r.resolveStream(stream)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("missing file content stream")
	}
	out.Stream = cs
	if isFileRef { // write back to the cache
		r.fileContents[fileEntryRef] = &out
	}
	return &out, err
}
