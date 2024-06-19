// Package formfill provides support for filling forms found
// in PDF files (aka AcroForm), reading forms input
// either form an FDF file or directly from memory.
package formfill

import (
	"errors"
	"strconv"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader"
	"github.com/benoitkugler/pdf/reader/file"
)

type FDFValue interface {
	isFDFValue()
}

func (FDFName) isFDFValue()    {}
func (FDFText) isFDFValue()    {}
func (FDFChoices) isFDFValue() {}

// FDFName is the value of a field with type `Btn`
type FDFName model.ObjName

// FDFText is the value of a field with type `Tx` or `Ch`
type FDFText string

// FDFChoices is the value of field with type `Ch`
type FDFChoices []string

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
	var walk func(FDFField, string, int)
	walk = func(fi FDFField, parentName string, index int) {
		name := fi.T
		if fi.T == "" {
			name = strconv.Itoa(index)
		}
		fullName := parentName + "." + name
		if parentName == "" { // exception for the root elements
			fullName = name
		}
		if fi.V != nil || fi.RV != "" {
			out[fullName] = fi.Values
		}
		for index, kid := range fi.Kids {
			walk(kid, fullName, index)
		}
	}
	for index, rootField := range f.Fields {
		walk(rootField, "", index)
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
func FillFormFromFDF(doc *model.Document, fdf file.FDFFile, lockForm bool) error {
	fields, err := processFDFFile(fdf)
	if err != nil {
		return err
	}
	return FillForm(doc, fields, lockForm)
}

func processFDFFile(fi file.FDFFile) (FDFDict, error) {
	catalog, ok := fi.XrefTable.ResolveObject(fi.Root).(model.ObjDict)
	if !ok {
		return FDFDict{}, errors.New("invalid type for Catalog")
	}
	fdf, ok := fi.XrefTable.ResolveObject(catalog["FDF"]).(model.ObjDict)
	if !ok {
		return FDFDict{}, errors.New("invalid type for FDF entry")
	}
	fields, ok := fi.XrefTable.ResolveObject(fdf["Fields"]).(model.ObjArray)
	if !ok {
		return FDFDict{}, errors.New("invalid type for Fields entry")
	}

	var resolveTree func(node model.Object) (FDFField, error)
	resolveTree = func(node model.Object) (FDFField, error) {
		fieldDict, ok := fi.XrefTable.ResolveObject(node).(model.ObjDict)
		if !ok {
			return FDFField{}, errors.New("invalid type for Fields item")
		}

		T, _ := file.IsString(fieldDict["T"])
		RV, _ := file.IsString(fieldDict["RV"])
		out := FDFField{
			T: reader.DecodeTextString(T),
			Values: Values{
				RV: reader.DecodeTextString(RV),
			},
		}

		// parse value V
		switch V := fi.XrefTable.ResolveObject(fieldDict["V"]).(type) {
		case model.ObjName:
			out.V = FDFName(V)
		case model.ObjArray:
			L := make(FDFChoices, len(V))
			for i, v := range V {
				s, _ := file.IsString(v)
				L[i] = reader.DecodeTextString(s)
			}
			out.V = L
		default:
			s, ok := file.IsString(V)
			if !ok {
				return FDFField{}, errors.New("invalid type for form field V entry")
			}
			out.V = FDFText(reader.DecodeTextString(s))
		}

		// recurse on kids
		kids, _ := fieldDict["Kids"].(model.ObjArray) // optional
		out.Kids = make([]FDFField, len(kids))
		for i, k := range kids {
			var err error
			out.Kids[i], err = resolveTree(k)
			if err != nil {
				return FDFField{}, err
			}
		}

		return out, nil
	}

	root := make([]FDFField, len(fields))
	for i, ref := range fields {
		var err error
		root[i], err = resolveTree(ref)
		if err != nil {
			return FDFDict{}, err
		}
	}

	return FDFDict{Fields: root}, nil
}
