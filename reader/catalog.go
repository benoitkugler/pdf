package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
)

func (r resolver) catalog() (model.Catalog, error) {
	var (
		out model.Catalog
		err error
	)
	d, ok := r.resolve(r.file.Root).(model.ObjDict)
	if !ok {
		return out, fmt.Errorf("can't resolve Catalog: expected dict, got %#v", r.resolve(r.file.Root))
	}

	out.AcroForm, err = r.processAcroForm(d["AcroForm"])
	if err != nil {
		return out, err
	}

	out.Pages, err = r.processPages(d["Pages"])
	if err != nil {
		return out, err
	}

	out.Dests, err = r.resolveDests(d["Dests"])
	if err != nil {
		return out, err
	}

	out.Names, err = r.processNameDict(d["Names"])
	if err != nil {
		return out, err
	}

	out.PageLayout, _ = r.resolveName(d["PageLayout"])
	out.PageMode, _ = r.resolveName(d["PageMode"])

	if pl := d["PageLabels"]; pl != nil {
		out.PageLabels = new(model.PageLabelsTree)
		err = r.resolveNumberTree(pl, pageLabelTree{out: out.PageLabels})
		if err != nil {
			return out, err
		}
	}

	// pages, annotations, xforms need to be resolved
	out.StructTreeRoot, err = r.resolveStructureTree(d["StructTreeRoot"])
	if err != nil {
		return out, err
	}

	// may need pages
	// HACK for #252
	out.Outlines, err = r.resolveOutline(d["Outlines"])
	if err != nil {
		return out, err
	}
	if out.Outlines == nil {
		out.Outlines, err = r.resolveOutline(d["Outline"])
		if err != nil {
			return out, err
		}
	}

	out.ViewerPreferences, err = r.resolveViewerPreferences(d["ViewerPreferences"])
	if err != nil {
		return out, err
	}

	out.MarkInfo, err = r.resolveMarkDict(d["MarkInfo"])
	if err != nil {
		return out, err
	}

	uriDict, ok := r.resolve(d["URI"]).(model.ObjDict)
	if ok {
		out.URI, _ = file.IsString(r.resolve(uriDict["Base"]))
	}

	out.OpenAction, err = r.resolveDestinationOrAction(d["OpenAction"])
	if err != nil {
		return out, err
	}

	lang, _ := file.IsString(r.resolve(d["Lang"]))
	out.Lang = DecodeTextString(lang)

	return out, nil
}

func (r resolver) rectangleFromArray(array model.Object) *model.Rectangle {
	ar, _ := r.resolveArray(array)
	if len(ar) < 4 {
		return nil
	}
	llx, _ := r.resolveNumber(ar[0])
	lly, _ := r.resolveNumber(ar[1])
	urx, _ := r.resolveNumber(ar[2])
	ury, _ := r.resolveNumber(ar[3])
	return &model.Rectangle{Llx: llx, Lly: lly, Urx: urx, Ury: ury}
}

func (r resolver) matrixFromArray(array model.Object) *model.Matrix {
	ar, _ := r.resolveArray(array)
	if len(ar) != 6 {
		return nil
	}
	var out model.Matrix
	for i := range out {
		out[i], _ = r.resolveNumber(ar[i])
	}
	return &out
}

func (r resolver) resolveAppearanceDict(o model.Object) (*model.AppearanceDict, error) {
	ref, isRef := o.(model.ObjIndirectRef)
	if isRef {
		if ff := r.appearanceDicts[ref]; ff != nil {
			return ff, nil
		}
		o = r.resolve(ref)
	}
	if o == nil {
		return nil, nil
	}
	a, isDict := o.(model.ObjDict)
	if !isDict {
		return nil, errType("AppearanceDict", o)
	}
	var (
		out model.AppearanceDict
		err error
	)
	if ap := a["N"]; ap != nil {
		out.N, err = r.resolveAppearanceEntry(ap)
		if err != nil {
			return nil, err
		}
	}
	if ap := a["R"]; ap != nil {
		out.R, err = r.resolveAppearanceEntry(ap)
		if err != nil {
			return nil, err
		}
	}
	if ap := a["D"]; ap != nil {
		out.D, err = r.resolveAppearanceEntry(ap)
		if err != nil {
			return nil, err
		}
	}
	if isRef { // write back to the cache
		r.appearanceDicts[ref] = &out
	}
	return &out, nil
}

func (r resolver) resolveAppearanceEntry(obj model.Object) (model.AppearanceEntry, error) {
	out := make(model.AppearanceEntry)

	// obj might be either a subdictionary or a streamdictionary
	if subDict, isResolvedDict := r.resolve(obj).(model.ObjDict); isResolvedDict {
		// subdictionary
		for name, stream := range subDict {
			formObj, err := r.resolveOneXObjectForm(stream)
			if err != nil {
				return nil, err
			}
			out[model.ObjName(name)] = formObj
		}
	} else { // stream (surely indirect)
		ap, err := r.resolveOneXObjectForm(obj)
		if err != nil {
			return nil, err
		}
		out = model.AppearanceEntry{"": ap}
	}
	return out, nil
}

