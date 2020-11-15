package model

import (
	"reflect"
	"testing"
)

func TestCloneFont(t *testing.T) {
	fonts := []FontType{
		Type0{Encoding: EmbeddedCMapEncoding{}},
		Type0{Encoding: PredefinedCMapEncoding("")},
		Type1{
			Encoding: MacRomanEncoding,
		},
		TrueType{
			Widths: make([]int, 12),
		},
		Type3{
			Encoding:  MacExpertEncoding,
			CharProcs: map[Name]ContentStream{"mkdmsk": {}},
		},
	}
	c := cloneCache{}
	for _, f := range fonts {
		f2 := f.clone(c)
		if !reflect.DeepEqual(f, f2) {
			t.Fatalf("expected deep equality, got %v and %v", f, f2)
		}
	}
}
