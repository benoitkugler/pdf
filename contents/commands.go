// This package defines the commands used in
// PDF content stream objects.
// They can be chained to build an arbitrary content
// (see WriteOperations).
// Reciprocally, they can be obtained from a content
// by parsing it.
package contents

import (
	"bytes"
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

// assert interface conformance
var _ = map[string]Operation{
	// "\"":  OpMoveSetShowText{},
	"'": OpMoveShowText{},
	// "B":   OpFillStroke{},
	// "B*":  OpEOFillStroke{},
	// "BDC": OpBeginMarkedContent{},
	// "BI":  OpBeginImage{},
	"BMC": OpBeginMarkedContent{},
	"BT":  OpBeginText{},
	// "BX":  OpBeginIgnoreUndef{},
	// "CS":  OpSetStrokeColorSpace{},
	// "DP":  OpMarkPoint{},
	// "Do":  OpXObject{},
	// "EI":  OpEndImage{},
	"EMC": OpEndMarkedContent{},
	"ET":  OpEndText{},
	// "EX":  OpEndIgnoreUndef{},
	// "F":   OpFill{},
	// "G":   OpSetStrokeGray{},
	// "ID":  OpImageData{},
	// "J":   OpSetLineCap{},
	// "K":   OpSetStrokeCMYKColor{},
	// "M":   OpSetMiterLimit{},
	// "MP":  OpMarkPoint{},
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
	// "cs":  OpSetFillColorSpace{},
	"d": OpSetDash{},
	// "d0":  OpSetCharWidth{},
	// "d1":  OpSetCacheDevice{},
	"f": OpFill{},
	// "f*":  OpEOFill{},
	"g": OpSetFillGray{},
	// "gs":  OpSetExtGState{},
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
	// "ri":  OpSetRenderingIntent{},
	// "s":   OpCloseStroke{},
	// "sc":  OpSetFillColor{},
	// "scn": OpSetFillColorN{},
	// "sh":  OpShFill{},
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
	fmt.Fprintf(out, "%.3f  g", o.G)
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
func floatArray(as []Fl) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = fmt.Sprintf("%.3f", a)
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
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

// BMC
type OpBeginMarkedContent struct {
	Tag model.Name
}

func (o OpBeginMarkedContent) Add(out *bytes.Buffer) {
	fmt.Fprintf(out, "%s BMC", o.Tag)
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
	out.WriteString(model.EspaceByteString(o.Text) + "''")
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
