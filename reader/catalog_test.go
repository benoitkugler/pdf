package reader

import (
	"fmt"
	"os"
	"testing"
)

func TestOutline(t *testing.T) {
	f, err := os.Open("test/ModeleRecuFiscalEditable.pdf")
	if err != nil {
		t.Fatal(err)
	}

	doc, _, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range doc.Catalog.Outlines.Flatten() {
		fmt.Println(o.Title)
		fmt.Println(utf16Dec.NewEncoder().Bytes([]byte(o.Title)))
	}
}
