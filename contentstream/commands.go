// This package defines the commands used in
// PDF content stream objects.
// They can be chained to build an arbitrary content
// (see `WriteOperations` and the higher level `Appearance` object).
// Reciprocally, they can be obtained from a content
// by parsing it, using for instance the 'parser' package.
package contentstream

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	pdfFonts "github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/textlayout/fonts"
)

type Fl = model.Fl

// Operation is a command and its related arguments.
type Operation interface {
	Add(out *bytes.Buffer)
}

// WriteOperations concatenate the given operations.
func WriteOperations(ops ...Operation) []byte {
	var out bytes.Buffer
	for _, op := range ops {
		op.Add(&out)
		out.WriteByte(' ')
	}
	return out.Bytes()
}

// PropertyList should be either a Name (refering to the resources dict)
// or dict object.
type PropertyList interface {
	contentStreamString() string
}

type PropertyListName model.ObjName

func (n PropertyListName) contentStreamString() string {
	return model.ObjName(n).String()
}

// PropertyListDict is a dictionary, where indirect
// references and streams are not allowed.
type PropertyListDict model.ObjDict

func (p PropertyListDict) contentStreamString() string {
	return model.ObjDict(p).Write(nil, 0)
}

// assert interface conformance
var _ = map[string]Operation{
	"\"":  OpMoveSetShowText{},
	"'":   OpMoveShowText{},
	"B":   OpFillStroke{},
	"B*":  OpEOFillStroke{},
	"BDC": OpBeginMarkedContent{},
	// ID and EI are processed together with BI
	"BI":  OpBeginImage{},
	"BMC": OpBeginMarkedContent{},
	"BT":  OpBeginText{},
	"BX":  OpBeginIgnoreUndef{},
	"CS":  OpSetStrokeColorSpace{},
	"DP":  OpMarkPoint{},
	"Do":  OpXObject{},
	"EMC": OpEndMarkedContent{},
	"ET":  OpEndText{},
	"EX":  OpEndIgnoreUndef{},
	"F":   OpFill{},
	"G":   OpSetStrokeGray{},
	"J":   OpSetLineCap{},
	"K":   OpSetStrokeCMYKColor{},
	"M":   OpSetMiterLimit{},
	"MP":  OpMarkPoint{},
	"Q":   OpRestore{},
	"RG":  OpSetStrokeRGBColor{},
	"S":   OpStroke{},
	"SC":  OpSetStrokeColor{},
	"SCN": OpSetStrokeColorN{},
	"T*":  OpTextNextLine{},
	"TD":  OpTextMoveSet{},
	"TJ":  OpShowSpaceText{},
	"TL":  OpSetTextLeading{},
	"Tc":  OpSetCharSpacing{},
	"Td":  OpTextMove{},
	"Tf":  OpSetFont{},
	"Tj":  OpShowText{},
	"Tm":  OpSetTextMatrix{},
	"Tr":  OpSetTextRender{},
	"Ts":  OpSetTextRise{},
	"Tw":  OpSetWordSpacing{},
	"Tz":  OpSetHorizScaling{},
	"W":   OpClip{},
	"W*":  OpEOClip{},
	"b":   OpCloseFillStroke{},
	"b*":  OpCloseEOFillStroke{},
	"c":   OpCubicTo{},
	"cm":  OpConcat{},
	"cs":  OpSetFillColorSpace{},
	"d":   OpSetDash{},
	"d0":  OpSetCharWidth{},
	"d1":  OpSetCacheDevice{},
	"f":   OpFill{},
	"f*":  OpEOFill{},
	"g":   OpSetFillGray{},
	"gs":  OpSetExtGState{},
	"h":   OpClosePath{},
	"i":   OpSetFlat{},
	"j":   OpSetLineJoin{},
	"k":   OpSetFillCMYKColor{},
	"l":   OpLineTo{},
	"m":   OpMoveTo{},
	"n":   OpEndPath{},
	"q":   OpSave{},
	"re":  OpRectangle{},
	"rg":  OpSetFillRGBColor{},
	"ri":  OpSetRenderingIntent{},
	"s":   OpCloseStroke{},
	"sc":  OpSetFillColor{},
	"scn": OpSetFillColorN{},
	"sh":  OpShFill{},
	"v":   OpCurveTo1{},
	"w":   OpSetLineWidth{},
	"y":   OpCurveTo{},
}

