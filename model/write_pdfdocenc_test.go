package model

import (
	"bytes"
	"testing"
)

var reversed = map[byte]rune{}

func init() {
	for r, b := range PdfDocRunes {
		reversed[b] = r
	}
}

// pdfDocEncodingToString decodes PDFDocEncoded byte slice `b` to unicode string.
func pdfDocEncodingToString(b []byte) string {
	var runes []rune
	for _, bval := range b {
		rune, has := reversed[bval]
		if !has {
			continue
		}

		runes = append(runes, rune)
	}

	return string(runes)
}

func TestPDFDocEncodingDecode(t *testing.T) {
	testcases := []struct {
		Encoded  []byte
		Expected string
	}{
		{[]byte{0x47, 0x65, 0x72, 0xfe, 0x72, 0xfa, 0xf0, 0x75, 0x72}, "Gerþrúður"},
		{[]byte("Ger\xfer\xfa\xf0ur"), "Gerþrúður"},
	}

	for _, testcase := range testcases {
		str := pdfDocEncodingToString(testcase.Encoded)
		if str != testcase.Expected {
			t.Fatalf("Mismatch %s != %s", str, testcase.Expected)
		}

		enc, _ := stringToPDFDocEncoding(str)
		if !bytes.Equal(enc, testcase.Encoded) {
			t.Fatalf("Encode mismatch %s (%X) != %s (%X)", enc, enc, testcase.Encoded, testcase.Encoded)
		}
	}

	return
}
