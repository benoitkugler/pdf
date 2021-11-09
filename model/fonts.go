package model

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// FontFlag specify various characteristics of a font.
// See Table 123 – Font flags of the PDF SPEC.
type FontFlag uint32

const (
	FixedPitch  FontFlag = 1 << (1 - 1)
	Serif       FontFlag = 1 << (2 - 1)
	Symbolic    FontFlag = 1 << (3 - 1)
	Script      FontFlag = 1 << (4 - 1)
	Nonsymbolic FontFlag = 1 << (6 - 1)
	Italic      FontFlag = 1 << (7 - 1)
	AllCap      FontFlag = 1 << (17 - 1)
	SmallCap    FontFlag = 1 << (18 - 1)
	ForceBold   FontFlag = 1 << (19 - 1)
)

type FontDescriptor struct {
	// Required. PostScript name of the font: the value of BaseFont in the font or
	// CIDFont dictionary that refers to this font descriptor
	FontName    Name
	Flags       FontFlag // Required
	ItalicAngle Fl       // Required

	FontFamily string    // byte string, optional
	FontBBox   Rectangle // specify the font bounding box, expressed in the glyph coordinate system
	// angle, expressed in degrees counterclockwise from
	// the vertical, of the dominant vertical strokes of the font.
	Ascent       Fl  // maximum height above the baseline reached by glyphs in this font
	Descent      Fl  // (negative number) maximum depth below the baseline reached by glyphs in this font
	Leading      Fl  // optional, default to 0. Spacing between baselines of consecutive lines of text
	CapHeight    Fl  // vertical coordinate of the top of flat capital letters, measured from the baseline
	XHeight      Fl  // optional, default to 0. Vertical coordinate of the top of flat nonascending lowercase letters
	StemV        Fl  // thickness, measured horizontally, of the dominant vertical stems of glyphs in the font
	StemH        Fl  // optional, default to 0. Thickness, measured vertically, of the dominant horizontal stems of glyphs in the font.
	AvgWidth     Fl  // optional, default to 0. Average width of glyphs in the font.
	MaxWidth     Fl  // optional, default to 0. Maximum width of glyphs in the font.
	MissingWidth int // optional, default to 0. Width to use for character codes whose widths are not specified

	FontFile *FontFile // optional, written in PDF under the key FontFile (for Type1), FontFile2 (for TrueType), FontFile3 (for Type 1 compact fonts, Type 0 compact CIDFonts or OpenType)
	CharSet  string    // optional, ASCII string or byte string. Meaningful only in Type 1 font
}

// font is used to choose the key for the potential FontFile
func (f FontDescriptor) pdfString(pdf pdfWriter, font Font, context Reference) string {
	b := newBuffer()
	b.line("<</Type/FontDescriptor/FontName %s/Flags %d/FontBBox %s/ItalicAngle %g/Ascent %g/Descent %g",
		f.FontName, f.Flags, f.FontBBox.String(), f.ItalicAngle, f.Ascent, f.Descent)
	if f.Leading != 0 {
		b.fmt("/Leading %g ", f.Leading)
	}
	b.fmt("/CapHeight %g ", f.CapHeight)
	if f.XHeight != 0 {
		b.fmt("/XHeight %g ", f.XHeight)
	}
	b.fmt("/StemV %g ", f.StemV)
	if f.StemH != 0 {
		b.fmt("/StemH %g ", f.StemH)
	}
	if f.AvgWidth != 0 {
		b.fmt("/AvgWidth %g ", f.AvgWidth)
	}
	if f.MaxWidth != 0 {
		b.fmt("/MaxWidth %g ", f.MaxWidth)
	}
	if f.MissingWidth != 0 {
		b.fmt("/MissingWidth %d ", f.MissingWidth)
	}
	if f.FontFile != nil {
		var key Name
		switch f.FontFile.Subtype {
		case "Type1C", "CIDFontType0C", "OpenType":
			key = "FontFile3"
		default:
			if _, isType1 := font.(FontType1); isType1 {
				key = "FontFile"
			} else {
				key = "FontFile2"
			}
		}
		ref := pdf.addStream(f.FontFile.pdfContent())
		b.fmt("%s %s ", key, ref)
	}
	if f.CharSet != "" {
		b.fmt("/CharSet %s", pdf.EncodeString(f.CharSet, ByteString, context))
	}
	b.fmt(">>")
	return b.String()
}

