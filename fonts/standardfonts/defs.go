package standardfonts

import (
	"log"

	"github.com/benoitkugler/pdf/fonts/simpleencodings"
	"github.com/benoitkugler/pdf/model"
)

// Metrics provide metrics for the font builtin encoding
type Metrics struct {
	Descriptor model.FontDescriptor
	Builtin    [256]string // builtin encoding
	// CharsWidths gives all the characters supported
	// by the font, and their widths
	// It can be used to change the encoding, see `AdaptEncoding`.
	CharsWidths map[string]int
}

// WidthsWithEncoding use the encoding (byte to name)
// given to generate a compatible Widths array
// An encoding can be the builtin encoding, a predefined encoding
// or a one obtained by applying a differences map.
// `widths` is an array of (lastChar − `firstChar` + 1) widths (that is, lastChar = firstChar + len(widths) - 1)
// Each element is the glyph width for the character code that equals
// `firstChar` plus the array index.
func (f Metrics) WidthsWithEncoding(encoding [256]string) (firstChar byte, widths []int) {
	var lastChar byte
	firstChar = 255
	// we first need to find the first and last char
	// var charcodes []byte
	for code, name := range encoding {
		if name == "" || name == ".undef" {
			continue
		}
		if byte(code) < firstChar {
			firstChar = byte(code)
		}
		if byte(code) > lastChar {
			lastChar = byte(code)
		}
	}
	widths = make([]int, lastChar-firstChar+1)
	for code, name := range encoding {
		if name == "" || name == ".notdef" {
			continue
		}
		width, ok := f.CharsWidths[name]
		if !ok {
			log.Printf("unsupported glyph name : %s", name)
		}
		index := code - int(firstChar)
		widths[index] = width
	}
	return firstChar, widths
}

// WesternType1Font return a version of the font
// using WinAnsi encoding (except for Symbol and ZapfDingbats)
func (m Metrics) WesternType1Font() model.FontType1 {
	if m.Descriptor.FontName == "ZapfDingbats" || m.Descriptor.FontName == "Symbol" {
		// keep the builtin encoding
		f, w := m.WidthsWithEncoding(m.Builtin)
		return model.FontType1{
			FirstChar:      f,
			Widths:         w,
			FontDescriptor: m.Descriptor,
			BaseFont:       m.Descriptor.FontName,
		}
	}

	// use WinAnsi
	f, w := m.WidthsWithEncoding(simpleencodings.WinAnsi.Names)
	return model.FontType1{
		FirstChar:      f,
		Widths:         w,
		FontDescriptor: m.Descriptor,
		BaseFont:       m.Descriptor.FontName,
		Encoding:       model.WinAnsiEncoding,
	}
}
