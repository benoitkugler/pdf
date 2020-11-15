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

func (f *Font) pdfContent(pdf pdfWriter, _ reference) (string, []byte) {
	return f.Subtype.fontPDFString(pdf), nil
}

type FontType interface {
	isFontType()
	fontPDFString(pdf pdfWriter) string
}

func (Type0) isFontType()    {}
func (Type1) isFontType()    {}
func (TrueType) isFontType() {}
func (Type3) isFontType()    {}

type Type1 struct {
	BaseFont  Name
	FirstChar byte
	// length (LastChar − FirstChar + 1) index i is char FirstChar + i
	// width are measured in units in which 1000 units correspond to 1 unit in text space
	Widths         []int
	FontDescriptor FontDescriptor
	Encoding       SimpleEncoding // optional
}

// LastChar return the last caracter encoded by the font (see Widths)
func (t Type1) LastChar() byte {
	return byte(len(t.Widths)) + t.FirstChar - 1
}

// font must be Type1 or TrueType,
// and is needed for the FontDescriptor
func t1orttPDFString(font FontType, pdf pdfWriter) string {
	var (
		t       Type1
		subtype Name
	)
	switch font := font.(type) {
	case Type1:
		t = font
		subtype = "Type1"
	case TrueType:
		t = Type1(font)
		subtype = "TrueType"
	}
	fd := pdf.addObject(t.FontDescriptor.pdfString(pdf, font), nil) // FontDescriptor need the type of font
	b := newBuffer()
	b.line("<</Type/Font/Subtype %s/BaseFont %s/FirstChar %d/LastChar %d",
		subtype, t.BaseFont, t.FirstChar, t.LastChar())
	b.line("/FontDescriptor %s", fd)
	b.line("/Widths %s", writeIntArray(t.Widths))
	if t.Encoding != nil {
		b.line("/Encoding %s", t.Encoding.simpleEncodingPDFString(pdf))
	}
	b.WriteString(">>")
	return b.String()
}

func (t Type1) fontPDFString(pdf pdfWriter) string {
	return t1orttPDFString(t, pdf)
}

type TrueType Type1

func (t TrueType) fontPDFString(pdf pdfWriter) string {
	return t1orttPDFString(t, pdf)

}

type Type3 struct {
	FontBBox       Rectangle
	FontMatrix     Matrix
	CharProcs      map[Name]ContentStream
	Encoding       SimpleEncoding // required
	FirstChar      byte
	Widths         []int           // length (LastChar − FirstChar + 1); index i is char code FirstChar + i
	FontDescriptor *FontDescriptor // required in TaggedPDF
	Resources      *ResourcesDict  // optional
	ToUnicode      *Stream         // optional
}

// LastChar return the last caracter encoded by the font (see Widths)
func (t Type3) LastChar() byte {
	return byte(len(t.Widths)) + t.FirstChar - 1
}

func (f Type3) fontPDFString(pdf pdfWriter) string {
	b := newBuffer()
	b.line("<</Type/Font/Subtype/Type3/FontBBox %s/FontMatrix %s",
		f.FontBBox.String(), f.FontMatrix.String())
	chunks := make([]string, 0, len(f.CharProcs))
	for name, stream := range f.CharProcs {
		ref := pdf.addObject(stream.PDFContent())
		chunks = append(chunks, fmt.Sprintf("%s %s", name, ref))
	}
	b.line("/CharProcs <<%s>>", strings.Join(chunks, ""))
	widthsRef := pdf.addObject(writeIntArray(f.Widths), nil)
	b.line("/Encoding %s/FirstChar %d/LastChar %d/Widths %s",
		f.Encoding.simpleEncodingPDFString(pdf), f.FirstChar, f.LastChar(), widthsRef)
	if f.FontDescriptor != nil {
		fdRef := pdf.addObject(f.FontDescriptor.pdfString(pdf, f), nil)
		b.fmt("/FontDescriptor %s", fdRef)
	}
	if f.Resources != nil {
		ref := pdf.addItem(f.Resources)
		b.fmt("/Resources %s", ref)
	}
	if f.ToUnicode != nil {
		ref := pdf.addObject(f.ToUnicode.PDFContent())
		b.fmt("/ToUnicode %s", ref)
	}
	b.WriteString(">>")
	return b.String()
}

