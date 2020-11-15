package apidemo

import (
	"os"
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func TestEmbeddedFiles(t *testing.T) {
	var doc model.Document

	f, err := os.Create("embedded.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = AddAttachments(&doc, f, []string{
		"api.go",
		"api_test.go,une description en français de ce superbé code",
	})
	if err != nil {
		t.Fatal(err)
	}

	list := ListAttachments(doc)
	expected := []string{"api.go", "api_test.go"}
	if !reflect.DeepEqual(list, expected) {
		t.Errorf("expected %v, got %v", expected, list)
	}
}
