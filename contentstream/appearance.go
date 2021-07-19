package contentstream

import (
	"errors"
	"fmt"
	"image/color"
	"math"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
)

var (
	errNoFont     = errors.New("no font is currently selected")
	errUnbalanced = errors.New("unbalanced save/restore state operators.")
)

type GraphicState struct {
	Font     fonts.Font // the current usable font
	FontSize Fl

	XTLM    Fl // The x position of the text line matrix.
	YTLM    Fl // The y position of the text line matrix.
	Leading Fl // The current text leading.
}

// Appearance is a buffer of graphics operation,
// with a state. It provides convenient methods
// to ease the creation of a content stream.
// Once ready, it can be transformed to an XObjectForm.
type Appearance struct {
	ops []Operation

	bBox model.Rectangle
	// matrix    model.Matrix
	resources model.ResourcesDict

	State     GraphicState
	stateList []GraphicState
}

// NewAppearance setup the BBox and initialize the
// resources dictionary.
func NewAppearance(width, height Fl) Appearance {
	return Appearance{
		bBox: model.Rectangle{Urx: width, Ury: height},
		resources: model.ResourcesDict{
			Font:      make(map[model.ObjName]*model.FontDict),
			XObject:   make(map[model.ObjName]model.XObject),
			Shading:   make(map[model.ObjName]*model.ShadingDict),
			ExtGState: make(map[model.ObjName]*model.GraphicState),
			Pattern:   make(map[model.ObjName]model.Pattern),
		},
	}
}

// ToXFormObject write the appearance to a new object,
// and associate it the resources, which are shallow copied.
// The content is optionaly compressed with the Flater filter.
func (ap Appearance) ToXFormObject(compress bool) *model.XObjectForm {
	out := new(model.XObjectForm)
	out.BBox = ap.bBox
	// out.Matrix = ap.matrix
	out.Resources = ap.resources.ShallowCopy()
	out.Content = WriteOperations(ap.ops...)
	if compress {
		out.Content = sliceCompress(out.Content)
		out.Filter = model.Filters{{Name: model.Flate}}
	}
	return out
}

// ToPageObject return a page with a single Content,
// build from the appearance.
// The content is optionaly compressed with the Flater filter.
func (ap Appearance) ToPageObject(compress bool) *model.PageObject {
	fo := ap.ToXFormObject(compress)
	return &model.PageObject{
		Contents: []model.ContentStream{
			fo.ContentStream,
		},
		MediaBox:  &fo.BBox,
		Resources: &fo.Resources,
	}
}

// Ops adds one or more graphic command. Some commands usually need
// to also update the state: see the other methods.
func (ap *Appearance) Ops(op ...Operation) {
	ap.ops = append(ap.ops, op...)
}

func (app *Appearance) SetColorFill(c color.Color) {
	r, g, b := colorRGB(c)
	app.Ops(OpSetFillRGBColor{R: r, G: g, B: b})
}

func (app *Appearance) SetColorStroke(c color.Color) {
	r, g, b := colorRGB(c)
	app.Ops(OpSetStrokeRGBColor{R: r, G: g, B: b})
}

