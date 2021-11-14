package formfill

import (
	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
)

type FDFValue interface {
	isFDFValue()
}

func (ButtonAppearanceName) isFDFValue() {}
func (Text) isFDFValue()                 {}
func (Choices) isFDFValue()              {}

// ButtonAppearanceName is the value of a field with type `Btn`
type ButtonAppearanceName model.ObjName

// Text is the value of a field with type `Tx`
type Text string

// Choices is the value of field with type `Ch`
type Choices []string

type Values struct {
	V  FDFValue
	RV string
}

type FDFField struct {
	Values
	Kids []FDFField
	T    string // partial field name
}

// FDFDict is the FDF entry of an FDF file catalog.
type FDFDict struct {
	Fields []FDFField
}

// walk the tree and construct the full names
func (f FDFDict) resolve() map[string]Values {
	out := map[string]Values{}
	var walk func(FDFField, string)
	walk = func(fi FDFField, parentName string) {
		fullName := parentName + "." + fi.T
		if parentName == "" { // exception for the root elements
			fullName = fi.T
		}
		if fi.V != nil || fi.RV != "" {
			out[fullName] = fi.Values
		}
		for _, kid := range fi.Kids {
			walk(kid, fullName)
		}
	}
	for _, rootField := range f.Fields {
		walk(rootField, "")
	}
	return out
}

// FillForm fill the AcroForm contained in the document
// using the value in `fdf`.
// If `lockForm` is true, all the fields are set ReadOnly (even the ones not filled).
// See FillFormFromFDF to use a FDF file as value input.
func FillForm(doc *model.Document, fdf FDFDict, lockForm bool) error {
	filler := newFiller()
	return filler.fillForm(&doc.Catalog.AcroForm, fdf, lockForm)
}

// FillFormFromFDF is the same as FillForm, but use the given `fdf` FDF file as input for
// filling the forms in `doc`.
// TODO:
func FillFormFromFDF(doc *model.Document, fdf file.FDFFile, lockForm bool) error {
	fields := processFDFFile(fdf)
	return FillForm(doc, fields, lockForm)
}

func processFDFFile(fi file.FDFFile) FDFDict {
	return FDFDict{}
	// TODO:
}
