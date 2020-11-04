package reader

import (
	"fmt"
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/phpdave11/gofpdf"
)

func TestGeneratePDF(t *testing.T) {
	f := gofpdf.New("", "", "", "")
	f.SetProtection(0, "aaa", "aaaa")
	f.OutputFileAndClose("Protected.pdf")
}

func TestOpen(t *testing.T) {
	f, err := os.Open("ModeleRecuFiscalEditable.pdf")
	// f, err := os.Open("SameFields.pdf")
	// f, err := os.Open("Protected.pdf")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ParsePDF(f, "aaa")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println((*doc.Catalog.AcroForm.Fields[9].AP.N)["Oui"].Resources)
	fmt.Println(doc.Catalog.AcroForm.Fields[9].AP.D)
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
