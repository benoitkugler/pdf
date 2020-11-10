package writer

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestFunction(t *testing.T) {
	out := &bytes.Buffer{}
	w := newWriter(out)
	processor := model.NewPDFWritter(w)
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
	fn.PDFBytes(processor)

	f2 := model.ExpInterpolationFunction{N: 1, C0: make([]float64, 5)}
	fn.FunctionType = f2
	fn.PDFBytes(processor)

	f3 := model.StitchingFunction{
		Functions: []model.Function{fn, fn},
	}
	fn.FunctionType = f3
	fn.PDFBytes(processor)

	fmt.Println(out.String())
}

func TestWrite(t *testing.T) {
	var doc model.Document
	doc.Trailer.Info.Author = strings.Repeat("ddd", 100)
	doc.Catalog.Pages.Kids = []model.PageNode{
		&model.PageObject{
			Contents: []model.ContentStream{
				{Content: []byte("mldskldm")},
			},
		},
		&model.PageTree{
			Kids: []model.PageNode{
				&model.PageObject{},
				&model.PageObject{},
			},
		},
	}
	out := &bytes.Buffer{}
	err := Write(doc, out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.String())

	inMemory := bytes.NewReader(out.Bytes())
	_, err = pdfcpu.Read(inMemory, nil)
	if err != nil {
		t.Fatal(err)
	}

	doc2, err := reader.ParsePDF(inMemory, "")
	if err != nil {
		t.Fatal(err)
	}
	if doc2.Trailer.Info.Author != doc.Trailer.Info.Author {
		t.Fatal()
	}
}

func TestWritePDFSpec(t *testing.T) {
	f, err := os.Open("../../reader/datatest/PDF_SPEC.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := reader.ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(doc.Catalog.Pages.Flatten()[0].Contents[0].Content[0:20]))
	fmt.Println(doc.Catalog.Pages.Flatten()[0].Contents[0].Content[0:20])

	out, err := os.Create("datatest/test.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	ti := time.Now()
	err = Write(doc, out)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("PDF  wrote in %s", time.Since(ti))

	_, err = pdfcpu.ReadFile("datatest/test.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}
