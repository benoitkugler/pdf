package model

import (
	"bytes"
	"testing"
)

func TestEmptyDocument(t *testing.T) {
	var d Document
	var b bytes.Buffer
	err := d.Write(&b)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Bytes()) != 297 {
		t.Fatalf("expected 297 bytes for an empty Document, got %d", len(b.Bytes()))
	}
}
