package model

import (
	"fmt"
	"strconv"
	"strings"
)

// implements basic types found in PDF files

// Object is a node of a PDF syntax tree.
//
// It serves two purposes:
//   - representing a PDF file in-memory, before turning it into a Document.
//     In this case, it is obtained from a PDF file by tokenizing and parsing its content,
//     and the concrete types used will be the basic PDF types defined in this file.
//   - allowing arbitrary user defined content, which is needed for some edge-cases like
//     property list or signature build information.
//     In this case, custom type may be used, but care should be taken to handle indirect objects:
//     when implementing WriteToPDF, new objects must be created using CreateObject.
//
// Note that the PDF null object is represented by its own concrete type,
// so Object must never be nil.
type Object interface {
	// Write must return a PDF string representation
	// of the object.
	// `PDFWritter` shall be use with strings and streams,
	// so that they are espaced and crypted accordingly.
	// This requires the `Reference` value (object number) of the parent object,
	// which should be forwarded to the `PDFWritter.EncodeString` method.
	//
	// When using indirect objects, `PDFWritter` `CreateObject` and `WriteObject`
	// methods must be used to create the necessary object and returns the string
	// form of the reference.
	//
	// When used in content stream, ecryption is disabled, so the `PDFWritter`
	// instance will be nil (and `Reference` invalid).
	Write(writter PDFWritter, parent Reference) string

	// Clone must return a deep copy of the object, preserving the concrete type.
	Clone() Object
}

type ObjNull struct{}

func (ObjNull) String() string { return "<null>" }

// String returns the PDF representation of a name
func (ObjNull) Write(PDFWritter, Reference) string { return "null" }

func (n ObjNull) Clone() Object { return n }

// ObjName is a symbol to be referenced,
// and it is included in PDF without encoding, by prepending/
type ObjName string

// String returns the PDF representation of a name
func (n ObjName) String() string {
	return "/" + string(n)
}

func (n ObjName) Clone() Object { return n }

// String returns the PDF representation of a name
func (n ObjName) Write(PDFWritter, Reference) string {
	return n.String()
}

// ObjFloat implements MaybeFloat
type ObjFloat Fl

func (f ObjFloat) Write(PDFWritter, Reference) string {
	return FmtFloat(float32(f))
}

func (f ObjFloat) Clone() Object { return f }

// ObjBool represents a PDF boolean object.
type ObjBool bool

func (boolean ObjBool) Clone() Object { return boolean }
func (boolean ObjBool) Write(PDFWritter, Reference) string {
	return fmt.Sprintf("%v", bool(boolean))
}

// ObjInt represents a PDF integer object.
type ObjInt int

func (i ObjInt) Clone() Object { return i }
func (i ObjInt) Write(PDFWritter, Reference) string {
	return strconv.Itoa(int(i))
}

// ObjStringLiteral represents a PDF string literal object.
// When required, text strings must be encoded and encrypted
// in a first step: the content of ObjStringLiteral will only be escaped.
type ObjStringLiteral string

func (s ObjStringLiteral) Clone() Object { return s }

func (s ObjStringLiteral) Write(w PDFWritter, r Reference) string {
	if w == nil { // content stream mode
		return EscapeByteString([]byte(s))
	}
	return w.EncodeString(string(s), ByteString, r)
}

// ObjHexLiteral represents a PDF hex literal object.
// Its content is stored not encoded, and will be transformed
// when writting to a PDF file.
// When required, text strings must be encoded and encrypted
// in a first step.
type ObjHexLiteral string

func (h ObjHexLiteral) Clone() Object { return h }

func (h ObjHexLiteral) Write(w PDFWritter, r Reference) string {
	if w == nil { // content stream mode
		return EspaceHexString([]byte(h))
	}
	return w.EncodeString(string(h), HexString, r)
}

// ObjIndirectRef represents a PDF indirect object.
// This type will be found in a parsed PDF, but not in the model
// (see the `Reference` type documentation).
type ObjIndirectRef struct {
	ObjectNumber     int
	GenerationNumber int
}

func (ir ObjIndirectRef) Clone() Object { return ir }

func (ir ObjIndirectRef) Write(PDFWritter, Reference) string {
	return fmt.Sprintf("%d %d R", ir.ObjectNumber, ir.GenerationNumber)
}

// ObjCommand is a PDF operation found in content streams.
type ObjCommand string

func (cmd ObjCommand) Clone() Object { return cmd }

func (cmd ObjCommand) Write(PDFWritter, Reference) string {
	return string(cmd)
}

// ObjArray represents a PDF array object.
type ObjArray []Object

