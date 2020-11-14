package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// name-tree like items
type nameTree interface {
	createKid() nameTree
	appendKid(kid nameTree) // kid will be the value returned by createKid
	// must handle the case where `value` is indirect
	resolveLeafValueAppend(r *resolver, name string, value pdfcpu.Object) error
}

// resolveNameTree is a "generic function" which walk a name tree
// and fill the given output
func (r *resolver) resolveNameTree(entry pdfcpu.Object, output nameTree) error {
	entry = r.resolve(entry)
	dict, isDict := entry.(pdfcpu.Dict)
	if !isDict {
		return errType("Name Tree value", entry)
	}

	if kids, _ := r.resolve(dict["Kids"]).(pdfcpu.Array); kids != nil {
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
	names, _ := r.resolve(dict["Names"]).(pdfcpu.Array)
	L := len(names)
	if L%2 != 0 {
		return fmt.Errorf("expected even length array in name tree, got %s", names)
	}
	for l := 0; l < L/2; l++ {
		name, _ := isString(r.resolve(names[2*l]))
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
func (d destNameTree) resolveLeafValueAppend(r *resolver, name string, value pdfcpu.Object) error {
	expDest, err := r.resolveOneNamedDest(value)
	d.out.Names = append(d.out.Names, model.NameToDest{Name: name, Destination: expDest})
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
func (d embFileNameTree) resolveLeafValueAppend(r *resolver, name string, value pdfcpu.Object) error {
	fileSpec, err := r.resolveFileSpec(value)
	*d.out = append(*d.out, model.NameToFile{Name: name, FileSpec: fileSpec})
	return err
}

// number tree

func (r *resolver) resolvePageLabelsTree(entry pdfcpu.Object, output *model.PageLabelsTree) error {
	entry = r.resolve(entry)
	dict, isDict := entry.(pdfcpu.Dict)
	if !isDict {
		return errType("Name Tree value", entry)
	}

	// limits is inferred from the content

	if kids, _ := r.resolve(dict["Kids"]).(pdfcpu.Array); kids != nil {
		// intermediate node
		// one node shouldn't be refered twice,
		// dont bother tracking ref
		for _, kid := range kids {
			kidModel := new(model.PageLabelsTree)
			err := r.resolvePageLabelsTree(kid, kidModel)
			if err != nil {
				return err
			}
			output.Kids = append(output.Kids, *kidModel)
		}
		return nil
	}

	// leaf node
	nums, _ := r.resolve(dict["Nums"]).(pdfcpu.Array)
	L := len(nums)
	if L%2 != 0 {
		return fmt.Errorf("expected even length array in number tree, got %s", nums)
	}
	for l := 0; l < L/2; l++ {
		num, _ := r.resolveInt(nums[2*l])
		value := nums[2*l+1]
		pageLabel, err := r.processPageLabel(value)
		if err != nil {
			return err
		}
		output.Nums = append(output.Nums, model.NumToPageLabel{Num: num, PageLabel: pageLabel})
	}
	return nil
}

func (r resolver) processPageLabel(entry pdfcpu.Object) (model.PageLabel, error) {
	entry = r.resolve(entry)
	entryDict, isDict := entry.(pdfcpu.Dict)
	if !isDict {
		return model.PageLabel{}, errType("Page Label", entry)
	}
	var out model.PageLabel
	if s, ok := r.resolveName(entryDict["S"]); ok {
		out.S = s
	}
	p, _ := isString(r.resolve(entryDict["P"]))
	out.P = decodeTextString(p)
	if st := entryDict.IntEntry("St"); st != nil {
		out.St = *st
	}
	return out, nil
}
