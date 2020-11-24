package formfill

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/formfill/pdftokenizer"
	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/standardfonts"
)

// port of pdftk library - BK 2020

type Fl = model.Fl

var defaultFont = &model.FontDict{Subtype: model.FontType1{
	FirstChar:      standardfonts.Helvetica.FirstChar,
	Widths:         standardfonts.Helvetica.Widths,
	FontDescriptor: standardfonts.Helvetica.Descriptor,
}}

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

	topFirst int

	fontCache map[model.Name]fonts.BuiltFont
}

func newAcroFields() acroFields {
	return acroFields{fontCache: make(map[model.Name]fonts.BuiltFont)}
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

func (ac *acroFields) buildAppearance(acro *model.AcroForm, fields model.FormFieldInheritable, widget model.FormFieldWidget, text string) (*model.XObjectForm, error) {
	ac.topFirst = 0
	tx := TextField{}
	tx.extraMarginLeft = ac.extraMarginLeft
	tx.extraMarginTop = ac.extraMarginTop

	// the text size and color
	var (
		fontSize Fl
		font     fonts.BuiltFont
	)
	if da := fields.DA; da != "" {
		dab, err := splitDAelements(da)
		if err != nil {
			return nil, fmt.Errorf("invalid DA string: %s", err)
		}
		if dab.size != 0 {
			fontSize = dab.size
		}
		if dab.color != nil {
			tx.textColor = dab.color
		}
		if dab.font != "" {
			if bf, has := ac.fontCache[dab.font]; has {
				font = bf
			} else { // build and cache
				fd := acro.DR.Font[dab.font]
				if fd == nil { // safely default to a standard font
					fd = defaultFont
				}
				font = fonts.BuildFont(fd)
				ac.fontCache[dab.font] = font
			}
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

	tx.options = fields.Ff
	if (fields.Ff & model.Comb) != 0 {
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
		return tx.buildAppearance(font, fontSize), nil
	case model.FormFieldChoice:
		opt := fieldType.Opt
		if (fields.Ff&model.Combo) != 0 && len(opt) == 0 {
			tx.text = text
			return tx.buildAppearance(font, fontSize), nil
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
		if (fields.Ff & model.Combo) != 0 {
			for k, choice := range choices {
				if text == choicesExp[k] {
					text = choice
					break
				}
			}
			tx.text = text
			return tx.buildAppearance(font, fontSize), nil
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
		app := tx.getListAppearance(font, fontSize)
		ac.topFirst = tx.topFirst
		return app, nil
	default:
		return nil, errors.New("an appearance was requested without a variable text field.")
	}
}

func (ac acroFields) buildWidgets(acro *model.AcroForm, item *model.FormFieldDict, inherited model.FormFieldInheritable, display string) error {
	for _, widget := range item.Widgets {
		app, err := ac.buildAppearance(acro, inherited, widget, display) // check last arg
		if err != nil {
			return err
		}
		appDic := widget.AP
		if appDic == nil {
			appDic = new(model.AppearanceDict)
		}
		appDic.N = model.AppearanceEntry{"": app}
		widget.AP = appDic // update the model
	}
	return nil
}

func (ac acroFields) setField(acro *model.AcroForm, item *model.FormFieldDict, value, display, richValue string) error {
	fields := acro.ResolveInheritance(item)

	item.RV = richValue
	switch type_ := fields.FT.(type) {
	case model.FormFieldText:
		if ml, _ := type_.MaxLen.(model.Int); ml > 0 {
			asRunes := []rune(value)
			value = string(asRunes[0:min(int(ml), len(asRunes))])
		}
		type_.V = value
		item.FT = type_
		return ac.buildWidgets(acro, item, fields, display)
	case model.FormFieldChoice:
		type_.V = []string{value}
		// ssteward; it might disagree w/ V in a Ch widget
		// PDF spec this shouldn't matter, but Reader 9 gives I precedence over V
		type_.I = nil
		err := ac.buildWidgets(acro, item, fields, display)
		type_.TI = ac.topFirst
		item.FT = type_
		return err
	case model.FormFieldButton:
		flags := fields.Ff
		if (flags & model.Pushbutton) != 0 {
			return nil
		}
		v := model.Name(value)
		if (flags & model.Radio) == 0 {
			type_.V = v
			for _, widget := range item.Widgets {
				widget.AS = v
				if isInAP(widget, v) {
					widget.AS = v
				} else {
					widget.AS = model.Name("Off")
				}
			}
		} else {
			vidx := -1
			for idx, vv := range type_.Opt {
				if vv == value {
					vidx = idx
				}
			}
			var vt model.Name
			if vidx >= 0 {
				vt = model.Name(strconv.Itoa(vidx))
				type_.V = model.Name(value)
			} else {
				vt = v
				type_.V = v
			}
			for _, widget := range item.Widgets {
				if isInAP(widget, vt) {
					widget.AS = vt
				} else {
					widget.AS = model.Name("Off")
				}
			}
		}
		item.FT = type_
	}
	return nil
}

func isInAP(widget model.FormFieldWidget, check model.Name) bool {
	if widget.AP == nil {
		return false
	}
	return widget.AP.N != nil && widget.AP.N[check] != nil
}
