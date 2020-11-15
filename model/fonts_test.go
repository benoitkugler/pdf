package model

import (
	"reflect"
	"testing"
)

func TestCloneFont(t *testing.T) {
	fonts := []Font{
		FontType0{Encoding: CMapEncodingEmbedded{}},
		FontType0{Encoding: CMapEncodingPredefined("")},
		FontType1{
			Encoding: MacRomanEncoding,
		},
		FontTrueType{
			Widths: make([]int, 12),
		},
		FontType3{
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
