package reader

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/phpdave11/gofpdf"
)

// we use the spec as a good test candidate
var pdfSpec model.Document

func init() {
	// loadPDFSpec()
}

func loadPDFSpec() {
	f, err := os.Open("datatest/PDF_SPEC.pdf")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	pdfSpec, _, err = ParsePDF(f, "")
	if err != nil {
		panic(err)
	}
}

func TestGeneratePDF(t *testing.T) {
	f := gofpdf.New("", "", "", "")
	f.SetProtection(0, "aaaa", "aaaa")
	if err := f.OutputFileAndClose("datatest/Protected.pdf"); err != nil {
		t.Fatal(err)
	}

	g := gofpdf.New("", "", "", "")
	g.AddPage()
	a := gofpdf.Attachment{
		Filename:    "Test.txt",
		Content:     []byte("AOIEPOZNSLKDSD"),
		Description: "Nice file !",
	}
	g.AddAttachmentAnnotation(&a, 10, 10, 20, 20)
	g.SetAttachments([]gofpdf.Attachment{
		a, a,
	})
	g.AddPage()
	l := g.AddLink()
	g.SetLink(l, 10, 1)
	g.Link(20, 30, 40, 50, l)
	g.Rect(20, 30, 40, 50, "D")
	if err := g.OutputFileAndClose("datatest/Links.pdf"); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateEmpty(t *testing.T) {
	g := gofpdf.New("", "", "", "")
	if err := g.OutputFileAndClose("datatest/Empty.pdf"); err != nil {
		t.Fatal(err)
	}
}

func TestOpen(t *testing.T) {
	// f, err := os.Open("datatest/descriptif.pdf")
	// f, err := os.Open("datatest/Links.pdf")
	// f, err := os.Open("datatest/f1118s1.pdf")
	// f, err := os.Open("datatest/transparents.pdf")
	// f, err := os.Open("datatest/ModeleRecuFiscalEditable.pdf")
	// f, err := os.Open("datatest/Protected.pdf")
	// f, err := os.Open("datatest/PDF_SPEC.pdf")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer f.Close()

	// doc, enc, err := ParsePDF(f, "")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	fmt.Println(pdfSpec.Catalog.OpenAction.ActionType)
	for _, p := range pdfSpec.Catalog.Pages.Flatten() {
		for _, an := range p.Annots {
			fmt.Println(an.C, an.M, an.NM)
		}
	}

}

func BenchmarkProcess(b *testing.B) {
	ctx, err := pdfcpu.ReadFile("datatest/PDF_SPEC.pdf", nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := ProcessContext(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDataset(t *testing.T) {
	files := [...]string{
		"datatest/descriptif.pdf",
		"datatest/Links.pdf",
		"datatest/f1118s1.pdf",
		"datatest/transparents.pdf",
		"datatest/ModeleRecuFiscalEditable.pdf",
		"datatest/CMYK_OP.pdf",
		"datatest/CMYKSpot_OP.pdf",
		"datatest/Shading.pdf",
		"datatest/Shading2.pdf",
		"datatest/Font_Substitution.pdf",
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

		fmt.Println("pages:", len(doc.Catalog.Pages.Flatten()))
	}
}

func TestType3(t *testing.T) {
	f, err := os.Open("datatest/type3.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	doc, _, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}
	type3Fonts := map[*model.FontDict]bool{}
	type3Refs := 0
	for _, page := range doc.Catalog.Pages.Flatten() {
		if res := page.Resources; res != nil {
			for _, font := range res.Font {
				if _, ok := font.Subtype.(model.FontType3); ok {
					type3Fonts[font] = true
					type3Refs++
				}
			}
		}
	}
	fmt.Println("type3 fonts found:", len(type3Fonts), "referenced", type3Refs, "times")
}

func TestProtected(t *testing.T) {
	f, err := os.Open("datatest/Protected.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, enc, err := ParsePDF(f, "aaaa")
	if err != nil {
		t.Fatal(err)
	}
	if enc == nil {
		t.Error("expected Encryption dictionary")
	}
	fmt.Println(*enc)
}

func TestWrite(t *testing.T) {
	var doc model.Document
	doc.Trailer.Info.Author = strings.Repeat("dd)d", 10)
	doc.Catalog.Pages.Kids = []model.PageNode{
		&model.PageObject{
			Contents: []model.ContentStream{
				{Stream: model.Stream{Content: []byte("mldskldm")}},
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
	err := doc.Write(out, nil)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(out.String())

	var inMemory io.ReadSeeker = bytes.NewReader(out.Bytes())

	doc2, _, err := ParsePDF(inMemory, "")
	if err != nil {
		t.Fatal(err)
	}
	if doc2.Trailer.Info.Author != doc.Trailer.Info.Author {
		t.Fatalf(doc2.Trailer.Info.Author)
	}
}

func TestReWrite(t *testing.T) {
	out, err := os.Create("datatest/PDF_SPEC_2.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	ti := time.Now()
	err = pdfSpec.Write(out, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("PDF wrote to disk in", time.Since(ti))

	out2 := bytes.Buffer{}
	ti = time.Now()
	err = pdfSpec.Write(&out2, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("PDF wrote to memory in", time.Since(ti))

	_, err = pdfcpu.ReadFile("datatest/PDF_SPEC_2.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out, err := os.Create("datatest/PDF_SPEC_bench.pdf")
		if err != nil {
			b.Fatal(err)
		}

		err = pdfSpec.Write(out, nil)
		if err != nil {
			b.Fatal(err)
		}
		out.Close()
	}
}