// Clone returns a deep copy of the font descriptor.
func (f FontDescriptor) Clone() FontDescriptor {
	out := f
	out.FontFile = out.FontFile.Clone()
	return out
}

// UnicodeCMapBase is either the name of
// a predifined CMap or an other UnicodeCMap stream
type UnicodeCMapBase interface {
	pdfString(pdf pdfWriter) string
	clone() UnicodeCMapBase
}

type UnicodeCMapBasePredefined Name

func (c UnicodeCMapBasePredefined) pdfString(pdf pdfWriter) string {
	return Name(c).String()
}

func (p UnicodeCMapBasePredefined) clone() UnicodeCMapBase { return p }

// UnicodeCMap is stream object containing a special kind of CMap file
// that maps character codes to Unicode values.
type UnicodeCMap struct {
	Stream

	UseCMap UnicodeCMapBase // optional
}

// Clone returns a deep copy
func (p *UnicodeCMap) Clone() *UnicodeCMap {
	if p == nil {
		return p // typed nil
	}
	out := *p
	out.Stream = p.Stream.Clone()
	if p.UseCMap != nil {
		out.UseCMap = p.UseCMap.clone()
	}
	return &out
}

func (p UnicodeCMap) clone() UnicodeCMapBase { return *p.Clone() }

func (c UnicodeCMap) pdfString(pdf pdfWriter) string {
	ref := pdf.CreateObject()
	dict := c.Stream.PDFCommonFields(true)
	dict.Fields["Type"] = "/CMap"
	if c.UseCMap != nil {
		dict.Fields["UseCMap"] = c.UseCMap.pdfString(pdf)
	}
	pdf.WriteStream(dict, c.Content, ref)
	return ref.String()
}

// FontDict is a PDF font Dictionary.
//
// It exposes only the information written and read in PDF
// files. Almost every text processing task will require
// more informations, especially when dealing with Unicode strings.
// See `pdf/fonts` for more information and tooling.
type FontDict struct {
	Subtype   Font
	ToUnicode *UnicodeCMap
}

func (f *FontDict) pdfContent(pdf pdfWriter, _ Reference) (StreamHeader, string, []byte) {
	sub := f.Subtype.fontPDFFields(pdf)
	if f.ToUnicode != nil {
		sub += "/ToUnicode " + f.ToUnicode.pdfString(pdf)
	}
	return StreamHeader{}, "<<" + sub + ">>", nil
}

// clone returns a deep copy, with concrete type `*Font`
func (f *FontDict) clone(cache cloneCache) Referenceable {
	if f == nil {
		return f
	}
	out := *f // shallow copy
	if f.Subtype != nil {
		out.Subtype = f.Subtype.clone(cache)
	}
	out.ToUnicode = f.ToUnicode.Clone()
	return &out
}

// Font is one of Type0, FontType1, TrueType or Type3
type Font interface {
	FontName() Name

	fontPDFFields(pdf pdfWriter) string
	// returns a deep copy, preserving the concrete type
	clone(cloneCache) Font
}

// FontSimple is either FontType1, TrueType or Type3
type FontSimple interface {
	Font
	SimpleEncoding() SimpleEncoding
}

type FontType1 struct {
	BaseFont  Name
	FirstChar byte
	// length (LastChar − FirstChar + 1) index i is char FirstChar + i
	// width are measured in units in which 1000 units correspond to 1 unit in text space
	Widths         []int
	FontDescriptor FontDescriptor
	Encoding       SimpleEncoding // optional
}

func (ft FontType1) FontName() Name                 { return ft.BaseFont }
func (ft FontType1) SimpleEncoding() SimpleEncoding { return ft.Encoding }

// LastChar return the last caracter encoded by the font (see Widths)
func (t FontType1) LastChar() byte {
	return byte(len(t.Widths) + int(t.FirstChar) - 1)
}

