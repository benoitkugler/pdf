package reader

import (
	"fmt"
	"log"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
)

func (r resolver) processAcroForm(acroForm model.Object) (model.AcroForm, error) {
	var out model.AcroForm
	acroForm = r.resolve(acroForm)
	if acroForm == nil || (acroForm == model.ObjNull{}) {
		return out, nil
	}
	form, ok := acroForm.(model.ObjDict)
	if !ok {
		return out, errType("AcroForm", acroForm)
	}
	fields, _ := r.resolveArray(form["Fields"])
	out.Fields = make([]*model.FormFieldDict, len(fields))
	for i, f := range fields {
		ff, err := r.resolveFormField(f, nil)
		if err != nil {
			return out, err
		}
		out.Fields[i] = ff
	}
	if na, ok := r.resolveBool(form["NeedAppearances"]); ok {
		out.NeedAppearances = na
	}
	if sig, ok := r.resolveInt(form["SigFlags"]); ok {
		out.SigFlags = model.SignatureFlag(sig)
	}
	if co, _ := r.resolveArray(form["CO"]); len(co) != 0 {
		out.CO = make([]*model.FormFieldDict, 0, len(co))
		for _, c := range co {
			ref, ok := c.(model.ObjIndirectRef)
			if !ok {
				return out, errType("Field reference for CO", c)
			}
			// we just ignore invalid reference
			if field := r.formFields[ref]; field != nil {
				out.CO = append(out.CO, field)
			}
		}
	}
	var err error
	if dr := form["DR"]; dr != nil {
		out.DR, err = r.resolveOneResourceDict(dr)
		if err != nil {
			return out, err
		}
	}
	out.DA, _ = file.IsString(r.resolve(form["DA"]))
	if q, ok := r.resolveInt(form["Q"]); ok {
		out.Q = model.Quadding(q)
	}
	return out, nil
}

// since a Widget dictionary may be merged into the Field dict
// there is no direct way to distinguish a FormField from a Widget
// to choose, we check is at least one attribute of a FormField is present
// is so, we are sure we have a form field (and maybe a Widget)
// if not, we have no certitudes, but the FormField would be useles
// so we will only use the Widget
// the boolean returned is true if `form` is actually a form field.
// the attibutes Parent,Kids, Widgets are not set
func (r resolver) isFormField(form model.ObjDict) (field model.FormFieldDict, isField bool) {
	if _, ok := r.resolveName(form["FT"]); ok {
		isField = true
		field.FT = r.processFormFieldType(form)
	}
	if t, ok := file.IsString(r.resolve(form["T"])); ok {
		isField = true
		field.T = DecodeTextString(t)
	}
	if t, ok := file.IsString(r.resolve(form["TU"])); ok {
		isField = true
		field.TU = t
	}
	if t, ok := file.IsString(r.resolve(form["TM"])); ok {
		isField = true
		field.TM = t
	}
	if ff, ok := r.resolveInt(form["Ff"]); ok {
		isField = true
		field.Ff = model.FormFlag(ff)
	}
	if aa := r.resolve(form["AA"]); aa != nil {
		field.AA = r.processFormAA(aa)
		if field.AA.IsEmpty() { // the AA entry may be the one of a widget
			isField = true
		}
	}
	if q, ok := r.resolveInt(form["Q"]); ok {
		isField = true
		field.Q = model.Quadding(q)
	}
	if da, ok := file.IsString(r.resolve(form["DA"])); ok {
		isField = true
		field.DA = da
	}
	if t, ok := file.IsString(r.resolve(form["DS"])); ok {
		isField = true
		field.DS = DecodeTextString(t)
	}
	if t, ok := file.IsString(r.resolve(form["RV"])); ok {
		isField = true
		field.RV = DecodeTextString(t)
	}
	return field, isField
}

func (r resolver) processFormAA(aa model.Object) model.FormFielAdditionalActions {
	aa = r.resolve(aa)
	aaDict, _ := aa.(model.ObjDict)
	var out model.FormFielAdditionalActions
	out.K, _ = r.processAction(aaDict["K"])
	out.F, _ = r.processAction(aaDict["F"])
	out.V, _ = r.processAction(aaDict["V"])
	out.C, _ = r.processAction(aaDict["C"])
	return out
}

// extract a text string from either a string or a stream object,
// after dereferencing
func (r resolver) textOrStream(object model.Object) string {
	content := r.resolve(object)
	var jsString string
	if stream, ok := content.(model.ObjStream); ok {
		s, ok, _ := r.resolveStream(stream)
		if ok {
			decoded, err := s.Decode()
			if err != nil { // best effort: we return the raw stream
				log.Println("failed to decode text stream", err)
				decoded = s.Content
			}
			jsString = string(decoded)
		}
	} else {
		jsString, _ = file.IsString(content)
	}
	return DecodeTextString(jsString)
}

