package formfill

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/parser/tokenizer"
)

// port of pdftk library - BK 2020

type Fl = model.Fl

var defaultFont = &model.FontDict{
	Subtype: standardfonts.Helvetica.WesternType1Font(),
}

type filler struct {
	fontCache map[model.Name]fonts.BuiltFont
}

func newFiller() filler {
	return filler{fontCache: make(map[model.Name]fonts.BuiltFont)}
}

type daConfig struct {
	font  model.Name
	color color.Color
	size  Fl
}

func splitDAelements(da string) (daConfig, error) {
	tk := tokenizer.NewTokenizer([]byte(da))
	var stack []tokenizer.Token
	var ret daConfig
	token, err := tk.NextToken()
	for ; token.Kind != tokenizer.EOF && err == nil; token, err = tk.NextToken() {
		if token.Kind == tokenizer.Other {
			switch token.Value {
			case "Tf":
				if len(stack) >= 2 {
					ret.font = model.Name(stack[len(stack)-2].Value)
					fl, err := stack[len(stack)-1].Float()
					if err != nil {
						return daConfig{}, err
					}
					ret.size = Fl(fl)
				}
			case "g":
				if len(stack) >= 1 {
					gray, err := stack[len(stack)-1].Float()
					if err != nil {
						return daConfig{}, err
					}
					if gray != 0 {
						ret.color = color.Gray{uint8(gray * 255)}
					}
				}
			case "rg":
				if len(stack) >= 3 {
					red, err := stack[len(stack)-3].Float()
					if err != nil {
						return daConfig{}, err
					}
					green, err := stack[len(stack)-2].Float()
					if err != nil {
						return daConfig{}, err
					}
					blue, err := stack[len(stack)-1].Float()
					if err != nil {
						return daConfig{}, err
					}
					ret.color = color.NRGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255)}
				}
			case "k":
				if len(stack) >= 4 {
					cyan, err := stack[len(stack)-4].Float()
					if err != nil {
						return daConfig{}, err
					}
					magenta, err := stack[len(stack)-3].Float()
					if err != nil {
						return daConfig{}, err
					}
					yellow, err := stack[len(stack)-2].Float()
					if err != nil {
						return daConfig{}, err
					}
					black, err := stack[len(stack)-1].Float()
					if err != nil {
						return daConfig{}, err
					}
					ret.color = color.CMYK{C: uint8(cyan * 255), M: uint8(magenta * 255), Y: uint8(yellow * 255), K: uint8(black * 255)}
				}
			}
			stack = stack[:0]
		} else {
			stack = append(stack, token)
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

func (ac *filler) buildAppearance(formResources model.ResourcesDict, fields model.FormFieldInheritable, widget model.FormFieldWidget, text string) (*model.XObjectForm, int, error) {
	appBuilder := fieldAppearanceBuilder{}

	// the text size and color
	var (
		fontSize Fl
		font     fonts.BuiltFont
	)
	if da := fields.DA; da != "" {
		dab, err := splitDAelements(da)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid DA string: %s", err)
		}
		if dab.size != 0 {
			fontSize = dab.size
		}
		if dab.color != nil {
			appBuilder.textColor = dab.color
		}
		if dab.font != "" {
			if bf, has := ac.fontCache[dab.font]; has {
				font = bf
			} else { // build and cache
				fd := formResources.Font[dab.font]
				if fd == nil { // safely default to a standard font
					log.Printf("can't resolve font %s -> using default", dab.font)
					fd = defaultFont
				}
				font, err = fonts.BuildFont(fd)
				if err != nil {
					return nil, 0, err
				}
				ac.fontCache[dab.font] = font
			}
		} else {
			log.Println("no font specified in DA string -> using default")
			// use a default font
			font, err = fonts.BuildFont(defaultFont)
			if err != nil {
				return nil, 0, err
			}
			ac.fontCache[dab.font] = font
		}
	}

	var annot model.AnnotationWidget
	if widget.AnnotationDict != nil {
		annot, _ = widget.Subtype.(model.AnnotationWidget)
	}

	// rotation, border and backgound color
	if annot.MK != nil {
		appBuilder.borderColor = annot.MK.BC.Color()
		if appBuilder.borderColor != nil {
			appBuilder.borderWidth = 1
		}
		appBuilder.backgroundColor = annot.MK.BG.Color()
		appBuilder.rotation = annot.MK.R.Degrees()
	}

	// multiline

	appBuilder.options = fields.Ff
	if (fields.Ff & model.Comb) != 0 {
		text, _ := fields.FT.(model.FormFieldText)
		appBuilder.maxCharacterLength = text.MaxLen
	}
	//alignment
	appBuilder.alignment = fields.Q

	//border styles
	if annot.BS != nil {
		if bw, ok := annot.BS.W.(model.Float); ok {
			appBuilder.borderWidth = Fl(bw)
		}
		appBuilder.borderStyle = annot.BS.S
	} else if widget.AnnotationDict != nil {
		if bd := widget.AnnotationDict.Border; bd != nil {
			appBuilder.borderWidth = bd.BorderWidth
			if bd.DashArray != nil {
				appBuilder.borderStyle = "D"
			}
		}
	}
	//rect
	var rect model.Rectangle
	if widget.AnnotationDict != nil {
		rect = widget.AnnotationDict.Rect
	}
	box := getNormalizedRectangle(rect)
	if appBuilder.rotation == 90 || appBuilder.rotation == 270 {
		box = rotate(box)
	}
	appBuilder.box = box

	switch fieldType := fields.FT.(type) {
	case model.FormFieldText:
		appBuilder.text = text
		return appBuilder.buildAppearance(font, fontSize), 0, nil
	case model.FormFieldChoice:
		opt := fieldType.Opt
		if (fields.Ff&model.Combo) != 0 && len(opt) == 0 {
			appBuilder.text = text
			return appBuilder.buildAppearance(font, fontSize), 0, nil
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
			appBuilder.text = text
			return appBuilder.buildAppearance(font, fontSize), 0, nil
		}
		var idx int
		for k, choiceExp := range choicesExp {
			if text == choiceExp {
				idx = k
				break
			}
		}
		appBuilder.choices = choices
		// tx.choiceExports = choicesExp
		appBuilder.choiceSelection = idx
		app, topFirst := appBuilder.getListAppearance(font, fontSize)
		return app, topFirst, nil
	default:
		return nil, 0, errors.New("an appearance was requested without a variable text field.")
	}
}

