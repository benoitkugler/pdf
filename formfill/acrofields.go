package formfill

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"

	"github.com/benoitkugler/pdf/formfill/pdftokenizer"
	"github.com/benoitkugler/pdf/model"
)

type Fl = model.Fl

// port of pdftk library - BK 2020

var stdFieldFontNames = map[string][]string{
	"CoBO": {"Courier-BoldOblique"},
	"CoBo": {"Courier-Bold"},
	"CoOb": {"Courier-Oblique"},
	"Cour": {"Courier"},
	"HeBO": {"Helvetica-BoldOblique"},
	"HeBo": {"Helvetica-Bold"},
	"HeOb": {"Helvetica-Oblique"},
	"Helv": {"Helvetica"},
	"Symb": {"Symbol"},
	"TiBI": {"Times-BoldItalic"},
	"TiBo": {"Times-Bold"},
	"TiIt": {"Times-Italic"},
	"TiRo": {"Times-Roman"},
	"ZaDb": {"ZapfDingbats"},
	"HySm": {"HYSMyeongJo-Medium", "UniKS-UCS2-H"},
	"HyGo": {"HYGoThic-Medium", "UniKS-UCS2-H"},
	"KaGo": {"HeiseiKakuGo-W5", "UniKS-UCS2-H"},
	"KaMi": {"HeiseiMin-W3", "UniJIS-UCS2-H"},
	"MHei": {"MHei-Medium", "UniCNS-UCS2-H"},
	"MSun": {"MSung-Light", "UniCNS-UCS2-H"},
	"STSo": {"STSong-Light", "UniGB-UCS2-H"},
}

type acroFields struct {
	extraMarginLeft Fl
	extraMarginTop  Fl

	// localFonts map[string]fonts.Font
	topFirst int
}

type daConfig struct {
	font  model.Name
	color color.Color
	size  Fl
}

func splitDAelements(da string) (daConfig, error) {
	tk := pdftokenizer.NewTokenizer([]byte(da))
	var stack []string
	var ret daConfig
	token, err := tk.NextToken()
	for ; token.Kind != pdftokenizer.EOF && err != nil; token, err = tk.NextToken() {
		if token.Kind == pdftokenizer.Comment {
			continue
		}
		if token.Kind == pdftokenizer.Other {
			switch token.Value {
			case "Tf":
				if len(stack) >= 2 {
					ret.font = model.Name(stack[len(stack)-2])
					fl, err := strconv.ParseFloat(stack[len(stack)-1], 64)
					if err != nil {
						return daConfig{}, err
					}
					ret.size = Fl(fl)
				}
			case "g":
				if len(stack) >= 1 {
					gray, err := strconv.ParseFloat(stack[len(stack)-1], 64)
					if err != nil {
						return daConfig{}, err
					}
					if gray != 0 {
						ret.color = color.Gray{uint8(gray * 255)}
					}
				}
			case "rg":
				if len(stack) >= 3 {
					red, err := strconv.ParseFloat(stack[len(stack)-3], 64)
					if err != nil {
						return daConfig{}, err
					}
					green, err := strconv.ParseFloat(stack[len(stack)-2], 64)
					if err != nil {
						return daConfig{}, err
					}
					blue, err := strconv.ParseFloat(stack[len(stack)-1], 64)
					if err != nil {
						return daConfig{}, err
					}
					ret.color = color.NRGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255)}
				}
			case "k":
				if len(stack) >= 4 {
					cyan, err := strconv.ParseFloat(stack[len(stack)-4], 64)
					if err != nil {
						return daConfig{}, err
					}
					magenta, err := strconv.ParseFloat(stack[len(stack)-3], 64)
					if err != nil {
						return daConfig{}, err
					}
					yellow, err := strconv.ParseFloat(stack[len(stack)-2], 64)
					if err != nil {
						return daConfig{}, err
					}
					black, err := strconv.ParseFloat(stack[len(stack)-1], 64)
					if err != nil {
						return daConfig{}, err
					}
					ret.color = color.CMYK{C: uint8(cyan * 255), M: uint8(magenta * 255), Y: uint8(yellow * 255), K: uint8(black * 255)}
				}
			}
			stack = stack[:0]
		} else {
			stack = append(stack, token.Value)
		}
	}
	return ret, err
}

// Normalizes a Rectangle so that llx and lly are smaller than urx and ury
func getNormalizedRectangle(box model.Rectangle) model.Rectangle {
	return model.Rectangle{
		Llx: minF(box.Llx, box.Urx),
		Lly: minF(box.Lly, box.Ury),
		Urx: maxF(box.Llx, box.Urx),
		Ury: maxF(box.Lly, box.Ury),
	}
}

