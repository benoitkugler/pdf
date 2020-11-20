package apidemo

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
)

func TestEmbeddedFiles(t *testing.T) {
	// fin, err := os.Open("type3.pdf")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer fin.Close()
	// doc, err := reader.ParsePDF(fin, "")
	// if err != nil {
	// 	t.Fatal(err)
	// }
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
