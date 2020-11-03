package model

type FormType uint8

const (
	Tx FormType = iota
	Btn
	Ch
	Sig
)

type FormField struct {
	Ft FormType
	T  string
}

type AcroForm struct {
	Fields          []FormField
	NeedAppearances bool
}
