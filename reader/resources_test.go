package reader

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestDifferences(t *testing.T) {
	expected := model.Differences{24: "breve", 25: "caron", 39: "quotesingle", 96: "grave", 128: "bullet", 129: "emdash"}
	ar := pdfcpu.Array{
		pdfcpu.Integer(39),
		pdfcpu.Name("quotesingle"),
		pdfcpu.Integer(24),
		pdfcpu.Name("breve"),
		pdfcpu.Name("caron"),
		pdfcpu.Integer(96),
		pdfcpu.Name("grave"),
		pdfcpu.Integer(128),
		pdfcpu.Name("bullet"),
		pdfcpu.Name("emdash"),
	}
	var r resolver
	diff := r.parseDiffArray(ar)
	if !reflect.DeepEqual(diff, expected) {
		t.Errorf("expected %v, got %v", expected, diff)
	}
}

func TestGradients(t *testing.T) {
	doc, _, err := ParsePDFFile("test/gradients.pdf", Options{})
	if err != nil {
		t.Fatal(err)
	}

	var processRes func(model.ResourcesDict)
	processRes = func(r model.ResourcesDict) {
		for name, pattern := range r.Pattern {
			if sh, ok := pattern.(*model.PatternShading); ok {
				fmt.Printf("Pattern shading %s: %T\n", name, sh.Shading.ShadingType)
				fmt.Println("Pattern state", sh.ExtGState, "pattern cm", sh.Matrix)
			}
		}
		for name, state := range r.ExtGState {
			if g := state.SMask.G; g != nil {
				b, _ := g.Decode()
				fmt.Println("SMask  for", name, ":", string(b))
				fmt.Println("BBox for soft max Xobject", g.BBox)
				processRes(g.Resources)
			}
		}
		for _, obj := range r.XObject {
			if f, ok := obj.(*model.XObjectForm); ok {
				processRes(f.Resources)
			}
		}
		for name, obj := range r.Shading {
			fmt.Printf("Shading %s: %T\n", name, obj.ShadingType)
		}
	}
	for _, page := range doc.Catalog.Pages.Flatten() {
		if page.Resources != nil {
			processRes(*page.Resources)
		}
	}
}
