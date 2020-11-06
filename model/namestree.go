package model

import (
	"sort"
)

// type NameTreeNode struct {
// 	Kids   []*NameTreeNode
// 	Names  []NamedObject
// 	Limits [2]string
// }

// type NameTree struct {
// 	Root NameTreeNode
// }

// type NamedObject struct {
// 	Name   string
// 	Object interface{}
// }

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
	Kids  []*DestTree
	Names []NameToDest
}

// Limits specify the (lexically) least and greatest keys included in the Names array of
// a leaf node or in the Names arrays of any leaf nodes that are descendants of an
// intermediate node.
func (d DestTree) Limits() [2]string {
	if len(d.Kids) == 0 { // leaf node
		if len(d.Names) == 0 {
			return [2]string{}
		}
		allNames := make([]string, len(d.Names))
		for i, v := range d.Names {
			allNames[i] = v.Name
		}
		sort.Strings(allNames)
		return [2]string{allNames[0], allNames[len(allNames)-1]}
	}
	limits := d.Kids[0].Limits()
	for _, kid := range d.Kids[1:] {
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

// ----------------------------------------------------------------------

type NameToFile struct {
	Name     string
	FileSpec *FileSpec // indirect object
}

// EmbeddedFileTree is written as a Name Tree in PDF,
// but, since it generally won't be big, is
// represented here as a flat list.
type EmbeddedFileTree []NameToFile

func (efs EmbeddedFileTree) Limits() [2]string {
	if len(efs) == 0 {
		return [2]string{}
	}
	allNames := make([]string, len(efs))
	for i, v := range efs {
		allNames[i] = v.Name
	}
	sort.Strings(allNames)
	return [2]string{allNames[0], allNames[len(allNames)-1]}
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
	Kids []*PageLabelsTree
	Nums []NumToPageLabel
}

// Limits specify the (numerically) least and greatest keys included in
// the Nums array of a leaf node or in the Nums arrays of any leaf nodes that are
// descendants of an intermediate node.
func (d PageLabelsTree) Limits() [2]int {
	if len(d.Kids) == 0 { // leaf node
		if len(d.Nums) == 0 {
			return [2]int{}
		}
		allNums := make([]int, len(d.Nums))
		for i, v := range d.Nums {
			allNums[i] = v.Num
		}
		sort.Ints(allNums)
		return [2]int{allNums[0], allNums[len(allNums)-1]}
	}
	limits := d.Kids[0].Limits()
	for _, kid := range d.Kids[1:] {
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
