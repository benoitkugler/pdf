package formfill

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

// func (t TextField) getAppearance() Appearance {
// 	app := t.BaseField.getBorderAppearance()
// 	app.beginVariableText()
// 	if t.text == "" {
// 		app.endVariableText()
// 		return app
// 	}

// 	ufont := t.getRealFont()

// 	borderExtra := t.borderStyle == borderStyleBeveled || t.borderStyle == borderStyleInset
// 	h := t.box.Height() - t.borderWidth*2
// 	bw2 := t.borderWidth
// 	if borderExtra {
// 		h -= t.borderWidth * 2
// 		bw2 *= 2
// 	}
// 	h -= t.extraMarginTop
// 	offsetX := t.borderWidth
// 	if borderExtra {
// 		offsetX = 2 * t.borderWidth
// 	}
// 	offsetX = math.Max(offsetX, 1)
// 	offX := math.Min(bw2, offsetX)
// 	app.saveState()
// 	app.rectangle(offX, offX, t.box.Width()-2*offX, t.box.Height()-2*offX)
// 	app.clip()
// 	app.newPath()
// 	if t.textColor == nil {
// 		app.setGrayFill(0)
// 	} else {
// 		app.setColorFill(t.textColor)
// 	}
// 	app.beginText()
// 	ptext := t.text // fixed by Kazuya Ujihara (ujihara.jp)
// 	ptextRunes := []rune(t.text)
// 	if (t.options & password) != 0 {
// 		ptext = strings.Repeat("*", len(ptextRunes))
// 	}
// 	if (t.options & multiline) != 0 {
// 		usize := t.fontSize
// 		width := t.box.Width() - 3*offsetX - t.extraMarginLeft
// 		breaks := getHardBreaks(ptext)
// 		lines := breaks
// 		factor := ufont.GetFontDescriptor(fonts.BBOXURY, 1) - ufont.GetFontDescriptor(fonts.BBOXLLY, 1)
// 		if usize == 0 {
// 			usize = h / Fl(len(breaks)) / factor
// 			if usize > 4 {
// 				if usize > 12 {
// 					usize = 12
// 				}
// 				step := math.Max((usize-4)/10, 0.2)
// 				for ; usize > 4; usize -= step {
// 					lines = breakLines(breaks, ufont, usize, width)
// 					if Fl(len(lines))*usize*factor <= h {
// 						break
// 					}
// 				}
// 			}
// 			if usize <= 4 {
// 				usize = 4
// 				lines = breakLines(breaks, ufont, usize, width)
// 			}
// 		} else {
// 			lines = breakLines(breaks, ufont, usize, width)
// 		}
// 		app.setFontAndSize(ufont, usize)
// 		app.setLeading(usize * factor)
// 		offsetY := offsetX + h - ufont.GetFontDescriptor(fonts.BBOXURY, usize)
// 		nt := lines[0]
// 		switch t.alignment {
// 		case alignRight:
// 			wd := ufont.GetWidthPoint(nt, usize)
// 			app.moveText(t.extraMarginLeft+t.box.Width()-2*offsetX-wd, offsetY)
// 		case alignCenter:
// 			nt = strings.TrimSpace(nt)
// 			wd := ufont.GetWidthPoint(nt, usize)
// 			app.moveText(t.extraMarginLeft+t.box.Width()/2-wd/2, offsetY)
// 		default:
// 			app.moveText(t.extraMarginLeft+2*offsetX, offsetY)
// 		}
// 		app.showText(nt)
// 		maxline := (int)(h/usize/factor) + 1
// 		if maxline > len(lines) {
// 			maxline = len(lines)
// 		}
// 		for k := 1; k < maxline; k++ {
// 			nt := lines[k]
// 			if t.alignment == alignRight {
// 				wd := ufont.GetWidthPoint(nt, usize)
// 				app.moveText(t.extraMarginLeft+t.box.Width()-2*offsetX-wd-app.state.xTLM, 0)
// 			} else if t.alignment == alignCenter {
// 				nt = strings.TrimSpace(nt)
// 				wd := ufont.GetWidthPoint(nt, usize)
// 				app.moveText(t.extraMarginLeft+t.box.Width()/2-wd/2-app.state.xTLM, 0)
// 			}
// 			app.newlineShowText(nt)
// 		}
// 	} else {
// 		usize := t.fontSize
// 		if usize == 0 {
// 			maxCalculatedSize := h / (ufont.GetFontDescriptor(fonts.BBOXURX, 1) - ufont.GetFontDescriptor(fonts.BBOXLLY, 1))
// 			wd := ufont.GetWidthPoint(ptext, 1)
// 			if wd == 0 {
// 				usize = maxCalculatedSize
// 			} else {
// 				usize = (t.box.Width() - t.extraMarginLeft - 2*offsetX) / wd
// 			}
// 			if usize > maxCalculatedSize {
// 				usize = maxCalculatedSize
// 			}
// 			if usize < 4 {
// 				usize = 4
// 			}
// 		}
// 		app.setFontAndSize(ufont, usize)
// 		offsetY := offX + ((t.box.Height()-2*offX)-ufont.GetFontDescriptor(fonts.ASCENT, usize))/2
// 		if offsetY < offX {
// 			offsetY = offX
// 		}
// 		if offsetY-offX < -ufont.GetFontDescriptor(fonts.DESCENT, usize) {
// 			ny := -ufont.GetFontDescriptor(fonts.DESCENT, usize) + offX
// 			dy := t.box.Height() - offX - ufont.GetFontDescriptor(fonts.ASCENT, usize)
// 			offsetY = math.Min(ny, math.Max(offsetY, dy))
// 		}
// 		if (t.options&comb) != 0 && t.maxCharacterLength > 0 {
// 			textLen := min(t.maxCharacterLength, len(ptextRunes))
// 			position := 0.
// 			if t.alignment == alignRight {
// 				position = Fl(t.maxCharacterLength - textLen)
// 			} else if t.alignment == alignCenter {
// 				position = Fl(t.maxCharacterLength-textLen) / 2
// 			}
// 			step := (t.box.Width() - t.extraMarginLeft) / Fl(t.maxCharacterLength)
// 			start := step/2 + position*step
// 			for k := 0; k < textLen; k++ {
// 				c := string(ptextRunes[k : k+1])
// 				wd := ufont.GetWidthPoint(c, usize)
// 				app.setTextMatrix2(t.extraMarginLeft+start-wd/2, offsetY-t.extraMarginTop)
// 				app.showText(c)
// 				start += step
// 			}
// 		} else {
// 			switch t.alignment {
// 			case alignRight:
// 				wd := ufont.GetWidthPoint(ptext, usize)
// 				app.moveText(t.extraMarginLeft+t.box.Width()-2*offsetX-wd, offsetY-t.extraMarginTop)
// 			case alignCenter:
// 				wd := ufont.GetWidthPoint(ptext, usize)
// 				app.moveText(t.extraMarginLeft+t.box.Width()/2-wd/2, offsetY-t.extraMarginTop)
// 			default:
// 				app.moveText(t.extraMarginLeft+2*offsetX, offsetY-t.extraMarginTop)
// 			}
// 			app.showText(ptext)
// 		}
// 	}
// 	app.endText()
// 	app.restoreState()
// 	app.endVariableText()
// 	return app
// }

