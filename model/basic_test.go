package model

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestFunction(t *testing.T) {
	var out bytes.Buffer
	w := newWriter(&out, Encrypt{})
	fn := FunctionDict{Domain: make([]Range, 4), Range: make([]Range, 3)}

	f1 := FunctionSampled{
		Stream:        Stream{Content: []byte("654646464456")},
		BitsPerSample: 12,
		Order:         3,
		Size:          []int{1, 2, 35},
		Decode:        []Range{{1, 2}, {0.45654, 0.65487}},
		Encode:        [][2]float64{{1, 2}, {0.45654, 0.65487}},
	}
	fn.FunctionType = f1
	w.addObject(fn.pdfContent(w))

	f2 := FunctionExpInterpolation{N: 1, C0: make([]float64, 5)}
	fn.FunctionType = f2
	w.addObject(fn.pdfContent(w))

	f3 := FunctionStitching{
		Functions: []FunctionDict{fn, fn},
	}
	fn.FunctionType = f3
	w.addObject(fn.pdfContent(w))

	f4 := FunctionPostScriptCalculator(f1.Stream)
	fn.FunctionType = f4
	w.addObject(fn.pdfContent(w))

	fmt.Println(out.String())
}

func TestClone(t *testing.T) {
	fn := FunctionDict{Domain: make([]Range, 4), Range: make([]Range, 3)}
	f1 := FunctionSampled{
		Stream:        Stream{Content: []byte("654646464456")},
		BitsPerSample: 12,
		Order:         3,
		Size:          []int{1, 2, 35},
		Decode:        []Range{{1, 2}, {0.45654, 0.65487}},
		Encode:        [][2]float64{{1, 2}, {0.45654, 0.65487}},
	}
	f2 := FunctionExpInterpolation{N: 1, C0: make([]float64, 5)}
	stitched := fn
	stitched.FunctionType = f2
	f3 := FunctionStitching{
		Functions: []FunctionDict{stitched, stitched},
	}
	f4 := FunctionPostScriptCalculator(f1.Stream)

	var types = []Function{f1, f2, f3, f4}

	for _, fnType := range types {
		fn.FunctionType = fnType
		fnClone := fn.Clone()
		if !reflect.DeepEqual(fn, fnClone) {
			t.Fatalf("expected equal functions, got %v and %v", fn, fnClone)
		}
	}
}
