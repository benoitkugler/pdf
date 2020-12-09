package reader

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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
	// loadPDFSpec()

	// generatePDFs()
}

func loadPDFSpec() {
	f, err := os.Open("test/PDF_SPEC.pdf")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	pdfSpec, _, err = ParsePDFReader(f, Options{})
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
			if meta, ok := pr["Metadata"].(model.MetadataStream); ok {
				fmt.Println("	Metadata length:", meta.Length())
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
		r := newResolver()
		r.xref = ctx.XRefTable
		_, _, err := r.processContext()
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

		doc, _, err := ParsePDFFile(file, Options{})
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("	Pages:", len(doc.Catalog.Pages.Flatten()))
		fmt.Println("	Dests name:", len(doc.Catalog.Dests))
		fmt.Println("	Dests string:", len(doc.Catalog.Names.Dests.LookupTable()))
		fmt.Println("	Pages strings:", len(doc.Catalog.Names.Pages.LookupTable()))
		fmt.Println("	Templates strings:", len(doc.Catalog.Names.Templates.LookupTable()))
	}
}

func TestProtected(t *testing.T) {
	f, err := os.Open("test/Protected.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, enc, err := ParsePDFReader(f, Options{UserPassword: password})
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

	doc, _, err := ParsePDFReader(f, Options{})
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

	doc2, _, err := ParsePDFReader(inMemory, Options{})
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
		"test/ttf.pdf",
	} {
		doc, _, err := ParsePDFFile(file, Options{})
		if err != nil {
			t.Fatal(err)
		}
		for _, page := range doc.Catalog.Pages.Flatten() {
			if r := page.Resources; r != nil {
				for _, font := range r.Font {
					if ttf, ok := font.Subtype.(model.FontTrueType); ok {
						_, err := fonts.BuildFont(font)
						if err != nil {
							t.Fatal(err)
						}
						fmt.Println(ttf.Encoding)
						// b, err := ttf.FontDescriptor.FontFile.Decode()
						// if err != nil {
						// 	t.Fatal(err)
						// }
						// err = ioutil.WriteFile("font.ttf", b, os.ModePerm)
						// if err != nil {
						// 	t.Error(err)
						// }
						// ft, err := sfnt.Parse(bytes.NewReader(b))
						// // ft, err := sfnt.Parse(b)
						// if err != nil {
						// 	t.Fatal(err)
						// }

						// fmt.Println(ft.HheaTable())
						// fmt.Println(ft.OS2Table())
						// fmt.Println(ft.GposTable())
						// fmt.Println(ft.CmapTable())
						// ft.Kern(&b, sfnt.GlyphIndex(b1), sfnt.GlyphIndex(b2))

					}
				}
			}
		}
	}
}

func TestMedia(t *testing.T) {
	for _, file := range []string{
		"test/mp3.pdf",
		"test/mp4.pdf",
	} {

		doc, _, err := ParsePDFFile(file, Options{})
		if err != nil {
			t.Fatal(err)
		}
		media := 0
		for _, page := range doc.Catalog.Pages.Flatten() {
			for _, annot := range page.Annots {
				if sc, ok := annot.Subtype.(model.AnnotationScreen); ok {
					if sc.P == nil {
						t.Error("missing page object in Screen Annotation")
					}
					if rd, ok := sc.A.ActionType.(model.ActionRendition); ok {
						if med, ok := rd.R.Subtype.(model.RenditionMedia); ok {
							fmt.Println(med.P)
							media++
						}
					}
				}
			}
		}
		if media != 1 {
			t.Errorf("expected one media file, got %d", media)
		}

		err = reWrite(doc, file+".pdf")
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestType1C(t *testing.T) {
	file := "test/UTF-32.pdf"
	// file := "test/type1C.pdf"
	doc, _, err := ParsePDFFile(file, Options{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[*model.FontDict]bool{} // avoid processing the same font twice
	for _, page := range doc.Catalog.Pages.Flatten() {
		if r := page.Resources; r != nil {
			for _, font := range r.Font {
				if seen[font] {
					continue
				}
				seen[font] = true
				var fontFile *model.FontFile
				if ttf, ok := font.Subtype.(model.FontType1); ok {
					if ft := ttf.FontDescriptor.FontFile; ft != nil && ft.Subtype == "Type1C" {
						fontFile = ft
					}
				} else if type0, ok := font.Subtype.(model.FontType0); ok {
					if ft := type0.DescendantFonts.FontDescriptor.FontFile; ft != nil && ft.Subtype == "CIDFontType0C" {
						fontFile = ft
					}
				}
				if fontFile == nil {
					continue
				}
				// _, err := fonts.BuildFont(font)
				// if err != nil {
				// 	t.Fatal(err)
				// }
				// fmt.Println(ttf.Encoding)
				b, err := fontFile.Decode()
				if err != nil {
					t.Fatal(err)
				}
				err = ioutil.WriteFile(string(font.Subtype.FontName())+".cff", b, os.ModePerm)
				if err != nil {
					t.Error(err)
				}
				// ft, err := sfnt.Parse(bytes.NewReader(b))
				// // ft, err := sfnt.Parse(b)
				// if err != nil {
				// 	t.Fatal(err)
				// }

				// fmt.Println(ft.HheaTable())
				// fmt.Println(ft.OS2Table())
				// fmt.Println(ft.GposTable())
				// fmt.Println(ft.CmapTable())
				// ft.Kern(&b, sfnt.GlyphIndex(b1), sfnt.GlyphIndex(b2))
			}
		}
	}
}