// font must be Type1 or TrueType,
// and is needed for the FontDescriptor
func t1orttWrite(font Font, pdf pdfWriter) string {
	var (
		t       FontType1
		subtype Name
	)
	switch font := font.(type) {
	case FontType1:
		t = font
		subtype = "Type1"
	case FontTrueType:
		t = FontType1(font)
		subtype = "TrueType"
	}
	fd := pdf.CreateObject()
	pdf.WriteObject(t.FontDescriptor.pdfString(pdf, font, fd), fd) // FontDescriptor need the type of font
	b := newBuffer()
	b.line("/Type/Font/Subtype %s/FirstChar %d/LastChar %d",
		subtype, t.FirstChar, t.LastChar())
	if t.BaseFont != "" {
		b.fmt("/BaseFont %s", t.BaseFont)
	}
	b.line("/FontDescriptor %s", fd)
	b.line("/Widths %s", writeIntArray(t.Widths))
	if t.Encoding != nil {
		b.line("/Encoding %s", t.Encoding.simpleEncodingWrite(pdf))
	}
	return b.String()
}

func (t FontType1) fontPDFFields(pdf pdfWriter) string {
	return t1orttWrite(t, pdf)
}

// returns a deep copy with concrete type `Type1`
func (t FontType1) clone(cache cloneCache) Font {
	out := t                                     // shallow copy
	out.Widths = append([]int(nil), t.Widths...) // preserve deep equal
	out.FontDescriptor = t.FontDescriptor.Clone()
	if t.Encoding != nil {
		out.Encoding = t.Encoding.cloneSE(cache)
	}
	return out
}

type FontTrueType FontType1

func (ft FontTrueType) FontName() Name                 { return ft.BaseFont }
func (ft FontTrueType) SimpleEncoding() SimpleEncoding { return ft.Encoding }

func (t FontTrueType) fontPDFFields(pdf pdfWriter) string {
	return t1orttWrite(t, pdf)
}

// returns a deep copy with concrete type `TrueType`
func (t FontTrueType) clone(cache cloneCache) Font {
	return FontTrueType(FontType1(t).clone(cache).(FontType1))
}

type FontType3 struct {
	FontBBox       Rectangle
	FontMatrix     Matrix
	CharProcs      map[Name]ContentStream
	Encoding       SimpleEncoding // required
	FirstChar      byte
	Widths         []int           // length (LastChar − FirstChar + 1); index i is char code FirstChar + i
	FontDescriptor *FontDescriptor // required in TaggedPDF
	Resources      ResourcesDict   // optional
}

func (ft FontType3) FontName() Name {
	if ft.FontDescriptor != nil {
		return ft.FontDescriptor.FontName
	}
	return ""
}

func (ft FontType3) SimpleEncoding() SimpleEncoding { return ft.Encoding }

// LastChar return the last caracter encoded by the font (see Widths)
func (t FontType3) LastChar() byte {
	return byte(len(t.Widths)) + t.FirstChar - 1
}

func (f FontType3) fontPDFFields(pdf pdfWriter) string {
	b := newBuffer()
	b.line("/Type/Font/Subtype/Type3/FontBBox %s/FontMatrix %s",
		f.FontBBox.String(), f.FontMatrix.String())
	chunks := make([]string, 0, len(f.CharProcs))
	for name, stream := range f.CharProcs {
		ref := pdf.addStream(stream.PDFContent())
		chunks = append(chunks, fmt.Sprintf("%s %s", name, ref))
	}
	b.line("/CharProcs <<%s>>", strings.Join(chunks, ""))
	widthsRef := pdf.addObject(writeIntArray(f.Widths))
	b.line("/Encoding %s/FirstChar %d/LastChar %d/Widths %s",
		f.Encoding.simpleEncodingWrite(pdf), f.FirstChar, f.LastChar(), widthsRef)
	if f.FontDescriptor != nil {
		fdRef := pdf.CreateObject()
		pdf.WriteObject(f.FontDescriptor.pdfString(pdf, f, fdRef), fdRef)
		b.fmt("/FontDescriptor %s", fdRef)
	}
	if !f.Resources.IsEmpty() {
		refResources := pdf.CreateObject()
		pdf.WriteObject(f.Resources.pdfString(pdf, refResources), refResources)
		b.fmt("/Resources %s", refResources)
	}
	return b.String()
}

