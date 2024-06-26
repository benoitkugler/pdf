package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
)

// name-tree like items
type nameTree interface {
	createKid() nameTree
	appendKid(kid nameTree) // kid will be the value returned by createKid
	// must handle the case where `value` is indirect
	// also, value may be 'null'
	resolveLeafValueAppend(r resolver, name string, value model.Object) error
}

// resolveNameTree is a "generic function" which walk a name tree
// and fill the given output
// if entry is (or resolve) to nil, return early
func (r resolver) resolveNameTree(entry model.Object, output nameTree) error {
	entry = r.resolve(entry)
	if entry == nil {
		return nil
	}
	dict, isDict := entry.(model.ObjDict)
	if !isDict {
		return errType("Name Tree value", entry)
	}

	if kids, _ := r.resolveArray(dict["Kids"]); kids != nil {
		// intermediate node
		// one node shouldn't be refered twice,
		// dont bother tracking ref
		for _, kid := range kids {
			kidModel := output.createKid()
			err := r.resolveNameTree(kid, kidModel)
			if err != nil {
				return err
			}
			output.appendKid(kidModel)
		}
		return nil
	}

	// leaf node
	names, _ := r.resolveArray(dict["Names"])
	L := len(names)
	if L%2 != 0 {
		return fmt.Errorf("expected even length array in name tree, got %s", names)
	}
	for l := 0; l < L/2; l++ {
		name, _ := file.IsString(r.resolve(names[2*l]))
		value := names[2*l+1]
		err := output.resolveLeafValueAppend(r, name, value)
		if err != nil {
			return err
		}
	}
	return nil
}

type destNameTree struct {
	out *model.DestTree // target which will be filled
}

func (d destNameTree) createKid() nameTree {
	return destNameTree{out: new(model.DestTree)}
}

func (d destNameTree) appendKid(kid nameTree) {
	d.out.Kids = append(d.out.Kids, *kid.(destNameTree).out)
}

func (d destNameTree) resolveLeafValueAppend(r resolver, name string, value model.Object) error {
	expDest, err := r.resolveOneNamedDest(value)
	d.out.Names = append(d.out.Names, model.NameToDest{Name: model.DestinationString(name), Destination: expDest})
	return err
}

type embFileNameTree struct {
	out *model.EmbeddedFileTree // target which will be filled
}

func (d embFileNameTree) createKid() nameTree {
	return embFileNameTree{out: new(model.EmbeddedFileTree)}
}

func (d embFileNameTree) appendKid(kid nameTree) {
	// we choose to flatten in the current node
	values := *kid.(embFileNameTree).out
	*d.out = append(*d.out, values...)
}

func (d embFileNameTree) resolveLeafValueAppend(r resolver, name string, value model.Object) error {
	fileSpec, err := r.resolveFileSpec(value)
	*d.out = append(*d.out, model.NameToFile{Name: name, FileSpec: fileSpec})
	return err
}

type appearanceNameTree struct {
	out *model.AppearanceTree // target which will be filled
}

func (d appearanceNameTree) createKid() nameTree {
	return appearanceNameTree{out: new(model.AppearanceTree)}
}

func (d appearanceNameTree) appendKid(kid nameTree) {
	d.out.Kids = append(d.out.Kids, *kid.(appearanceNameTree).out)
}

func (d appearanceNameTree) resolveLeafValueAppend(r resolver, name string, value model.Object) error {
	// Some trees may have a key but a null value (see issue #7) :
	// simply ignore thoose
	if value == (model.ObjNull{}) {
		return nil
	}
	form, err := r.resolveOneXObjectForm(value)
	d.out.Names = append(d.out.Names, model.NameToAppearance{Name: name, Appearance: form})
	return err
}

type idTree struct {
	out *model.IDTree // target which will be filled
}

func (d idTree) createKid() nameTree {
	return idTree{out: new(model.IDTree)}
}

func (d idTree) appendKid(kid nameTree) {
	d.out.Kids = append(d.out.Kids, *kid.(idTree).out)
}

func (d idTree) resolveLeafValueAppend(r resolver, name string, value model.Object) error {
	ref, isRef := value.(model.ObjIndirectRef)
	if !isRef {
		return errType("IDTree value", value)
	}
	st := r.structure[ref]
	d.out.Names = append(d.out.Names, model.NameToStructureElement{Name: name, Structure: st})
	return nil
}

// number-tree like items
type numberTree interface {
	createKid() numberTree
	appendKid(kid numberTree) // kid will be the value returned by createKid
	// must handle the case where `value` is indirect
	resolveLeafValueAppend(r resolver, number int, value model.Object) error
}

