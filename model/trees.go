package model

import (
	"sort"
)

// generic code for limits

type numTree interface {
	nums() []int
	kids() []numTree
	Limits() [2]int
}

func limitsNum(n numTree) [2]int {
	kids := n.kids()
	if len(kids) == 0 { // leaf node
		nums := n.nums()
		if len(nums) == 0 {
			return [2]int{}
		}
		sort.Ints(nums)
		return [2]int{nums[0], nums[len(nums)-1]}
	}
	limits := kids[0].Limits()
	for _, kid := range kids[1:] {
		kidLimits := kid.Limits()
		if kidLimits[0] < limits[0] {
			limits[0] = kidLimits[0]
		}
		if kidLimits[1] > limits[1] {
			limits[1] = kidLimits[1]
		}
	}
	return limits
}

type nameTree interface {
	names() []string
	kids() []nameTree
	Limits() [2]string
}

func limitsName(n nameTree) [2]string {
	kids := n.kids()
	if len(kids) == 0 { // leaf node
		names := n.names()
		if len(names) == 0 {
			return [2]string{}
		}
		sort.Strings(names)
		return [2]string{names[0], names[len(names)-1]}
	}
	limits := kids[0].Limits()
	for _, kid := range kids[1:] {
		kidLimits := kid.Limits()
		if kidLimits[0] < limits[0] {
			limits[0] = kidLimits[0]
		}
		if kidLimits[1] > limits[1] {
			limits[1] = kidLimits[1]
		}
	}
	return limits
}

// NameToDest associate an explicit destination
// to a name.
type NameToDest struct {
	Name        string
	Destination *ExplicitDestination // indirect object
}

// DestTree links a serie of arbitrary name
// to explicit destination, enabling `NamedDestination`
// to reference them
type DestTree struct {
	Kids  []DestTree
	Names []NameToDest
}

func (d DestTree) names() []string {
	out := make([]string, len(d.Names))
	for i, k := range d.Names {
		out[i] = k.Name
	}
	return out
}

func (d DestTree) kids() []nameTree {
	out := make([]nameTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

// Limits specify the (lexically) least and greatest keys included in the Names array of
// a leaf node or in the Names arrays of any leaf nodes that are descendants of an
// intermediate node.
func (d DestTree) Limits() [2]string {
	return limitsName(d)
}

// LookupTable walks the name tree and
// accumulates the result into one map
func (d DestTree) LookupTable() map[string]*ExplicitDestination {
	out := make(map[string]*ExplicitDestination)
	for _, v := range d.Names {
		out[v.Name] = v.Destination
	}
	for _, kid := range d.Kids {
		for name, dest := range kid.LookupTable() {
			out[name] = dest
		}
	}
	return out
}

//TODO: dest tree
func (p DestTree) pdfString(pdf PDFWriter) string {
	return "<<>>"
}

// ----------------------------------------------------------------------

type NameToFile struct {
	Name     string
	FileSpec *FileSpec // indirect object
}

// EmbeddedFileTree is written as a Name Tree in PDF,
// but, since it generally won't be big, is
// represented here as a flat list.
type EmbeddedFileTree []NameToFile

func (d EmbeddedFileTree) names() []string {
	out := make([]string, len(d))
	for i, k := range d {
		out[i] = k.Name
	}
	return out
}

func (d EmbeddedFileTree) kids() []nameTree {
	return nil
}

func (efs EmbeddedFileTree) Limits() [2]string {
	return limitsName(efs)
}

// TODO: EmbeddedFileTree
func (p EmbeddedFileTree) pdfString(pdf PDFWriter) string {
	return "<<>>"
}

// -----------------------------------------------------------------------

type PageLabel struct {
	S  Name
	P  string
	St int // optionnal default to 1
}

type NumToPageLabel struct {
	Num       int
	PageLabel PageLabel // rather a direct object
}

type PageLabelsTree struct {
	Kids []PageLabelsTree
	Nums []NumToPageLabel
}

func (d PageLabelsTree) nums() []int {
	out := make([]int, len(d.Nums))
	for i, k := range d.Nums {
		out[i] = k.Num
	}
	return out
}

func (d PageLabelsTree) kids() []numTree {
	out := make([]numTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

// Limits specify the (numerically) least and greatest keys included in
// the Nums array of a leaf node or in the Nums arrays of any leaf nodes that are
// descendants of an intermediate node.
func (d PageLabelsTree) Limits() [2]int {
	return limitsNum(d)
}

// LookupTable walks the number tree and
// accumulates the result into one map
func (d PageLabelsTree) LookupTable() map[int]PageLabel {
	out := make(map[int]PageLabel)
	for _, v := range d.Nums {
		out[v.Num] = v.PageLabel
	}
	for _, kid := range d.Kids {
		for name, dest := range kid.LookupTable() {
			out[name] = dest
		}
	}
	return out
}

// TODO: PageLabelsTree
func (p PageLabelsTree) pdfString(pdf PDFWriter) string {
	return "<<>>"
}

// ------------------------------------------------------------
type NameToStructureElement struct {
	Name      string
	Structure *StructureElement
}

type IDTree struct {
	Kids  []IDTree
	Names []NameToStructureElement
}

func (d IDTree) names() []string {
	out := make([]string, len(d.Names))
	for i, k := range d.Names {
		out[i] = k.Name
	}
	return out
}

func (d IDTree) kids() []nameTree {
	out := make([]nameTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

// Limits specify the (lexically) least and greatest keys included in the Names array of
// a leaf node or in the Names arrays of any leaf nodes that are descendants of an
// intermediate node.
func (d IDTree) Limits() [2]string {
	return limitsName(d)
}

type NumToParent struct {
	Num    int
	Parent []*StructureElement // 1-element array may be written directly in PDF
}

type ParentTree struct {
	Kids []ParentTree
	Nums []NumToParent
}

func (d ParentTree) nums() []int {
	out := make([]int, len(d.Nums))
	for i, k := range d.Nums {
		out[i] = k.Num
	}
	return out
}

func (d ParentTree) kids() []numTree {
	out := make([]numTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

func (d ParentTree) Limits() [2]int {
	return limitsNum(d)
}