func (arr ObjArray) Clone() Object {
	out := make(ObjArray, len(arr))
	for i, v := range arr {
		out[i] = v.Clone()
	}
	return out
}

func (arr ObjArray) Write(w PDFWritter, r Reference) string {
	chunks := make([]string, len(arr))
	for i, o := range arr {
		chunks[i] = o.Write(w, r)
	}
	return "[" + strings.Join(chunks, " ") + "]"
}

// ObjDict represents a PDF dict object.
type ObjDict map[Name]Object

func (d ObjDict) Clone() Object {
	out := make(ObjDict, len(d))
	for k, v := range d {
		out[k] = v.Clone()
	}
	return out
}

func (d ObjDict) Write(w PDFWritter, r Reference) string {
	chunks := make([]string, 0, len(d))
	for i, o := range d {
		chunks = append(chunks, i.Write(w, r), o.Write(w, r))
	}
	return "<<\n" + strings.Join(chunks, "\n") + "\n>>"
}

// ObjStream is a stream
type ObjStream struct {
	Args    ObjDict
	Content []byte // as written in a PDF file (that is, encoded)
}

func (stream ObjStream) Clone() Object {
	return ObjStream{
		Args:    stream.Args.Clone().(ObjDict),
		Content: append([]byte(nil), stream.Content...),
	}
}

func (stream ObjStream) bypassEncrypt() bool {
	fs := stream.Args["Filter"]
	if fs, ok := fs.(ObjArray); ok {
		return len(fs) == 1 && fs[1] == ObjName("Crypt")
	}
	return fs == ObjName("Crypt")
}

func (stream ObjStream) Write(w PDFWritter, r Reference) string {
	if w == nil { // shoud never happen
		return ""
	}
	ref := w.CreateObject()

	streamDict := make(map[Name]string, len(stream.Args))
	for i, o := range stream.Args {
		streamDict[i] = o.Write(w, r)
	}

	w.WriteStream(StreamHeader{Fields: streamDict, BypassCrypt: stream.bypassEncrypt()}, stream.Content, ref)
	return ref.String()
}

// ----------------------- utils commonly used -----------------------

// Name is so used that it deservers a shorted alias
type Name = ObjName

// Fl is the numeric type used for float values.
type Fl = float32

// MaybeInt is an Int or nothing
// It'a an other way to specify *int,
// safer to use and pass by value.
type MaybeInt interface {
	isMaybeInt()
}

func (i ObjInt) isMaybeInt() {}

// MaybeFloat is a Float or nothing
// It'a an other way to specify *Fl,
// safer to use and pass by value.
type MaybeFloat interface {
	isMaybeFloat()
}

func (f ObjFloat) isMaybeFloat() {}

// MaybeBool is a Bool or nothing
// It'a an other way to specify *Fl,
// safer to use and pass by value.
type MaybeBool interface {
	isMaybeBool()
}

func (b ObjBool) isMaybeBool() {}

// IsString return `true` is `o` is either a StringLitteral
// or an HexLitteral
func IsString(o Object) (string, bool) {
	switch s := o.(type) {
	case ObjStringLiteral:
		return string(s), true
	case ObjHexLiteral:
		return string(s), true
	default:
		return "", false
	}
}

// IsNumber return `true` is `o` is either a Float
// or an Int
func IsNumber(o Object) (Fl, bool) {
	switch t := o.(type) {
	case ObjFloat:
		return Fl(t), true
	case ObjInt:
		return Fl(t), true
	default:
		return 0, false
	}
}

type Rectangle struct {
	Llx, Lly, Urx, Ury Fl // lower-left x, lower-left y, upper-right x, and upper-right y coordinates of the rectangle
}

func (r Rectangle) String() string {
	return writeFloatArray([]Fl{r.Llx, r.Lly, r.Urx, r.Ury})
}

// Height returns the absolute value of the height of the rectangle.
func (r Rectangle) Height() Fl {
	h := r.Ury - r.Lly
	if h < 0 {
		return -h
	}
	return h
}

// Width returns the absolute value of the width of the rectangle.
func (r Rectangle) Width() Fl {
	w := r.Urx - r.Llx
	if w < 0 {
		return -w
	}
	return w
}

// Rotation encodes an optional clock-wise rotation.
type Rotation uint8

const (
	Unset Rotation = iota // use the inherited value
	Zero
	Quarter
	Half
	ThreeQuarter
)

// NewRotation validate the input and returns
// a rotation, which may be unset.
func NewRotation(degrees int) Rotation {
	if degrees%90 != 0 {
		return Unset
	}
	r := Rotation((degrees / 90) % 4)
	return r + 1
}

func (r Rotation) Degrees() int {
	if r == Unset {
		return 0
	}
	return 90 * int(r-1)
}
