package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

const (
	fieldBtn    = "Btn"
	fieldText   = "Tx"
	fieldChoice = "Ch"
	fieldSig    = "Sig"
)

func (r resolver) processAcroForm(acroForm pdfcpu.Object) (*model.AcroForm, error) {
	acroForm = r.resolve(acroForm)
	if acroForm == nil {
		return nil, nil
	}
	form, ok := acroForm.(pdfcpu.Dict)
	if !ok {
		return nil, errType("AcroForm", acroForm)
	}
	var (
		out model.AcroForm
		err error
	)
	fields, _ := r.resolveArray(form["Fields"])
	out.Fields = make([]*model.FormFieldDict, len(fields))
	for i, f := range fields {
		ff, err := r.resolveFormField(f, nil)
		if err != nil {
			return nil, err
		}
		out.Fields[i] = ff
	}
	if na, ok := r.resolveBool(form["NeedAppearances"]); ok {
		out.NeedAppearances = bool(na)
	}
	if sig, ok := r.resolveInt(form["SigFlags"]); ok {
		out.SigFlags = model.SignatureFlag(sig)
	}
	if co, _ := r.resolveArray(form["CO"]); len(co) != 0 {
		out.CO = make([]*model.FormFieldDict, 0, len(co))
		for _, c := range co {
			ref, ok := c.(pdfcpu.IndirectRef)
			if !ok {
				return nil, errType("Field reference for CO", c)
			}
			// we just ignore invalid reference
			if field := r.formFields[ref]; field != nil {
				out.CO = append(out.CO, field)
			}
		}
	}
	if dr := form["DR"]; dr != nil {
		out.DR, err = r.resolveOneResourceDict(dr)
		if err != nil {
			return nil, err
		}
	}
	out.DA, _ = isString(r.resolve(form["DA"]))
	if q, ok := r.resolveInt(form["Q"]); ok {
		out.Q = int(q)
	}
	return &out, nil
}

// since a Widget dictionary may be merged into the Field dict
// there is no direct way to distinguish a FormField from a Widget
// to choose, we check is at least one attribute of a FormField is present
// is so, we are sure we have a form field (and maybe a Widget)
// if not, we have no certitudes, but the FormField would be useles
// so we will only use the Widget
// the boolean returned is true if `form` is actually a form field.
// the attibutes Parent,Kids, Widgets are not set
func (r resolver) isFormField(form pdfcpu.Dict) (field model.FormFieldDict, isField bool) {
	if _, ok := r.resolveName(form["FT"]); ok {
		isField = true
		field.FT = r.processFormFieldType(form)
	}
	if t, ok := isString(r.resolve(form["T"])); ok {
		isField = true
		field.T = decodeTextString(t)
	}
	if t, ok := isString(r.resolve(form["TU"])); ok {
		isField = true
		field.TU = t
	}
	if t, ok := isString(r.resolve(form["TM"])); ok {
		isField = true
		field.TM = t
	}
	if ff, ok := r.resolveInt(form["Ff"]); ok {
		isField = true
		field.Ff = model.FormFlag(ff)
	}
	if aa := r.resolve(form["AA"]); aa != nil {
		isField = true
		field.AA = r.processAA(aa)
	}
	if q, ok := r.resolveInt(form["Q"]); ok {
		isField = true
		field.Q = uint8(q)
	}
	if da, ok := isString(r.resolve(form["DA"])); ok {
		isField = true
		field.DA = da
	}
	if t, ok := isString(r.resolve(form["DS"])); ok {
		isField = true
		field.DS = decodeTextString(t)
	}
	if t, ok := isString(r.resolve(form["RV"])); ok {
		isField = true
		field.RV = decodeTextString(t)
	}
	return field, isField
}

func (r resolver) processAA(aa pdfcpu.Object) *model.FormFielAdditionalActions {
	aa = r.resolve(aa)
	aaDict, ok := aa.(pdfcpu.Dict)
	if !ok {
		return nil
	}
	var out model.FormFielAdditionalActions
	out.K = r.processJSAction(aaDict["K"])
	out.F = r.processJSAction(aaDict["F"])
	out.V = r.processJSAction(aaDict["V"])
	out.C = r.processJSAction(aaDict["C"])
	return &out
}

