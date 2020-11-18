// PDF basic types used when run-time types
// are needed.
// Indirect object and streams are not represented
package values

import (
	"fmt"
	"strings"

	"github.com/benoitkugler/pdf/model"
)

// Name is a pdf name object
type Name model.Name

func (v Name) PDFString(model.PDFStringEncoder, model.Reference) string {
	return model.Name(v).String()
}

func (v Name) Clone() model.UPValue { return v }

// String is a text string
type String string

func (v String) PDFString(enc model.PDFStringEncoder, context model.Reference) string {
	return enc.EncodeString(string(v), model.TextString, context)
}
func (v String) Clone() model.UPValue { return v }

// Float is a float
type Float float64

func (v Float) PDFString(model.PDFStringEncoder, model.Reference) string {
	return fmt.Sprintf("%.3f", v)
}
func (v Float) Clone() model.UPValue { return v }

// Int is an int
type Int int

func (v Int) PDFString(model.PDFStringEncoder, model.Reference) string {
	return fmt.Sprintf("%d", v)
}
func (v Int) Clone() model.UPValue { return v }

// Bool is a bool
type Bool bool

func (v Bool) PDFString(model.PDFStringEncoder, model.Reference) string {
	return fmt.Sprintf("%v", v)
}
func (v Bool) Clone() model.UPValue { return v }

// Array is a PDF array
type Array []model.UPValue

func (v Array) PDFString(enc model.PDFStringEncoder, context model.Reference) string {
	chunks := make([]string, len(v))
	for i, a := range v {
		chunks[i] = a.PDFString(enc, context)
	}
	return "[" + strings.Join(chunks, " ") + "]"
}

func (v Array) Clone() model.UPValue {
	if v == nil { // preserve nil
		return v
	}
	out := make(Array, len(v))
	for i, a := range v {
		out[i] = a.Clone()
	}
	return out
}

// Dict is a PDF dictionary
type Dict map[Name]model.UPValue

func (v Dict) PDFString(enc model.PDFStringEncoder, context model.Reference) string {
	chunks := make([]string, 0, len(v))
	for name, a := range v {
		chunks = append(chunks, name.PDFString(enc, context)+" "+a.PDFString(enc, context))
	}
	return "<<" + strings.Join(chunks, "") + ">>"
}

func (v Dict) Clone() model.UPValue {
	if v == nil { // preserve nil
		return v
	}
	out := make(Dict, len(v))
	for name, a := range v {
		out[name] = a.Clone()
	}
	return out
}
