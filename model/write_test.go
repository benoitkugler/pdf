package model

import (
	"fmt"
	"testing"

	"golang.org/x/text/encoding/unicode"
)

func TestUnicode(t *testing.T) {
	enc := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder()

	encoded, err := enc.String("dlmkskmdskld")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(encoded))
}
