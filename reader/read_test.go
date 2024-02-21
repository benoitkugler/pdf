package reader

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
)

// the SPEC is a good test candidate
var pdfSpec model.Document

const password = "78eoln_-(_àç_-')"

func init() {
	// the PDF spec is used in several tests, but is heavy
	// so, when working on isolated test, you may want to avoid loading it
	// by commenting this line
	loadPDFSpec()
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
	fmt.Println("PDF spec loaded.")
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
	ctx, err := file.ReadFile("test/PDF_SPEC.pdf", nil)
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

func filesFromDir(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	var out []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		out = append(out, filepath.Join(dir, f.Name()))
	}
	return out
}

func TestDataset(t *testing.T) {
	for _, file := range filesFromDir("test/corpus") {
		fmt.Println("Parsing", file)

		doc, _, err := ParsePDFFile(file, Options{})
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("	Pages:", len(doc.Catalog.Pages.Flatten()))
		if len(doc.Catalog.Pages.FlattenInherit()) != len(doc.Catalog.Pages.Flatten()) {
			t.Fatalf("invalid page flatten")
		}
		fmt.Println("	Dests name:", len(doc.Catalog.Dests))
		fmt.Println("	Dests string:", len(doc.Catalog.Names.Dests.LookupTable()))
		fmt.Println("	Pages strings:", len(doc.Catalog.Names.Pages.LookupTable()))
		fmt.Println("	Templates strings:", len(doc.Catalog.Names.Templates.LookupTable()))
	}
}

func TestProtected(t *testing.T) {
	f, err := os.Open("test/ProtectedRC4.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, enc, err := ParsePDFReader(f, Options{UserPassword: password})
	if err != nil {
		t.Fatal(err)
	}
	if enc == nil {
		t.Fatal("expected Encryption dictionary")
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

func reWrite(doc model.Document, filename string) error {
	out, err := os.Create(filename)
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

	_, err = file.ReadFile(filename, nil)
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

func TestImportPage(t *testing.T) {
	// const file = "test/Venez le célébrer.pdf"
	const file = "test/Prince-de-paix (Dieu tu es saint).pdf"
	doc, _, err := ParsePDFFile(file, Options{})
	if err != nil {
		t.Fatal(err)
	}

	// for _, p := range doc.Catalog.Pages.Flatten() {
	// b, _ := p.DecodeAllContents()
	// fmt.Println(string(b))
	// fmt.Println(p.Resources.XObject["Im1"].(*model.XObjectImage).Filter)
	// fmt.Println(p.Resources.XObject["Im0"].(*model.XObjectImage).Filter)
	// }

	err = reWrite(doc, file+".pdf")
	if err != nil {
		t.Fatal(err)
	}
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
		"test/corpus/ModeleRecuFiscalEditable.pdf",
		"test/ttf.pdf",
		"test/ttf_kerning.pdf",
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

func TestDecrompress(t *testing.T) {
	file := "test/gradient_spread.pdf"
	doc, _, err := ParsePDFFile(file, Options{})
	if err != nil {
		t.Fatal(err)
	}
	decrompress := func(s *model.ContentStream) {
		fmt.Println("found content stream")
		b, err := s.Decode()
		if err != nil {
			t.Fatal(err)
		}
		s.Stream = model.Stream{ // remove filters
			Content: b,
		}
	}
	for _, page := range doc.Catalog.Pages.Flatten() {
		for i := range page.Contents {
			decrompress(&page.Contents[i])
		}
		if page.Resources == nil {
			continue
		}
		for _, xo := range page.Resources.XObject {
			if form, ok := xo.(*model.XObjectForm); ok {
				decrompress(&form.ContentStream)
			}
		}
	}

	err = doc.WriteFile(file+"_clear.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFreeObjects(t *testing.T) {
	for range [10]int{} {
		f, err := os.Open("test/JEM-0943.pdf")
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = ParsePDFReader(f, Options{})
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

func TestWrite2(t *testing.T) {
	doc, _, err := ParsePDFFile("test/text_orig.pdf", Options{})
	if err != nil {
		t.Fatal(err)
	}

	err = doc.WriteFile("test/text.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestJapanseOutline(t *testing.T) {
	// https://github.com/benoitkugler/pdf/issues/5
	file := "samples/JapaneseOutline.pdf"
	doc, _, err := ParsePDFFile(file, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Catalog.Outlines != nil {
		t.Fatal()
	}
}

func TestCerfaForm(t *testing.T) {
	// https://github.com/benoitkugler/pdf/issues/7
	file := "samples/CerfaArret.pdf"
	doc, _, err := ParsePDFFile(file, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if L := doc.Catalog.Pages.Count(); L != 5 {
		t.Fatal()
	}
	ap := doc.Catalog.Names.AP.LookupTable()
	if len(ap) != 0 {
		t.Fatal()
	}
}