// func (tx *TextField) getListAppearance() Appearance {
// 	app := tx.getBorderAppearance()
// 	app.beginVariableText()
// 	if len(tx.choices) == 0 {
// 		app.endVariableText()
// 		return app
// 	}
// 	topChoice := tx.choiceSelection
// 	if topChoice >= len(tx.choices) {
// 		topChoice = len(tx.choices) - 1
// 	}
// 	if topChoice < 0 {
// 		topChoice = 0
// 	}
// 	ufont := tx.getRealFont()
// 	usize := tx.fontSize
// 	if usize == 0 {
// 		usize = 12
// 	}
// 	borderExtra := tx.borderStyle == borderStyleBeveled || tx.borderStyle == borderStyleInset
// 	h := tx.box.Height() - tx.borderWidth*2
// 	if borderExtra {
// 		h -= tx.borderWidth * 2
// 	}
// 	offsetX := tx.borderWidth
// 	if borderExtra {
// 		offsetX *= 2
// 	}
// 	leading := ufont.GetFontDescriptor(fonts.BBOXURY, usize) - ufont.GetFontDescriptor(fonts.BBOXLLY, usize)
// 	maxFit := int(h/leading) + 1
// 	first := 0
// 	last := 0
// 	last = topChoice + maxFit/2 + 1
// 	first = last - maxFit
// 	if first < 0 {
// 		last += first
// 		first = 0
// 	}
// 	//        first = topChoice;
// 	last = first + maxFit
// 	if last > len(tx.choices) {
// 		last = len(tx.choices)
// 	}
// 	tx.topFirst = first
// 	app.saveState()
// 	app.rectangle(offsetX, offsetX, tx.box.Width()-2*offsetX, tx.box.Height()-2*offsetX)
// 	app.clip()
// 	app.newPath()
// 	mColor := tx.textColor
// 	if mColor == nil {
// 		mColor = color.Gray{}
// 	}
// 	app.setColorFill(color.NRGBA{R: 10, G: 36, B: 106, A: 255})
// 	app.rectangle(offsetX, offsetX+h-Fl(topChoice-first+1)*leading, tx.box.Width()-2*offsetX, leading)
// 	app.fill()
// 	app.beginText()
// 	app.setFontAndSize(ufont, usize)
// 	app.setLeading(leading)
// 	app.moveText(offsetX*2, offsetX+h-ufont.GetFontDescriptor(fonts.BBOXURY, usize)+leading)
// 	app.setColorFill(mColor)
// 	for idx := first; idx < last; idx++ {
// 		if idx == topChoice {
// 			app.setGrayFill(1)
// 			app.newlineShowText(tx.choices[idx])
// 			app.setColorFill(mColor)
// 		} else {
// 			app.newlineShowText(tx.choices[idx])
// 		}
// 	}
// 	app.endText()
// 	app.restoreState()
// 	app.endVariableText()
// 	return app
// }

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
//         PdfAppearance tp = getAppearance();
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
//             int top = math.Min(uchoices.length, choiceExports.length);
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
//             tp = getAppearance();
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
