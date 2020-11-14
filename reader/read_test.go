package reader

import (
	"bytes"
	"encoding/ascii85"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/phpdave11/gofpdf"
)

func TestGeneratePDF(t *testing.T) {
	f := gofpdf.New("", "", "", "")
	f.SetProtection(0, "aaa", "aaaa")
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

func decodeStream(stream model.ContentStream) ([]byte, error) {
	var current io.Reader = bytes.NewReader(stream.Content)
	for i, f := range stream.Filter {
		params := map[string]int{}
		for n, v := range stream.ParamsForFilter(i) {
			params[string(n)] = v
		}
		fp, err := filter.NewFilter(string(f), params)
		if err != nil {
			return nil, err
		}
		current, err = fp.Decode(current)
		if err != nil {
			return nil, err
		}
	}
	return ioutil.ReadAll(current)
}

func TestOpen(t *testing.T) {
	// f, err := os.Open("datatest/descriptif.pdf")
	// f, err := os.Open("datatest/Links.pdf")
	// f, err := os.Open("datatest/f1118s1.pdf")
	// f, err := os.Open("datatest/transparents.pdf")
	f, err := os.Open("datatest/ModeleRecuFiscalEditable.pdf")
	// f, err := os.Open("datatest/Protected.pdf")
	// f, err := os.Open("datatest/PDF_SPEC.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	doc, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(doc.Trailer.Encrypt)

	// fmt.Println(doc.Catalog.Outlines.Count())
	// fmt.Println(doc.Catalog.Outlines.First.Count())
	// fmt.Println(doc.Catalog.Outlines.Last().Count())
	// fmt.Println(string(doc.Catalog.Pages.Flatten()[0].Contents[0].Content[0:20]))
	// fmt.Println(doc.Catalog.Pages.Flatten()[0].Contents[0].Content[0:20])

	// for _, field := range doc.Catalog.AcroForm.Flatten() {
	// 	fmt.Println(field.FT, field.FullFieldName())
	// 	fmt.Printf("%T", field.FT)
	// }

}

func TestDataset(t *testing.T) {
	files := [...]string{
		"datatest/descriptif.pdf",
		"datatest/Links.pdf",
		"datatest/f1118s1.pdf",
		"datatest/transparents.pdf",
		"datatest/ModeleRecuFiscalEditable.pdf",
		"datatest/PDF_SPEC.pdf",
		"datatest/type3.pdf",
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}

		doc, err := ParsePDF(f, "")
		if err != nil {
			t.Fatal(err)
		}

		f.Close()

		fmt.Println("pages:", len(doc.Catalog.Pages.Flatten()))
	}
}

// func parseContentSteam(content []byte) ([]pdfcpu.Object, error) {
// 	var out []pdfcpu.Object
// 	s := string(bytes.TrimSpace(content))
// 	for len(s) > 0 {
// 		obj, err := pdfcpu.ParseNextObject(&s)
// 		if err != nil {
// 			return nil, err
// 		}
// 		out = append(out, obj)
// 	}
// 	return out, nil
// }

func TestType3(t *testing.T) {
	f, err := os.Open("datatest/type3.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	doc, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}
	type3Fonts := map[*model.Font]bool{}
	type3Refs := 0
	for _, page := range doc.Catalog.Pages.Flatten() {
		if res := page.Resources; res != nil {
			for _, font := range res.Font {
				if _, ok := font.Subtype.(model.Type3); ok {
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

	doc, err := ParsePDF(f, "aaa")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(doc.Trailer.Encrypt)
}
func TestStream(t *testing.T) {
	ctx, err := pdfcpu.ReadFile("datatest/PDF_SPEC.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	stream, _, err := ctx.XRefTable.DereferenceStreamDict(*pdfcpu.NewIndirectRef(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("datatest/PDF_SPEC.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Println(string(stream.Raw[0:20]))

}

func TestAlterFields(t *testing.T) {
	ctx, err := pdfcpu.ReadFile("ModeleRecuFiscalEditable.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	d, err := ctx.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	ac, _ := ctx.DereferenceDict(d["AcroForm"])
	f := ac["Fields"].(pdfcpu.Array)
	f[1] = pdfcpu.IndirectRef{ObjectNumber: 214}
	ac["Fields"] = f
	d["AcroForm"] = ac
	ref, err := ctx.XRefTable.IndRefForNewObject(d)
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.Root = ref
	ctx.Write.FileName = "SameFields.pdf"
	err = pdfcpu.Write(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBytes(t *testing.T) {
	bs := []byte{82, 201, 80, 85, 66, 76, 73, 81, 85, 69, 32, 70, 82, 65, 78, 199, 65, 73, 83, 69}
	for _, b := range bs {
		fmt.Println(string(rune(b)))
	}
	fmt.Println(string(bs))
}

func TestUnicode(t *testing.T) {
	b := byte(155)
	r := rune(0x203a)
	fmt.Println([]byte(string(r)))
	fmt.Println(string(b), string(r))

	fmt.Println(int(0x7d) - 20)

	s := "http://www.iso.org/iso/iso_catalogue/catalogue_tc/catalogue_detail.htm?csnumber=51502"
	var bu bytes.Buffer
	w := ascii85.NewEncoder(&bu)
	_, err := w.Write([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	fmt.Println(bu.Bytes())
	fmt.Println(bu.String())
}

func TestWrite(t *testing.T) {
	var doc model.Document
	doc.Trailer.Info.Author = strings.Repeat("dd)d", 10)
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
	err := doc.Write(out)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(out.String())

	var inMemory io.ReadSeeker = bytes.NewReader(out.Bytes())

	doc2, err := ParsePDF(inMemory, "")
	if err != nil {
		t.Fatal(err)
	}
	if doc2.Trailer.Info.Author != doc.Trailer.Info.Author {
		t.Fatalf(doc2.Trailer.Info.Author)
	}
}

func TestWritePDFSpec(t *testing.T) {
	// f, err := os.Open("datatest/PDF_SPEC.pdf")
	f, err := os.Open("datatest/type3.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(doc.Catalog.Pages.Flatten()[0].Contents[0].Content[0:20]))
	fmt.Println(doc.Catalog.Pages.Flatten()[0].Contents[0].Content[0:20])

	out, err := os.Create("datatest/type3_2.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	ti := time.Now()
	err = doc.Write(out)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("PDF wrote in %s", time.Since(ti))

	_, err = pdfcpu.ReadFile("datatest/type3_2.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}