// BX
type OpBeginIgnoreUndef struct{}

// EX
func (o OpBeginIgnoreUndef) Add(out *bytes.Buffer) { out.WriteString("BX") }

type OpEndIgnoreUndef struct{}

func (o OpEndIgnoreUndef) Add(out *bytes.Buffer) { out.WriteString("EX") }

// g
type OpSetFillGray struct {
	G Fl
}

func (o OpSetFillGray) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g g", o.G)
}

// G
type OpSetStrokeGray OpSetFillGray

func (o OpSetStrokeGray) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g G", o.G)
}

// rg
type OpSetFillRGBColor struct {
	R, G, B Fl
}

func (o OpSetFillRGBColor) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g %g rg", o.R, o.G, o.B)
}

// RG
type OpSetStrokeRGBColor OpSetFillRGBColor

func (o OpSetStrokeRGBColor) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g %g RG", o.R, o.G, o.B)
}

// k
type OpSetFillCMYKColor struct {
	C, M, Y, K Fl
}

func (o OpSetFillCMYKColor) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g %g %g k", o.C, o.M, o.Y, o.K)
}

// K
type OpSetStrokeCMYKColor OpSetFillCMYKColor

func (o OpSetStrokeCMYKColor) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g %g %g K", o.C, o.M, o.Y, o.K)
}

// J
type OpSetLineCap struct {
	Style uint8
}

func (o OpSetLineCap) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%d J", o.Style)
}

// j
type OpSetLineJoin struct {
	Style uint8
}

func (o OpSetLineJoin) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%d j", o.Style)
}

// M
type OpSetMiterLimit struct {
	Limit Fl
}

func (o OpSetMiterLimit) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g M", o.Limit)
}

// w
type OpSetLineWidth struct {
	W Fl
}

func (o OpSetLineWidth) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g w", o.W)
}

// i
type OpSetFlat struct {
	Flatness Fl // between 0 to 100
}

func (o OpSetFlat) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.2f i", o.Flatness)
}

// d
type OpSetDash struct {
	Dash model.DashPattern
}

func (o OpSetDash) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "[%s] %g d", floatArray(o.Dash.Array...), o.Dash.Phase)
}

// without the enclosing []
func floatArray(as ...Fl) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = fmt.Sprintf("%g", a)
	}
	return strings.Join(b, " ")
}

// Tf
type OpSetFont struct {
	Font model.ObjName
	Size Fl
}

func (o OpSetFont) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%s %g Tf", o.Font, o.Size)
}

// TL
type OpSetTextLeading struct {
	L Fl
}

func (o OpSetTextLeading) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g TL", o.L)
}

// Tc
type OpSetCharSpacing struct {
	CharSpace Fl
}

func (o OpSetCharSpacing) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g Tc", o.CharSpace)
}

// Tw
type OpSetWordSpacing struct {
	WordSpace Fl
}

func (o OpSetWordSpacing) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g Tw", o.WordSpace)
}

// Tz
type OpSetHorizScaling struct {
	Scale Fl
}

func (o OpSetHorizScaling) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g Tz", o.Scale)
}

// Tr
type OpSetTextRender struct {
	Render Fl
}

func (o OpSetTextRender) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g Tr", o.Render)
}

// Ts
type OpSetTextRise struct {
	Rise Fl
}

func (o OpSetTextRise) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g Ts", o.Rise)
}

// n
// OpEndPath is the same as new path.
type OpEndPath struct{}

func (o OpEndPath) Add(out *bytes.Buffer) { out.WriteByte('n') }

// m
type OpMoveTo struct {
	X, Y Fl
}

func (o OpMoveTo) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g m", o.X, o.Y)
}

// l
type OpLineTo struct {
	X, Y Fl
}

func (o OpLineTo) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g l", o.X, o.Y)
}

// c
type OpCubicTo struct {
	// P1 et P2 are the control points
	X1, Y1, X2, Y2, X3, Y3 Fl
}

func (o OpCubicTo) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.X1, o.Y1, o.X2, o.Y2, o.X3, o.Y3) + " c")
}

