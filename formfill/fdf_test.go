package formfill

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/model"
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

func TestFill1(t *testing.T) {
	const path = "test/sample1.pdf"
	doc, _, err := reader.ParsePDFFile(path, reader.Options{})
	if err != nil {
		t.Fatal(err)
	}
	fields := doc.Catalog.AcroForm.Flatten()
	if L := len(fields); L != 65 {
		t.Errorf("expected 65 fields, got %d", L)
	}

	if got := fields["z9"].Field.AppearanceKeys(); !reflect.DeepEqual(got, []model.Name{"Oui"}) {
		t.Error(got)
	}

	err = FillForm(&doc, FDFDict{Fields: data}, true)
	if err != nil {
		t.Fatal(err)
	}

	out, err := os.Create("test/sample1_filled.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err = doc.Write(out, nil); err != nil {
		t.Error(err)
	}
}

func TestFillPDFjs(t *testing.T) {
	const path = "test/sample2.pdf"
	doc, _, err := reader.ParsePDFFile(path, reader.Options{})
	if err != nil {
		t.Fatal(err)
	}

	page := doc.Catalog.Pages.Flatten()[0]
	annots := make(map[*model.AnnotationDict]bool)
	for _, annot := range page.Annots {
		annots[annot] = true
	}
	for _, field := range doc.Catalog.AcroForm.Flatten() {
		if len(field.Field.Widgets) != 1 {
			t.Fatal("expected one widget only")
		}
		annot := field.Field.Widgets[0].AnnotationDict
		if !annots[annot] {
			t.Fatal("missing annotation widget in page Annots")
		}
	}

	err = FillForm(&doc, FDFDict{Fields: []FDFField{
		{
			T:      "Text1",
			Values: Values{V: FDFText("My text sample 1")},
		},
		{
			T:      "Text2",
			Values: Values{V: FDFText("My text sample 2")},
		},
	}}, true)
	if err != nil {
		t.Fatal(err)
	}

	if err = doc.WriteFile("test/sample2_filled.pdf", nil); err != nil {
		t.Fatal(err)
	}
}

func TestFill3(t *testing.T) {
	// https://github.com/benoitkugler/pdf/issues/8
	doc, _, err := reader.ParsePDFFile("test/sample3.pdf", reader.Options{})
	if err != nil {
		t.Fatal(err)
	}
	const fieldName = "3.2 Companies' House Reg No"
	field := doc.Catalog.AcroForm.Flatten()[fieldName]
	_, isText := field.Field.FT.(model.FormFieldText)
	if !isText {
		t.Fatal(err)
	}
	err = FillForm(&doc, FDFDict{Fields: []FDFField{
		{
			T: "3", Kids: []FDFField{
				{
					T:      "2 Companies' House Reg No",
					Values: Values{V: FDFText("A sample number")},
				},
			},
		},
	}}, true)
	if err != nil {
		t.Fatal(err)
	}

	if err = doc.WriteFile("test/sample3_filled.pdf", nil); err != nil {
		t.Fatal(err)
	}
}

func TestFill4(t *testing.T) {
	doc, _, err := reader.ParsePDFFile("test/sample4.pdf", reader.Options{})
	if err != nil {
		t.Fatal(err)
	}

	fields := doc.Catalog.AcroForm.Flatten()
	if got := fields["SOR A"].Field.AppearanceKeys(); !reflect.DeepEqual(got, []model.Name{"NON", "Oui"}) {
		t.Fatal(got)
	}
	if got := fields["SOR B"].Field.AppearanceKeys(); !reflect.DeepEqual(got, []model.Name{"NON", "Oui"}) {
		t.Fatal(got)
	}

	err = FillForm(&doc, FDFDict{Fields: []FDFField{
		{
			T: "SOR A", Values: Values{V: FDFName("Oui")},
		},
		{
			T: "SOR B", Values: Values{V: FDFName("Oui")},
		},
	}}, true)
	if err != nil {
		t.Fatal(err)
	}

	if err = doc.WriteFile("test/sample4_filled.pdf", nil); err != nil {
		t.Fatal(err)
	}
}
