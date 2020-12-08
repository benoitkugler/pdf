package type1

import (
	"log"
	"strings"

	"github.com/benoitkugler/pdf/fonts/simpleencodings"
	"github.com/benoitkugler/pdf/model"
)

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
	KernPairs map[string][]KernPair
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

type Fl = model.Fl

type charMetric struct {
	code     *byte // nil for not encoded glyphs
	width    int
	name     string
	charBBox [4]int
}

// KernPair represents a kerning pair, from
// an implicit first first glyph.
type KernPair struct {
	SndChar string // glyph name
	// KerningDistance is expressed in glyph units.
	// It is most often negative,
	// that is, a negative value indicates that chars
	// should be closer.
	KerningDistance int
}

// AFMFont represents a type1 font as found in a .afm
// file.
// Most user should probably be satisfied by the `Metrics` summary,
// but more informations are available through this type.
type AFMFont struct {
	Ascender    Fl
	CapHeight   Fl
	Descender   Fl
	ItalicAngle Fl // the italic angle of the font, usually 0.0 or negative.
	Llx         Fl // the llx of the FontBox
	Lly         Fl // the lly of the FontBox
	Urx         Fl // the urx of the FontBox
	Ury         Fl // the ury of the FontBox

	// the Postscript font name.
	FontName string
	// the full name of the font.
	FullName string
	// the family name of the font.
	FamilyName string
	// the Weight of the font: normal, bold, etc.
	Weight string
	// true if all the characters have the same width.
	IsFixedPitch bool
	// the character set of the font.
	CharacterSet string

	// the underline position.
	UnderlinePosition int
	// the underline thickness.
	UnderlineThickness int

	encodingScheme string

	XHeight int

	StdHw int
	StdVw int

	// Represents the section CharMetrics in the AFM file.
	// The key is the name of the char.
	// Even not encoded chars are present
	charMetrics        map[string]charMetric
	charCodeToCharName [256]string // encoded chars

	kernPairs map[string][]KernPair
}

// widthsStats collect the mean and the maximum values
// of the glyphs width
func (f AFMFont) widthsStats() (mean, max Fl) {
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

// returns a string listing the character names defined in the font subset.
// The names in this string shall be in PDF syntax—that is, each name preceded by a slash (/).
// The names may appear in any order. The name .notdef shall be
// omitted; it shall exist in the font subset.
func (f AFMFont) charSet() string {
	var v strings.Builder
	for name := range f.charMetrics {
		if name != ".notdef" {
			v.WriteString(model.ObjName(name).String())
		}
	}
	return v.String()
}

// synthetize a fontDescriptor from various
// font metrics.
func (f AFMFont) fontDescriptor() model.FontDescriptor {
	if f.CapHeight == 0 {
		f.CapHeight = f.Ascender
	}
	isSymbolic := f.FontName == "Symbol" || f.FontName == "ZapfDingbats"

	flag := model.Nonsymbolic
	if isSymbolic {
		flag = model.Symbolic
	}

	if f.IsFixedPitch {
		flag |= model.FixedPitch
	}
	if f.ItalicAngle != 0 {
		flag |= model.Italic
	}
	if f.StdVw == 0 {
		isBold := f.Weight == "bold" || f.Weight == "black"
		if isBold {
			f.StdVw = 120
		} else {
			f.StdVw = 80
		}
	}

	out := model.FontDescriptor{
		FontName:    model.ObjName(f.FontName),
		FontFamily:  f.FamilyName,
		Flags:       flag,
		FontBBox:    model.Rectangle{Llx: f.Llx, Lly: f.Lly, Urx: f.Urx, Ury: f.Ury},
		ItalicAngle: f.ItalicAngle,
		Ascent:      f.Ascender,
		Descent:     f.Descender,
		Leading:     0, // unknown
		CapHeight:   f.CapHeight,
		XHeight:     Fl(f.XHeight),
		StemV:       Fl(f.StdVw),
		StemH:       Fl(f.StdHw),
	}

	// use its width as missing width
	if notdef, ok := f.charMetrics[".notdef"]; ok {
		out.MissingWidth = notdef.width
	}

	out.AvgWidth, out.MaxWidth = f.widthsStats()

	out.CharSet = f.charSet()

	return out
}

// only widths
func (f AFMFont) simplifiedMetrics() map[string]int {
	out := make(map[string]int, len(f.charMetrics))
	for name, m := range f.charMetrics {
		out[name] = m.width
	}
	return out
}

// Metrics returns the essential information from the font.
func (f AFMFont) Metrics() Metrics {
	return Metrics{
		Descriptor:  f.fontDescriptor(),
		CharsWidths: f.simplifiedMetrics(),
		Builtin:     f.charCodeToCharName,
		KernPairs:   f.kernPairs,
	}
}