// v
type OpCurveTo1 struct {
	// P2 is the control point
	X2, Y2, X3, Y3 Fl
}

func (o OpCurveTo1) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.X2, o.Y2, o.X3, o.Y3) + " v")
}

// y
type OpCurveTo struct {
	// P1 is the control point
	X1, Y1, X3, Y3 Fl
}

func (o OpCurveTo) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.X1, o.Y1, o.X3, o.Y3) + " y")
}

// re
type OpRectangle struct {
	X, Y, W, H Fl
}

func (o OpRectangle) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g %g %g re", o.X, o.Y, o.W, o.H)
}

// f
type OpFill struct{}

func (o OpFill) Add(out *bytes.Buffer) { out.WriteByte('f') }

// f*
type OpEOFill struct{}

func (o OpEOFill) Add(out *bytes.Buffer) { out.WriteString("f*") }

// S
type OpStroke struct{}

func (o OpStroke) Add(out *bytes.Buffer) { out.WriteByte('S') }

// B
type OpFillStroke struct{}

func (o OpFillStroke) Add(out *bytes.Buffer) { out.WriteByte('B') }

// B*
type OpEOFillStroke struct{}

func (o OpEOFillStroke) Add(out *bytes.Buffer) { out.WriteString("B*") }

// b
type OpCloseFillStroke struct{}

func (o OpCloseFillStroke) Add(out *bytes.Buffer) { out.WriteByte('b') }

// b*
type OpCloseEOFillStroke struct{}

func (o OpCloseEOFillStroke) Add(out *bytes.Buffer) { out.WriteString("b*") }

// BMC or BDC depending on Properties
type OpBeginMarkedContent struct {
	Properties PropertyList
	Tag        model.ObjName
}

func (o OpBeginMarkedContent) Add(out *bytes.Buffer) {
	if o.Properties == nil {
		fmt.Fprintf(out, "%s BMC", o.Tag)
	} else {
		fmt.Fprintf(out, "%s %s BDC", o.Tag, o.Properties.contentStreamString())
	}
}

// EMC
type OpEndMarkedContent struct{}

func (o OpEndMarkedContent) Add(out *bytes.Buffer) {
	out.WriteString("EMC")
}

// BT
type OpBeginText struct{}

func (o OpBeginText) Add(out *bytes.Buffer) {
	out.WriteString("BT")
}

// ET
type OpEndText struct{}

func (o OpEndText) Add(out *bytes.Buffer) {
	out.WriteString("ET")
}

// Td
type OpTextMove struct {
	X, Y Fl
}

func (o OpTextMove) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g Td", o.X, o.Y)
}

// TD
type OpTextMoveSet OpTextMove

func (o OpTextMoveSet) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g TD", o.X, o.Y)
}

// T*
type OpTextNextLine struct{}

func (o OpTextNextLine) Add(out *bytes.Buffer) { out.WriteString("T*") }

// Tj
type OpShowText struct {
	Text string // unescaped
}

func (o OpShowText) Add(out *bytes.Buffer) {
	out.WriteString(model.EscapeByteString([]byte(o.Text)) + "Tj")
}

// TJ - OpShowSpaceText enables font kerning
type OpShowSpaceText struct {
	// Texts store a "normalized version" of texts and spaces
	// SpaceSubtractedAfter fields of 0 are ignored.
	Texts []pdfFonts.TextSpaced
}

func (o OpShowSpaceText) Add(out *bytes.Buffer) {
	out.WriteByte('[')
	for _, ts := range o.Texts {
		out.WriteString(model.EscapeByteString([]byte(ts.CharCodes)))
		if ts.SpaceSubtractedAfter != 0 {
			fmt.Fprintf(out, "%d", ts.SpaceSubtractedAfter)
		}
	}
	out.WriteString("]TJ")
}

// TJ - OpShowSpaceGlyphs enables font kerning
type SpacedGlyph struct {
	SpaceSubtractedBefore int
	GID                   fonts.GID // will be hex encoded in the content stream
	SpaceSubtractedAfter  int
}

// TJ - OpShowSpaceGlyph is the same as OpShowSpaceText
// but with input specified as glyph number.
// This should be used in conjonction with a font using
// an identity encoding
type OpShowSpaceGlyph struct {
	Glyphs []SpacedGlyph
}