// check is the font is in the resources map or generate a new name and add the font
func (ap Appearance) addFont(newFont *model.FontDict) model.ObjName {
	for name, f := range ap.resources.Font {
		if f == newFont {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.ObjName(fmt.Sprintf("FT%d", len(ap.resources.Font)))
	ap.resources.Font[name] = newFont
	return name
}

// SetFontAndSize sets the font and the size (in points) for the subsequent text writing.
func (ap *Appearance) SetFontAndSize(font fonts.BuiltFont, size Fl) {
	ap.State.Font = font.Font // usable part
	ap.State.FontSize = size
	name := ap.addFont(font.Meta) // store and add the PDF name
	ap.Ops(OpSetFont{Font: name, Size: size})
}

// SetLeading sets the text leading parameter, which is measured in text space units.
// It specifies the vertical distance
// between the baselines of adjacent lines of text.
func (ap *Appearance) SetLeading(leading Fl) {
	ap.State.Leading = leading
	ap.Ops(OpSetTextLeading{L: leading})
}

// BeginVariableText starts a MarkedContent sequence of text
func (ap *Appearance) BeginVariableText() {
	ap.Ops(OpBeginMarkedContent{Tag: "Tx"})
}

// EndVariableText end a MarkedContent sequence of text
func (ap *Appearance) EndVariableText() {
	ap.Ops(OpEndMarkedContent{})
}

// BeginText starts the writing of text.
func (ap *Appearance) BeginText() {
	ap.State.XTLM = 0
	ap.State.YTLM = 0
	ap.Ops(OpBeginText{})
}

// EndText ends the writing of text
func (ap *Appearance) EndText() {
	ap.State.XTLM = 0
	ap.State.YTLM = 0
	ap.Ops(OpEndText{})
}

// MoveText moves to the start of the next line, offset from the start of the current line.
func (ap *Appearance) MoveText(x, y Fl) {
	ap.State.XTLM += x
	ap.State.YTLM += y
	ap.Ops(OpTextMove{X: x, Y: y})
}

// ShowText shows the `text`, after encoding it
// according to the current font.
// And error is returned (only) if a font has not been setup.
// A typical text drawing should apply the following methods ;
//	- BeginText
//	- SetFontAndSize
//	- ShowText
//	- EndText
func (ap *Appearance) ShowText(text string) error {
	if ap.State.Font == nil {
		return errNoFont
	}
	s := string(ap.State.Font.Encode([]rune(text)))
	ap.Ops(OpShowText{Text: s})
	return nil
}

// NewlineShowText moves to the next line and shows text.
func (ap *Appearance) NewlineShowText(text string) error {
	if ap.State.Font == nil {
		return errNoFont
	}
	ap.State.YTLM -= ap.State.Leading
	s := string(ap.State.Font.Encode([]rune(text)))
	ap.Ops(OpMoveShowText{Text: s})
	return nil
}

// SetTextMatrix changes the text matrix.
// This operation also initializes the current point position.
func (ap *Appearance) SetTextMatrix(a, b, c, d, x, y Fl) {
	ap.State.XTLM = x
	ap.State.YTLM = y
	ap.Ops(OpSetTextMatrix{Matrix: model.Matrix{a, b, c, d, x, y}})
}

// Saves the graphic state. SaveState and RestoreState must be balanced.
func (ap *Appearance) SaveState() {
	ap.Ops(OpSave{})
	ap.stateList = append(ap.stateList, ap.State)
}

// RestoreState restores the graphic state.
// An error is returned (only) if the call are not balanced.
func (ap *Appearance) RestoreState() error {
	idx := len(ap.stateList) - 1
	if idx < 0 {
		return errUnbalanced
	}
	ap.Ops(OpRestore{})
	ap.State = ap.stateList[idx]
	ap.stateList = ap.stateList[:idx]
	return nil
}

// check if the image or content is in the resources map or generate a new name and add the object
func (ap *Appearance) addXobject(xobj model.XObject) model.Name {
	for name, obj := range ap.resources.XObject {
		if obj == xobj {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.ObjName(fmt.Sprintf("XO%d", len(ap.resources.XObject)))
	ap.resources.XObject[name] = xobj
	return name
}

// AddXObject puts an image or an XObjectForm in the current page, at the given position,
// with the given dimentions.
// See `RenderingDims` for several ways of specifying image dimentions.
func (ap *Appearance) AddXObject(img model.XObject, x, y, width, height Fl) {
	imgName := ap.addXobject(img)
	ap.Ops(
		OpSave{},
		OpConcat{Matrix: model.Matrix{width, 0, 0, height, x, y}},
		OpXObject{XObject: imgName},
		OpRestore{},
	)
}

// approximate arc for border radius
const myArc = (4.0 / 3.0) * (math.Sqrt2 - 1.0)

// RoundedRectPath returns a rectangle path with rounded corners.
// The rectangle is of width `w` and height `h`. Its upper left corner is positioned at point (`x`, `y`).
// The radius for each corner are given by `rTL` (top-left), `rTR` (top-right)
// `rBR` (bottom-right), `rBL` (bottom-left) (0 means square corners)
func RoundedRectPath(x, y, w, h, rTL, rTR, rBR, rBL float64) []Operation {
	out := make([]Operation, 0, 4)

	out = append(out, OpMoveTo{X: x + rTL, Y: y})
	xc := x + w - rTR
	yc := y + rTR
	out = append(out, OpLineTo{X: xc, Y: y})
	if rTR > 0 {
		out = append(out, OpCubicTo{X1: xc + rTR*myArc, Y1: yc - rTR, X2: xc + rTR, Y2: yc - rTR*myArc, X3: xc + rTR, Y3: yc})
	}

	xc = x + w - rBR
	yc = y + h - rBR
	out = append(out, OpLineTo{X: x + w, Y: yc})
	if rBR > 0 {
		out = append(out, OpCubicTo{X1: xc + rBR, Y1: yc + rBR*myArc, X2: xc + rBR*myArc, Y2: yc + rBR, X3: xc, Y3: yc + rBR})
	}

	xc = x + rBL
	yc = y + h - rBL
	out = append(out, OpLineTo{X: xc, Y: y + h})
	if rBL != 0 {
		out = append(out, OpCubicTo{X1: xc - rBL*myArc, Y1: yc + rBL, X2: xc - rBL, Y2: yc + rBL*myArc, X3: xc - rBL, Y3: yc})
	}

	xc = x + rTL
	yc = y + rTL
	out = append(out, OpLineTo{X: x, Y: yc})
	if rTL != 0 {
		out = append(out, OpCubicTo{X1: xc - rTL, Y1: yc - rTL*myArc, X2: xc - rTL*myArc, Y2: yc - rTL, X3: xc, Y3: yc - rTL})
	}
	return out
}
