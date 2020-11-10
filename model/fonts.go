package model

import (
	"fmt"
	"sort"
	"strings"
)

// Font is a PDF font Dictionary
type Font struct {
	Subtype FontType
}

func (f *Font) PDFBytes(pdf PDFWriter) []byte {
	return f.Subtype.fontPDFBytes(pdf)
}

type FontType interface {
	isFontType()
	fontPDFBytes(pdf PDFWriter) []byte
}

func (Type0) isFontType()    {}
func (Type1) isFontType()    {}
func (Type3) isFontType()    {}
func (TrueType) isFontType() {}

type Type1 struct {
	BaseFont            Name
	FirstChar, LastChar byte
	Widths              []float64 // length (LastChar − FirstChar + 1) index i is char FirstChar + i
	FontDescriptor      FontDescriptor
	Encoding            SimpleEncoding // optional
}

func (t Type1) fontPDFBytes(pdf PDFWriter) []byte {
	fd := pdf.addObject(t.FontDescriptor.pdfBytes())
	b := newBuffer()
	b.line("<</Type /Font /Subtype /Type1 /BaseFont %s /FirstChar %d /LastChar %d",
		t.BaseFont, t.FirstChar, t.LastChar)
	b.line("/FontDescriptor %s", fd)
	b.line("/Widths %s", writeFloatArray(t.Widths))
	if t.Encoding != nil {
		enc := writeSimpleEncoding(t.Encoding, pdf)
		b.line("/Encoding %s", enc)
	}
	b.WriteString(">>")
	return b.Bytes()
}

type TrueType Type1

func (t TrueType) fontPDFBytes(pdf PDFWriter) []byte {
	return Type1(t).fontPDFBytes(pdf)
}

type Type3 struct {
	FontBBox            Rectangle
	FontMatrix          Matrix
	CharProcs           map[Name]ContentStream
	Encoding            SimpleEncoding
	FirstChar, LastChar byte
	Widths              []float64 // length (LastChar − FirstChar + 1) index i is char FirstChar + i
	FontDescriptor      FontDescriptor
	Resources           ResourcesDict
}

// TODO: type3 font
func (f Type3) fontPDFBytes(pdf PDFWriter) []byte {
	return []byte("<<>>")
}

type FontFlag uint32

const (
	FixedPitch  FontFlag = 1
	Serif       FontFlag = 1 << 2
	Symbolic    FontFlag = 1 << 3
	Script      FontFlag = 1 << 4
	Nonsymbolic FontFlag = 1 << 6
	Italic      FontFlag = 1 << 7
	AllCap      FontFlag = 1 << 17
	SmallCap    FontFlag = 1 << 18
	ForceBold   FontFlag = 1 << 19
)

type FontDescriptor struct {
	FontName        Name
	Flags           uint32
	FontBBox        Rectangle
	ItalicAngle     float64
	Ascent, Descent float64
	Leading         float64
	CapHeight       float64
	XHeight         float64
	StemV, StemH    float64
	AvgWidth        float64
	MaxWidth        float64
	MissingWidth    float64

	//TODO: check stream fontfile
}

func (f FontDescriptor) pdfBytes() []byte {
	b := newBuffer()
	b.line("<</Type /FontDescriptor /FontName %s /Flags %d /FontBBox %s /ItalicAngle %.3f /Ascent %.3f /Descent %.3f",
		f.FontName, f.Flags, f.FontBBox.PDFstring(), f.ItalicAngle, f.Ascent, f.Descent)
	if f.Leading != 0 {
		b.fmt("/Leading %.3f ", f.Leading)
	}
	b.fmt("/CapHeight %.3f ", f.CapHeight)
	if f.XHeight != 0 {
		b.fmt("/XHeight %.3f ", f.XHeight)
	}
	b.fmt("/StemV %.3f ", f.StemV)
	if f.StemH != 0 {
		b.fmt("/StemH %.3f ", f.StemH)
	}
	if f.AvgWidth != 0 {
		b.fmt("/AvgWidth %.3f ", f.AvgWidth)
	}
	if f.MaxWidth != 0 {
		b.fmt("/MaxWidth %.3f ", f.MaxWidth)
	}
	if f.MissingWidth != 0 {
		b.fmt("/MissingWidth %.3f ", f.MissingWidth)
	}
	b.fmt(">>")
	return b.Bytes()
}