func (o OpShowSpaceGlyph) Add(out *bytes.Buffer) {
	out.WriteByte('[')
	for _, ts := range o.Glyphs {
		if ts.SpaceSubtractedBefore != 0 {
			fmt.Fprintf(out, "%d ", ts.SpaceSubtractedBefore)
		}
		out.WriteString(fmt.Sprintf("<%04x>", ts.GID))
		if ts.SpaceSubtractedAfter != 0 {
			fmt.Fprintf(out, " %d ", ts.SpaceSubtractedAfter)
		}
	}
	out.WriteString("]TJ")
}

// '
type OpMoveShowText struct {
	Text string // unescaped
}

func (o OpMoveShowText) Add(out *bytes.Buffer) {
	out.WriteString(model.EscapeByteString([]byte(o.Text)) + "'")
}

// \"
type OpMoveSetShowText struct {
	Text                          string // unescaped
	WordSpacing, CharacterSpacing Fl
}

func (o OpMoveSetShowText) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%g %g %s \"", o.WordSpacing, o.CharacterSpacing, model.EscapeByteString([]byte(o.Text)))
}

// Tm
type OpSetTextMatrix struct {
	Matrix model.Matrix
}

func (o OpSetTextMatrix) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.Matrix[:]...) + " Tm")
}

// cm
type OpConcat struct {
	Matrix model.Matrix
}

func (o OpConcat) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.Matrix[:]...) + " cm")
}

// Q
type OpRestore struct{}

func (o OpRestore) Add(out *bytes.Buffer) {
	out.WriteByte('Q')
}

// q
type OpSave struct{}

func (o OpSave) Add(out *bytes.Buffer) {
	out.WriteByte('q')
}

// s
type OpCloseStroke struct{}

func (o OpCloseStroke) Add(out *bytes.Buffer) { out.WriteByte('s') }

// h
type OpClosePath struct{}

func (o OpClosePath) Add(out *bytes.Buffer) { out.WriteByte('h') }

// W
type OpClip struct{}

func (o OpClip) Add(out *bytes.Buffer) { out.WriteByte('W') }

// W*
type OpEOClip struct{}

func (o OpEOClip) Add(out *bytes.Buffer) { out.WriteString("W*") }

// CS
type OpSetStrokeColorSpace struct {
	ColorSpace model.ObjName // either a ColorSpaceName, or a name of a resource
}

func (o OpSetStrokeColorSpace) Add(out *bytes.Buffer) {
	out.WriteString(o.ColorSpace.String() + " CS")
}

// cs
type OpSetFillColorSpace OpSetStrokeColorSpace

func (o OpSetFillColorSpace) Add(out *bytes.Buffer) {
	out.WriteString(o.ColorSpace.String() + " cs")
}

// gs
type OpSetExtGState struct {
	Dict model.ObjName
}

func (o OpSetExtGState) Add(out *bytes.Buffer) {
	out.WriteString(o.Dict.String() + " gs")
}

// sh
type OpShFill struct {
	Shading model.ObjName
}

func (o OpShFill) Add(out *bytes.Buffer) {
	out.WriteString(o.Shading.String() + " sh")
}

// sc
type OpSetFillColor struct {
	Color []Fl
}

func (o OpSetFillColor) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.Color...) + " sc")
}

// SC
type OpSetStrokeColor OpSetFillColor

func (o OpSetStrokeColor) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.Color...) + " SC")
}

// scn
type OpSetFillColorN struct {
	Pattern model.ObjName // optional
	Color   []Fl
}

func (o OpSetFillColorN) Add(out *bytes.Buffer) {
	var n string
	if o.Pattern != "" {
		n = o.Pattern.String()
	}
	out.WriteString(floatArray(o.Color...) + n + " scn")
}

// SCN
type OpSetStrokeColorN OpSetFillColorN

func (o OpSetStrokeColorN) Add(out *bytes.Buffer) {
	var n string
	if o.Pattern != "" {
		n = o.Pattern.String()
	}
	out.WriteString(floatArray(o.Color...) + n + " SCN")
}

// Do
type OpXObject struct {
	XObject model.ObjName
}

