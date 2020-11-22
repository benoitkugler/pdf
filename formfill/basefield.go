package formfill

import (
	"image/color"
	"strings"

	"github.com/benoitkugler/pdf/contents"
	"github.com/benoitkugler/pdf/model"
)

// const (
// 	alignLeft = iota
// 	alignRight
// 	alignCenter
// )

type BaseField struct {
	box       model.Rectangle
	fieldName string
	text      string

	textColor       color.Color
	backgroundColor color.Color

	borderStyle model.Name
	borderWidth Fl
	borderColor color.Color

	alignment uint8
	rotation  int
	options   model.FormFlag // options flag

	maxCharacterLength model.MaybeInt // value of property maxCharacterLength

	font     *model.FontDict
	fontSize Fl
}

const brightScale = 0.7

func darker(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(Fl(r) * brightScale), G: uint8(Fl(g) * brightScale), B: uint8(Fl(b) * brightScale), A: uint8(a)}
}

func (b BaseField) drawTopFrame(app *Appearance) {
	app.ops = append(app.ops, contents.OpMoveTo{X: b.borderWidth, Y: b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.borderWidth, Y: b.box.Height() - b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.box.Width() - b.borderWidth, Y: b.box.Height() - b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.box.Width() - 2*b.borderWidth, Y: b.box.Height() - 2*b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: 2 * b.borderWidth, Y: b.box.Height() - 2*b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: 2 * b.borderWidth, Y: 2 * b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.borderWidth, Y: b.borderWidth})
	app.ops = append(app.ops, contents.OpFill{})
}

func (b BaseField) drawBottomFrame(app *Appearance) {
	app.ops = append(app.ops, contents.OpMoveTo{X: b.borderWidth, Y: b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.box.Width() - b.borderWidth, Y: b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.box.Width() - b.borderWidth, Y: b.box.Height() - b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.box.Width() - 2*b.borderWidth, Y: b.box.Height() - 2*b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.box.Width() - 2*b.borderWidth, Y: 2 * b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: 2 * b.borderWidth, Y: 2 * b.borderWidth})
	app.ops = append(app.ops, contents.OpLineTo{X: b.borderWidth, Y: b.borderWidth})
	app.ops = append(app.ops, contents.OpFill{})
}

func (b BaseField) getBorderAppearance() Appearance {
	app := newAppearance(b.box.Width(), b.box.Height())
	switch b.rotation {
	case 90:
		app.ops = append(app.ops, contents.OpSetTextMatrix{Matrix: model.Matrix{0, 1, -1, 0, b.box.Height(), 0}})
	case 180:
		app.ops = append(app.ops, contents.OpSetTextMatrix{Matrix: model.Matrix{-1, 0, 0, -1, b.box.Width(), b.box.Height()}})
	case 270:
		app.ops = append(app.ops, contents.OpSetTextMatrix{Matrix: model.Matrix{0, -1, 1, 0, 0, b.box.Width()}})
	}
	// background
	if b.backgroundColor != nil {
		app.setColorFill(b.backgroundColor)
		app.ops = append(app.ops, contents.OpRectangle{X: 0, Y: 0, W: b.box.Width(), H: b.box.Height()})
		app.ops = append(app.ops, contents.OpFill{})
	}
	// border
	switch b.borderStyle {
	case "U":
		if b.borderWidth != 0 && b.borderColor != nil {
			app.setColorStroke(b.borderColor)
			app.ops = append(app.ops, contents.OpSetLineWidth{W: b.borderWidth})
			app.ops = append(app.ops, contents.OpMoveTo{X: 0, Y: b.borderWidth / 2})
			app.ops = append(app.ops, contents.OpLineTo{X: b.box.Width(), Y: b.borderWidth / 2})
			app.ops = append(app.ops, contents.OpStroke{})
		}
	case "B":
		if b.borderWidth != 0 && b.borderColor != nil {
			app.setColorStroke(b.borderColor)
			app.ops = append(app.ops, contents.OpSetLineWidth{W: b.borderWidth})
			app.ops = append(app.ops, contents.OpRectangle{X: b.borderWidth / 2, Y: b.borderWidth / 2, W: b.box.Width() - b.borderWidth, H: b.box.Height() - b.borderWidth})
			app.ops = append(app.ops, contents.OpStroke{})
		}
		// beveled
		var actual color.Color = color.White
		if b.backgroundColor != nil {
			actual = b.backgroundColor
		}
		app.ops = append(app.ops, contents.OpSetFillGray{G: 1})
		b.drawTopFrame(&app)
		app.setColorFill(darker(actual))
		b.drawBottomFrame(&app)
	case "I":
		if b.borderWidth != 0 && b.borderColor != nil {
			app.setColorStroke(b.borderColor)
			app.ops = append(app.ops, contents.OpSetLineWidth{W: b.borderWidth})
			app.ops = append(app.ops, contents.OpRectangle{X: b.borderWidth / 2, Y: b.borderWidth / 2, W: b.box.Width() - b.borderWidth, H: b.box.Height() - b.borderWidth})
			app.ops = append(app.ops, contents.OpStroke{})
		}
		// inset
		app.ops = append(app.ops, contents.OpSetFillGray{G: 0.5})
		b.drawTopFrame(&app)
		app.ops = append(app.ops, contents.OpSetFillGray{G: 0.75})
		b.drawBottomFrame(&app)
	default:
		if b.borderWidth != 0 && b.borderColor != nil {
			if b.borderStyle == "D" {
				app.ops = append(app.ops, contents.OpSetDash{Dash: model.DashPattern{Array: []Fl{3}, Phase: 0}})
			}
			app.setColorStroke(b.borderColor)
			app.ops = append(app.ops, contents.OpSetLineWidth{W: b.borderWidth})
			app.ops = append(app.ops, contents.OpRectangle{X: b.borderWidth / 2, Y: b.borderWidth / 2,
				W: b.box.Width() - b.borderWidth, H: b.box.Height() - b.borderWidth})
			app.ops = append(app.ops, contents.OpStroke{})
			if m, ok := b.maxCharacterLength.(model.Int); (b.options&model.Comb) != 0 && (ok && m > 1) {
				step := b.box.Width() / Fl(m)
				yb := b.borderWidth / 2
				yt := b.box.Height() - b.borderWidth/2
				for k := 1; k < int(m); k++ {
					x := step * Fl(k)
					app.ops = append(app.ops, contents.OpMoveTo{X: x, Y: yb})
					app.ops = append(app.ops, contents.OpLineTo{X: x, Y: yt})
				}
				app.ops = append(app.ops, contents.OpStroke{})
			}
		}
	}
	return app
}

