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
	setLimits(limits [2]string)
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

	limits := dict.ArrayEntry("Limits")
	if len(limits) == 2 {
		low, _ := limits[0].(pdfcpu.StringLiteral)
		high, _ := limits[1].(pdfcpu.StringLiteral)
		output.setLimits([2]string{decodeStringLit(low), decodeStringLit(high)})
	}

	if kids := dict.ArrayEntry("Kids"); kids != nil {
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
	names := dict.ArrayEntry("Names")
	L := len(names)
	if L%2 != 0 {
		return fmt.Errorf("expected even length array in name tree, got %s", names)
	}
	for l := 0; l < L/2; l++ {
		name, _ := names[2*l].(pdfcpu.StringLiteral)
		value := names[2*l+1]
		err := output.resolveLeafValueAppend(r, decodeStringLit(name), value)
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
	d.out.Kids = append(d.out.Kids, kid.(destNameTree).out)
}
func (d destNameTree) setLimits(limits [2]string) {
	d.out.Limits = limits
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
func (d embFileNameTree) setLimits(limits [2]string) {} // no op
func (d embFileNameTree) resolveLeafValueAppend(r *resolver, name string, value pdfcpu.Object) error {
	fileSpec, err := r.resolveFileSpec(value)
	*d.out = append(*d.out, model.NameToFile{Name: name, FileSpec: fileSpec})
	return err
}
