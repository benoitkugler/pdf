package formfill

import (
	"image/color"
	"strings"

	"github.com/benoitkugler/pdf/contents"
	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
)

// Port from the code from Paulo Soares (psoares@consiste.pt)

// Supports text, combo and list fields generating the correct appearances.
type TextField struct {
	BaseField

	// Holds value of property defaultText.
	defaultText string

	// Holds value of property choices.
	choices []string

	// Holds value of property choiceExports.
	choiceExports []string

	// Holds value of property choiceSelection.
	choiceSelection int

	topFirst int

	extraMarginLeft Fl
	extraMarginTop  Fl
}

func stringSize(s string, ft fonts.Font, size Fl) Fl {
	var out Fl
	for _, r := range s {
		out += ft.GetWidth(r, size)
	}
	return out
}

func (t TextField) buildAppearance(ufont fonts.BuiltFont, fontSize Fl) *model.XObjectForm {
	app := t.BaseField.getBorderAppearance()
	app.BeginVariableText()
	if t.text == "" {
		app.EndVariableText()
		return app.ToXFormObject()
	}

	fd := ufont.Desc()

	borderExtra := t.borderStyle == "B" || t.borderStyle == "I"
	h := t.box.Height() - t.borderWidth*2
	bw2 := t.borderWidth
	if borderExtra {
		h -= t.borderWidth * 2
		bw2 *= 2
	}
	h -= t.extraMarginTop
	offsetX := t.borderWidth
	if borderExtra {
		offsetX = 2 * t.borderWidth
	}
	offsetX = maxF(offsetX, 1)
	offX := minF(bw2, offsetX)

	app.SaveState()

	app.Op(contents.OpRectangle{X: offX, Y: offX, W: t.box.Width() - 2*offX, H: t.box.Height() - 2*offX})
	app.Op(contents.OpClip{})
	app.Op(contents.OpEndPath{})
	if t.textColor == nil {
		app.Op(contents.OpSetFillGray{})
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
		width := t.box.Width() - 3*offsetX - t.extraMarginLeft
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
			app.MoveText(t.extraMarginLeft+t.box.Width()-2*offsetX-wd, offsetY)
		case model.Centered:
			nt = strings.TrimSpace(nt)
			wd := stringSize(nt, ufont, usize)
			app.MoveText(t.extraMarginLeft+t.box.Width()/2-wd/2, offsetY)
		default:
			app.MoveText(t.extraMarginLeft+2*offsetX, offsetY)
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
				app.MoveText(t.extraMarginLeft+t.box.Width()-2*offsetX-wd-app.State.XTLM, 0)
			} else if t.alignment == model.Centered {
				nt = strings.TrimSpace(nt)
				wd := stringSize(nt, ufont, usize)
				app.MoveText(t.extraMarginLeft+t.box.Width()/2-wd/2-app.State.XTLM, 0)
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
				usize = (t.box.Width() - t.extraMarginLeft - 2*offsetX) / wd
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
		if maxL, _ := t.maxCharacterLength.(model.Int); (t.options&model.Comb) != 0 && maxL > 0 {
			textLen := min(int(maxL), len(ptextRunes))
			var position Fl
			if t.alignment == model.RightJustified {
				position = Fl(int(maxL) - textLen)
			} else if t.alignment == model.Centered {
				position = Fl(int(maxL)-textLen) / 2
			}
			step := (t.box.Width() - t.extraMarginLeft) / Fl(int(maxL))
			start := step/2 + position*step
			for k := 0; k < textLen; k++ {
				c := ptextRunes[k]
				wd := ufont.GetWidth(c, usize)
				app.SetTextMatrix(1, 0, 0, 1, t.extraMarginLeft+start-wd/2, offsetY-t.extraMarginTop)
				_ = app.ShowText(string(c)) // its clear font size was set
				start += step
			}
		} else {
			switch t.alignment {
			case model.RightJustified:
				wd := stringSize(ptext, ufont, usize)
				app.MoveText(t.extraMarginLeft+t.box.Width()-2*offsetX-wd, offsetY-t.extraMarginTop)
			case model.Centered:
				wd := stringSize(ptext, ufont, usize)
				app.MoveText(t.extraMarginLeft+t.box.Width()/2-wd/2, offsetY-t.extraMarginTop)
			default:
				app.MoveText(t.extraMarginLeft+2*offsetX, offsetY-t.extraMarginTop)
			}
			_ = app.ShowText(ptext) // its clear font size was set
		}
	}
	app.EndText()
	_ = app.RestoreState() // it's clear the call are balanced
	app.EndVariableText()
	return app.ToXFormObject()
}