// return an error if obj is nil
func (r resolver) resolveOneXObjectForm(obj model.Object) (*model.XObjectForm, error) {
	xObjRef, isRef := obj.(model.ObjIndirectRef)
	if out := r.xObjectForms[xObjRef]; isRef && out != nil {
		return out, nil
	}

	// some PDF have Object form with ressources pointing to themselves
	// thus, we first register the current object
	out := new(model.XObjectForm)
	if isRef {
		r.xObjectForms[xObjRef] = out
	}

	err := r.resolveXFormObjectFields(obj, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// return an error if resolved obj is nil
// do not register the ref
func (r resolver) resolveXFormObjectFields(obj model.Object, out *model.XObjectForm) error {
	obj = r.resolve(obj)
	cs, ok, err := r.resolveStream(obj)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("missing Form XObject")
	}
	out.ContentStream = model.ContentStream{Stream: cs}

	stream, _ := obj.(model.ObjStream) // here, we are sure obj is a stream
	if rect := r.rectangleFromArray(r.resolve(stream.Args["BBox"])); rect != nil {
		out.BBox = *rect
	}
	if mat := r.matrixFromArray(r.resolve(stream.Args["Matrix"])); mat != nil {
		out.Matrix = *mat
	}
	if res := stream.Args["Resources"]; res != nil {
		var err error
		out.Resources, err = r.resolveOneResourceDict(res)
		if err != nil {
			return err
		}
	}
	if st, ok := r.resolveInt(stream.Args["StructParent"]); ok {
		out.StructParent = model.ObjInt(st)
	} else if st, ok := r.resolveInt(stream.Args["StructParents"]); ok {
		out.StructParents = model.ObjInt(st)
	}

	return nil
}

func (r *resolver) resolveOneXObjectGroup(obj model.Object) (*model.XObjectTransparencyGroup, error) {
	xObjRef, isRef := obj.(model.ObjIndirectRef)
	if out := r.xObjectsGroups[xObjRef]; isRef && out != nil {
		return out, nil
	}

	// some PDF have Object form with ressources pointing to themselves
	// thus, we first register the current object
	out := new(model.XObjectTransparencyGroup)
	if isRef {
		r.xObjectsGroups[xObjRef] = out
	}

	err := r.resolveXFormObjectFields(obj, &out.XObjectForm)
	if err != nil {
		return nil, err
	}
	// here we known resolved obj is a valid StreamDict
	gDict := r.resolve(obj).(model.ObjStream).Args
	group, _ := r.resolve(gDict["Group"]).(model.ObjDict)
	out.CS, err = r.resolveOneColorSpace(group["CS"])
	if err != nil {
		return out, err
	}
	out.I, _ = r.resolveBool(group["I"])
	out.K, _ = r.resolveBool(group["K"])

	return out, nil
}

// The value of this entry shall be a dictionary in which
// each key is a destination name and the corresponding value is either an array defining the destination, using
// the syntax shown in Table 151, or a dictionary with a D entry whose value is such an array.
func (r resolver) resolveOneNamedDest(dest model.Object) (model.DestinationExplicit, error) {
	dest = r.resolve(dest)
	switch dest := dest.(type) {
	case model.ObjArray:
		return r.resolveExplicitDestination(dest)
	case model.ObjDict:
		D, isArray := r.resolveArray(dest["D"])
		if !isArray {
			return nil, errType("(Dests value).D", dest["D"])
		}
		return r.resolveExplicitDestination(D)
	default:
		return nil, errType("Dests value", dest)
	}
}

// the entry is a simple dictonnary
// In PDF 1.1, the correspondence between name objects and destinations shall be defined by the Dests entry in
// the document catalogue (see 7.7.2, “Document Catalog”).
func (r resolver) processDictDests(entry model.Object) (model.DestTree, error) {
	entry = r.resolve(entry)
	if entry == nil {
		return model.DestTree{}, nil
	}
	nameDict, isDict := entry.(model.ObjDict)
	if !isDict {
		return model.DestTree{}, errType("Dests", entry)
	}
	var out model.DestTree
	for name, dest := range nameDict {
		expDest, err := r.resolveOneNamedDest(dest)
		if err != nil {
			return model.DestTree{}, err
		}
		out.Names = append(out.Names, model.NameToDest{Name: model.DestinationString(name), Destination: expDest})
	}
	return out, nil
}

func (r resolver) processNameDict(entry model.Object) (model.NameDictionary, error) {
	var out model.NameDictionary

	dict, _ := r.resolve(entry).(model.ObjDict)
	if tree := dict["Dests"]; tree != nil {
		err := r.resolveNameTree(tree, destNameTree{out: &out.Dests})
		if err != nil {
			return out, err
		}
	}

	if tree := dict["EmbeddedFiles"]; tree != nil {
		err := r.resolveNameTree(tree, embFileNameTree{out: &out.EmbeddedFiles})
		if err != nil {
			return out, err
		}
	}

	if tree := dict["AP"]; tree != nil {
		err := r.resolveNameTree(tree, appearanceNameTree{out: &out.AP})
		if err != nil {
			return out, err
		}
	}

	if tree := dict["Pages"]; tree != nil {
		err := r.resolveNameTree(tree, templatesNameTree{out: &out.Pages})
		if err != nil {
			return out, err
		}
	}

	if tree := dict["Templates"]; tree != nil {
		err := r.resolveNameTree(tree, templatesNameTree{out: &out.Templates})
		if err != nil {
			return out, err
		}
	}

	// TODO: other names
	return out, nil
}

func (r resolver) resolveViewerPreferences(entry model.Object) (*model.ViewerPreferences, error) {
	entry = r.resolve(entry)
	if entry == nil {
		return nil, nil
	}
	dict, ok := entry.(model.ObjDict)
	if !ok {
		return nil, errType("ViewerPreferences", entry)
	}
	var out model.ViewerPreferences
	if ft, ok := r.resolveBool(dict["FitWindow"]); ok {
		out.FitWindow = bool(ft)
	}
	if ct, ok := r.resolveBool(dict["CenterWindow"]); ok {
		out.CenterWindow = bool(ct)
	}
	if ct, _ := r.resolveName(dict["Direction"]); ct == "R2L" {
		out.DirectionRTL = true
	}
	return &out, nil
}

func (r resolver) resolveOutline(entry model.Object) (*model.Outline, error) {
	entry = r.resolve(entry)
	if entry == nil {
		return nil, nil
	}
	dict, ok := entry.(model.ObjDict)
	if !ok {
		return nil, errType("Outlines", entry)
	}
	var (
		out model.Outline
		err error
	)
	out.First, err = r.resolveOutlineItem(dict["First"], &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (r resolver) resolveOutlineItem(object model.Object, parent model.OutlineNode) (*model.OutlineItem, error) {
	object = r.resolve(object)
	dict, ok := object.(model.ObjDict)
	if !ok {
		return nil, errType("Outline item", object)
	}
	var (
		out model.OutlineItem
		err error
	)
	title, _ := file.IsString(r.resolve(dict["Title"]))
	out.Title = DecodeTextString(title)
	out.Parent = parent
	if first := dict["First"]; first != nil {
		out.First, err = r.resolveOutlineItem(dict["First"], &out)
		if err != nil {
			return nil, err
		}
	}
	if next := dict["Next"]; next != nil {
		out.Next, err = r.resolveOutlineItem(dict["Next"], parent)
		if err != nil {
			return nil, err
		}
	}
	if c, _ := r.resolveInt(dict["Count"]); c >= 0 {
		out.Open = true
	}
	if dest := r.resolve(dict["Dest"]); dest != nil {
		out.Dest, err = r.processDestination(dest)
		if err != nil {
			return nil, err
		}
	} else if action, _ := r.resolve(dict["Action"]).(model.ObjDict); action != nil {
		out.A, err = r.processAction(action)
		if err != nil {
			return nil, err
		}
	}
	// TODO: SE entry (structure hierarchy)
	if c, _ := r.resolveArray(dict["C"]); len(c) == 3 {
		out.C[0], _ = r.resolveNumber(c[0])
		out.C[1], _ = r.resolveNumber(c[1])
		out.C[2], _ = r.resolveNumber(c[2])
	}
	if f, ok := r.resolveInt(dict["F"]); ok {
		out.F = model.OutlineFlag(f)
	}
	return &out, nil
}

func (r resolver) resolveMarkDict(object model.Object) (*model.MarkDict, error) {
	object = r.resolve(object)
	if object == nil {
		return nil, nil
	}
	dict, ok := object.(model.ObjDict)
	if !ok {
		return nil, errType("MarkInfo", object)
	}
	var out model.MarkDict
	out.Marked, _ = r.resolveBool(dict["Marked"])
	out.UserProperties, _ = r.resolveBool(dict["UserProperties"])
	out.Suspects, _ = r.resolveBool(dict["Suspects"])
	return &out, nil
}

func (r resolver) resolveDestinationOrAction(object model.Object) (model.Action, error) {
	object = r.resolve(object)
	switch object := object.(type) {
	case model.ObjArray: // explicit destination
		dest, err := r.resolveExplicitDestination(object)
		if err != nil {
			return model.Action{}, err
		}
		// we see simple destination as GoTo actions
		return model.Action{ActionType: model.ActionGoTo{D: dest}}, nil
	case model.ObjDict:
		return r.processAction(object)
	}
	return model.Action{}, nil
}

func (r resolver) resolveDests(object model.Object) (map[model.ObjName]model.DestinationExplicit, error) {
	dict, _ := r.resolve(object).(model.ObjDict)
	out := make(map[model.ObjName]model.DestinationExplicit, len(dict))
	for name, dest := range dict {
		if ar, ok := r.resolveArray(dest); ok {
			exp, err := r.resolveExplicitDestination(ar)
			if err != nil {
				return nil, err
			}
			out[model.ObjName(name)] = exp
		}
	}
	return out, nil
}
