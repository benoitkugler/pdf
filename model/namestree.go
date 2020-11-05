package model

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
	Destination *ExplicitDestination // indirect obj
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
