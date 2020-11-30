// This package defines the commands used in
// PDF content stream objects.
// They can be chained to build an arbitrary content
// (see WriteOperations).
// Reciprocally, they can be obtained from a content
// by parsing it, using for instance the 'parser' package.
package contents

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/benoitkugler/pdf/model"
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

type PropertyList interface {
	PDFString() string
}

// assert interface conformance
var _ = map[string]Operation{
	// "\"":  OpMoveSetShowText{},
	"'": OpMoveShowText{},
	// "B":   OpFillStroke{},
	// "B*":  OpEOFillStroke{},
	"BDC": OpBeginMarkedContent{},
	"BI":  OpBeginImage{}, // ID and EI are processed together with BI
	"BMC": OpBeginMarkedContent{},
	"BT":  OpBeginText{},
	// "BX":  OpBeginIgnoreUndef{},
	"CS":  OpSetStrokeColorSpace{},
	"DP":  OpMarkPoint{},
	"Do":  OpXObject{},
	"EMC": OpEndMarkedContent{},
	"ET":  OpEndText{},
	// "EX":  OpEndIgnoreUndef{},
	// "F":   OpFill{},
	// "G":   OpSetStrokeGray{},
	// "J":   OpSetLineCap{},
	// "K":   OpSetStrokeCMYKColor{},
	// "M":   OpSetMiterLimit{},
	"MP": OpMarkPoint{},
	"Q":  OpRestore{},
	"RG": OpSetStrokeRGBColor{},
	"S":  OpStroke{},
	// "SC":  OpSetStrokeColor{},
	// "SCN": OpSetStrokeColorN{},
	// "T*":  OpTextNextLine{},
	// "TD":  OpTextMoveSet{},
	// "TJ":  OpShowSpaceText{},
	"TL": OpSetTextLeading{},
	// "Tc":  OpSetCharSpacing{},
	"Td": OpTextMove{},
	"Tf": OpSetFont{},
	"Tj": OpShowText{},
	"Tm": OpSetTextMatrix{},
	// "Tr":  OpSetTextRender{},
	// "Ts":  OpSetTextRise{},
	// "Tw":  OpSetWordSpacing{},
	// "Tz":  OpSetHorizScaling{},
	"W": OpClip{},
	// "W*":  OpEOClip{},
	// "b":   OpCloseFillStroke{},
	// "b*":  OpCloseEOFillStroke{},
	// "c":   OpCurveTo{},
	// "cm":  OpConcat{},
	"cs": OpSetFillColorSpace{},
	"d":  OpSetDash{},
	// "d0":  OpSetCharWidth{},
	// "d1":  OpSetCacheDevice{},
	"f": OpFill{},
	// "f*":  OpEOFill{},
	"g":  OpSetFillGray{},
	"gs": OpSetExtGState{},
	// "h":   OpClosePath{},
	// "i":   OpSetFlat{},
	// "j":   OpSetLineJoin{},
	// "k":   OpSetFillCMYKColor{},
	"l":  OpLineTo{},
	"m":  OpMoveTo{},
	"n":  OpEndPath{},
	"q":  OpSave{},
	"re": OpRectangle{},
	"rg": OpSetFillRGBColor{},
	"ri": OpSetRenderingIntent{},
	// "s":   OpCloseStroke{},
	"sc":  OpSetFillColor{},
	"scn": OpSetFillColorN{},
	"sh":  OpShFill{},
	// "v":   OpCurveTo1{},
	"w": OpSetLineWidth{},
	// "y":   OpCurveTo{},
}

// rg
type OpSetFillRGBColor struct {
	R, G, B Fl
}

func (o OpSetFillRGBColor) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f %.3f %.3f rg", o.R, o.G, o.B)
}

// g
type OpSetFillGray struct {
	G Fl
}

func (o OpSetFillGray) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f g", o.G)
}

// RG
type OpSetStrokeRGBColor struct {
	R, G, B Fl
}

func (o OpSetStrokeRGBColor) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f %.3f %.3f RG", o.R, o.G, o.B)
}

// w
type OpSetLineWidth struct {
	W Fl
}

func (o OpSetLineWidth) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f w", o.W)
}

