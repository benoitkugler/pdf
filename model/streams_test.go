package model

import (
	"fmt"
	"testing"
)

func TestStream(t *testing.T) {
	s := ContentStream{
		Filter:  []Filter{JPX, ASCII85, ASCIIHex, JBIG2, Flate},
		Content: make([]byte, 245),
	}
	st1 := s.PDFCommonFields()

	s.DecodeParms = []map[Name]int{
		{"P1": 1, "EndOfLine": 0, "EncodedByteAlign": 1},
		nil,
		{"P1": 1, "EndOfLine": 0, "EncodedByteAlign": 1},
		nil,
		nil,
	}
	st2 := s.PDFCommonFields()
	fmt.Println(st1)
	fmt.Println(st2)
}
