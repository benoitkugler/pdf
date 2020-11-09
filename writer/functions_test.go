package writer

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func TestFunction(t *testing.T) {
	out := &bytes.Buffer{}
	w := newWriter(out)
	fn := model.Function{Domain: make([]model.Range, 4), Range: make([]model.Range, 3)}

	f1 := model.SampledFunction{
		ContentStream: model.ContentStream{Content: []byte("654646464456")},
		BitsPerSample: 12,
		Order:         3,
		Size:          []int{1, 2, 35},
		Decode:        []model.Range{{1, 2}, {0.45654, 0.65487}},
		Encode:        [][2]float64{{1, 2}, {0.45654, 0.65487}},
	}
	fn.FunctionType = f1
	w.writeFunction(fn)

	f2 := model.ExpInterpolationFunction{N: 1, C0: make([]float64, 5)}
	fn.FunctionType = f2
	w.writeFunction(fn)

	f3 := model.StitchingFunction{
		Functions: []model.Function{fn, fn},
	}
	fn.FunctionType = f3
	w.writeFunction(fn)

	fmt.Println(out.String())
}
