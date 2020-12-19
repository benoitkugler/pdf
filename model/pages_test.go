package model

import (
	"reflect"
	"testing"
)

func TestCloneResources(t *testing.T) {
	r := ResourcesDict{
		ExtGState: map[Name]*GraphicState{
			"lmdsm": {
				D: &DashPattern{
					Array: []Fl{78, 9},
				},
			},
		},
		ColorSpace: map[Name]ColorSpace{"eee": nil},
	}

	cache := newCloneCache()

	r2 := r.clone(cache)

	if !reflect.DeepEqual(r, r2) {
		t.Errorf("expected %v, got %v2", r, r2)
	}
}

func TestShallowClone(t *testing.T) {
	r := ResourcesDict{
		ExtGState: map[Name]*GraphicState{
			"lmdsm": {
				D: &DashPattern{
					Array: []Fl{78, 9},
				},
			},
		},
		ColorSpace: map[Name]ColorSpace{"eee": nil},
	}

	r2 := r.ShallowCopy()
	r2.ColorSpace["eee"] = ColorSpaceCMYK

	if r.ColorSpace["eee"] != nil {
		t.Error("expected no impact")
	}

	r.ExtGState["lmdsm"].Ca = ObjFloat(4.)
	if r2.ExtGState["lmdsm"].Ca == nil {
		t.Error("expected impact")
	}
}
