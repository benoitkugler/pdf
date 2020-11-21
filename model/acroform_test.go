package model

import (
	"reflect"
	"testing"
)

func TestCloneAcro(t *testing.T) {
	a := &AcroForm{
		Fields: []*FormFieldDict{
			{
				FT:      FormFieldText{},
				Widgets: []Widget{{}},
			},
		},
		NeedAppearances: true,
		DR:              &ResourcesDict{ColorSpace: map[Name]ColorSpace{"eee": nil}},
	}

	cache := newCloneCache()
	a2 := a.clone(cache)

	if !reflect.DeepEqual(a, a2) {
		t.Errorf("expected %v, got %v", a, a2)
	}
}

func TestCloneForm(t *testing.T) {
	a := &FormFieldDict{
		FT: FormFieldText{},
		Widgets: []Widget{
			{
				BaseAnnotation: BaseAnnotation{
					Contents: "sldml",
					Border:   &Border{DashArray: []float64{4, 5, 6, 8}},
				},
				Subtype: AnnotationWidget{
					BS: &BorderStyle{
						S: "sd24",
					},
				},
			},
		},
		AA: &FormFielAdditionalActions{},
	}

	cache := newCloneCache()
	a2 := a.clone(cache)

	if !reflect.DeepEqual(a, a2) {
		t.Errorf("expected %v, got %v", a, a2)
	}
}
