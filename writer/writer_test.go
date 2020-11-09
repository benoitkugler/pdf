package writer

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestWrite(t *testing.T) {
	var doc model.Document
	doc.Trailer.Info.Author = strings.Repeat("ddd", 100)
	doc.Catalog.Pages.Kids = []model.PageNode{
		&model.PageObject{
			Contents: model.Contents{
				model.ContentStream{Content: []byte("mldskldm")},
			},
		},
		model.PageTree{
			Kids: []model.PageNode{
				&model.PageObject{},
				&model.PageObject{},
			},
		},
	}
	out := &bytes.Buffer{}
	err := Write(doc, out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.String())

	inMemory := bytes.NewReader(out.Bytes())
	_, err = pdfcpu.Read(inMemory, nil)
	if err != nil {
		t.Fatal(err)
	}

	doc2, err := reader.ParsePDF(inMemory, "")
	if err != nil {
		t.Fatal(err)
	}
	if doc2.Trailer.Info.Author != doc.Trailer.Info.Author {
		t.Fatal()
	}
}