// resolveNumberTree is a "generic function" which walk a number tree
// and fill the given output
func (r resolver) resolveNumberTree(entry model.Object, output numberTree) error {
	entry = r.resolve(entry)
	dict, isDict := entry.(model.ObjDict)
	if !isDict {
		return errType("Number Tree value", entry)
	}

	if kids, _ := r.resolveArray(dict["Kids"]); kids != nil {
		// intermediate node
		// one node shouldn't be refered twice,
		// so dont bother tracking ref
		for _, kid := range kids {
			kidModel := output.createKid()
			err := r.resolveNumberTree(kid, kidModel)
			if err != nil {
				return err
			}
			output.appendKid(kidModel)
		}
		return nil
	}

	// leaf node
	nums, _ := r.resolveArray(dict["Nums"])
	L := len(nums)
	if L%2 != 0 {
		return fmt.Errorf("expected even length array in number tree, got %s", nums)
	}
	for l := 0; l < L/2; l++ {
		number, _ := r.resolveInt(nums[2*l])
		value := nums[2*l+1]
		err := output.resolveLeafValueAppend(r, number, value)
		if err != nil {
			return err
		}
	}
	return nil
}

type pageLabelTree struct {
	out *model.PageLabelsTree // will be filled
}

func (d pageLabelTree) createKid() numberTree {
	return pageLabelTree{out: new(model.PageLabelsTree)}
}

func (d pageLabelTree) appendKid(kid numberTree) {
	d.out.Kids = append(d.out.Kids, *kid.(pageLabelTree).out)
}

func (d pageLabelTree) resolveLeafValueAppend(r resolver, number int, value model.Object) error {
	label, err := r.processPageLabel(value)
	d.out.Nums = append(d.out.Nums, model.NumToPageLabel{Num: number, PageLabel: label})
	return err
}

func (r resolver) processPageLabel(entry model.Object) (model.PageLabel, error) {
	entry = r.resolve(entry)
	entryDict, isDict := entry.(model.ObjDict)
	if !isDict {
		return model.PageLabel{}, errType("Page Label", entry)
	}
	var out model.PageLabel
	if s, ok := r.resolveName(entryDict["S"]); ok {
		out.S = s
	}
	p, _ := file.IsString(r.resolve(entryDict["P"]))
	out.P = DecodeTextString(p)
	out.St = 1 // default value
	if st, ok := r.resolveInt(entryDict["St"]); ok {
		out.St = st
	}
	return out, nil
}

type parentTree struct {
	out *model.ParentTree // will be filled
}

func (d parentTree) createKid() numberTree {
	return parentTree{out: new(model.ParentTree)}
}

func (d parentTree) appendKid(kid numberTree) {
	d.out.Kids = append(d.out.Kids, *kid.(parentTree).out)
}

func (d parentTree) resolveLeafValueAppend(r resolver, number int, value model.Object) error {
	var parent model.NumToParent
	parent.Num = number
	// value must be either an indirect ref, or a direct array of indirect ref
	if ref, isRef := value.(model.ObjIndirectRef); isRef {
		parent.Parent = r.structure[ref]
	} else if array, ok := value.(model.ObjArray); ok {
		parent.Parents = make([]*model.StructureElement, 0, len(array))
		for _, p := range array {
			ref, ok := p.(model.ObjIndirectRef)
			if !ok { // invalid: ignore
				continue
			}
			parent.Parents = append(parent.Parents, r.structure[ref])
		}
	} else {
		return errType("value in ParentTree", value)
	}
	d.out.Nums = append(d.out.Nums, parent)
	return nil
}

type templatesNameTree struct {
	out *model.TemplateTree // target which will be filled
}

func (d templatesNameTree) createKid() nameTree {
	return templatesNameTree{out: new(model.TemplateTree)}
}

func (d templatesNameTree) appendKid(kid nameTree) {
	d.out.Kids = append(d.out.Kids, *kid.(templatesNameTree).out)
}

func (d templatesNameTree) resolveLeafValueAppend(r resolver, name string, value model.Object) error {
	var page *model.PageObject
	if pageRef, isRef := value.(model.ObjIndirectRef); isRef {
		page = r.pages[pageRef]
	}
	if page == nil { // template -> create a new object
		page = new(model.PageObject)
		pageDict, _ := r.resolve(value).(model.ObjDict)
		err := r.resolvePageObject(pageDict, page)
		if err != nil {
			return err
		}
	}
	d.out.Names = append(d.out.Names, model.NameToPage{Name: name, Page: page})
	return nil
}
