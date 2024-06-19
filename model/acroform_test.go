package model

import (
	"reflect"
	"testing"
)

func TestCloneAcro(t *testing.T) {
	a := AcroForm{
		Fields: []*FormFieldDict{
			{
				FormFieldInheritable: FormFieldInheritable{FT: FormFieldText{}},
				Widgets:              []FormFieldWidget{{}},
			},
		},
		NeedAppearances: true,
		DR:              ResourcesDict{ColorSpace: map[ColorSpaceName]ColorSpace{"eee": nil}},
	}

	cache := newCloneCache()
	a2 := a.clone(cache)

	if !reflect.DeepEqual(a, a2) {
		t.Errorf("expected %v, got %v", a, a2)
	}
}

func TestCloneForm(t *testing.T) {
	a := &FormFieldDict{
		FormFieldInheritable: FormFieldInheritable{FT: FormFieldText{}},
		Widgets: []FormFieldWidget{
			{
				AnnotationDict: &AnnotationDict{
					BaseAnnotation: BaseAnnotation{
						Contents: "sldml",
						Border:   &Border{DashArray: []Fl{4, 5, 6, 8}},
					},
					Subtype: AnnotationWidget{
						BS: &BorderStyle{
							S: "sd24",
						},
					},
				},
			},
		},
		AA: FormFielAdditionalActions{
			K: Action{ActionType: ActionJavaScript{JS: "sdlmsmd"}},
		},
	}

	cache := newCloneCache()
	a2 := a.clone(cache)

	if !reflect.DeepEqual(a, a2) {
		t.Errorf("expected %v, got %v", a, a2)
	}
}

func TestResolve(t *testing.T) {
	a1 := FormFieldDict{
		T:                    "a1",
		FormFieldInheritable: FormFieldInheritable{FT: FormFieldText{}},
	}
	a2 := a1
	a2.T = "5"
	b := &FormFieldDict{
		Kids: []*FormFieldDict{&a1, &a2},
		FormFieldInheritable: FormFieldInheritable{
			DA: "564",
		},
	}
	a1.Parent = b
	a2.Parent = b

	ac := AcroForm{
		Fields: []*FormFieldDict{b},
	}
	m := ac.Flatten()
	if L := len(m); L != 3 {
		t.Errorf("expected 3 fields, got %d", L)
	}
	for _, f := range m {
		if f.Merged.DA != "564" {
			t.Error()
		}
	}
}

func TestAppearanceKeys(t *testing.T) {
	var f FormFieldDict
	f.FT = FormFieldButton{}
	f.Widgets = []FormFieldWidget{
		{&AnnotationDict{BaseAnnotation: BaseAnnotation{AP: &AppearanceDict{N: AppearanceEntry{
			"Yes": &XObjectForm{},
			"Off": &XObjectForm{},
		}}}}},
		{&AnnotationDict{BaseAnnotation: BaseAnnotation{AP: &AppearanceDict{N: AppearanceEntry{
			"Yes": &XObjectForm{},
			"No":  &XObjectForm{},
		}}}}},
	}
	if !reflect.DeepEqual(f.AppearanceKeys(), []Name{"No", "Off", "Yes"}) {
		t.Error()
	}

	f.Widgets = nil
	if len(f.AppearanceKeys()) != 0 {
		t.Error()
	}
}
