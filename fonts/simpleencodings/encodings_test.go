package simpleencodings

import (
	"fmt"
	"testing"
)

var encs = [...]*Encoding{
	&MacExpert, &MacRoman, &PdfDoc, &Standard, &Symbol, &WinAnsi, &ZapfDingbats,
}

func TestNames(t *testing.T) {
	for _, e := range encs {
		fmt.Println(len(e.NameToRune()))
	}
}
