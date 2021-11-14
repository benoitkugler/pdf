package reader

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/text/encoding/unicode"
)

func TestOutline(t *testing.T) {
	f, err := os.Open("test/corpus/ModeleRecuFiscalEditable.pdf")
	if err != nil {
		t.Fatal(err)
	}

	doc, _, err := ParsePDFReader(f, Options{})
	if err != nil {
		t.Fatal(err)
	}

	enc := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder()
	for _, o := range doc.Catalog.Outlines.Flatten() {
		fmt.Println(o.Title)
		fmt.Println(enc.Bytes([]byte(o.Title)))
	}
}