// FontFlag specify various characteristics of a font.
type FontFlag uint32

const (
	FixedPitch  FontFlag = 1
	Serif       FontFlag = 1 << 1
	Symbolic    FontFlag = 1 << 2
	Script      FontFlag = 1 << 3
	Nonsymbolic FontFlag = 1 << 5
	Italic      FontFlag = 1 << 6
	AllCap      FontFlag = 1 << 16
	SmallCap    FontFlag = 1 << 17
	ForceBold   FontFlag = 1 << 18
)

type FontDescriptor struct {
	// PostScript name of the font: the value of BaseFont in the font or
	// CIDFont dictionary that refers to this font descriptor
	FontName   Name
	FontFamily string // byte string, optional
	Flags      FontFlag
	FontBBox   Rectangle // specify the font bounding box, expressed in the glyph coordinate system
	// angle, expressed in degrees counterclockwise from
	// the vertical, of the dominant vertical strokes of the font.
	ItalicAngle  float64
	Ascent       float64 // maximum height above the baseline reached by glyphs in this font
	Descent      float64 // (negative number) maximum depth below the baseline reached by glyphs in this font
	Leading      float64 // optional, default to 0. Spacing between baselines of consecutive lines of text
	CapHeight    float64 // vertical coordinate of the top of flat capital letters, measured from the baseline
	XHeight      float64 // optional, default to 0. Vertical coordinate of the top of flat nonascending lowercase letters
	StemV        float64 // thickness, measured horizontally, of the dominant vertical stems of glyphs in the font
	StemH        float64 // optional, default to 0. Thickness, measured vertically, of the dominant horizontal stems of glyphs in the font.
	AvgWidth     float64 // optional, default to 0. Average width of glyphs in the font.
	MaxWidth     float64 // optional, default to 0. Maximum width of glyphs in the font.
	MissingWidth float64 // optional, default to 0. Width to use for character codes whose widths are not specified

	FontFile *FontFile // optional, written in PDF under the key FontFile (for Type1), FontFile2 (for TrueType), FontFile3 (for Type 1 compact fonts, Type 0 compact CIDFonts or OpenType)
	CharSet  string    // optional, ASCII string or byte string. Meaningful only in Type 1 font
}

// font is used to choose the key for the potential FontFile
func (f FontDescriptor) pdfString(pdf pdfWriter, font FontType) string {
	b := newBuffer()
	b.line("<</Type/FontDescriptor/FontName %s/Flags %d/FontBBox %s/ItalicAngle %.3f/Ascent %.3f/Descent %.3f",
		f.FontName, f.Flags, f.FontBBox.String(), f.ItalicAngle, f.Ascent, f.Descent)
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
	if f.FontFile != nil {
		var key Name
		switch font.(type) {
		case Type1:
			key = "FontFile"
		case TrueType:
			key = "FontFile2"
		case Type3:
			key = "FontFile3"
		}
		ref := pdf.addObject(f.FontFile.pdfContent())
		b.fmt("%s %s ", key, ref)
	}
	b.fmt(">>")
	return b.String()
}

// SimpleEncoding is a font encoding for simple fonts
type SimpleEncoding interface {
	// return either a name or an indirect ref
	simpleEncodingPDFString(pdf pdfWriter) string
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

func (enc PredefinedEncoding) simpleEncodingPDFString(pdf pdfWriter) string {
	return Name(enc).String()
}

// Differences describes the differences from the encoding specified by BaseEncoding
// It is written in a PDF file as a more condensed form: it is an array:
// 	[ code1, name1_1, name1_2, code2, name2_1, name2_2, name2_3 ... ]
// where code1 -> name1_1 ; code1 + 1 -> name1_2 ; ...
type Differences map[byte]Name

// PDFString pack the differences again, to obtain a compact
// representation of the mappgin, as specified in the SPEC.
func (d Differences) PDFString() string {
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
			chunks = append(chunks, fmt.Sprintf("%d ", k), name)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(chunks, ""))
}

