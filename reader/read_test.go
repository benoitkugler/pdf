package reader

import (
	"bytes"
	"encoding/ascii85"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

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

func decodeStream(stream model.ContentStream) ([]byte, error) {
	var current io.Reader = bytes.NewReader(stream.Content)
	for i, f := range stream.Filters {
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
	// f, err := os.Open("datatest/transparents.pdf")
	// f, err := os.Open("datatest/ModeleRecuFiscalEditable.pdf")
	// f, err := os.Open("datatest/Protected.pdf")
	f, err := os.Open("datatest/PDF_SPEC.pdf")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(doc.Trailer.Info)

	fontUsage := map[*model.Font]int{}
	iccUsage := map[*model.ICCBasedColorSpace]int{}
	patUsage := map[model.Pattern]int{}

	inspectCs := func(cs model.ColorSpace) {
		switch cs := cs.(type) {
		case *model.ICCBasedColorSpace:
			iccUsage[cs]++
		case model.UncoloredTilingPattern:
			if underlyingCs, ok := cs.UnderlyingColorSpace.(*model.ICCBasedColorSpace); ok {
				iccUsage[underlyingCs]++
			}
		case model.IndexedColorSpace:
			if baseCs, ok := cs.Base.(*model.ICCBasedColorSpace); ok {
				iccUsage[baseCs]++
			}
		}
	}

	for _, page := range doc.Catalog.Pages.Flatten() {
		for _, font := range page.Resources.Font {
			fontUsage[font]++
		}
		for _, sh := range page.Resources.Shading {
			inspectCs(sh.ColorSpace)
		}
		for _, pat := range page.Resources.Pattern {
			patUsage[pat]++
			if tiling, ok := pat.(*model.TilingPatern); ok {
				for _, cs := range tiling.Resources.ColorSpace {
					inspectCs(cs)
				}
			}
		}
		for _, cs := range page.Resources.ColorSpace {
			inspectCs(cs)
		}
	}
	fmt.Println(fontUsage)
	fmt.Println(iccUsage)
	fmt.Println(patUsage)

	// ct, err := decodeStream(doc.Catalog.Pages.Flatten()[15].Contents[0])
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println(string(ct))
	// fmt.Println(doc.Catalog.Names.Dests.LookupTable())
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
