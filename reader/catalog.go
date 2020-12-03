package reader

import (
	"errors"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) rectangleFromArray(array pdfcpu.Object) *model.Rectangle {
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

func (r resolver) matrixFromArray(array pdfcpu.Object) *model.Matrix {
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

func (r resolver) resolveAppearanceDict(o pdfcpu.Object) (*model.AppearanceDict, error) {
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		if ff := r.appearanceDicts[ref]; ff != nil {
			return ff, nil
		}
		o = r.resolve(ref)
	}
	if o == nil {
		return nil, nil
	}
	a, isDict := o.(pdfcpu.Dict)
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

func (r resolver) resolveAppearanceEntry(obj pdfcpu.Object) (model.AppearanceEntry, error) {
	out := make(model.AppearanceEntry)

	// obj might be either a subdictionary or a streamdictionary
	if subDict, isResolvedDict := r.resolve(obj).(pdfcpu.Dict); isResolvedDict {
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
func (r resolver) resolveOneXObjectForm(obj pdfcpu.Object) (*model.XObjectForm, error) {
	xObjRef, isRef := obj.(pdfcpu.IndirectRef)
	if out := r.xObjectForms[xObjRef]; isRef && out != nil {
		return out, nil
	}
	obj = r.resolve(obj)
	cs, err := r.resolveStream(obj)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, errors.New("missing Form XObject")
	}
	stream, _ := obj.(pdfcpu.StreamDict) // here, we are sure obj is a stream
	ap := model.XObjectForm{ContentStream: model.ContentStream{Stream: *cs}}
	if rect := r.rectangleFromArray(r.resolve(stream.Dict["BBox"])); rect != nil {
		ap.BBox = *rect
	}
	if mat := r.matrixFromArray(r.resolve(stream.Dict["Matrix"])); mat != nil {
		ap.Matrix = *mat
	}
	if res := stream.Dict["Resources"]; res != nil {
		var err error
		ap.Resources, err = r.resolveOneResourceDict(res)
		if err != nil {
			return nil, err
		}
	}
	if st, ok := r.resolveInt(stream.Dict["StructParent"]); ok {
		ap.StructParent = model.ObjInt(st)
	} else if st, ok := r.resolveInt(stream.Dict["StructParents"]); ok {
		ap.StructParents = model.ObjInt(st)
	}

	if isRef {
		r.xObjectForms[xObjRef] = &ap
	}
	return &ap, nil
}

// The value of this entry shall be a dictionary in which
// each key is a destination name and the corresponding value is either an array defining the destination, using
// the syntax shown in Table 151, or a dictionary with a D entry whose value is such an array.
func (r resolver) resolveOneNamedDest(dest pdfcpu.Object) (model.DestinationExplicit, error) {
	dest = r.resolve(dest)
	switch dest := dest.(type) {
	case pdfcpu.Array:
		return r.resolveExplicitDestination(dest)
	case pdfcpu.Dict:
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
func (r resolver) processDictDests(entry pdfcpu.Object) (model.DestTree, error) {
	entry = r.resolve(entry)
	if entry == nil {
		return model.DestTree{}, nil
	}
	nameDict, isDict := entry.(pdfcpu.Dict)
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

func (r resolver) processNameDict(entry pdfcpu.Object) (model.NameDictionary, error) {
	var out model.NameDictionary

	dict, _ := r.resolve(entry).(pdfcpu.Dict)
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

	// TODO: other names
	return out, nil
}

func (r resolver) resolveViewerPreferences(entry pdfcpu.Object) (*model.ViewerPreferences, error) {
	entry = r.resolve(entry)
	if entry == nil {
		return nil, nil
	}
	dict, ok := entry.(pdfcpu.Dict)
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

func (r resolver) resolveOutline(entry pdfcpu.Object) (*model.Outline, error) {
	entry = r.resolve(entry)
	if entry == nil {
		return nil, nil
	}
	dict, ok := entry.(pdfcpu.Dict)
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

func (r resolver) resolveOutlineItem(object pdfcpu.Object, parent model.OutlineNode) (*model.OutlineItem, error) {
	object = r.resolve(object)
	dict, ok := object.(pdfcpu.Dict)
	if !ok {
		return nil, errType("Outline item", object)
	}
	var (
		out model.OutlineItem
		err error
	)
	title, _ := isString(r.resolve(dict["Title"]))
	out.Title = decodeTextString(title)
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
	} else if action, _ := r.resolve(dict["Action"]).(pdfcpu.Dict); action != nil {
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

func (r resolver) resolveMarkDict(object pdfcpu.Object) (*model.MarkDict, error) {
	object = r.resolve(object)
	if object == nil {
		return nil, nil
	}
	dict, ok := object.(pdfcpu.Dict)
	if !ok {
		return nil, errType("MarkInfo", object)
	}
	var out model.MarkDict
	out.Marked, _ = r.resolveBool(dict["Marked"])
	out.UserProperties, _ = r.resolveBool(dict["UserProperties"])
	out.Suspects, _ = r.resolveBool(dict["Suspects"])
	return &out, nil
}

func (r resolver) resolveDestinationOrAction(object pdfcpu.Object) (model.Action, error) {
	object = r.resolve(object)
	switch object := object.(type) {
	case pdfcpu.Array: // explicit destination
		dest, err := r.resolveExplicitDestination(object)
		if err != nil {
			return model.Action{}, err
		}
		// we see simple destination as GoTo actions
		return model.Action{ActionType: model.ActionGoTo{D: dest}}, nil
	case pdfcpu.Dict:
		return r.processAction(object)
	}
	return model.Action{}, nil
}

func (r resolver) resolveDests(object pdfcpu.Object) (map[model.ObjName]model.DestinationExplicit, error) {
	dict, _ := r.resolve(object).(pdfcpu.Dict)
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