func rotate(r model.Rectangle) model.Rectangle {
	return model.Rectangle{Llx: r.Lly, Lly: r.Llx, Urx: r.Ury, Ury: r.Urx}
}

const (
	comb = 1 << 8 // 256 // combo box flag.

	// The field may contain multiple lines of text.
	// This flag is only meaningful with text fields.
	multiline = 1 << 2

	// The field is intended for entering a secure password that should
	// not be echoed visibly to the screen.
	password = 1 << 4
)

func (ac acroFields) getAppearance(acro *model.AcroForm, field *model.FormFieldDict, widget model.FormFieldWidget, text, fieldName string) (Appearance, error) {
	ac.topFirst = 0
	tx := TextField{}
	tx.extraMarginLeft = ac.extraMarginLeft
	tx.extraMarginTop = ac.extraMarginTop
	fields := acro.ResolveInheritance(field)

	// the text size and color
	if da := fields.DA; da != "" {
		dab, err := splitDAelements(da)
		if err != nil {
			return Appearance{}, fmt.Errorf("invalid DA string: %s", err)
		}
		if dab.size != 0 {
			tx.fontSize = dab.size
		}
		if dab.color != nil {
			tx.textColor = dab.color
		}
		if dab.font != "" {
			tx.font = acro.DR.Font[dab.font]
		}
	}

	var annot model.AnnotationWidget
	if widget.AnnotationDict != nil {
		annot, _ = widget.Subtype.(model.AnnotationWidget)
	}

	// rotation, border and backgound color
	if annot.MK != nil {
		tx.borderColor = annot.MK.BC.Color()
		if tx.borderColor != nil {
			tx.borderWidth = 1
		}
		tx.backgroundColor = annot.MK.BG.Color()
		tx.rotation = annot.MK.R.Degrees()
	}

	// multiline
	flags := fields.Ff

	// f1 := multiline
	// if (flags & model.Multiline) == 0 {
	// 	f1 = 0
	// }
	// f2 := comb
	// if (flags & ffComb) == 0 {
	// 	f2 = 0
	// }
	// tx.options = f1 | f2
	tx.options = field.Ff
	if (flags & model.Comb) != 0 {
		text, _ := fields.FT.(model.FormFieldText)
		tx.maxCharacterLength = text.MaxLen
	}
	//alignment
	tx.alignment = fields.Q

	//border styles
	if annot.BS != nil {
		if bw, ok := annot.BS.W.(model.Float); ok {
			tx.borderWidth = Fl(bw)
		}
		tx.borderStyle = annot.BS.S
	} else if widget.AnnotationDict != nil {
		if bd := widget.AnnotationDict.Border; bd != nil {
			tx.borderWidth = bd.BorderWidth
			if bd.DashArray != nil {
				tx.borderStyle = "D"
			}
		}
	}
	//rect
	var rect model.Rectangle
	if widget.AnnotationDict != nil {
		rect = widget.AnnotationDict.Rect
	}
	box := getNormalizedRectangle(rect)
	if tx.rotation == 90 || tx.rotation == 270 {
		box = rotate(box)
	}
	tx.box = box

	switch fieldType := fields.FT.(type) {
	case model.FormFieldText:
		tx.text = text
		return tx.getAppearance(), nil
	case model.FormFieldChoice:
		opt := fieldType.Opt
		if (flags&model.Combo) != 0 && len(opt) == 0 {
			tx.text = text
			return tx.getAppearance(), nil
		}
		choices := make([]string, len(opt))
		choicesExp := make([]string, len(opt))
		for k, obj := range opt {
			choices[k] = obj.Name
			if obj.Export == "" {
				choicesExp[k] = obj.Name
			} else {
				choicesExp[k] = obj.Export
			}
		}
		if (flags & model.Combo) != 0 {
			for k, choice := range choices {
				if text == choicesExp[k] {
					text = choice
					break
				}
			}
			tx.text = text
			return tx.getAppearance(), nil
		}
		var idx int
		for k, choiceExp := range choicesExp {
			if text == choiceExp {
				idx = k
				break
			}
		}
		tx.choices = choices
		tx.choiceExports = choicesExp
		tx.choiceSelection = idx
		app := tx.getListAppearance()
		ac.topFirst = tx.topFirst
		return app, nil
	default:
		return Appearance{}, errors.New("an appearance was requested without a variable text field.")
	}
}

// func (ac acroFields) setField(item pdfcpu.Dict, value string) {
// if len(item) == 0 {
// 	return
// }
// type_, _ := item["FT"].(pdfcpu.Name)
// if type_ == "Tx" {
// 	len, _ := item["MaxLen"].(pdfcpu.Integer)
// 	if (len > 0) {
// 		asRunes := []rune(value)
// 		value = string(asRunes[0:min(len, len(asRunes))])
// 	}
// }
// switch type_ {
// case "Tx", "Ch":
// 	PdfString v = new PdfString(value, PdfObject.TEXT_UNICODE);

