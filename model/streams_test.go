package model

import (
	"fmt"
	"testing"
)

func TestStream(t *testing.T) {
	s := Stream{
		Filter: []Filter{
			{Name: JPX, DecodeParams: map[Name]int{"P1": 1, "EndOfLine": 0, "EncodedByteAlign": 1}},
			{Name: ASCII85},
			{Name: ASCIIHex, DecodeParams: map[Name]int{"P1": 1, "EndOfLine": 0, "EncodedByteAlign": 1}},
			{Name: JBIG2},
			{Name: Flate},
		},
		Content: make([]byte, 245),
	}

	st2 := s.PDFCommonFields()
	fmt.Println(st2)
}