// clone returns a deep copy, with concrete type `Type3`
func (t FontType3) clone(cache cloneCache) Font {
	out := t
	if t.CharProcs != nil { // preserve reflect.DeepEqual
		out.CharProcs = make(map[Name]ContentStream, len(t.CharProcs))
	}
	for n, cs := range t.CharProcs {
		out.CharProcs[n] = cs.Clone()
	}
	if t.Encoding != nil {
		out.Encoding = t.Encoding.cloneSE(cache)
	}
	out.Widths = append([]int(nil), t.Widths...)
	if t.FontDescriptor != nil {
		tf := t.FontDescriptor.Clone()
		out.FontDescriptor = &tf
	}
	out.Resources = t.Resources.clone(cache)
	return out
}

// SimpleEncoding is a font encoding for simple fonts
type SimpleEncoding interface {
	// return either a name or an indirect ref
	simpleEncodingWrite(pdf pdfWriter) string
	// cloneSE returns a deep copy, preserving the concrete type
	cloneSE(cache cloneCache) SimpleEncoding
}

const (
	MacRomanEncoding  SimpleEncodingPredefined = "MacRomanEncoding"
	MacExpertEncoding SimpleEncodingPredefined = "MacExpertEncoding"
	WinAnsiEncoding   SimpleEncodingPredefined = "WinAnsiEncoding"
)

type SimpleEncodingPredefined Name

// NewSimpleEncodingPredefined validated the string `s`
// and return either a valid `PredefinedEncoding` or nil
func NewSimpleEncodingPredefined(s string) SimpleEncoding {
	e := SimpleEncodingPredefined(s)
	switch e {
	case MacExpertEncoding, MacRomanEncoding, WinAnsiEncoding:
		return e
	default:
		return nil
	}
}

func (enc SimpleEncodingPredefined) simpleEncodingWrite(pdf pdfWriter) string {
	return Name(enc).String()
}

// Clone returns a deep copy with concrete type `PredefinedEncoding`
func (enc SimpleEncodingPredefined) cloneSE(cloneCache) SimpleEncoding { return enc }

// Differences describes the differences from the encoding specified by BaseEncoding
// It is written in a PDF file as a more condensed form: it is an array:
// 	[ code1, name1_1, name1_2, code2, name2_1, name2_2, name2_3 ... ]
// where code1 -> name1_1 ; code1 + 1 -> name1_2 ; ...
type Differences map[byte]Name

// PDFString pack the differences again, to obtain a compact
// representation of the mappgin, as specified in the SPEC.
func (d Differences) Write() string {
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
			chunks = append(chunks, fmt.Sprintf(" %d", k), name)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(chunks, ""))
}

// Clone returns a deep copy of `d`
func (d Differences) Clone() Differences {
	if d == nil { // preserve deep equal
		return nil
	}
	out := make(Differences, len(d))
	for k, v := range d {
		out[k] = v
	}
	return out
}

// Apply applies the difference to a base encoding, represented by glyph names.
func (d Differences) Apply(encoding [256]string) [256]string {
	// encoding is copied, since it's an array
	for b, n := range d {
		encoding[b] = string(n)
	}
	return encoding
}

type SimpleEncodingDict struct {
	BaseEncoding SimpleEncodingPredefined // optionnal
	Differences  Differences              // optionnal
}

func (e *SimpleEncodingDict) pdfContent(pdfWriter pdfWriter, _ Reference) (StreamHeader, string, []byte) {
	out := "<<"
	if e.BaseEncoding != "" {
		out += "/BaseEncoding " + Name(e.BaseEncoding).String()
	}
	if len(e.Differences) != 0 {
		out += "/Differences " + e.Differences.Write()
	}
	out += ">>"
	return StreamHeader{}, out, nil
}

