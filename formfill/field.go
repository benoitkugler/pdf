package formfill

// Port from the code from Paulo Soares (psoares@consiste.pt)

import (
	"image/color"
	"strings"

	"github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
)

// Supports text, combo and list fields generating the correct appearances.
type fieldAppearanceBuilder struct {
	box  model.Rectangle
	text string

	textColor       color.Color
	backgroundColor color.Color

	borderStyle model.ObjName
	borderWidth Fl
	borderColor color.Color

	alignment model.Quadding
	rotation  int
	options   model.FormFlag // options flag

	maxCharacterLength model.MaybeInt // value of property maxCharacterLength

	// Holds value of property choices.
	choices []string

	// Holds value of property choiceSelection.
	choiceSelection int

	topFirst int
}

const brightScale = 0.7

func darker(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(Fl(r) * brightScale), G: uint8(Fl(g) * brightScale), B: uint8(Fl(b) * brightScale), A: uint8(a)}
}

func (b fieldAppearanceBuilder) drawTopFrame(app *contentstream.Appearance) {
	app.Ops(
		contentstream.OpMoveTo{X: b.borderWidth, Y: b.borderWidth},
		contentstream.OpLineTo{X: b.borderWidth, Y: b.box.Height() - b.borderWidth},
		contentstream.OpLineTo{X: b.box.Width() - b.borderWidth, Y: b.box.Height() - b.borderWidth},
		contentstream.OpLineTo{X: b.box.Width() - 2*b.borderWidth, Y: b.box.Height() - 2*b.borderWidth},
		contentstream.OpLineTo{X: 2 * b.borderWidth, Y: b.box.Height() - 2*b.borderWidth},
		contentstream.OpLineTo{X: 2 * b.borderWidth, Y: 2 * b.borderWidth},
		contentstream.OpLineTo{X: b.borderWidth, Y: b.borderWidth},
		contentstream.OpFill{},
	)
}

func (b fieldAppearanceBuilder) drawBottomFrame(app *contentstream.Appearance) {
	app.Ops(
		contentstream.OpMoveTo{X: b.borderWidth, Y: b.borderWidth},
		contentstream.OpLineTo{X: b.box.Width() - b.borderWidth, Y: b.borderWidth},
		contentstream.OpLineTo{X: b.box.Width() - b.borderWidth, Y: b.box.Height() - b.borderWidth},
		contentstream.OpLineTo{X: b.box.Width() - 2*b.borderWidth, Y: b.box.Height() - 2*b.borderWidth},
		contentstream.OpLineTo{X: b.box.Width() - 2*b.borderWidth, Y: 2 * b.borderWidth},
		contentstream.OpLineTo{X: 2 * b.borderWidth, Y: 2 * b.borderWidth},
		contentstream.OpLineTo{X: b.borderWidth, Y: b.borderWidth},
		contentstream.OpFill{},
	)
}

