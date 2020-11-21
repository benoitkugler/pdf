package reader

import (
	"os"
	"testing"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/exp/errors/fmt"
)

func TestWidgets(t *testing.T) {
	file := "datatest/ModeleRecuFiscalEditable.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	doc, _, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	for _, page := range doc.Catalog.Pages.Flatten() {
		for _, annot := range page.Annots {
			if w, ok := annot.Subtype.(model.AnnotationWidget); ok {
				fmt.Println(w.MK, w.AA)
			}
		}
	}
	for _, field := range doc.Catalog.AcroForm.Flatten() {
		fmt.Println(field.Widgets)
	}
}
