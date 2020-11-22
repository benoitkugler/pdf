package reader

import (
	"os"
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func annotsToPageNumber(doc model.Document) map[model.FormFieldWidget]int {
	out := map[model.FormFieldWidget]int{}
	pages := doc.Catalog.Pages.Flatten()
	for nb, page := range pages {
		for _, annot := range page.Annots {
			if _, ok := annot.Subtype.(model.AnnotationWidget); ok {
				out[model.FormFieldWidget{AnnotationDict: annot}] = nb
			}
		}
	}
	return out
}

func checkAnnotsWidget(t *testing.T, doc model.Document, annots map[model.FormFieldWidget]int) {
	for _, field := range doc.Catalog.AcroForm.Flatten() {
		for _, w := range field.Widgets {
			_, ok := annots[w]
			if !ok {
				t.Errorf("annotation widget not found in the Annots lists %v", w)
			}
		}
	}
}

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

	annots := annotsToPageNumber(doc)

	widgetPerPages := map[int]int{}
	for _, pageNb := range annots {
		widgetPerPages[pageNb]++
	}
	if nbP1, nbP2 := widgetPerPages[0], widgetPerPages[1]; nbP1 != 39 || nbP2 != 26 {
		t.Errorf("expected 39 widgets on page 0 and 26 on page 1, got %d and %d", nbP1, nbP2)
	}
}

func TestForms(t *testing.T) {
	files := []string{
		// "datatest/f990se.pdf",
		// "datatest/f1041t.pdf",
		// "datatest/f1118s1.pdf",
		// "datatest/f4506c.pdf",
		"datatest/ModeleRecuFiscalEditable.pdf",
	}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		doc, _, err := ParsePDF(f, "")
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		annots := annotsToPageNumber(doc)
		checkAnnotsWidget(t, doc, annots)

		err = reWrite(doc, file+".pdf")
		if err != nil {
			t.Error(err)
		}

	}
}