func (b fieldAppearanceBuilder) getBorderAppearance() contentstream.Appearance {
	app := contentstream.NewAppearance(model.Rectangle{Llx: 0, Lly: 0, Urx: b.box.Width(), Ury: b.box.Height()})
	switch b.rotation {
	case 90:
		app.SetTextMatrix(0, 1, -1, 0, b.box.Height(), 0)
	case 180:
		app.SetTextMatrix(-1, 0, 0, -1, b.box.Width(), b.box.Height())
	case 270:
		app.SetTextMatrix(0, -1, 1, 0, 0, b.box.Width())
	}
	// background
	if b.backgroundColor != nil {
		app.SetColorFill(b.backgroundColor)
		app.Ops(contentstream.OpRectangle{X: 0, Y: 0, W: b.box.Width(), H: b.box.Height()})
		app.Ops(contentstream.OpFill{})
	}
	// border
	switch b.borderStyle {
	case "U":
		if b.borderWidth != 0 && b.borderColor != nil {
			app.SetColorStroke(b.borderColor)
			app.Ops(
				contentstream.OpSetLineWidth{W: b.borderWidth},
				contentstream.OpMoveTo{X: 0, Y: b.borderWidth / 2},
				contentstream.OpLineTo{X: b.box.Width(), Y: b.borderWidth / 2},
				contentstream.OpStroke{},
			)
		}
	case "B":
		if b.borderWidth != 0 && b.borderColor != nil {
			app.SetColorStroke(b.borderColor)
			app.Ops(
				contentstream.OpSetLineWidth{W: b.borderWidth},
				contentstream.OpRectangle{X: b.borderWidth / 2, Y: b.borderWidth / 2, W: b.box.Width() - b.borderWidth, H: b.box.Height() - b.borderWidth},
				contentstream.OpStroke{},
			)
		}
		// beveled
		var actual color.Color = color.White
		if b.backgroundColor != nil {
			actual = b.backgroundColor
		}
		app.Ops(contentstream.OpSetFillGray{G: 1})
		b.drawTopFrame(&app)
		app.SetColorFill(darker(actual))
		b.drawBottomFrame(&app)
	case "I":
		if b.borderWidth != 0 && b.borderColor != nil {
			app.SetColorStroke(b.borderColor)
			app.Ops(
				contentstream.OpSetLineWidth{W: b.borderWidth},
				contentstream.OpRectangle{X: b.borderWidth / 2, Y: b.borderWidth / 2, W: b.box.Width() - b.borderWidth, H: b.box.Height() - b.borderWidth},
				contentstream.OpStroke{},
			)
		}
		// inset
		app.Ops(contentstream.OpSetFillGray{G: 0.5})
		b.drawTopFrame(&app)
		app.Ops(contentstream.OpSetFillGray{G: 0.75})
		b.drawBottomFrame(&app)
	default:
		if b.borderWidth != 0 && b.borderColor != nil {
			if b.borderStyle == "D" {
				app.Ops(contentstream.OpSetDash{Dash: model.DashPattern{Array: []Fl{3}, Phase: 0}})
			}
			app.SetColorStroke(b.borderColor)
			app.Ops(contentstream.OpSetLineWidth{W: b.borderWidth})
			app.Ops(contentstream.OpRectangle{
				X: b.borderWidth / 2, Y: b.borderWidth / 2,
				W: b.box.Width() - b.borderWidth, H: b.box.Height() - b.borderWidth,
			})
			app.Ops(contentstream.OpStroke{})
			if m, ok := b.maxCharacterLength.(model.ObjInt); (b.options&model.Comb) != 0 && (ok && m > 1) {
				step := b.box.Width() / Fl(m)
				yb := b.borderWidth / 2
				yt := b.box.Height() - b.borderWidth/2
				for k := 1; k < int(m); k++ {
					x := step * Fl(k)
					app.Ops(contentstream.OpMoveTo{X: x, Y: yb})
					app.Ops(contentstream.OpLineTo{X: x, Y: yt})
				}
				app.Ops(contentstream.OpStroke{})
			}
		}
	}
	return app
}

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