func (enc *SimpleEncodingDict) simpleEncodingWrite(pdf pdfWriter) string {
	ref := pdf.addItem(enc)
	return ref.String()
}

// clone returns a deep copy with concrete type *EncodingDict
func (enc *SimpleEncodingDict) clone(cloneCache) Referenceable {
	if enc == nil {
		return enc
	}
	out := *enc // shallow copy
	out.Differences = enc.Differences.Clone()
	return &out
}

func (enc *SimpleEncodingDict) cloneSE(cache cloneCache) SimpleEncoding {
	return cache.checkOrClone(enc).(*SimpleEncodingDict)
}

// -------------------------- Type 0 --------------------------

type FontType0 struct {
	BaseFont        Name
	Encoding        CMapEncoding      // required
	DescendantFonts CIDFontDictionary // in PDF, array of one indirect object
}

func (f FontType0) FontName() Name {
	return f.BaseFont
}

func (f FontType0) fontPDFFields(pdf pdfWriter) string {
	desc := pdf.CreateObject()
	pdf.WriteObject(f.DescendantFonts.pdfString(pdf, desc), desc)
	out := fmt.Sprintf("/Type/Font/Subtype/Type0/BaseFont %s/DescendantFonts [%s]",
		f.BaseFont, desc)
	if f.Encoding != nil {
		out += "/Encoding " + f.Encoding.cMapEncodingWrite(pdf)
	}
	return out
}

// returns a deep copy with concrete type `Type0`
func (t FontType0) clone(cloneCache) Font {
	out := t
	out.Encoding = t.Encoding.Clone()
	out.DescendantFonts = t.DescendantFonts.Clone()
	return out
}

// CMapEncoding maps character codes to font numbers and CIDs
type CMapEncoding interface {
	cMapEncodingWrite(pdf pdfWriter) string
	// Clone returns a deep copy, preserving the concrete type
	Clone() CMapEncoding
}

type CMapEncodingPredefined Name

func (c CMapEncodingPredefined) cMapEncodingWrite(pdf pdfWriter) string {
	return Name(c).String()
}

// Clone returns a deep copy with concrete type `PredefinedCMapEncoding`
func (p CMapEncodingPredefined) Clone() CMapEncoding { return p }

type CMapEncodingEmbedded struct {
	Stream

	CMapName      Name
	CIDSystemInfo CIDSystemInfo
	WMode         bool         // optional, true for vertical
	UseCMap       CMapEncoding // optional
}

// Clone returns a deep copy with concrete type `EmbeddedCMapEncoding`
func (p CMapEncodingEmbedded) Clone() CMapEncoding {
	out := p
	out.Stream = p.Stream.Clone()
	if p.UseCMap != nil {
		out.UseCMap = p.UseCMap.Clone()
	}
	return out
}

func (c CMapEncodingEmbedded) cMapEncodingWrite(pdf pdfWriter) string {
	ref := pdf.CreateObject()
	base := c.Stream.PDFCommonFields(true)
	base.Fields["Type"] = "/CMap"
	base.Fields["CMapName"] = c.CMapName.String()
	base.Fields["CIDSystemInfo"] = c.CIDSystemInfo.pdfString(pdf, ref)
	if c.WMode {
		base.Fields["WMode"] = "1"
	}
	if c.UseCMap != nil {
		base.Fields["UseCMap"] = c.UseCMap.cMapEncodingWrite(pdf)
	}
	pdf.WriteStream(base, c.Content, ref)
	return ref.String()
}

// CIDToGIDMap is either /Identity or a (binary) stream
type CIDToGIDMap interface {
	cidToGIDMapString(pdf pdfWriter) string
	// Clone returns a deep copy, preserving the concrete type.
	Clone() CIDToGIDMap
}

type CIDToGIDMapIdentity struct{}

func (c CIDToGIDMapIdentity) cidToGIDMapString(pdfWriter) string {
	return Name("Identity").String()
}

func (c CIDToGIDMapIdentity) Clone() CIDToGIDMap { return c }

type CIDToGIDMapStream struct {
	Stream
}

func (c CIDToGIDMapStream) cidToGIDMapString(pdf pdfWriter) string {
	ref := pdf.addStream(c.Stream.PDFContent())
	return ref.String()
}

