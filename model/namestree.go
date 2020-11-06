package model

import "sort"

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
	Kids   []*DestTree
	Names  []NameToDest
	Limits [2]string
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
