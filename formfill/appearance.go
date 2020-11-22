package formfill

import (
	"fmt"
	"image/color"

	"github.com/benoitkugler/pdf/contents"
	"github.com/benoitkugler/pdf/model"
)

type graphicState struct {
	// fontDetails fonts.FontDetails // this is the font in use

	// This is the color in use
	//  colorDetails ColorDetails

	model.FontStyle

	xTLM    Fl // The x position of the text line matrix.
	yTLM    Fl // The y position of the text line matrix.
	leading Fl // The current text leading.
}

// Appearance is a temporary object
// used to build an appearance content stream
type Appearance struct {
	ops []contents.Operation

	bBox   model.Rectangle
	matrix model.Matrix
	// state  graphicState

	// stateList []graphicState
	resources model.ResourcesDict
}

func newAppearance(width, height Fl) Appearance {
	return Appearance{
		bBox: model.Rectangle{Urx: width, Ury: height},
		resources: model.ResourcesDict{
			Font: make(map[model.Name]*model.FontDict),
		},
	}
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

func (app *Appearance) setColorFill(c color.Color) {
	r, g, b := colorRGB(c)
	app.ops = append(app.ops, contents.OpSetFillRGBColor{R: r, G: g, B: b})
}

func (app *Appearance) setColorStroke(c color.Color) {
	r, g, b := colorRGB(c)
	app.ops = append(app.ops, contents.OpSetStrokeRGBColor{R: r, G: g, B: b})
}

// check is the font is in the map or generate a new name and add the font
func addFont(fonts map[model.Name]*model.FontDict, newFont *model.FontDict) model.Name {
	for name, f := range fonts {
		if f == newFont {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.Name(fmt.Sprintf("Font%d", len(fonts)))
	fonts[name] = newFont
	return name
}

// set the font and the size (in points) for the subsequent text writing.
func (ap *Appearance) setFontAndSize(font *model.FontDict, size Fl) {
	// ap.state.FontStyle.Size = size
	// switch bf := bf.(type) {
	// case *document.DocumentFont:
	// 	ap.state.fontDetails = fonts.FontDetails{FontName: "", IndirectReference: bf.RefFont, Font: bf}
	// default:
	// 	ap.state.fontDetails = writer.addSimple(bf)
	// }

	name := addFont(ap.resources.Font, font)
	ap.ops = append(ap.ops, contents.OpSetFont{Font: name, Size: size})
}

// sets the text leading parameter, which is measured in text space units.
// It specifies the vertical distance
// between the baselines of adjacent lines of text.
// func (ap *Appearance) setLeading(leading Fl) {
// 	ap.state.leading = leading
// 	ap.content.WriteString(fmt.Sprintf("%.3f TL ", leading))
// }

//

// func (ap Appearance) beginVariableText() {
// 	ap.content.WriteString("/Tx BMC ")
// }

// func (ap *Appearance) beginText() {
// 	ap.state.xTLM = 0
// 	ap.state.yTLM = 0
// 	ap.content.WriteString("BT ")
// }

// func (ap *Appearance) endText() {
// 	ap.state.xTLM = 0
// 	ap.state.yTLM = 0
// 	ap.content.WriteString("ET ")
// }

// // moves to the start of the next line, offset from the start of the current line.
// func (ap *Appearance) moveText(x, y Fl) {
// 	ap.state.xTLM += x
// 	ap.state.yTLM += y
// 	ap.content.WriteString(fmt.Sprintf("%.3f %.3f Td "))
// }

// // helper to insert into the content stream the `text`
// // converted to bytes according to the font's encoding.
// func (ap Appearance) encodeShow(text string) {
// 	if ap.state.fontDetails.Font == nil {
// 		panic("Font and size must be set before writing any text")
// 	}
// 	b := ap.state.fontDetails.convertToBytes(text)
// 	escapeString(b, content)
// }

// shows the `text`
// func (ap Appearance) showText(text string) {
// 	ap.encodeShow(text)
// 	ap.content.WriteString("Tj ")
// }

// // moves to the next line and shows text
// func (ap *Appearance) newlineShowText(text string) {
// 	ap.state.yTLM -= ap.state.leading
// 	ap.encodeShow(text)
// 	ap.content.WriteString("' ")
// }

// Changes the text matrix.
// Remark: this operation also initializes the current point position.</P>
// func (ap *Appearance) setTextMatrix(a, b, c, d, x, y Fl) {
// 	ap.state.xTLM = x
// 	ap.state.yTLM = y

// 	ap.content.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f Tm ", a, b, c, d, x, y))
// }

// func (ap Appearance) setTextMatrix2(x, y Fl) {
// 	ap.setTextMatrix(1, 0, 0, 1, x, y)
// }

// func (ap *Appearance) saveState() {
// 	ap.content.WriteString("q ")
// 	ap.stateList = append(ap.stateList, ap.state)
// }

// restores the graphic state.
// will panic if saveState and restoreState are not balanced.
// func (ap *Appearance) restoreState() {
// 	ap.content.WriteString("Q ")
// 	idx := len(ap.stateList) - 1
// 	if idx < 0 {
// 		panic("unbalanced save/restore state operators.")
// 	}
// 	ap.state = ap.stateList[idx]
// 	ap.stateList = ap.stateList[:idx]
// }