func (c CIDToGIDMapStream) Clone() CIDToGIDMap {
	return CIDToGIDMapStream{Stream: c.Stream.Clone()}
}

type CIDFontDictionary struct {
	Subtype        Name // CIDFontType0 or CIDFontType2
	BaseFont       Name
	CIDSystemInfo  CIDSystemInfo
	FontDescriptor FontDescriptor      // indirect object
	DW             int                 // optionnal, default to 1000
	W              []CIDWidth          // optionnal
	DW2            [2]int              // optionnal, default to [ 880 −1000 ]
	W2             []CIDVerticalMetric // optionnal
	CIDToGIDMap    CIDToGIDMap         // optional
}

// Widths resolve the mapping from CID to glypth widths
// CID not found should be mapped to the default width
func (c CIDFontDictionary) Widths() map[CID]int {
	out := make(map[CID]int, len(c.W)) // at least
	for _, w := range c.W {
		w.Widths(out)
	}
	return out
}

func (c CIDFontDictionary) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	fD := pdf.CreateObject()
	pdf.WriteObject(c.FontDescriptor.pdfString(pdf, FontType0{}, fD), fD)
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
	if c.CIDToGIDMap != nil {
		b.fmt("/CIDToGIDMap %s", c.CIDToGIDMap.cidToGIDMapString(pdf))
	}
	b.fmt(">>")
	return b.String()
}

// Clone returns a deep copy of the CIDFontDictionary
func (c CIDFontDictionary) Clone() CIDFontDictionary {
	out := c
	out.FontDescriptor = c.FontDescriptor.Clone()
	if c.W != nil { // preserve deep equal
		out.W = make([]CIDWidth, len(c.W))
	}
	for i, w := range c.W {
		out.W[i] = w.Clone()
	}
	if c.W2 != nil { // preserve deep equal
		out.W2 = make([]CIDVerticalMetric, len(c.W2))
		for i, w := range c.W2 {
			out.W2[i] = w.Clone()
		}
	}
	if c.CIDToGIDMap != nil {
		out.CIDToGIDMap = c.CIDToGIDMap.Clone()
	}
	return out
}

type CIDSystemInfo struct {
	Registry   string // must be ASCII string
	Ordering   string // must be ASCII string
	Supplement int
}

// ToUnicodeCMapName construct a Unicode CMap name in the format `registry–ordering–UCS2`
func (c CIDSystemInfo) ToUnicodeCMapName() Name {
	return Name(fmt.Sprintf("%s-%s-UCS2", c.Registry, c.Ordering))
}

// returns a dictionary representation
func (c CIDSystemInfo) pdfString(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("<</Registry %s/Ordering %s/Supplement %d>>",
		pdf.EncodeString(c.Registry, ByteString, ref),
		pdf.EncodeString(c.Ordering, ByteString, ref), c.Supplement)
}

// CID is a character code that correspond to one glyph
// It will be obtained (from the bytes of a string) through a CMap, and will be use
// as index in a CIDFont object.
// It is not the same as an Unicode point.
type CID uint16 // See Table C.1 – Architectural limits for the 2^16 limit.

// CIDWidth groups the two ways of defining widths for CID :
// either CIDWidthRange or CIDWidthArray
type CIDWidth interface {
	// Widths accumulate the widths for each character, defined in user units
	Widths(map[CID]int)
	// String returns a PDF representation of the width
	String() string
	// Clone returns a deepcopy, preserving the concrete type
	Clone() CIDWidth
}

// CIDWidthRange is written in PDF as
//	c_first c_last w
type CIDWidthRange struct {
	First, Last CID
	Width       int
}

func (c CIDWidthRange) Widths(out map[CID]int) {
	for r := c.First; r <= c.Last; r++ {
		out[r] = c.Width
	}
}

func (c CIDWidthRange) String() string {
	return fmt.Sprintf("%d %d %d", c.First, c.Last, c.Width)
}

// Clone return a deep copy of `c`, with concrete type `CIDWidthRange`
func (c CIDWidthRange) Clone() CIDWidth { return c }