// // TODO: support for more font ?
// // for now Helvetica is always returned
// func (b BaseField) getRealFont() fonts.Font {
// 	return type1.Helvetica
// }

func getHardBreaks(text string) (arr []string) {
	cs := []rune(text)
	var buf strings.Builder
	for k := 0; k < len(cs); k++ {
		c := cs[k]
		if c == '\r' {
			if k+1 < len(cs) && cs[k+1] == '\n' {
				k++
			}
			arr = append(arr, buf.String())
			buf.Reset()
		} else if c == '\n' {
			arr = append(arr, buf.String())
			buf.Reset()
		} else {
			buf.WriteRune(c)
		}
	}
	arr = append(arr, buf.String())
	return arr
}

func breakLines(breaks []string, font *model.FontDict, fontSize, width Fl) (lines []string) {
	var buf []rune
	for _, break_ := range breaks {
		buf = buf[:0]
		var w Fl
		cs := []rune(break_)
		// 0 inline first, 1 inline, 2 spaces
		state := 0
		lastspace := -1
		refk := 0
		for k := 0; k < len(cs); k++ {
			c := cs[k]
			switch state {
			case 0:
				w += font.GetWidth(c, fontSize)
				buf = append(buf, c)
				if w > width {
					w = 0
					if len(buf) > 1 {
						k--
						buf = buf[:len(buf)-1]
					}
					lines = append(lines, string(buf))
					buf = buf[:0]
					refk = k
					if c == ' ' {
						state = 2
					} else {
						state = 1
					}
				} else {
					if c != ' ' {
						state = 1
					}
				}
			case 1:
				w += font.GetWidth(c, fontSize)
				buf = append(buf, c)
				if c == ' ' {
					lastspace = k
				}
				if w > width {
					w = 0
					if lastspace >= 0 {
						k = lastspace
						buf = buf[:lastspace-refk]
						lines = append(lines, strings.TrimRight(string(buf), " "))
						buf = buf[:0]
						refk = k
						lastspace = -1
						state = 2
					} else {
						if len(buf) > 1 {
							k--
							buf = buf[:len(buf)-1]
						}
						lines = append(lines, string(buf))
						buf = buf[:0]
						refk = k
						if c == ' ' {
							state = 2
						}
					}
				}
			case 2:
				if c != ' ' {
					w = 0
					k--
					state = 1
				}
			}
		}
		lines = append(lines, strings.TrimRight(string(buf), " "))
	}
	return lines
}
