package model

import (
	"bytes"
	"fmt"
	"testing"
)

type inMemoryOutput struct {
	*bytes.Buffer
	n Reference
}

func newOut() *inMemoryOutput {
	return &inMemoryOutput{Buffer: &bytes.Buffer{}}
}

func (*inMemoryOutput) EncodeString(s string, mode StringEncoding) string {
	return s
}
func (p *inMemoryOutput) CreateObject() Reference {
	p.n++
	return p.n
}
func (p *inMemoryOutput) WriteObject(content string, stream []byte, ref Reference) {
	p.WriteString(fmt.Sprintf("%d 0 obj\n", ref))
	p.WriteString(content)
	if stream != nil {
		p.WriteString("\nstream")
		p.Write(stream)
		p.WriteString("\nendstream")
	}
	p.WriteString("\nendobj")
}

func TestFunction(t *testing.T) {
	out := newOut()
	w := newWriter(out)
	fn := Function{Domain: make([]Range, 4), Range: make([]Range, 3)}

	f1 := SampledFunction{
		ContentStream: ContentStream{Content: []byte("654646464456")},
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
