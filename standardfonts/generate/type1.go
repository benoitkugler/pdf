// Implements the parsing of a type1 font configuration.
// This package is only used as code generator:
// most user will use the package standardfonts/fonts
package type1

import (
	"strings"

	"github.com/benoitkugler/pdf/model"
)

type Fl = float32

const spaces = " \t\n\r\f"

var defautFontValues = Font{
	underlinePosition:  -100,
	underlineThickness: 50,
	encodingScheme:     "FontSpecific",
	xHeight:            480,
	stdVw:              80,
}

type charMetric struct {
	code     byte
	width    int
	name     string
	charBBox [4]int
}

type kernPair struct {
	sndChar         string
	kerningDistance int
}

// Font represents a type1 font as found in a .afm
// file.
type Font struct {
	Ascender    Fl
	CapHeight   Fl
	Descender   Fl
	ItalicAngle Fl // the italic angle of the font, usually 0.0 or negative.
	Llx         Fl // the llx of the FontBox
	Lly         Fl // the lly of the FontBox
	Urx         Fl // the urx of the FontBox
	Ury         Fl // the ury of the FontBox

	// the Postscript font name.
	fontName string
	// the full name of the font.
	fullName string
	// the family name of the font.
	familyName string
	// the weight of the font: normal, bold, etc.
	weight string
	// true if all the characters have the same width.
	isFixedPitch bool
	// the character set of the font.
	characterSet string

	// the underline position.
	underlinePosition int
	// the underline thickness.
	underlineThickness int

	// the font's encoding name. This encoding is 'StandardEncoding' or
	//  'AdobeStandardEncoding' for a font that can be totally encoded
	//  according to the characters names. For all other names the
	//  font is treated as symbolic.
	encodingScheme string

	xHeight int

	stdHw int
	stdVw int

	// Represents the section CharMetrics in the AFM file.
	// The key is the name of the char and also an Integer with the char number.
	charMetrics        map[string]charMetric
	charCodeToCharName map[byte]string

	// Represents the section KernPairs in the AFM file. The key is
	// the name of the first character and the value is a <CODE>Object[]</CODE>
	// with 2 elements for each kern pair. Position 0 is the name of
	// the second character and position 1 is the kerning distance. This is
	// repeated for all the pairs.
	kernPairs map[string][]kernPair
}

// WidthsStats collect the mean and the maximum values
// of the glyphs width
func (f Font) WidthsStats() (mean, max Fl) {
	for _, c := range f.charMetrics {
		w := Fl(c.width)
		if w > max {
			max = w
		}
		mean += w
	}
	mean /= Fl(len(f.charMetrics))
	return mean, max
}

// CharSet returns a string listing the character names defined in the font subset.
// The names in this string shall be in PDF syntax—that is, each name preceded by a slash (/).
// The names may appear in any order. The name .notdef shall be
// omitted; it shall exist in the font subset.
func (f Font) CharSet() string {
	var v strings.Builder
	for name := range f.charMetrics {
		if name != ".notdef" {
			v.WriteString(model.Name(name).String())
		}
	}
	return v.String()
}

func (f Font) FontDescriptor() model.FontDescriptor {
	if f.CapHeight == 0 {
		f.CapHeight = f.Ascender
	}
	isSymbolic := f.fontName == "Symbol" || f.fontName == "ZapfDingbats"

	flag := model.Nonsymbolic
	if isSymbolic {
		flag = model.Symbolic
	}

	if f.isFixedPitch {
		flag |= model.FixedPitch
	}
	if f.ItalicAngle != 0 {
		flag |= model.Italic
	}
	if f.stdVw == 0 {
		isBold := f.weight == "bold" || f.weight == "black"
		if isBold {
			f.stdVw = 120
		} else {
			f.stdVw = 80
		}
	}

	out := model.FontDescriptor{
		FontName:    model.Name(f.fontName),
		FontFamily:  f.familyName,
		Flags:       flag,
		FontBBox:    model.Rectangle{Llx: f.Llx, Lly: f.Lly, Urx: f.Urx, Ury: f.Ury},
		ItalicAngle: f.ItalicAngle,
		Ascent:      f.Ascender,
		Descent:     f.Descender,
		Leading:     0, // unknown
		CapHeight:   f.CapHeight,
		XHeight:     Fl(f.xHeight),
		StemV:       Fl(f.stdVw),
		StemH:       Fl(f.stdHw),
	}

	// use its width as missing width
	if notdef, ok := f.charMetrics[".notdef"]; ok {
		out.MissingWidth = Fl(notdef.width)
	}

	out.AvgWidth, out.MaxWidth = f.WidthsStats()

	// it seems not to be required
	// out.CharSet = f.CharSet()

	return out
}

// Widths returns the first and last character encoded, and
// an array of (lastChar − firstChar + 1) widths, each
// element being the glyph width for the character code that equals
// firstChar plus the array index.
func (f Font) Widths() (firstChar byte, widths []int) {
	var lastChar byte
	firstChar = 255
	// we first need to find the first and last char
	// var charcodes []byte
	for _, cm := range f.charMetrics {
		if cm.name != ".undef" {
			if cm.code < firstChar {
				firstChar = cm.code
			}
			if cm.code > lastChar {
				lastChar = cm.code
			}
			// charcodes = append(charcodes, cm.code)
		}
	}
	widths = make([]int, lastChar-firstChar+1)
	for _, cm := range f.charMetrics {
		if cm.name != ".undef" {
			index := cm.code - firstChar
			widths[index] = cm.width
		}
	}
	return firstChar, widths
}