func (tx *TextField) getListAppearance(ufont fonts.BuiltFont, fontSize Fl) *model.XObjectForm {
	app := tx.getBorderAppearance()
	app.BeginVariableText()
	if len(tx.choices) == 0 {
		app.EndVariableText()
		return app.ToXFormObject()
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

	app.Op(contents.OpRectangle{X: offsetX, Y: offsetX, W: tx.box.Width() - 2*offsetX, H: tx.box.Height() - 2*offsetX})
	app.Op(contents.OpClip{})
	app.Op(contents.OpEndPath{})
	mColor := tx.textColor
	if mColor == nil {
		mColor = color.Gray{}
	}
	app.SetColorFill(color.NRGBA{R: 10, G: 36, B: 106, A: 255})
	app.Op(contents.OpRectangle{X: offsetX, Y: offsetX + h - Fl(topChoice-first+1)*leading, W: tx.box.Width() - 2*offsetX, H: leading})
	app.Op(contents.OpFill{})
	app.BeginText()
	app.SetFontAndSize(ufont, usize)
	app.SetLeading(leading)
	app.MoveText(offsetX*2, offsetX+h-fd.FontBBox.Ury*usize/1000+leading)
	app.SetColorFill(mColor)
	for idx := first; idx < last; idx++ {
		if idx == topChoice {
			app.Op(contents.OpSetFillGray{G: 1})
			_ = app.NewlineShowText(tx.choices[idx]) // font was setup
			app.SetColorFill(mColor)
		} else {
			_ = app.NewlineShowText(tx.choices[idx]) // font was setup
		}
	}
	app.EndText()
	_ = app.RestoreState() // calls are balanced
	app.EndVariableText()
	return app.ToXFormObject()
}

//     /** Gets a new text field.
//      * @throws IOException on error
//      * @throws DocumentException on error
//      * @return a new text field
//      */
//     public PdfFormField getTextField() throws IOException, DocumentException {
//         if (maxCharacterLength <= 0)
//             t.options &= ~COMB;
//         if ((t.options & COMB) != 0)
//             t.options &= ~MULTILINE;
//         PdfFormField field = PdfFormField.createTextField(writer, false, false, maxCharacterLength);
//         field.setWidget(box, PdfAnnotation.HIGHLIGHT_INVERT);
//         switch (alignment) {
//             case Element.ALIGN_CENTER:
//                 field.setQuadding(PdfFormField.Q_CENTER);
//                 break;
//             case Element.ALIGN_RIGHT:
//                 field.setQuadding(PdfFormField.Q_RIGHT);
//                 break;
//         }
//         if (rotation != 0)
//             field.setMKRotation(rotation);
//         if (fieldName != nil) {
//             field.setFieldName(fieldName);
//             field.setValueAsString(text);
//             if (defaultText != nil)
//                 field.setDefaultValueAsString(defaultText);
//             if ((t.options & READ_ONLY) != 0)
//                 field.setFieldFlags(PdfFormField.FF_READ_ONLY);
//             if ((t.options & REQUIRED) != 0)
//                 field.setFieldFlags(PdfFormField.FF_REQUIRED);
//             if ((t.options & MULTILINE) != 0)
//                 field.setFieldFlags(PdfFormField.FF_MULTILINE);
//             if ((t.options & DO_NOT_SCROLL) != 0)
//                 field.setFieldFlags(PdfFormField.FF_DONOTSCROLL);
//             if ((t.options & PASSWORD) != 0)
//                 field.setFieldFlags(PdfFormField.FF_PASSWORD);
//             if ((t.options & FILE_SELECTION) != 0)
//                 field.setFieldFlags(PdfFormField.FF_FILESELECT);
//             if ((t.options & DO_NOT_SPELL_CHECK) != 0)
//                 field.setFieldFlags(PdfFormField.FF_DONOTSPELLCHECK);
//             if ((t.options & COMB) != 0)
//                 field.setFieldFlags(PdfFormField.FF_COMB);
//         }
//         field.setBorderStyle(new PdfBorderDictionary(borderWidth, borderStyle, new PdfDashPattern(3)));
//         PdfAppearance tp = buildAppearance();
//         field.setAppearance(PdfAnnotation.APPEARANCE_NORMAL, tp);
//         PdfAppearance da = (PdfAppearance)tp.getDuplicate();
//         da.setFontAndSize(getRealFont(), fontSize);
//         if (textColor == nil)
//             da.setGrayFill(0);
//         else
//             da.setColorFill(textColor);
//         field.setDefaultAppearanceString(da);
//         if (borderColor != nil)
//             field.setMKBorderColor(borderColor);
//         if (backgroundColor != nil)
//             field.setMKBackgroundColor(backgroundColor);
//         switch (visibility) {
//             case HIDDEN:
//                 field.setFlags(PdfAnnotation.FLAGS_PRINT | PdfAnnotation.FLAGS_HIDDEN);
//                 break;
//             case VISIBLE_BUT_DOES_NOT_PRINT:
//                 break;
//             case HIDDEN_BUT_PRINTABLE:
//                 field.setFlags(PdfAnnotation.FLAGS_PRINT | PdfAnnotation.FLAGS_NOVIEW);
//                 break;
//             default:
//                 field.setFlags(PdfAnnotation.FLAGS_PRINT);
//                 break;
//         }
//         return field;
//     }

//     /** Gets a new combo field.
//      * @throws IOException on error
//      * @throws DocumentException on error
//      * @return a new combo field
//      */
//     public PdfFormField getComboField() throws IOException, DocumentException {
//         return getChoiceField(false);
//     }

//     /** Gets a new list field.
//      * @throws IOException on error
//      * @throws DocumentException on error
//      * @return a new list field
//      */
//     public PdfFormField getListField() throws IOException, DocumentException {
//         return getChoiceField(true);
//     }

//     protected PdfFormField getChoiceField(boolean isList) throws IOException, DocumentException {
//         t.options &= (~MULTILINE) & (~COMB);
//         String uchoices[] = choices;
//         if (uchoices == nil)
//             uchoices = new String[0];
//         int topChoice = choiceSelection;
//         if (topChoice >= uchoices.length)
//             topChoice = uchoices.length - 1;
//         if (text == nil) text = ""; //fixed by Kazuya Ujihara (ujihara.jp)
//         if (topChoice >= 0)
//             text = uchoices[topChoice];
//         if (topChoice < 0)
//             topChoice = 0;
//         PdfFormField field = nil;
//         String mix[][] = nil;
//         if (choiceExports == nil) {
//             if (isList)
//                 field = PdfFormField.createList(writer, uchoices, topChoice);
//             else
//                 field = PdfFormField.createCombo(writer, (t.options & EDIT) != 0, uchoices, topChoice);
//         }
//         else {
//             mix = new String[uchoices.length][2];
//             for (int k = 0; k < mix.length; ++k)
//                 mix[k][0] = mix[k][1] = uchoices[k];
//             int top = minF(uchoices.length, choiceExports.length);
//             for (int k = 0; k < top; ++k) {
//                 if (choiceExports[k] != nil)
//                     mix[k][0] = choiceExports[k];
//             }
//             if (isList)
//                 field = PdfFormField.createList(writer, mix, topChoice);
//             else
//                 field = PdfFormField.createCombo(writer, (t.options & EDIT) != 0, mix, topChoice);
//         }
//         field.setWidget(box, PdfAnnotation.HIGHLIGHT_INVERT);
//         if (rotation != 0)
//             field.setMKRotation(rotation);
//         if (fieldName != nil) {
//             field.setFieldName(fieldName);
//             if (uchoices.length > 0) {
//                 if (mix != nil) {
//                     field.setValueAsString(mix[topChoice][0]);
//                     field.setDefaultValueAsString(mix[topChoice][0]);
//                 }
//                 else {
//                     field.setValueAsString(text);
//                     field.setDefaultValueAsString(text);
//                 }
//             }
//             if ((t.options & READ_ONLY) != 0)
//                 field.setFieldFlags(PdfFormField.FF_READ_ONLY);
//             if ((t.options & REQUIRED) != 0)
//                 field.setFieldFlags(PdfFormField.FF_REQUIRED);
//             if ((t.options & DO_NOT_SPELL_CHECK) != 0)
//                 field.setFieldFlags(PdfFormField.FF_DONOTSPELLCHECK);
//         }
//         field.setBorderStyle(new PdfBorderDictionary(borderWidth, borderStyle, new PdfDashPattern(3)));
//         PdfAppearance tp;
//         if (isList) {
//             tp = getListAppearance();
//             if (topFirst > 0)
//                 field.put(PdfName.TI, new PdfNumber(topFirst));
//         }
//         else
//             tp = buildAppearance();
//         field.setAppearance(PdfAnnotation.APPEARANCE_NORMAL, tp);
//         PdfAppearance da = (PdfAppearance)tp.getDuplicate();
//         da.setFontAndSize(getRealFont(), fontSize);
//         if (textColor == nil)
//             da.setGrayFill(0);
//         else
//             da.setColorFill(textColor);
//         field.setDefaultAppearanceString(da);
//         if (borderColor != nil)
//             field.setMKBorderColor(borderColor);
//         if (backgroundColor != nil)
//             field.setMKBackgroundColor(backgroundColor);
//         switch (visibility) {
//             case HIDDEN:
//                 field.setFlags(PdfAnnotation.FLAGS_PRINT | PdfAnnotation.FLAGS_HIDDEN);
//                 break;
//             case VISIBLE_BUT_DOES_NOT_PRINT:
//                 break;
//             case HIDDEN_BUT_PRINTABLE:
//                 field.setFlags(PdfAnnotation.FLAGS_PRINT | PdfAnnotation.FLAGS_NOVIEW);
//                 break;
//             default:
//                 field.setFlags(PdfAnnotation.FLAGS_PRINT);
//                 break;
//         }
//         return field;
//     }

//     /** Gets the default text.
//      * @return the default text
//      */
//     public String getDefaultText() {
//         return this.defaultText;
//     }

//     /** Sets the default text. It is only meaningful for text fields.
//      * @param defaultText the default text
//      */
//     public void setDefaultText(String defaultText) {
//         this.defaultText = defaultText;
//     }

//     /** Gets the choices to be presented to the user in list/combo
//      * fields.
//      * @return the choices to be presented to the user
//      */
//     public String[] getChoices() {
//         return this.choices;
//     }

//     /** Sets the choices to be presented to the user in list/combo
//      * fields.
//      * @param choices the choices to be presented to the user
//      */
//     public void setChoices(String[] choices) {
//         this.choices = choices;
//     }

//     /** Gets the export values in list/combo fields.
//      * @return the export values in list/combo fields
//      */
//     public String[] getChoiceExports() {
//         return this.choiceExports;
//     }

//     /** Sets the export values in list/combo fields. If this array
//      * is <CODE>nil</CODE> then the choice values will also be used
//      * as the export values.
//      * @param choiceExports the export values in list/combo fields
//      */
//     public void setChoiceExports(String[] choiceExports) {
//         this.choiceExports = choiceExports;
//     }

//     /** Gets the zero based index of the selected item.
//      * @return the zero based index of the selected item
//      */
//     public int getChoiceSelection() {
//         return this.choiceSelection;
//     }

//     /** Sets the zero based index of the selected item.
//      * @param choiceSelection the zero based index of the selected item
//      */
//     public void setChoiceSelection(int choiceSelection) {
//         this.choiceSelection = choiceSelection;
//     }

//     int getTopFirst() {
//         return topFirst;
//     }

//     /**
//      * Sets extra margins in text fields to better mimic the Acrobat layout.
//      * @param t.extraMarginLeft the extra marging left
//      * @param extraMarginTop the extra margin top
//      */
//     public void setExtraMargin(float t.extraMarginLeft, float extraMarginTop) {
//         this.extraMarginLeft = t.extraMarginLeft;
//         this.extraMarginTop = extraMarginTop;
//     }
