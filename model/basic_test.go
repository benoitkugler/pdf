package model

import (
	"bytes"
	"fmt"
	"testing"
)

func TestFunction(t *testing.T) {
	var out bytes.Buffer
	w := newWriter(&out, Encrypt{})
	fn := Function{Domain: make([]Range, 4), Range: make([]Range, 3)}

	f1 := SampledFunction{
		Stream:        Stream{Content: []byte("654646464456")},
		BitsPerSample: 12,
		Order:         3,
		Size:          []int{1, 2, 35},
		Decode:        []Range{{1, 2}, {0.45654, 0.65487}},
		Encode:        [][2]float64{{1, 2}, {0.45654, 0.65487}},
	}
	fn.FunctionType = f1
	w.addObject(fn.pdfContent(w))

	f2 := ExpInterpolationFunction{N: 1, C0: make([]float64, 5)}
	fn.FunctionType = f2
	w.addObject(fn.pdfContent(w))

	f3 := StitchingFunction{
		Functions: []Function{fn, fn},
	}
	fn.FunctionType = f3
	w.addObject(fn.pdfContent(w))

	fmt.Println(out.String())
}
