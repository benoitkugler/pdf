package reader

import (
	"fmt"
	"log"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/model/values"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveStructureTree(obj pdfcpu.Object) (*model.StructureTree, error) {
	obj = r.resolve(obj)
	if obj == nil {
		return nil, nil
	}
	structDict, ok := obj.(pdfcpu.Dict)
	if !ok {
		return nil, errType("StructTreeRoot", obj)
	}
	var (
		out model.StructureTree
		err error
	)

	switch K := r.resolve(structDict["K"]).(type) {
	case pdfcpu.Dict: // one child
		elemn, err := r.resolveOneStructureElement(K, nil)
		if err != nil {
			return nil, err
		}
		out.K = []*model.StructureElement{elemn}
	case pdfcpu.Array: // many
		out.K = make([]*model.StructureElement, len(K))
		for i, v := range K {
			out.K[i], err = r.resolveOneStructureElement(v, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	// the structure tree need to be resolved first
	out.IDTree, err = r.resolveIDTree(structDict["IDTree"])
	if err != nil {
		return nil, err
	}

	out.ParentTree, err = r.resolveParentTree(structDict["ParentTree"])
	if err != nil {
		return nil, err
	}

	roles, _ := r.resolve(structDict["RoleMap"]).(pdfcpu.Dict)
	out.RoleMap = make(map[model.Name]model.Name, len(roles))
	for k, v := range roles {
		out.RoleMap[model.Name(k)], _ = r.resolveName(v)
	}

	class, _ := r.resolve(structDict["ClassMap"]).(pdfcpu.Dict)
	out.ClassMap = make(map[model.Name][]model.AttributeObject, len(class))
	for k, v := range class {
		var attrs []model.AttributeObject
		switch v := r.resolve(v).(type) {
		case pdfcpu.Array: // many attributes, with potential revision numbers
			attrs, err = r.resolveAttributeObjects(v)
			if err != nil {
				return nil, err
			}
		case pdfcpu.Dict: // only one attribute
			a, err := r.resolveAttributObject(v)
			if err != nil {
				return nil, err
			}
			attrs = []model.AttributeObject{a}
		case pdfcpu.StreamDict:
			log.Println("unsupported attribute type : stream. skipping")
		default:
			return nil, errType("structure Attribute", v)
		}
		out.ClassMap[model.Name(k)] = attrs
	}

	return &out, nil
}

// resolve indirect object and map pdfcpu types to model/values types
func (r resolver) processCustomObject(object pdfcpu.Object) (model.UPValue, error) {
	var err error
	object = r.resolve(object)
	switch o := object.(type) {
	case pdfcpu.Name:
		return values.Name(o), nil
	case pdfcpu.Float:
		return values.Float(o), nil
	case pdfcpu.Integer:
		return values.Int(o), nil
	case pdfcpu.Boolean:
		return values.Bool(o), nil
	case pdfcpu.Array:
		out := make(values.Array, len(o))
		for i, v := range o {
			out[i], err = r.processCustomObject(v)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case pdfcpu.Dict:
		out := make(values.Dict, len(o))
		for name, v := range o {
			out[values.Name(name)], err = r.processCustomObject(v)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	// simplify strings
	if s, ok := isString(object); ok {
		s = decodeTextString(s)
		return values.String(s), nil
	}
	return nil, fmt.Errorf("unsupported custom type %T in custom object", object)
}

func (r resolver) resolveAttributObject(attr pdfcpu.Object) (out model.AttributeObject, err error) {
	attr = r.resolve(attr)
	attrDict, ok := attr.(pdfcpu.Dict)
	if !ok {
		return model.AttributeObject{}, errType("Attribute Object", attr)
	}
	out.O, _ = r.resolveName(attrDict["O"])
	if out.O == "UserProperties" { // special case
		err = r.resolveUserProperties(attrDict, &out)
		if err != nil {
			return out, err
		}
	} else {
		atts := make(map[model.Name]model.Attribute)
		for name, v := range attrDict {
			if name == "O" { // already handled
				continue
			}
			atts[model.Name(name)], err = r.processCustomObject(v)
			if err != nil {
				return out, err
			}
		}
		out.Attributes = atts
	}
	return out, nil
}

func (r resolver) resolveAttributeObjects(ar pdfcpu.Array) ([]model.AttributeObject, error) {
	// many attributes, with potential revision numbers
	// the minimum number of attributes is len(ar) /2, if all items have a revision number
	attrs := make([]model.AttributeObject, 0, len(ar)/2)
	for i := 0; i < len(ar); {
		// the first element of the potential pair must be an attribute
		att, err := r.resolveAttributObject(ar[i])
		if err != nil {
			return nil, err
		}
		// look ahead for a potential revision number
		if i+1 < len(ar) {
			if rev, ok := r.resolve(ar[i+1]).(pdfcpu.Integer); ok {
				att.RevisionNumber = int(rev)
				i++ // now, skip the revision number
			}
		}
		i++
		attrs = append(attrs, att)
	}
	return attrs, nil
}

func (r resolver) resolveUserProperties(dict pdfcpu.Dict, out *model.AttributeObject) (err error) {
	ps, _ := r.resolveArray(dict["P"])
	attr := make(model.AttributeUserProperties, len(ps))
	for i, prop := range ps {
		prop = r.resolve(prop)
		propDict, ok := prop.(pdfcpu.Dict)
		if !ok {
			return errType("UserProperty", prop)
		}
		attr[i].F, _ = isString(r.resolve(propDict["F"]))
		attr[i].N, _ = isString(r.resolve(propDict["N"]))
		attr[i].H, _ = r.resolveBool(propDict["H"])
		attr[i].V, err = r.processCustomObject(propDict["V"])
		if err != nil {
			return err
		}
	}
	out.Attributes = map[model.Name]model.Attribute{"P": attr}
	return nil
}

func (r resolver) resolveOneStructureElement(element pdfcpu.Object, parent *model.StructureElement) (*model.StructureElement, error) {
	ref, isRef := element.(pdfcpu.IndirectRef)
	if out := r.structure[ref]; isRef && out != nil {
		return out, nil
	}

	element = r.resolve(element)
	dict, ok := element.(pdfcpu.Dict)
	if !ok {
		return nil, errType("Structure element", element)
	}
	var (
		out model.StructureElement
		err error
	)

	if isRef { // register the structure element
		r.structure[ref] = &out
	}

	out.P = parent
	out.S, _ = r.resolveName(dict["S"])
	out.ID, _ = isString(dict["ID"])
	if pageRef, ok := dict["Pg"].(pdfcpu.IndirectRef); ok { // if it's not a ref, ignore it
		out.Pg = r.pages[pageRef]
	}

	// K entry is either and array or a dict
	kid := dict["K"]
	if array, ok := r.resolve(kid).(pdfcpu.Array); ok {
		out.K = make([]model.ContentItem, len(array))
		for i, kid := range array {
			out.K[i], err = r.resolveContentItem(kid, &out)
			if err != nil {
				return nil, err
			}
		}
	} else if kid != nil {
		kid, err := r.resolveContentItem(kid, &out) // dont resolve to keep track of the potential indirect object
		if err != nil {
			return nil, err
		}
		out.K = []model.ContentItem{kid}
	}

	switch a := r.resolve(dict["A"]).(type) {
	case pdfcpu.Dict: // one attribute
		att, err := r.resolveAttributObject(a)
		if err != nil {
			return nil, err
		}
		out.A = []model.AttributeObject{att}
	case pdfcpu.Array: // many attributes, with possible revision number
		out.A, err = r.resolveAttributeObjects(a)
		if err != nil {
			return nil, err
		}
	} // nil or invalid, ignore

	switch c := r.resolve(dict["C"]).(type) {
	case pdfcpu.Name: // only one class
		out.C = []model.ClassName{{Name: model.Name(c)}}
	case pdfcpu.Array: // many class, with potential revision number
		// the minimum number of classes is len(ar) /2, if all items have a revision number
		out.C = make([]model.ClassName, 0, len(c)/2)
		for i := 0; i < len(c); {
			// the first element of the potential pair must be an attribute
			name, ok := r.resolveName(c[i])
			if !ok {
				return nil, errType("Class Name for structure element", c[i])
			}
			className := model.ClassName{Name: name}
			// look ahead for a potential revision number
			if i+1 < len(c) {
				if rev, ok := r.resolve(c[i+1]).(pdfcpu.Integer); ok {
					className.RevisionNumber = int(rev)
					i++ // now, skip the revision number
				}
			}
			i++
			out.C = append(out.C, className)
		}
	}

	out.R, _ = r.resolveInt(dict["R"])
	if s, ok := isString(r.resolve(dict["T"])); ok {
		out.T = decodeTextString(s)
	}
	if s, ok := isString(r.resolve(dict["Lang"])); ok {
		out.Lang = decodeTextString(s)
	}
	if s, ok := isString(r.resolve(dict["Alt"])); ok {
		out.Alt = decodeTextString(s)
	}
	if s, ok := isString(r.resolve(dict["E"])); ok {
		out.E = decodeTextString(s)
	}
	if s, ok := isString(r.resolve(dict["ActualText"])); ok {
		out.ActualText = decodeTextString(s)
	}

	return &out, nil
}

func (r resolver) resolveContentItem(object pdfcpu.Object, parent *model.StructureElement) (model.ContentItem, error) {
	resolved := r.resolve(object)                 // keep the potential indirect object
	if mci, ok := resolved.(pdfcpu.Integer); ok { // integer marked-content identifier denoting a marked-content sequence
		return model.ContentItemMarkedReference{MCID: int(mci)}, nil
	}
	// now, must be a dict
	contentDict, ok := resolved.(pdfcpu.Dict)
	if !ok {
		return nil, errType("Content Item", resolved)
	}
	typeName, _ := r.resolveName(contentDict["Type"])
	switch typeName {
	case "OBJR":
		return r.resolveObjectReference(contentDict)
	case "MCR":
		return r.resolveMarkedReference(contentDict), nil
	default:
		// If the value of K is a dictionary containing no Type entry,
		// it shall be assumed to be a structure element dictionary
		return r.resolveOneStructureElement(object, parent)
	}
}

func (r resolver) resolveObjectReference(dict pdfcpu.Dict) (out model.ContentItemObjectReference, err error) {
	if pageRef, ok := dict["Pg"].(pdfcpu.IndirectRef); ok {
		out.Pg = r.pages[pageRef]
	}
	objRef, ok := dict["Obj"].(pdfcpu.IndirectRef)
	if !ok {
		return out, errType("Obj entry in object reference", dict["Obj"])
	}
	if annot := r.annotations[objRef]; annot != nil {
		out.Obj = annot
	} else if form := r.xObjectForms[objRef]; form != nil {
		out.Obj = form
	} else if img := r.images[objRef]; img != nil {
		out.Obj = img
	} else { // invalid reference
		return out, fmt.Errorf("invalid type for object reference : %v", r.resolve(objRef))
	}
	return out, nil
}

func (r resolver) resolveMarkedReference(dict pdfcpu.Dict) (out model.ContentItemMarkedReference) {
	out.MCID, _ = r.resolveInt(dict["MCID"])
	if pageRef, isRef := dict["Pg"].(pdfcpu.IndirectRef); isRef {
		out.Container = r.pages[pageRef]
	} else if formRef, isRef := dict["Stm"].(pdfcpu.IndirectRef); isRef {
		out.Container = r.xObjectForms[formRef]
	}
	return out
}

func (r resolver) resolveIDTree(tree pdfcpu.Object) (model.IDTree, error) {
	var out model.IDTree
	err := r.resolveNameTree(tree, idTree{out: &out})
	return out, err
}

func (r resolver) resolveParentTree(tree pdfcpu.Object) (model.ParentTree, error) {
	var out model.ParentTree
	err := r.resolveNumberTree(tree, parentTree{out: &out})
	return out, err
}