func (o OpXObject) Add(out *bytes.Buffer) {
	out.WriteString(o.XObject.String() + " Do")
}

// d0
type OpSetCharWidth struct {
	WX, WY int // glyph units
}

func (o OpSetCharWidth) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%d %d d0", o.WX, o.WY)
}

// d1
type OpSetCacheDevice struct {
	WX, WY, LLX, LLY, URX, URY int // glyph units
}

func (o OpSetCacheDevice) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%d %d %d %d %d %d d1",
		o.WX, o.WY, o.LLX, o.LLY, o.URX, o.URY)
}

// ri
type OpSetRenderingIntent struct {
	Intent model.ObjName
}

func (o OpSetRenderingIntent) Add(out *bytes.Buffer) {
	out.WriteString(o.Intent.String() + " ri")
}

// MP or DP depending on Properties
type OpMarkPoint struct {
	Properties PropertyList // optional
	Tag        model.ObjName
}

func (o OpMarkPoint) Add(out *bytes.Buffer) {
	if o.Properties == nil {
		fmt.Fprintf(out, "%s MP", o.Tag)
	} else {
		fmt.Fprintf(out, "%s %s DP", o.Tag, o.Properties.contentStreamString())
	}
}

// ImageColorSpace is either:
// 	- a device color space
//	- a limited form of Indexed colour space whose base colour space is a device space
// 		and whose colour table is specified by a byte string
// 	- a name
type ImageColorSpace interface {
	isColorSpace()
	Write() string
}

func (ImageColorSpaceName) isColorSpace()    {}
func (ImageColorSpaceIndexed) isColorSpace() {}

// ImageColorSpaceName is a custom name or a device
type ImageColorSpaceName struct {
	model.ColorSpaceName
}

func (c ImageColorSpaceName) Write() string {
	return model.ObjName(c.ColorSpaceName).String()
}

// ImageColorSpaceIndexed is written in PDF as
// [/Indexed base hival lookup ]
type ImageColorSpaceIndexed struct {
	Base   model.ColorSpaceName // required, must be a Device CS
	Lookup model.ColorTableBytes
	Hival  uint8
}

func (c ImageColorSpaceIndexed) Write() string {
	return fmt.Sprintf("[/Indexed %s %d %s]",
		model.ObjName(c.Base), c.Hival, c.Lookup)
}

func (c ImageColorSpaceIndexed) ToColorSpace() model.ColorSpace {
	return model.ColorSpaceIndexed{Base: c.Base, Hival: c.Hival, Lookup: c.Lookup}
}

// BI ... ID ... EI
type OpBeginImage struct {
	ColorSpace ImageColorSpace
	Image      model.Image
}

func (o OpBeginImage) Add(out *bytes.Buffer) {
	out.WriteString("BI ")
	for k, v := range o.Image.PDFFields(true).Fields {
		out.WriteString(k.String() + " " + v)
	}
	if o.ColorSpace != nil {
		out.WriteString(" /CS " + o.ColorSpace.Write())
	}
	out.WriteString(" ID ") // one space
	out.Write(o.Image.Content)
	out.WriteString("EI")
}

// Metrics returns the number of color components and the number of bits for each.
// An error is returned if the color space can't be resolved from the resources dictionary.
func (img OpBeginImage) Metrics(res model.ResourcesColorSpace) (comps, bits int, err error) {
	bits = int(img.Image.BitsPerComponent)
	if img.Image.ImageMask {
		bits = 1
	}
	colorSpace, err := img.resolveColorSpace(res)
	if err != nil {
		return 0, 0, err
	}
	// get decode map
	if len(img.Image.Decode) == 0 {
		comps = colorSpace.NbColorComponents()
	} else {
		comps = len(img.Image.Decode) / 2
	}
	return comps, bits, nil
}

func (img OpBeginImage) resolveColorSpace(resources model.ResourcesColorSpace) (model.ColorSpace, error) {
	switch cs := img.ColorSpace.(type) {
	case ImageColorSpaceName:
		return resources.Resolve(model.ObjName(cs.ColorSpaceName))
	case ImageColorSpaceIndexed:
		return cs.ToColorSpace(), nil
	default:
		return nil, errors.New("missing color space")
	}
}
