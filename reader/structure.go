package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
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
		elemn, err := r.resolveStructureElement(K)
		if err != nil {
			return nil, err
		}
		out.K = []*model.StructureElement{elemn}
	case pdfcpu.Array: // many
		out.K = make([]*model.StructureElement, len(K))
		for i, v := range K {
			out.K[i], err = r.resolveStructureElement(v)
			if err != nil {
				return nil, err
			}
		}
	}
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
		//TODO: class map
		switch v := v.(type) {
		case pdfcpu.Array: // more than one attribute

		case pdfcpu.IndirectRef: // only one attribute

		default:
			fmt.Println("class map", k, v)

		}
	}
	return &out, nil
}

// implements model.Attribute
// indirefect are resolved when creating
type attribute struct {
	pdfcpu.Object
}

func (a attribute) Clone() model.Attribute {
	return attribute{Object: a.Object.Clone()}
}

// walk and look for string, and return the encoded strings
// avoid to reimplement PDFString from sratch
func (a attribute) encryptStrings(enc model.PDFStringEncoder, context model.Reference) attribute

func (a attribute) PDFString(enc model.PDFStringEncoder, context model.Reference) string {
	switch a := a.Object.(type) {
	case pdfcpu.Array:

	}
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
		for name, v := range attrDict {

		}
	}
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
		switch v := r.resolve(propDict["V"]).(type) {
		case pdfcpu.Float:
			attr[i].V = model.UPNumber(v)
		case pdfcpu.Integer:
			attr[i].V = model.UPNumber(v)
		case pdfcpu.Boolean:
			attr[i].V = model.UPBool(v)
		default:
			vs, _ := isString(v)
			attr[i].V = model.UPString(vs)
		}
	}
	out.Attributes = map[model.Name]model.Attribute{"P": attr}
	return nil
}

// TODO:
func (r resolver) resolveStructureElement(element pdfcpu.Object) (*model.StructureElement, error) {
	return nil, nil
}

// TODO:
func (r resolver) resolveIDTree(tree pdfcpu.Object) (model.IDTree, error) {
	return model.IDTree{}, nil
}

// TODO:
func (r resolver) resolveParentTree(tree pdfcpu.Object) (model.ParentTree, error) {
	return model.ParentTree{}, nil
}
