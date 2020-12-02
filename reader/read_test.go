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

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/phpdave11/gofpdf"
)

// the SPEC is a good test candidate
var pdfSpec model.Document

const password = "78eoln_-(_àç_-')"

func init() {
	// the PDF spec is used in several tests, but is heavy
	// so, when working on isolated test, you may want to avoid loading it
	// by commenting this line
	loadPDFSpec()

	// generatePDFs()
}

func loadPDFSpec() {
	f, err := os.Open("test/PDF_SPEC.pdf")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	pdfSpec, _, err = ParsePDF(f, "")
	if err != nil {
		panic(err)
	}
}

func generatePDFs() {
	f := gofpdf.New("", "", "", "")
	f.SetProtection(0, password, "aaaa")
	if err := f.OutputFileAndClose("test/Protected.pdf"); err != nil {
		log.Fatal(err)
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
	if err := g.OutputFileAndClose("test/Links.pdf"); err != nil {
		log.Fatal(err)
	}

	g = gofpdf.New("", "", "", "")
	if err := g.OutputFileAndClose("test/Empty.pdf"); err != nil {
		log.Fatal(err)
	}
}

func TestOpen(t *testing.T) {
	if L := len(pdfSpec.Catalog.Pages.Flatten()); L != 756 {
		t.Errorf("expected 756 pages, got %d", L)
	}

	for _, p := range pdfSpec.Catalog.Pages.Flatten() {
		for _, pr := range p.Resources.Properties {
			if pr.Metadata != nil {
				fmt.Println(pr.Metadata.Length())
			}
		}
	}

}

func BenchmarkProcess(b *testing.B) {
	ctx, err := pdfcpu.ReadFile("test/PDF_SPEC.pdf", nil)
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
		"test/Links.pdf",
		"test/Empty.pdf",
		"test/descriptif.pdf",
		"test/f1118s1.pdf",
		"test/transparents.pdf",
		"test/ModeleRecuFiscalEditable.pdf",
		"test/CMYK_OP.pdf",
		"test/CMYKSpot_OP.pdf",
		"test/Shading.pdf",
		"test/Shading4.pdf",
		"test/Font_Substitution.pdf",
	}

	for _, file := range files {
		fmt.Println("Parsing", file)

		doc, _, err := ParseFile(file, "")
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("	Pages:", len(doc.Catalog.Pages.Flatten()))
		fmt.Println("	Dests string:", len(doc.Catalog.Names.Dests.LookupTable()))
		fmt.Println("	Dests name:", len(doc.Catalog.Dests))
	}
}

func TestProtected(t *testing.T) {
	f, err := os.Open("test/Protected.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, enc, err := ParsePDF(f, password)
	if err != nil {
		t.Fatal(err)
	}
	if enc == nil {
		t.Error("expected Encryption dictionary")
	}
	if enc.Filter != "Standard" {
		t.Errorf("expected Standard encryption, got %s", enc.Filter)
	}
}

func TestType3(t *testing.T) {
	f, err := os.Open("test/type3.pdf")
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

func reWrite(doc model.Document, file string) error {
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()

	ti := time.Now()
	err = doc.Write(out, nil)
	if err != nil {
		return err
	}
	fmt.Println("PDF wrote to disk in", time.Since(ti))

	_, err = pdfcpu.ReadFile(file, nil)
	return err
}

func TestReWrite(t *testing.T) {
	err := reWrite(pdfSpec, "test/PDF_SPEC.pdf.pdf")
	if err != nil {
		t.Error(err)
	}

	out2 := bytes.Buffer{}
	ti := time.Now()
	err = pdfSpec.Write(&out2, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("PDF wrote to memory in", time.Since(ti))
}

func BenchmarkWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out, err := os.Create("test/PDF_SPEC_bench.pdf")
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

func TestEmbeddedTTF(t *testing.T) {
	for _, file := range [...]string{
		"test/symbolic_ttf.pdf",
		"test/ModeleRecuFiscalEditable.pdf",
	} {
		doc, _, err := ParseFile(file, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, page := range doc.Catalog.Pages.Flatten() {
			if r := page.Resources; r != nil {
				for _, font := range r.Font {
					if _, ok := font.Subtype.(model.FontTrueType); ok {
						_, err = fonts.BuildFont(font)
						if err != nil {
							t.Fatal(err)
						}
						fmt.Println(font.Subtype.FontName())
					}
				}
			}
		}
	}
}

func TestFlate(t *testing.T) {
	for _, page := range pdfSpec.Catalog.Pages.Flatten() {
		for _, s := range page.Contents {
			if len(s.Filter) == 1 && s.Filter[0].Name == model.Flate {
				fmt.Println(s.Content[s.Length()-8:])
			}
		}
	}
}
