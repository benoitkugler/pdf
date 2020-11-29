package parser

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/benoitkugler/pdf/model"
)

// Object is a node of a PDF syntax tree.
// It is obtained from a PDF file by tokenizing and parsing its content.
// The PDF null object is represented by a nil interface.
type Object interface {
	// PDFString must return a valid PDF syntax
	PDFString() string
}

// Boolean represents a PDF boolean object.
type Boolean bool

func (boolean Boolean) PDFString() string {
	return fmt.Sprintf("%v", bool(boolean))
}

// Float represents a PDF float object.
type Float float64

func (f Float) PDFString() string {
	// The max precision encountered so far has been 12 (fontType3 fontmatrix components).
	return strconv.FormatFloat(float64(f), 'f', 12, 64)
}

// Integer represents a PDF integer object.
type Integer int

func (i Integer) PDFString() string {
	return strconv.Itoa(int(i))
}

// Name represents a PDF name object.
type Name string

func (nameObject Name) PDFString() string {
	return "/" + string(nameObject)
}

// StringLiteral represents a PDF string literal object.
// When required, text strings must be encoded and encrypted
// in a first step: the content of StringLiteral will only be escaped.
type StringLiteral string

func (s StringLiteral) PDFString() string {
	return model.EspaceByteString(string(s))
}

// HexLiteral represents a PDF hex literal object.
// When required, text strings must be encoded and encrypted
// in a first step.
type HexLiteral string

func (h HexLiteral) PDFString() string {
	return "<" + hex.EncodeToString([]byte(h)) + ">"
}

// IndirectRef represents a PDF indirect object.
type IndirectRef struct {
	ObjectNumber     Integer
	GenerationNumber Integer
}

func (ir IndirectRef) PDFString() string {
	return fmt.Sprintf("%d %d R", ir.ObjectNumber, ir.GenerationNumber)
}

// Command is a PDF operation found in content streams.
type Command string

func (ir Command) PDFString() string {
	return string(ir)
}

// Array represents a PDF array object.
type Array []Object

func (arr Array) PDFString() string {
	chunks := make([]string, len(arr))
	for i, o := range arr {
		if o == nil {
			chunks[i] = "null"
		} else {
			chunks[i] = o.PDFString()
		}
	}
	return "[" + strings.Join(chunks, " ") + "]"
}

// Dict represents a PDF dict object.
type Dict map[Name]Object

func (d Dict) PDFString() string {
	chunks := make([]string, 0, len(d))
	for i, o := range d {
		if o == nil {
			chunks = append(chunks, "null")
		} else {
			chunks = append(chunks, i.PDFString(), o.PDFString())
		}
	}
	return "<<\n" + strings.Join(chunks, "\n") + "\n>>"
}