// 	item.Update("V", value)
// 	// markUsed(item_value);

// 	// ssteward; it might disagree w/ V in a Ch widget
// 	// PDF spec this shouldn't matter, but Reader 9 gives I precedence over V
// 	item.Remove("I")

// 		PdfDictionary widget = (PdfDictionary)item.widgets.get(idx);
// 		if (generateAppearances) {
// 			PdfAppearance app = getAppearance(merged, display, name);
// 			if (PdfName.CH.equals(type)) {
// 				PdfNumber n = new PdfNumber(topFirst);
// 				widget.put(PdfName.TI, n);
// 				merged.put(PdfName.TI, n);
// 			}
// 			PdfDictionary appDic = (PdfDictionary)PdfReader.getPdfObject(widget.get(PdfName.AP));
// 			if (appDic == nil) {
// 				appDic = new PdfDictionary();
// 				widget.put(PdfName.AP, appDic);
// 				merged.put(PdfName.AP, appDic);
// 			}
// 			appDic.put(PdfName.N, app.getIndirectReference());
// 			writer.releaseTemplate(app);
// 		}
// 		// else {
// 		// 	widget.remove(PdfName.AP);
// 		// 	merged.remove(PdfName.AP);
// 		// }
// 		// markUsed(widget);
// 	return true;
// }
// else if (PdfName.BTN.equals(type)) {
// 	PdfNumber ff = (PdfNumber)PdfReader.getPdfObject(((PdfDictionary)item.merged.get(0)).get(PdfName.FF));
// 	int flags = 0;
// 	if (ff != nil)
// 		flags = ff.intValue();
// 	if ((flags & ffPushbutton) != 0)
// 		return true;
// 	PdfName v = new PdfName(value);
// 	if ((flags & ffRadio) == 0) {
// 		for (int idx = 0; idx < item.values.size(); ++idx) {
// 			((PdfDictionary)item.values.get(idx)).put(PdfName.V, v);
// 			markUsed((PdfDictionary)item.values.get(idx));
// 			PdfDictionary merged = (PdfDictionary)item.merged.get(idx);
// 			merged.put(PdfName.V, v);
// 			merged.put(PdfName.AS, v);
// 			PdfDictionary widget = (PdfDictionary)item.widgets.get(idx);
// 			if (isInAP(widget,  v))
// 				widget.put(PdfName.AS, v);
// 			else
// 				widget.put(PdfName.AS, PdfName.Off);
// 			markUsed(widget);
// 		}
// 	}
// 	else {
// 		ArrayList lopt = new ArrayList();
// 		PdfObject opts = PdfReader.getPdfObject(((PdfDictionary)item.values.get(0)).get(PdfName.OPT));
// 		if (opts != nil && opts.isArray()) {
// 			ArrayList list = ((PdfArray)opts).getArrayList();
// 			for (int k = 0; k < list.size(); ++k) {
// 				PdfObject vv = PdfReader.getPdfObject((PdfObject)list.get(k));
// 				if (vv != nil && vv.isString())
// 					lopt.add(((PdfString)vv).toUnicodeString());
// 				else
// 					lopt.add(nil);
// 			}
// 		}
// 		int vidx = lopt.indexOf(value);
// 		PdfName valt = nil;
// 		PdfName vt;
// 		if (vidx >= 0) {
// 			vt = valt = new PdfName(String.valueOf(vidx));
// 		}
// 		else
// 			vt = v;
// 		for (int idx = 0; idx < item.values.size(); ++idx) {
// 			PdfDictionary merged = (PdfDictionary)item.merged.get(idx);
// 			PdfDictionary widget = (PdfDictionary)item.widgets.get(idx);
// 			markUsed((PdfDictionary)item.values.get(idx));
// 			if (valt != nil) {
// 				PdfString ps = new PdfString(value, PdfObject.TEXT_UNICODE);
// 				((PdfDictionary)item.values.get(idx)).put(PdfName.V, ps);
// 				merged.put(PdfName.V, ps);
// 			}
// 			else {
// 				((PdfDictionary)item.values.get(idx)).put(PdfName.V, v);
// 				merged.put(PdfName.V, v);
// 			}
// 			markUsed(widget);
// 			if (isInAP(widget,  vt)) {
// 				merged.put(PdfName.AS, vt);
// 				widget.put(PdfName.AS, vt);
// 			}
// 			else {
// 				merged.put(PdfName.AS, PdfName.Off);
// 				widget.put(PdfName.AS, PdfName.Off);
// 			}
// 		}
// 	}
// 	return true;
// }
// return false;
// }