func (r resolver) processJSAction(action pdfcpu.Object) model.ActionJavaScript {
	var js string
	if K, ok := r.resolve(action).(pdfcpu.Dict); ok {
		js = r.textOrStream(K["JS"])
	}
	return model.ActionJavaScript{JS: js}
}

// extract a text string from either a string or a stream object,
// after dereferencing
func (r resolver) textOrStream(object pdfcpu.Object) string {
	content := r.resolve(object)
	// TODO: decode stream to be in part with strings
	// this requires to check pdfcpu filters
	if stream, ok := content.(pdfcpu.StreamDict); ok {
		s, _ := r.resolveStream(stream)
		if s != nil {
			return string(s.Content)
		}
	}
	jsString, _ := isString(content)
	return decodeTextString(jsString)
}

// TODO: fix form tree parsing
// `parent` will be nil for the top-level fields
// if not, its type maybe be checked to find the field type by inheritance
func (r resolver) resolveFormField(o pdfcpu.Object, parent *model.FormFieldDict) (*model.FormFieldDict, error) {
	var err error
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		// did we already resolve this value ?
		if ff := r.formFields[ref]; ff != nil {
			return ff, nil
		}
		// we haven't resolved it yet: do it
		o = r.resolve(ref)
	}
	if o == nil {
		return nil, nil
	}
	f, isDict := o.(pdfcpu.Dict)
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
		kidDict, _ := r.resolve(kid).(pdfcpu.Dict)
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
			widget, _, err := r.resolveWidget(kidDict)
			if err != nil {
				return nil, err
			}
			fi.Widgets = append(fi.Widgets, widget)
		}
	}

	// check for merged widget annotation
	widget, isWidget, err := r.resolveWidget(f)
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
func (r resolver) resolveWidget(form pdfcpu.Dict) (model.Widget, bool, error) {
	annot, err := r.resolveAnnotationFields(form)
	if err != nil {
		return model.Widget{}, false, err
	}
	if widget, isWidget := annot.Subtype.(model.AnnotationWidget); isWidget {
		// we found a merged widget
		return model.Widget{BaseAnnotation: annot.BaseAnnotation, AnnotationWidget: widget}, true, nil
	}
	return model.Widget{}, false, nil
}

// ------------------- specialization of form fields -------------------

// may return nil it the type if inherited
func (r resolver) processFormFieldType(form pdfcpu.Dict) model.FormField {
	ft, _ := r.resolveName(form["FT"])
	switch ft {
	case fieldBtn:
		var out model.FormFieldButton
		v, _ := r.resolveName(form["V"])
		out.V = model.Name(v)
		opt, _ := r.resolveArray(form["Opt"])
		out.Opt = make([]string, len(opt))
		for i, o := range opt {
			os, _ := isString(r.resolve(o))
			out.Opt[i] = decodeTextString(os)
		}
		return out
	case fieldChoice:
		var out model.FormFieldChoice
		v := r.resolve(form["V"])
		if str, is := isString(v); is {
			out.V = []string{decodeTextString(str)}
		} else if ar, ok := v.(pdfcpu.Array); ok {
			out.V = make([]string, len(ar))
			for i, a := range ar {
				s, _ := isString(r.resolve(a))
				out.V[i] = decodeTextString(s)
			}
		}
		opts, _ := r.resolveArray(form["Opt"])
		out.Opt = make([]model.Option, len(opts))
		for i, o := range opts {
			o = r.resolve(o)
			if s, ok := isString(o); ok { // a single text string
				out.Opt[i].Name = decodeTextString(s)
			} else if s, _ := o.(pdfcpu.Array); len(s) == 2 { // [export name]
				export, _ := isString(r.resolve(s[0]))
				name, _ := isString(r.resolve(s[1]))
				out.Opt[i].Export = decodeTextString(export)
				out.Opt[i].Name = decodeTextString(name)
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
	case fieldSig:
		return r.processSignatureField(form)
	case fieldText:
		var out model.FormFieldText
		out.V = r.textOrStream(form["V"])
		if ml, ok := r.resolveInt(form["MaxLen"]); ok {
			out.MaxLen = model.Int(ml)
		}
		return out
	default: // nil or invalid
		return nil
	}
}

// TODO: process signature field
func (r resolver) processSignatureField(form pdfcpu.Dict) model.FormFieldSignature {
	fmt.Println("TODO", form)
	return model.FormFieldSignature{}
}