// `parent` will be nil for the top-level fields
// if not, its type maybe be checked to find the field type by inheritance
func (r resolver) resolveFormField(o model.Object, parent *model.FormFieldDict) (*model.FormFieldDict, error) {
	var err error
	ref, isRef := o.(model.ObjIndirectRef)
	if ff := r.formFields[ref]; isRef && ff != nil {
		return ff, nil
	}
	resolved := r.resolve(ref)
	if resolved == nil {
		return nil, nil
	}
	f, isDict := resolved.(model.ObjDict)
	if !isDict {
		return nil, errType("FormField", o)
	}

	fi, _ := r.isFormField(f) // fill the simple attributes

	fi.Parent = parent

	kids, _ := r.resolveArray(f["Kids"])
	for _, kid := range kids {
		// a kid may be either another FormField or a Widget Annotation
		// we need a first exploration of the kid dict
		// before doing it for good with the recursive call to `resolveFormField`
		kidDict, _ := r.resolve(kid).(model.ObjDict)
		if kidDict == nil { // ignore the invalid entry
			continue
		}
		_, isField := r.isFormField(kidDict) // could be optimized not to resolve entry
		if isField {
			kidField, err := r.resolveFormField(kid, &fi) // surely indirect ref
			if err != nil {
				return nil, err
			}
			fi.Kids = append(fi.Kids, kidField)
		} else {
			widget, _, err := r.resolveWidget(kid)
			if err != nil {
				return nil, err
			}
			fi.Widgets = append(fi.Widgets, widget)
		}
	}

	// check for merged widget annotation
	// we need to pass the indirect ref not to duplicate annotation dicts
	widget, isWidget, err := r.resolveWidget(o)
	if err != nil {
		return nil, err
	}
	if isWidget {
		fi.Widgets = append(fi.Widgets, widget)
	}

	if isRef {
		r.formFields[ref] = &fi
	}

	return &fi, nil
}

// resolveWidget return true if form is an annotation widget
// (it will be false for form field which have no merged fields)
func (r resolver) resolveWidget(obj model.Object) (model.FormFieldWidget, bool, error) {
	annot, err := r.resolveAnnotation(obj)
	if err != nil {
		return model.FormFieldWidget{}, false, err
	}
	// check the dynamic type
	if _, isWidget := annot.Subtype.(model.AnnotationWidget); isWidget {
		// we found a widget
		return model.FormFieldWidget{AnnotationDict: annot}, true, nil
	}
	// ignore the invalid form widget
	return model.FormFieldWidget{}, false, nil
}

// ------------------- specialization of form fields -------------------

// may return nil it the type if inherited
func (r resolver) processFormFieldType(form model.ObjDict) model.FormField {
	ft, _ := r.resolveName(form["FT"])
	switch ft {
	case "Btn":
		var out model.FormFieldButton
		v, _ := r.resolveName(form["V"])
		out.V = model.ObjName(v)
		opt, _ := r.resolveArray(form["Opt"])
		out.Opt = make([]string, len(opt))
		for i, o := range opt {
			os, _ := file.IsString(r.resolve(o))
			out.Opt[i] = DecodeTextString(os)
		}
		return out
	case "Ch":
		var out model.FormFieldChoice
		v := r.resolve(form["V"])
		if str, is := file.IsString(v); is {
			out.V = []string{DecodeTextString(str)}
		} else if ar, ok := v.(model.ObjArray); ok {
			out.V = make([]string, len(ar))
			for i, a := range ar {
				s, _ := file.IsString(r.resolve(a))
				out.V[i] = DecodeTextString(s)
			}
		}
		opts, _ := r.resolveArray(form["Opt"])
		out.Opt = make([]model.Option, len(opts))
		for i, o := range opts {
			o = r.resolve(o)
			if s, ok := file.IsString(o); ok { // a single text string
				out.Opt[i].Name = DecodeTextString(s)
			} else if s, _ := o.(model.ObjArray); len(s) == 2 { // [export name]
				export, _ := file.IsString(r.resolve(s[0]))
				name, _ := file.IsString(r.resolve(s[1]))
				out.Opt[i].Export = DecodeTextString(export)
				out.Opt[i].Name = DecodeTextString(name)
			}
		}
		if ti, ok := r.resolveInt(form["TI"]); ok {
			out.TI = ti
		}
		is, _ := r.resolveArray(form["I"])
		out.I = make([]int, len(is))
		for i, ii := range is {
			out.I[i], _ = r.resolveInt(ii)
		}
		return out
	case "Sig":
		return r.processSignatureField(form)
	case "Tx":
		var out model.FormFieldText
		out.V = r.textOrStream(form["V"])
		if ml, ok := r.resolveInt(form["MaxLen"]); ok {
			out.MaxLen = model.ObjInt(ml)
		}
		return out
	default: // nil or invalid
		return nil
	}
}

// TODO: process signature field
func (r resolver) processSignatureField(form model.ObjDict) model.FormFieldSignature {
	fmt.Println("TODO Signature field", form)
	return model.FormFieldSignature{}
}
