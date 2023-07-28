package formfill

import (
	"fmt"
	"os"
	"testing"

	"github.com/benoitkugler/pdf/reader"
	"github.com/benoitkugler/pdf/reader/file"
)

var data = []FDFField{
	{T: "z1", Values: Values{V: FDFText("879-sde9-898")}},
	{T: "z2", Values: Values{V: FDFText("ACVE")}},
	{T: "z4", Values: Values{V: FDFText("La Maison du Rocher")}},
	{T: "z5", Values: Values{V: FDFText("26160")}},
	{T: "z5b", Values: Values{V: FDFText("CHAMALOC")}},
	{T: "z6", Values: Values{V: FDFText("Créer et gérer des séjours pour enfants, adolescents et adultes.")}},
	{T: "z7", Values: Values{V: FDFText("Faire connaître, à travers des animations adaptées à l’âge des participants, les valeurs chrétiennes.")}},
	{T: "z9", Values: Values{V: FDFName("Oui")}},
	{T: "d3", Values: Values{V: FDFText("1957")}},
	{T: "d3b", Values: Values{V: FDFText("1957")}},
	{T: "d1", Values: Values{V: FDFText("5")}},
	{T: "d1b", Values: Values{V: FDFText("29")}},
	{T: "d2", Values: Values{V: FDFText("1")}},
	{T: "d2b", Values: Values{V: FDFText("1")}},
	{T: "z29", Values: Values{V: FDFText("')='à=(kmlrk'")}},
	{T: "z30", Values: Values{V: FDFText("mldmskld8+-*")}},
	{T: "z31", Values: Values{V: FDFText("lmemzkd\ndlss\nzlkdsmkmdkmsdk")}},
	{T: "z32", Values: Values{V: FDFText("kdskdl")}},
	{T: "z33", Values: Values{V: FDFText("ùmdslsùmd")}},
	{T: "z34", Values: Values{V: FDFText("1457.46")}},
	{T: "z35", Values: Values{V: FDFText("mille quatre cent cinquante-sept euros et quarante-six centimes")}},
	{T: "z36", Values: Values{V: FDFText("25")}},
	{T: "z37", Values: Values{V: FDFText("11")}},
	{T: "z38", Values: Values{V: FDFText("2020")}},
	{T: "z50", Values: Values{V: FDFName("Oui")}},
	{T: "z39", Values: Values{V: FDFName("Oui")}},
	{T: "z46", Values: Values{V: FDFName("Oui")}},
	{T: "z44", Values: Values{V: FDFName("Oui")}},
	{T: "z52", Values: Values{V: FDFText("25")}},
	{T: "z53", Values: Values{V: FDFText("11")}},
	{T: "z54", Values: Values{V: FDFText("2020")}},
}

func TestFDF(t *testing.T) {
	const path = "test/ModeleRecuFiscalEditable.pdf"
	doc, _, err := reader.ParsePDFFile(path, reader.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if L := len(doc.Catalog.AcroForm.Flatten()); L != 65 {
		t.Errorf("expected 65 fields, got %d", L)
	}

	// ft := doc.Catalog.AcroForm.DR.Font["Helv"]
	// fmt.Println(fonts.BuildFont(ft).Font.GetWidth('à', 10))
	// t1 := ft.Subtype.(model.FontType1)
	// fmt.Println(t1.Widths[224-t1.FirstChar])
	// fmt.Println(enc.BaseEncoding)
	// fmt.Println(enc.Differences)

	err = FillForm(&doc, FDFDict{Fields: data}, true)
	if err != nil {
		t.Fatal(err)
	}

	out, err := os.Create("test/filled.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err = doc.Write(out, nil); err != nil {
		t.Error(err)
	}
}

func TestFDFFile(t *testing.T) {
	fi, err := file.ReadFDFFile("test/sample.fdf")
	if err != nil {
		t.Fatal(err)
	}
	out, err := processFDFFile(fi)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Fields) != 5 {
		t.Fatal()
	}
	fmt.Println(out)
}
