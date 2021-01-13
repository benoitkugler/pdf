package standardfonts

import (
	"log"

	"github.com/benoitkugler/fonts/simpleencodings"
	"github.com/benoitkugler/fonts/type1"
	"github.com/benoitkugler/pdf/model"
)

var PredefinedEncodings = map[model.SimpleEncodingPredefined]*simpleencodings.Encoding{
	model.MacExpertEncoding: &simpleencodings.MacExpert,
	model.MacRomanEncoding:  &simpleencodings.MacRoman,
	model.WinAnsiEncoding:   &simpleencodings.WinAnsi,
}

// Metrics provide metrics for Type1 fonts (such as the predefined).
// Such metrics are usually extracted from .afm files.
// PDF writter may need the Kerning entry to support font kerning.
type Metrics struct {
	Descriptor model.FontDescriptor
	Builtin    [256]string // builtin encoding
	// CharsWidths gives all the characters supported
	// by the font, and their widths
	// It can be used to change the encoding, see `WidthsWithEncoding`.
	CharsWidths map[string]int

	// Represents the section KernPairs in the AFM file. The key is
	// the name of the first character and the value is an array of each kern pair.
	KernPairs map[string][]type1.KernPair
}

// KernsWithEncoding uses the given encoding (byte to name)
// and the available `KernPairs` field to build a condensed map of kerns,
// to be used with the encoding.
func (m Metrics) KernsWithEncoding(enc simpleencodings.Encoding) map[uint16]int {
	out := make(map[uint16]int)
	nameToByte := enc.NameToByte()
	for b, name := range enc {
		if name == "" || name == type1.Notdef {
			continue
		}
		for _, kern := range m.KernPairs[name] {
			// we only keep encoded kern pairs
			if b2, ok := nameToByte[kern.SndChar]; ok {
				key := uint16(b)<<8 | uint16(b2)
				out[key] = kern.KerningDistance
			}
		}
	}
	return out
}

// WidthsWithEncoding uses the given encoding (byte to name)
// to generate a compatible Widths array
// An encoding can be the builtin encoding, a predefined encoding
// or a one obtained by applying a differences map.
// `widths` is an array of (lastChar − `firstChar` + 1) widths (that is, lastChar = firstChar + len(widths) - 1)
// Each element is the glyph width for the character code that equals
// `firstChar` plus the array index.
func (m Metrics) WidthsWithEncoding(encoding simpleencodings.Encoding) (firstChar byte, widths []int) {
	var lastChar byte
	firstChar = 255
	// we first need to find the first and last char
	// var charcodes []byte
	for code, name := range encoding {
		if name == "" || name == type1.Notdef {
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
		if name == "" || name == type1.Notdef {
			continue
		}
		width, ok := m.CharsWidths[name]
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
	f, w := m.WidthsWithEncoding(simpleencodings.WinAnsi)
	return model.FontType1{
		FirstChar:      f,
		Widths:         w,
		FontDescriptor: m.Descriptor,
		BaseFont:       m.Descriptor.FontName,
		Encoding:       model.WinAnsiEncoding,
	}
}
