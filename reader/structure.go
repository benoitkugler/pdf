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