type EncodingDict struct {
	BaseEncoding Name        // optionnal
	Differences  Differences // optionnal
}

func (e *EncodingDict) pdfContent(pdfWriter pdfWriter, _ reference) (string, []byte) {
	out := "<<"
	if e.BaseEncoding != "" {
		out += "/BaseEncoding " + e.BaseEncoding.String()
	}
	if len(e.Differences) != 0 {
		out += "/Differences " + e.Differences.PDFString()
	}
	out += ">>"
	return out, nil
}

func (enc *EncodingDict) simpleEncodingPDFString(pdf pdfWriter) string {
	ref := pdf.addItem(enc)
	return ref.String()
}

// -------------------------- Type 0 --------------------------

type Type0 struct {
	BaseFont        Name
	Encoding        CMapEncoding
	DescendantFonts CIDFontDictionary // in PDF, array of one indirect object
	ToUnicode       *Stream           // optionnal, as indirect object
}

func (f Type0) fontPDFString(pdf pdfWriter) string {
	enc := writeCMapEncoding(f.Encoding, pdf)
	desc := pdf.createObject()
	pdf.writeObject(f.DescendantFonts.pdfString(pdf, desc), nil, desc)
	out := fmt.Sprintf("<</Type/Font/Subtype/Type0/BaseFont %s/Encoding %s/DescendantFonts [%s]",
		f.BaseFont, enc, desc)
	if f.ToUnicode != nil {
		toU := pdf.addObject(f.ToUnicode.PDFContent())
		out += "/ToUnicode " + toU.String()
	}
	out += ">>"
	return out
}

// CMapEncoding maps character codes to font numbers and CIDs
type CMapEncoding interface {
	isCMapEncoding()
}

func (PredefinedCMapEncoding) isCMapEncoding() {}
func (EmbeddedCMapEncoding) isCMapEncoding()   {}

type PredefinedCMapEncoding Name

type EmbeddedCMapEncoding Stream

// return either a ref or a name
func writeCMapEncoding(enc CMapEncoding, pdf pdfWriter) string {
	switch enc := enc.(type) {
	case PredefinedCMapEncoding:
		return Name(enc).String()
	case EmbeddedCMapEncoding:
		ref := pdf.addObject(Stream(enc).PDFContent())
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

func (c CIDFontDictionary) pdfString(pdf pdfWriter, ref reference) string {
	b := newBuffer()
	fD := pdf.addObject(c.FontDescriptor.pdfString(pdf, Type0{}), nil)
	b.line("<</Type/Font/Subtype %s/BaseFont %s/CIDSystemInfo %s/FontDescriptor %s",
		c.Subtype, c.BaseFont, c.CIDSystemInfo.pdfString(pdf, ref), fD)
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
	return b.String()
}

type CIDSystemInfo struct {
	Registry   string
	Ordering   string
	Supplement int
}

// returns a dictionary representation
func (c CIDSystemInfo) pdfString(pdf pdfWriter, ref reference) string {
	return fmt.Sprintf("<</Registry %s/Ordering %s/Supplement %d>>",
		pdf.EncodeString(c.Registry, ASCIIString, ref),
		pdf.EncodeString(c.Ordering, ASCIIString, ref), c.Supplement)
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

type FontFile struct {
	Stream

	Length1 int
	Length2 int
	Length3 int
	Subtype Name // optional, one of Type1C for Type 1 compact fonts, CIDFontType0C for Type 0 compact CIDFonts, or OpenType
}

func (f *FontFile) pdfContent() (string, []byte) {
	args := f.Stream.PDFCommonFields()
	out := fmt.Sprintf("<<%s/Length1 %d/Length2 %d/Length3 %d",
		args, f.Length1, f.Length2, f.Length3)
	if f.Subtype != "" {
		out += fmt.Sprintf("/Subtype %s", f.Subtype)
	}
	out += ">>"
	return out, f.Content
}