// buildWidgets update item
func (ac filler) buildWidgets(formResources model.ResourcesDict, field model.FormFieldInherited, display string) (int, error) {
	var topFirst int
	for _, widget := range field.FormFieldDict.Widgets {
		var (
			app *model.XObjectForm
			err error
		)
		app, topFirst, err = ac.buildAppearance(formResources, field.Merged, widget, display)
		if err != nil {
			return 0, err
		}
		appDic := widget.AP
		if appDic == nil {
			appDic = new(model.AppearanceDict)
		}
		appDic.N = model.AppearanceEntry{"": app}
		widget.AP = appDic // update the model
	}
	return topFirst, nil
}

// fields contains the inherited currentvalues, values are the values to write to the field
func (ac filler) setField(formResources model.ResourcesDict, field model.FormFieldInherited, values Values) error {
	field.FormFieldDict.RV = values.RV
	switch type_ := field.Merged.FT.(type) {
	case model.FormFieldText:
		value, ok := values.V.(Text)
		if !ok {
			return fmt.Errorf("unexpected value type for text field: %T", values.V)
		}
		if ml, _ := type_.MaxLen.(model.Int); ml > 0 {
			asRunes := []rune(value)
			value = Text(asRunes[0:min(int(ml), len(asRunes))])
		}
		type_.V = string(value)
		_, err := ac.buildWidgets(formResources, field, string(value))
		if err != nil {
			return err
		}
		field.FormFieldDict.FT = type_ // update
	case model.FormFieldChoice:
		value, ok := values.V.(Choices)
		if !ok {
			return fmt.Errorf("unexpected value type for choices field: %T", values.V)
		}
		type_.V = []string(value)
		// ssteward; it might disagree w/ V in a Ch widget
		// PDF spec this shouldn't matter, but Reader 9 gives I precedence over V
		type_.I = nil
		display := strings.Join(type_.V, ", ")
		topFirst, err := ac.buildWidgets(formResources, field, display)
		if err != nil {
			return err
		}
		type_.TI = topFirst
		field.FormFieldDict.FT = type_ // update
	case model.FormFieldButton:
		value, ok := values.V.(ButtonAppearanceName)
		if !ok {
			return fmt.Errorf("unexpected value type for button field: %T", values.V)
		}
		flags := field.Merged.Ff
		if (flags & model.Pushbutton) != 0 {
			return nil
		}
		v := model.Name(value)
		if (flags & model.Radio) == 0 {
			type_.V = v
			setStateAS(field.FormFieldDict, v)
		} else {
			vidx := -1
			for idx, vv := range type_.Opt {
				if vv == string(value) {
					vidx = idx
				}
			}
			vt := v
			if vidx >= 0 {
				vt = model.Name(strconv.Itoa(vidx))
			}
			type_.V = v
			setStateAS(field.FormFieldDict, vt)
		}
		field.FormFieldDict.FT = type_ // update
	}
	return nil
}

func setStateAS(field *model.FormFieldDict, state model.Name) {
	for _, widget := range field.Widgets {
		if isInAP(widget, state) {
			widget.AS = state
		} else {
			widget.AS = model.Name("Off")
		}
	}
}

func isInAP(widget model.FormFieldWidget, check model.Name) bool {
	if widget.AP == nil {
		return false
	}
	return widget.AP.N != nil && widget.AP.N[check] != nil
}

// update `acro` in place, accorcding to the value in `fdf`
func (ac filler) fillForm(acro *model.AcroForm, fdf FDFDict, lockForm bool) error {
	// we first walk the fdf tree into a map
	values := fdf.resolve()

	// we also walk the current tree into a map
	fields := acro.Flatten()

	for fullName, fdfValue := range values {
		if acroValue, ok := fields[fullName]; ok {
			// match with value, do fill the field
			err := ac.setField(acro.DR, acroValue, fdfValue)
			if err != nil {
				return err
			}
		}
	}
	acro.NeedAppearances = false

	if lockForm {
		// lock all the fields, not only the ones filled
		for _, field := range fields {
			field.Ff |= model.ReadOnly
		}
	}

	return nil
}