// SimpleEncoding is a font encoding for simple fonts
type SimpleEncoding interface {
	isSimpleEncoding()
}

func (PredefinedEncoding) isSimpleEncoding() {}
func (*EncodingDict) isSimpleEncoding()      {}

// return either a name or an indirect ref
func writeSimpleEncoding(enc SimpleEncoding, pdf PDFWriter) string {
	switch enc := enc.(type) {
	case PredefinedEncoding:
		return Name(enc).String()
	case *EncodingDict:
		ref := pdf.addItem(enc)
		return ref.String()
	default:
		panic("exhaustive switch")
	}
}

type PredefinedEncoding Name

const (
	MacRomanEncoding  PredefinedEncoding = "MacRomanEncoding"
	MacExpertEncoding PredefinedEncoding = "MacExpertEncoding"
	WinAnsiEncoding   PredefinedEncoding = "WinAnsiEncoding"
)

// NewPrededinedEncoding validated the string `s`
// and return either a valid `PredefinedEncoding` or nil
func NewPrededinedEncoding(s string) SimpleEncoding {
	e := PredefinedEncoding(s)
	switch e {
	case MacExpertEncoding, MacRomanEncoding, WinAnsiEncoding:
		return e
	default:
		return nil
	}
}

// Differences describes the differences from the encoding specified by BaseEncoding
// It is written in a PDF file as a more condensed form: it is an array:
// 	[ code1, name1_1, name1_2, code2, name2_1, name2_2, name2_3 ... ]
// where code1 -> name1_1 ; code1 + 1 -> name1_2 ; ...
type Differences map[byte]Name

