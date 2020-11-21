package model

import (
	"reflect"
	"testing"
)

func TestEncodingString(t *testing.T) {
	diffs := Differences{
		1:  "dsd",
		2:  "mldsks",
		3:  "mdsùldùs",
		10: "ee",
		11: "sd",
		12: "ee",
		7:  "88",
	}
	exp := "[ 1/dsd/mldsks/mdsùldùs 7/88 10/ee/sd/ee]"
	if d := diffs.PDFString(); d != exp {
		t.Errorf("expected %s, got %v", exp, d)
	}
}

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
