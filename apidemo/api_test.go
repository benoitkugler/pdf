package apidemo

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader"
)

func TestEmbeddedFiles(t *testing.T) {
	// doc, _, err := reader.ParsePDFFile("test/descriptif.pdf", reader.Options{})
	var doc model.Document

	out, err := os.Create("test/embedded.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	// out := new(bytes.Buffer)

	files := []string{
		"api.go",
		"api_test.go, une description en français de ce superbé code",
	}
	ti := time.Now()
	err = AddAttachments(&doc, nil, out, files)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("done in", time.Since(ti))

	list := ListAttachments(doc)
	expected := []string{"api.go", "api_test.go"}
	if !reflect.DeepEqual(list, expected) {
		t.Errorf("expected %v, got %v", expected, list)
	}
}

func TestExtractContent(t *testing.T) {
	doc, _, err := reader.ParsePDFFile("test/descriptif.pdf", reader.Options{})
	if err != nil {
		t.Fatal(err)
	}
	err = ExtractContent(doc, "test/", nil)
	if err != nil {
		t.Error(err)
	}
}