// pack the differences again
func (d Differences) pdfString() string {
	keys := make([]byte, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	var chunks []string
	for i, k := range keys {
		name := d[k].String()
		if i >= 1 && keys[i-1] == k-1 { // consecutive -> add name to the same serie
			chunks = append(chunks, name)
		} else { // start a new serie
			chunks = append(chunks, fmt.Sprintf("%d", k), name)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(chunks, " "))
}

type EncodingDict struct {
	BaseEncoding Name        // optionnal
	Differences  Differences // optionnal
}

func (e *EncodingDict) PDFBytes(PDFWriter) []byte {
	out := "<<"
	if e.BaseEncoding != "" {
		out += "/BaseEncoding " + e.BaseEncoding.String()
	}
	if len(e.Differences) != 0 {
		out += "/Differences " + e.Differences.pdfString()
	}
	out += ">>"
	return []byte(out)
}

// -------------------------- Type 0 --------------------------

type Type0 struct {
	BaseFont        Name
	Encoding        CMapEncoding
	DescendantFonts CIDFontDictionary // in PDF, array of one indirect object
	ToUnicode       *ContentStream    // optionnal, as indirect object
}

func (f Type0) fontPDFBytes(pdf PDFWriter) []byte {
	enc := writeCMapEncoding(f.Encoding, pdf)
	desc := pdf.addObject(f.DescendantFonts.pdfBytes(pdf))
	out := fmt.Sprintf("<</Type /Font /Subtype /Type0 /BaseFont %s /Encoding %s /DescendantFonts [%s]",
		f.BaseFont, enc, desc)
	if f.ToUnicode != nil {
		toU := pdf.addObject(f.ToUnicode.PDFBytes())
		out += " /ToUnicode " + toU.String()
	}
	out += ">>"
	return []byte(out)
}

// CMapEncoding maps character codes to font numbers and CIDs
type CMapEncoding interface {
	isCMapEncoding()
}

func (PredefinedCMapEncoding) isCMapEncoding() {}
func (EmbeddedCMapEncoding) isCMapEncoding()   {}

type PredefinedCMapEncoding Name

type EmbeddedCMapEncoding ContentStream

// return either a ref or a name
func writeCMapEncoding(enc CMapEncoding, pdf PDFWriter) string {
	switch enc := enc.(type) {
	case PredefinedCMapEncoding:
		return Name(enc).String()
	case EmbeddedCMapEncoding:
		ref := pdf.addObject(ContentStream(enc).PDFBytes())
		return ref.String()
	default:
		panic("exhaustive switch")
	}
}

type CIDFontDictionary struct {
	Subtype        Name // CIDFontType0 or CIDFontType2
	BaseFont       Name
	CIDSystemInfo  CIDSystemInfo
	FontDescriptor FontDescriptor // indirect object
	DW             int            // optionnal, default to 1000
	W              []CIDWidth     // optionnal
	DW2            [2]int         // optionnal, default to [ 880 −1000 ]
	W2             []CIDWidth     // optionnal
}

func (c CIDFontDictionary) pdfBytes(pdf PDFWriter) []byte {
	b := newBuffer()
	fD := pdf.addObject(c.FontDescriptor.pdfBytes())
	b.line("<</Type /Font /Subtype %s /BaseFont %s /CIDSystemInfo %s /FontDescriptor %s",
		c.Subtype, c.BaseFont, c.CIDSystemInfo.pdfString(pdf), fD)
	if c.DW != 0 {
		b.line("/DW %d", c.DW)
	}
	if len(c.W) != 0 {
		chunks := make([]string, len(c.W))
		for i, c := range c.W {
			chunks[i] = c.String()
		}
		b.line("/W [%s]", strings.Join(chunks, " "))
	}
	if c.DW2 != [2]int{} {
		b.line("/DW2 %s", writeIntArray(c.DW2[:]))
	}
	if len(c.W2) != 0 {
		chunks := make([]string, len(c.W2))
		for i, c := range c.W2 {
			chunks[i] = c.String()
		}
		b.line("/W2 [%s]", strings.Join(chunks, " "))
	}
	b.fmt(">>")
	return b.Bytes()
}

type CIDSystemInfo struct {
	Registry   string
	Ordering   string
	Supplement int
}

// String returns a dictionary representation
func (c CIDSystemInfo) pdfString(pdf PDFWriter) string {
	return fmt.Sprintf("<</Registry %s /Ordering %s /Supplement %d>>",
		pdf.ASCIIString(c.Registry), pdf.ASCIIString(c.Ordering), c.Supplement)
}

// CIDWidth groups the two ways of defining widths for CID
type CIDWidth interface {
	// Widths returns the widths for each character, defined in user units
	Widths() map[rune]int
	// String returns a PDF representation of the width
	String() string
}

// CIDWidthRange is written in PDF as
//	c_first c_last w
type CIDWidthRange struct {
	First, Last rune
	Width       int
}

func (c CIDWidthRange) Widths() map[rune]int {
	out := make(map[rune]int, c.Last-c.First)
	for r := c.First; r <= c.Last; r++ {
		out[r] = c.Width
	}
	return out
}

func (c CIDWidthRange) String() string {
	return fmt.Sprintf("%d %d %d", c.First, c.Last, c.Width)
}

// CIDWidthArray is written in PDF as
//	c [ w_1 w_2 ... w_n ]
type CIDWidthArray struct {
	Start rune
	W     []int
}

func (c CIDWidthArray) Widths() map[rune]int {
	out := make(map[rune]int, len(c.W))
	for i, w := range c.W {
		out[c.Start+rune(i)] = w
	}
	return out
}

func (c CIDWidthArray) String() string {
	return fmt.Sprintf("%d %s", c.Start, writeIntArray(c.W))
}