// d
type OpSetDash struct {
	Dash model.DashPattern
}

func (o OpSetDash) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "[%s] %.3f d", floatArray(o.Dash.Array), o.Dash.Phase)
}

// without the enclosing []
func floatArray(as []Fl) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = fmt.Sprintf("%f", a)
	}
	return strings.Join(b, " ")
}

// Tf
type OpSetFont struct {
	Font model.Name
	Size Fl
}

func (o OpSetFont) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%s %.3f Tf", o.Font, o.Size)
}

// TL
type OpSetTextLeading struct {
	L Fl
}

func (o OpSetTextLeading) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f TL", o.L)
}

// n
// OpEndPath is the same as new path.
type OpEndPath struct{}

func (o OpEndPath) Add(out *bytes.Buffer) {
	out.WriteByte('n')
}

// m
type OpMoveTo struct {
	X, Y Fl
}

func (o OpMoveTo) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f %.3f m", o.X, o.Y)
}

// l
type OpLineTo struct {
	X, Y Fl
}

func (o OpLineTo) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f %.3f l", o.X, o.Y)
}

// re
type OpRectangle struct {
	X, Y, W, H Fl
}

func (o OpRectangle) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f %.3f %.3f %.3f re", o.X, o.Y, o.W, o.H)
}

// f
type OpFill struct{}

func (o OpFill) Add(out *bytes.Buffer) {
	out.WriteByte('f')
}

// S
type OpStroke struct{}

func (o OpStroke) Add(out *bytes.Buffer) {
	out.WriteByte('S')
}

// BMC or BDC depending on Properties
type OpBeginMarkedContent struct {
	Tag        model.Name
	Properties PropertyList
}

func (o OpBeginMarkedContent) Add(out *bytes.Buffer) {
	if o.Properties == nil {
		fmt.Fprintf(out, "%s BMC", o.Tag)
	} else {
		fmt.Fprintf(out, "%s %s BDC", o.Tag, o.Properties.PDFString())
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
	fmt.Fprintf(out, "%.3f %.3f Td", o.X, o.Y)
}

// Tj
type OpShowText struct {
	Text string // unescaped
}

func (o OpShowText) Add(out *bytes.Buffer) {
	out.WriteString(model.EspaceByteString(o.Text) + "Tj")
}

// '
type OpMoveShowText struct {
	Text string // unescaped
}

func (o OpMoveShowText) Add(out *bytes.Buffer) {
	out.WriteString(model.EspaceByteString(o.Text) + "'")
}

// Tm
type OpSetTextMatrix struct {
	Matrix model.Matrix
}

func (o OpSetTextMatrix) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%.3f %.3f %.3f %.3f %.3f %.3f Tm",
		o.Matrix[0], o.Matrix[1], o.Matrix[2], o.Matrix[3], o.Matrix[4], o.Matrix[5])
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

// W
type OpClip struct{}

func (o OpClip) Add(out *bytes.Buffer) {
	out.WriteByte('W')
}

// CS
type OpSetStrokeColorSpace struct {
	ColorSpace model.Name
}

func (o OpSetStrokeColorSpace) Add(out *bytes.Buffer) {
	out.WriteString(o.ColorSpace.String() + " CS")
}

// cs
type OpSetFillColorSpace struct {
	ColorSpace model.Name
}

func (o OpSetFillColorSpace) Add(out *bytes.Buffer) {
	out.WriteString(o.ColorSpace.String() + " cs")
}

// gs
type OpSetExtGState struct {
	Dict model.Name
}

func (o OpSetExtGState) Add(out *bytes.Buffer) {
	out.WriteString(o.Dict.String() + " gs")
}

// sh
type OpShFill struct {
	Shading model.Name
}

func (o OpShFill) Add(out *bytes.Buffer) {
	out.WriteString(o.Shading.String() + " sh")
}

// sc
type OpSetFillColor struct {
	Color []Fl
}

func (o OpSetFillColor) Add(out *bytes.Buffer) {
	out.WriteString(floatArray(o.Color) + " sc")
}

// scn
type OpSetFillColorN struct {
	Color   []Fl
	Pattern model.Name // optional
}

