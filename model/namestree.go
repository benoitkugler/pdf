package model

type NameTreeNode struct {
	Kids   []*NameTreeNode
	Names  []NamedObject
	Limits [2]string
}

type NameTree struct {
	Root NameTreeNode
}

type NamedObject struct {
	Name   string
	Object interface{}
}