// CIDWidthArray is written in PDF as
//	start [ w_1 w_2 ... w_n ]
type CIDWidthArray struct {
	Start CID
	W     []int
}

func (c CIDWidthArray) Widths(out map[CID]int) {
	for i, w := range c.W {
		out[c.Start+CID(i)] = w
	}
}

func (c CIDWidthArray) String() string {
	return fmt.Sprintf("%d %s", c.Start, writeIntArray(c.W))
}

// Clone return a deep copy of `c`, with concrete type `CIDWidthArray`
func (c CIDWidthArray) Clone() CIDWidth {
	out := c
	out.W = append([]int(nil), c.W...) // nil to preserve deep equal
	return out
}

// VerticalMetric is found in PDF as 3 numbers
type VerticalMetric struct {
	Vertical int
	Position [2]int // position vector v [v_x, v_y]
}

func (v VerticalMetric) String() string {
	return fmt.Sprintf("%d %d %d", v.Vertical, v.Position[0], v.Position[1])
}

// CIDVerticalMetric groups the two ways of defining vertical displacement for CID
type CIDVerticalMetric interface {
	// VerticalMetrics accumulate the vertical metrics for each character, defined in user units
	VerticalMetrics(map[CID]VerticalMetric)
	// String returns a PDF representation of the width
	String() string
	// Clone returns a deepcopy, preserving the concrete type
	Clone() CIDVerticalMetric
}

// CIDVerticalMetricRange is written in PDF as
//	c_first c_last w v_w v_y
type CIDVerticalMetricRange struct {
	First, Last    CID
	VerticalMetric VerticalMetric
}

func (c CIDVerticalMetricRange) VerticalMetrics(out map[CID]VerticalMetric) {
	for r := c.First; r <= c.Last; r++ {
		out[r] = c.VerticalMetric
	}
}

func (c CIDVerticalMetricRange) String() string {
	return fmt.Sprintf("%d %d %s", c.First, c.Last, c.VerticalMetric)
}

// Clone return a deep copy of `c`, with concrete type `CIDVerticalMetricRange`
func (c CIDVerticalMetricRange) Clone() CIDVerticalMetric { return c }

// CIDVerticalMetricArray is written in PDF as
//	c [ w_1 w_2 ... w_n ]
type CIDVerticalMetricArray struct {
	Start     CID
	Verticals []VerticalMetric
}

func (c CIDVerticalMetricArray) VerticalMetrics(out map[CID]VerticalMetric) {
	for i, v := range c.Verticals {
		out[c.Start+CID(i)] = v
	}
}

func (c CIDVerticalMetricArray) String() string {
	chunks := make([]string, len(c.Verticals))
	for i, w := range c.Verticals {
		chunks[i] = w.String()
	}
	return fmt.Sprintf("%d [%s]", c.Start, strings.Join(chunks, " "))
}

// Clone return a deep copy of `c`, with concrete type `CIDVerticalMetricArray`
func (c CIDVerticalMetricArray) Clone() CIDVerticalMetric {
	out := c
	out.Verticals = append([]VerticalMetric(nil), c.Verticals...) // nil to preserve deep equal
	return out
}

type FontFile struct {
	Stream

	Length1 int
	Length2 int
	Length3 int
	Subtype Name // optional, one of Type1C for Type 1 compact fonts, CIDFontType0C for Type 0 compact CIDFonts, or OpenType
}

func (f *FontFile) pdfContent() (StreamHeader, []byte) {
	out := f.Stream.PDFCommonFields(true)
	out.Fields["Length1"] = strconv.Itoa(f.Length1)
	out.Fields["Length2"] = strconv.Itoa(f.Length2)
	out.Fields["Length3"] = strconv.Itoa(f.Length3)
	if f.Subtype != "" {
		out.Fields["Subtype"] = f.Subtype.String()
	}
	return out, f.Content
}

// Clone returns a deep copy of the font file.
func (f *FontFile) Clone() *FontFile {
	if f == nil {
		return nil
	}
	out := *f // shallow copy
	out.Stream = f.Stream.Clone()
	return &out
}