func (o OpSetFillColorN) Add(out *bytes.Buffer) {
	var n string
	if o.Pattern != "" {
		n = o.Pattern.String()
	}
	out.WriteString(floatArray(o.Color) + n + " scn")
}

// Do
type OpXObject struct {
	XObject model.Name
}

func (o OpXObject) Add(out *bytes.Buffer) {
	out.WriteString(o.XObject.String() + " Do")
}

// ri
type OpSetRenderingIntent struct {
	Intent model.Name
}

func (o OpSetRenderingIntent) Add(out *bytes.Buffer) {
	out.WriteString(o.Intent.String() + " ri")
}

// MP or DP depending on Properties
type OpMarkPoint struct {
	Tag        model.Name
	Properties PropertyList // optional
}

func (o OpMarkPoint) Add(out *bytes.Buffer) {
	if o.Properties == nil {
		fmt.Fprintf(out, "%s MP", o.Tag)
	} else {
		fmt.Fprintf(out, "%s %s DP", o.Tag, o.Properties.PDFString())
	}
}

// ImageColorSpace is either:
// 	- a device color space
//	- a limited form of Indexed colour space whose base colour space is a device space
// 		and whose colour table is specified by a byte string
// 	- a name
type ImageColorSpace interface {
	isColorSpace()
	PDFString() string
}

func (ImageColorSpaceName) isColorSpace()    {}
func (ImageColorSpaceIndexed) isColorSpace() {}

// ImageColorSpaceName is a custom name or a device
type ImageColorSpaceName struct {
	model.ColorSpaceName
}

func (c ImageColorSpaceName) PDFString() string {
	return model.Name(c.ColorSpaceName).String()
}

// ImageColorSpaceIndexed is written in PDF as
// [/Indexed base hival lookup ]
type ImageColorSpaceIndexed struct {
	Base   model.ColorSpaceName // required
	Hival  uint8
	Lookup model.ColorTableBytes
}

func (c ImageColorSpaceIndexed) PDFString() string {
	return fmt.Sprintf("[/Indexed %s %d %s]",
		model.Name(c.Base), c.Hival, c.Lookup)
}

func (c ImageColorSpaceIndexed) ToColorSpace() model.ColorSpace {
	return model.ColorSpaceIndexed{Base: c.Base, Hival: c.Hival, Lookup: c.Lookup}
}

// BI ... ID ... EI
type OpBeginImage struct {
	Image      model.Image
	ColorSpace ImageColorSpace
}

func (o OpBeginImage) Add(out *bytes.Buffer) {
	out.WriteString("BI " + o.Image.PDFFields(true))
	if o.ColorSpace != nil {
		out.WriteString(" /CS " + o.ColorSpace.PDFString())
	}
	out.WriteString(" ID ") // one space
	out.Write(o.Image.Content)
	out.WriteString("EI")
}

// Metrics returns the number of color components and the number of bits for each.
// An error is returned if the color space can't be resolved from the resources dictionary.
func (img OpBeginImage) Metrics(res model.ResourcesDict) (comps, bits int, err error) {
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

func (img OpBeginImage) resolveColorSpace(res model.ResourcesDict) (model.ColorSpace, error) {
	switch cs := img.ColorSpace.(type) {
	case ImageColorSpaceName:
		// resolve the default color spaces
		var defa model.ColorSpace
		switch cs.ColorSpaceName {
		case model.ColorSpaceGray:
			defa = res.ColorSpace["DefaultGray"]
		case model.ColorSpaceRGB:
			defa = res.ColorSpace["DefaultRGB"]
		case model.ColorSpaceCMYK:
			defa = res.ColorSpace["DefaultCMYK"]
		default:
			c := res.ColorSpace[model.Name(cs.ColorSpaceName)]
			if c == nil {
				return nil, fmt.Errorf("missing color space for name %s", cs.ColorSpaceName)
			}
			return c, nil
		}
		if defa == nil { // use the "normal" Device CS
			return cs.ColorSpaceName, nil
		}
		return defa, nil
	case ImageColorSpaceIndexed:
		return cs.ToColorSpace(), nil
	default:
		return nil, errors.New("missing color space")
	}
}
