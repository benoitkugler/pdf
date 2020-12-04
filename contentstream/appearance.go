package contentstream

import (
	"errors"
	"fmt"
	"image/color"

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

	// XObjectForm fields
	BBox      model.Rectangle
	Matrix    model.Matrix
	Resources model.ResourcesDict

	State     GraphicState
	stateList []GraphicState
}

func NewAppearance(width, height Fl) Appearance {
	return Appearance{
		BBox: model.Rectangle{Urx: width, Ury: height},
		Resources: model.ResourcesDict{
			Font: make(map[model.ObjName]*model.FontDict),
		},
	}
}

// ToXFormObject write the appearance to a new object,
// and associate it the resources.
// The content is not filtred
func (ap Appearance) ToXFormObject() *model.XObjectForm {
	out := new(model.XObjectForm)
	out.BBox = ap.BBox
	out.Matrix = ap.Matrix
	out.Resources = ap.Resources
	out.Content = WriteOperations(ap.ops...)
	return out
}

// Op adds a command. Some commands usually need
// to also update the state: see the other methods.
func (ap *Appearance) Op(op Operation) {
	ap.ops = append(ap.ops, op)
}

func clamp(ch, a uint32) Fl {
	if ch < 0 {
		return 0
	}
	if ch > a {
		return 1
	}
	return Fl(ch) / Fl(a)
}

func colorRGB(c color.Color) (r, g, b Fl) {
	cr, cg, cb, ca := c.RGBA()
	return clamp(cr, ca), clamp(cg, ca), clamp(cb, ca)
}

func (app *Appearance) SetColorFill(c color.Color) {
	r, g, b := colorRGB(c)
	app.Op(OpSetFillRGBColor{R: r, G: g, B: b})
}

func (app *Appearance) SetColorStroke(c color.Color) {
	r, g, b := colorRGB(c)
	app.Op(OpSetStrokeRGBColor{R: r, G: g, B: b})
}

// check is the font is in the map or generate a new name and add the font
func (ap Appearance) addFont(newFont *model.FontDict) model.ObjName {
	for name, f := range ap.Resources.Font {
		if f == newFont {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.ObjName(fmt.Sprintf("Font%d", len(ap.Resources.Font)))
	ap.Resources.Font[name] = newFont
	return name
}

// SetFontAndSize sets the font and the size (in points) for the subsequent text writing.
func (ap *Appearance) SetFontAndSize(font fonts.BuiltFont, size Fl) {
	ap.State.Font = font.Font // usable part
	ap.State.FontSize = size
	name := ap.addFont(font.Meta) // store and add the PDF name
	ap.Op(OpSetFont{Font: name, Size: size})
}

// SetLeading sets the text leading parameter, which is measured in text space units.
// It specifies the vertical distance
// between the baselines of adjacent lines of text.
func (ap *Appearance) SetLeading(leading Fl) {
	ap.State.Leading = leading
	ap.Op(OpSetTextLeading{L: leading})
}

// BeginVariableText starts a MarkedContent sequence of text
func (ap *Appearance) BeginVariableText() {
	ap.Op(OpBeginMarkedContent{Tag: "Tx"})
}

// EndVariableText end a MarkedContent sequence of text
func (ap *Appearance) EndVariableText() {
	ap.Op(OpEndMarkedContent{})
}

// BeginText starts the writing of text.
func (ap *Appearance) BeginText() {
	ap.State.XTLM = 0
	ap.State.YTLM = 0
	ap.Op(OpBeginText{})
}

// EndText ends the writing of text
func (ap *Appearance) EndText() {
	ap.State.XTLM = 0
	ap.State.YTLM = 0
	ap.Op(OpEndText{})
}

// MoveText moves to the start of the next line, offset from the start of the current line.
func (ap *Appearance) MoveText(x, y Fl) {
	ap.State.XTLM += x
	ap.State.YTLM += y
	ap.Op(OpTextMove{X: x, Y: y})
}

// ShowText shows the `text`, after encoding it
// according to the current font.
// And error is returned (only) if a font has not been setup.
func (ap *Appearance) ShowText(text string) error {
	if ap.State.Font == nil {
		return errNoFont
	}
	s := string(ap.State.Font.Encode([]rune(text)))
	ap.Op(OpShowText{Text: s})
	return nil
}

// NewlineShowText moves to the next line and shows text.
func (ap *Appearance) NewlineShowText(text string) error {
	if ap.State.Font == nil {
		return errNoFont
	}
	ap.State.YTLM -= ap.State.Leading
	s := string(ap.State.Font.Encode([]rune(text)))
	ap.Op(OpMoveShowText{Text: s})
	return nil
}

// SetTextMatrix changes the text matrix.
// This operation also initializes the current point position.
func (ap *Appearance) SetTextMatrix(a, b, c, d, x, y Fl) {
	ap.State.XTLM = x
	ap.State.YTLM = y
	ap.Op(OpSetTextMatrix{Matrix: model.Matrix{a, b, c, d, x, y}})
}

// Saves the graphic state. SaveState and RestoreState must be balanced.
func (ap *Appearance) SaveState() {
	ap.Op(OpSave{})
	ap.stateList = append(ap.stateList, ap.State)
}

// RestoreState restores the graphic state.
// An error is returned (only) if the call are not balanced.
func (ap *Appearance) RestoreState() error {
	idx := len(ap.stateList) - 1
	if idx < 0 {
		return errUnbalanced
	}
	ap.Op(OpRestore{})
	ap.State = ap.stateList[idx]
	ap.stateList = ap.stateList[:idx]
	return nil
}