func breakLines(breaks []string, font fonts.Font, fontSize, width Fl) (lines []string) {
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

// extra margins could be use in text fields to better mimic the Acrobat layout.
const (
	extraMarginLeft = 0
	extraMarginTop  = 0
)

func stringSize(s string, ft fonts.Font, size Fl) Fl {
	var out Fl
	for _, r := range s {
		out += ft.GetWidth(r, size)
	}
	return out
}

func (t fieldAppearanceBuilder) buildAppearance(ufont fonts.BuiltFont, fontSize Fl) *model.XObjectForm {
	app := t.getBorderAppearance()
	app.BeginVariableText()
	if t.text == "" {
		app.EndVariableText()
		return app.ToXFormObject(true)
	}

	fd := ufont.Desc()

	borderExtra := t.borderStyle == "B" || t.borderStyle == "I"
	h := t.box.Height() - t.borderWidth*2
	bw2 := t.borderWidth
	if borderExtra {
		h -= t.borderWidth * 2
		bw2 *= 2
	}
	h -= extraMarginTop
	offsetX := t.borderWidth
	if borderExtra {
		offsetX = 2 * t.borderWidth
	}
	offsetX = maxF(offsetX, 1)
	offX := minF(bw2, offsetX)

	app.SaveState()

	app.Ops(contentstream.OpRectangle{X: offX, Y: offX, W: t.box.Width() - 2*offX, H: t.box.Height() - 2*offX})
	app.Ops(contentstream.OpClip{})
	app.Ops(contentstream.OpEndPath{})
	if t.textColor == nil {
		app.Ops(contentstream.OpSetFillGray{})
	} else {
		app.SetColorFill(t.textColor)
	}
	app.BeginVariableText()
	ptext := t.text // fixed by Kazuya Ujihara (ujihara.jp)
	ptextRunes := []rune(t.text)
	if (t.options & model.Password) != 0 {
		ptext = strings.Repeat("*", len(ptextRunes))
	}
	if (t.options & model.Multiline) != 0 {
		usize := fontSize
		width := t.box.Width() - 3*offsetX - extraMarginLeft
		breaks := getHardBreaks(ptext)
		lines := breaks
		factor := (fd.FontBBox.Urx - fd.FontBBox.Lly) / 1000
		if usize == 0 {
			usize = h / Fl(len(breaks)) / factor
			if usize > 4 {
				if usize > 12 {
					usize = 12
				}
				step := maxF((usize-4)/10, 0.2)
				for ; usize > 4; usize -= step {
					lines = breakLines(breaks, ufont, usize, width)
					if Fl(len(lines))*usize*factor <= h {
						break
					}
				}
			}
			if usize <= 4 {
				usize = 4
				lines = breakLines(breaks, ufont, usize, width)
			}
		} else {
			lines = breakLines(breaks, ufont, usize, width)
		}
		app.SetFontAndSize(ufont, usize)
		app.SetLeading(usize * factor)
		offsetY := offsetX + h - fd.FontBBox.Ury*usize/1000
		nt := lines[0]
		switch t.alignment {
		case model.RightJustified:
			wd := stringSize(nt, ufont, usize)
			app.MoveText(extraMarginLeft+t.box.Width()-2*offsetX-wd, offsetY)
		case model.Centered:
			nt = strings.TrimSpace(nt)
			wd := stringSize(nt, ufont, usize)
			app.MoveText(extraMarginLeft+t.box.Width()/2-wd/2, offsetY)
		default:
			app.MoveText(extraMarginLeft+2*offsetX, offsetY)
		}
		_ = app.ShowText(nt) // its clear font size was set
		maxline := int(h/usize/factor) + 1
		if maxline > len(lines) {
			maxline = len(lines)
		}
		for k := 1; k < maxline; k++ {
			nt := lines[k]
			if t.alignment == model.RightJustified {
				wd := stringSize(nt, ufont, usize)
				app.MoveText(extraMarginLeft+t.box.Width()-2*offsetX-wd-app.State.XTLM, 0)
			} else if t.alignment == model.Centered {
				nt = strings.TrimSpace(nt)
				wd := stringSize(nt, ufont, usize)
				app.MoveText(extraMarginLeft+t.box.Width()/2-wd/2-app.State.XTLM, 0)
			}
			app.NewlineShowText(nt)
		}
	} else {
		usize := fontSize
		if usize == 0 {
			maxCalculatedSize := h / ((fd.FontBBox.Urx - fd.FontBBox.Lly) / 1000)
			wd := stringSize(ptext, ufont, 1)
			if wd == 0 {
				usize = maxCalculatedSize
			} else {
				usize = (t.box.Width() - extraMarginLeft - 2*offsetX) / wd
			}
			if usize > maxCalculatedSize {
				usize = maxCalculatedSize
			}
			if usize < 4 {
				usize = 4
			}
		}
		app.SetFontAndSize(ufont, usize)
		offsetY := offX + ((t.box.Height()-2*offX)-(fd.Ascent*usize/1000))/2
		if offsetY < offX {
			offsetY = offX
		}
		if offsetY-offX < -(fd.Descent * usize / 1000) {
			ny := -(fd.Descent * usize / 1000) + offX
			dy := t.box.Height() - offX - (fd.Ascent * usize / 1000)
			offsetY = minF(ny, maxF(offsetY, dy))
		}
		if maxL, _ := t.maxCharacterLength.(model.ObjInt); (t.options&model.Comb) != 0 && maxL > 0 {
			textLen := min(int(maxL), len(ptextRunes))
			var position Fl
			if t.alignment == model.RightJustified {
				position = Fl(int(maxL) - textLen)
			} else if t.alignment == model.Centered {
				position = Fl(int(maxL)-textLen) / 2
			}
			step := (t.box.Width() - extraMarginLeft) / Fl(int(maxL))
			start := step/2 + position*step
			for k := 0; k < textLen; k++ {
				c := ptextRunes[k]
				wd := ufont.GetWidth(c, usize)
				app.SetTextMatrix(1, 0, 0, 1, extraMarginLeft+start-wd/2, offsetY-extraMarginTop)
				_ = app.ShowText(string(c)) // its clear font size was set
				start += step
			}
		} else {
			switch t.alignment {
			case model.RightJustified:
				wd := stringSize(ptext, ufont, usize)
				app.MoveText(extraMarginLeft+t.box.Width()-2*offsetX-wd, offsetY-extraMarginTop)
			case model.Centered:
				wd := stringSize(ptext, ufont, usize)
				app.MoveText(extraMarginLeft+t.box.Width()/2-wd/2, offsetY-extraMarginTop)
			default:
				app.MoveText(extraMarginLeft+2*offsetX, offsetY-extraMarginTop)
			}
			_ = app.ShowText(ptext) // its clear font size was set
		}
	}
	app.EndText()
	_ = app.RestoreState() // it's clear the call are balanced
	app.EndVariableText()
	return app.ToXFormObject(true)
}

func (tx *fieldAppearanceBuilder) getListAppearance(ufont fonts.BuiltFont, fontSize Fl) (*model.XObjectForm, int) {
	app := tx.getBorderAppearance()
	app.BeginVariableText()
	if len(tx.choices) == 0 {
		app.EndVariableText()
		return app.ToXFormObject(true), tx.topFirst
	}
	topChoice := tx.choiceSelection
	if topChoice >= len(tx.choices) {
		topChoice = len(tx.choices) - 1
	}
	if topChoice < 0 {
		topChoice = 0
	}

	fd := ufont.Desc()

	usize := fontSize
	if usize == 0 {
		usize = 12
	}
	borderExtra := tx.borderStyle == "B" || tx.borderStyle == "I"
	h := tx.box.Height() - tx.borderWidth*2
	if borderExtra {
		h -= tx.borderWidth * 2
	}
	offsetX := tx.borderWidth
	if borderExtra {
		offsetX *= 2
	}
	leading := fd.FontBBox.Urx*usize/1000 - fd.FontBBox.Lly*usize/1000
	maxFit := int(h/leading) + 1
	first := 0
	last := 0
	last = topChoice + maxFit/2 + 1
	first = last - maxFit
	if first < 0 {
		last += first
		first = 0
	}
	//        first = topChoice;
	last = first + maxFit
	if last > len(tx.choices) {
		last = len(tx.choices)
	}
	tx.topFirst = first

	app.SaveState()

	app.Ops(contentstream.OpRectangle{X: offsetX, Y: offsetX, W: tx.box.Width() - 2*offsetX, H: tx.box.Height() - 2*offsetX})
	app.Ops(contentstream.OpClip{})
	app.Ops(contentstream.OpEndPath{})
	mColor := tx.textColor
	if mColor == nil {
		mColor = color.Gray{}
	}
	app.SetColorFill(color.NRGBA{R: 10, G: 36, B: 106, A: 255})
	app.Ops(contentstream.OpRectangle{X: offsetX, Y: offsetX + h - Fl(topChoice-first+1)*leading, W: tx.box.Width() - 2*offsetX, H: leading})
	app.Ops(contentstream.OpFill{})
	app.BeginText()
	app.SetFontAndSize(ufont, usize)
	app.SetLeading(leading)
	app.MoveText(offsetX*2, offsetX+h-fd.FontBBox.Ury*usize/1000+leading)
	app.SetColorFill(mColor)
	for idx := first; idx < last; idx++ {
		if idx == topChoice {
			app.Ops(contentstream.OpSetFillGray{G: 1})
			_ = app.NewlineShowText(tx.choices[idx]) // font was setup
			app.SetColorFill(mColor)
		} else {
			_ = app.NewlineShowText(tx.choices[idx]) // font was setup
		}
	}
	app.EndText()
	_ = app.RestoreState() // calls are balanced
	app.EndVariableText()
	return app.ToXFormObject(true), tx.topFirst
}
