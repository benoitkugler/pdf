package model

import (
	"reflect"
	"testing"
)

func TestCloneAcro(t *testing.T) {
	a := &AcroForm{
		Fields: []*FormFieldDict{
			{
				FormFieldInheritable: FormFieldInheritable{FT: FormFieldText{}},
				Widgets:              []FormFieldWidget{{}},
			},
		},
		NeedAppearances: true,
		DR:              ResourcesDict{ColorSpace: map[Name]ColorSpace{"eee": nil}},
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
