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
