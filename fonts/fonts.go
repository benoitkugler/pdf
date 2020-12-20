// This package provides tooling for exploiting the
// fonts defined (and embedded) in a PDF file and (TODO) to add new ones.
//
// PDF supports 4 kinds of fonts: the Simples (Type1, TrueType and Type3)
// and the Composite (Type0) and divides the text representation
// in 3 differents objects:
//	1- Glyph (selector): it is either a name (for Simples) or an integer called CID (for Composite)
//	2- Chars (character code): it is a slice of bytes (1 byte for Simples, 1 to 4 bytes for Composite)
//	3- Unicode (point): the Unicode point of a character, coded in Go as runes.
//
// The Glyphs are mapped to Chars (which are the bytes written in the PDF in content streams)
// by an Encoding entry (and also the 'buitlin' encoding of a font).
// Going from Chars to Glyphs is well-defined, but in general, there is no clear mapping
// from Unicode to Glyph (or Chars). Thus, to be able to write an Unicode string (such as UTF-8 strings, which are
// the default in Go), a writter need to build a mapping between Unicode and Glyph.
// It is possible (and automatic) for many fonts (thanks to predifined encodings),
// but some custom fonts may require user inputs.
package fonts

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/benoitkugler/pdf/fonts/cmaps"
	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/fonts/truetype"
	"github.com/benoitkugler/pdf/fonts/type1"
	"github.com/benoitkugler/pdf/model"
)

type Fl = model.Fl

// TextSpaced subtracts space after showing the text
// See 9.4.3 - Text-Showing Operators
type TextSpaced struct {
	Text                 string // unescaped
	SpaceSubtractedAfter int    // value in thousands of text space unit
}

// BuiltFont associate the built font
// to its origin data.
type BuiltFont struct {
	Font
	Meta *model.FontDict
}

// Font provides metric related to a font,
// and a way to encode utf-8 strings to a compatible byte string.
// Since fetching such informations has a cost,
// one font should be build once and reused as often as possible.
// TODO: Support font kerning:
//		- fetch the information from various files
// 		- add the EncodeKern method
//		- add GetWidthWithKerning([]rune) ?
type Font interface {
	// GetWidth return the size, in points, needed to display the character `c`
	// using the font size `size`.
	// Note that this method can't handle kerning.
	GetWidth(c rune, size Fl) Fl

	// Encode transform a slice of unicode points to a
	// slice of bytes, conform to the font expectation.
	// See `EncodeKern` for kerning support.
	Encode(cs []rune) []byte

	// EncodeKern encodes the given unicode points
	// but also adds kerning information, if available from the font.
	// EncodeKern(cs []rune) []TextSpaced

	// Desc return the font descriptor
	Desc() model.FontDescriptor
}

// Type1, TrueType, Type3
type simpleFont struct {
	desc      model.FontDescriptor
	firstChar byte
	widths    []int
	charMap   map[rune]byte
	// key = first<< 8 + second; negative value for closer glyphs
	kerns map[uint16]int
}

func (ft simpleFont) GetWidth(c rune, size Fl) Fl {
	by := ft.charMap[c] // = FirstChar + i
	index := int(by) - int(ft.firstChar)
	var w int
	if index < 0 || index >= len(ft.widths) { // not encoded
		w = ft.desc.MissingWidth
	} else {
		w = ft.widths[index]
	}
	return Fl(w) * 0.001 * size
}

// simple font: use a simple map algorithm
// unsuported runes are mapped to the byte '.'
func (ft simpleFont) Encode(cs []rune) []byte {
	out := make([]byte, len(cs))
	for i, c := range cs {
		switch c {
		case '\n', '\r', '\t', '\f': // the caracters are not encoded, write them and dont warn
			out[i] = byte(c)
		default:
			b, ok := ft.charMap[c]
			if !ok {
				log.Printf("unsupported rune %s %d", string(c), c)
				b = '.'
			}
			out[i] = b
		}
	}
	return out
}

// func (ft simpleFont) EncodeKern(cs []rune) []TextSpaced {
// 	encoded := ft.Encode(cs)
// 	var current TextSpaced
// 	for i := 0; i < len(encoded)-1; i++ {
// 		first, second := encoded[i], encoded[i+1]
// 		kern, has := ft.kerns[uint16(first<<8)|uint16(second)]
// 		if has {
// 			// add the first char to the current text
// 			current.Text = append(current.Text, first)
// 			// close the current and add the kern space
// 			current.SpaceSubtractedAfter =
// 		}
// 	}
// }

func (ft simpleFont) Desc() model.FontDescriptor {
	return ft.desc
}

// type0
type compositeFont struct {
	desc        model.FontDescriptor
	fromUnicode map[rune]model.CID
	widths      map[model.CID]int

	// the special case of the Identity CMap
	// is handled by setting cmap to nil
	reversedCMap map[model.CID]cmaps.CharCode
}

func (ft compositeFont) Encode(cs []rune) []byte {
	out := make([]byte, 0, len(cs)) // at least
	for _, r := range cs {
		cid, ok := ft.fromUnicode[r]
		if !ok {
			log.Printf("unsupported rune %s %d", string(r), r)
		}
		var charCode cmaps.CharCode
		if ft.reversedCMap == nil { // identity
			charCode = cmaps.CharCode(cid)
		} else {
			charCode, ok = ft.reversedCMap[cid]
			if !ok {
				log.Printf("unsupported CID %d", cid)
			}
		}
		charCode.Append(&out)
	}
	return out
}

func (ft compositeFont) GetWidth(c rune, size Fl) Fl {
	cid := ft.fromUnicode[c]
	w, ok := ft.widths[cid]
	if !ok {
		w = ft.desc.MissingWidth
	}
	return Fl(w) * 0.001 * size
}

func (ct compositeFont) Desc() model.FontDescriptor {
	return ct.desc
}

// BuildFont compiles an existing FontDictionary, as found in a PDF,
// to a usefull font metrics. When needed the font builtin encoding is parsed
// and used.
// TODO: New font should be created from a font file using `NewFont`
func BuildFont(f *model.FontDict) (BuiltFont, error) {
	// 9.10.2 - Mapping Character Codes to Unicode Values
	var (
		toUnicode map[model.CID]rune
		err       error
	)
	if f.ToUnicode != nil {
		toUnicode, err = resolveToUnicode(*f.ToUnicode)
		if err != nil {
			return BuiltFont{}, err
		}
	}

	if ft, ok := f.Subtype.(model.FontSimple); ok {
		enc := resolveSimpleEncoding(ft)
		simpleCharMap := buildSimpleFromUnicode(&enc, toUnicode)
		var out simpleFont
		switch ft := f.Subtype.(type) {
		case model.FontType1:
			out = simpleFont{
				desc:      ft.FontDescriptor,
				charMap:   simpleCharMap,
				firstChar: ft.FirstChar,
				widths:    ft.Widths,
			}
			// we add kerning information for the standards font
			if met, ok := standardfonts.Fonts[string(ft.FontName())]; ok {
				out.kerns = met.KernsWithEncoding(enc)
			}
		case model.FontTrueType:
			out = simpleFont{
				desc:      ft.FontDescriptor,
				charMap:   simpleCharMap,
				firstChar: ft.FirstChar,
				widths:    ft.Widths,
			}
			out.kerns, err = fetchSimpleTrueTypeKerning(ft.FontDescriptor.FontFile, simpleCharMap)
			if err != nil {
				return BuiltFont{}, err
			}
		case model.FontType3:
			out = simpleFont{
				desc:      buildType3FontDesc(ft),
				charMap:   simpleCharMap,
				firstChar: ft.FirstChar,
				widths:    ft.Widths,
			}
		default:
			panic("should be an exhaustive switch")
		}
		if len(out.widths) == 0 {
			out.firstChar, out.widths = fallbackWidths(out.desc).WidthsWithEncoding(enc)
		}
		return BuiltFont{Meta: f, Font: out}, nil
	}

	// TODO:
	if ft, ok := f.Subtype.(model.FontType0); ok {
		var fromUnicode map[rune]model.CID
		if toUnicode != nil {
			fromUnicode = reverseToUnicode(toUnicode)
		} else {
			resolveCharMapType0(ft)
		}
		return BuiltFont{Meta: f, Font: compositeFont{
			fromUnicode: fromUnicode,
			desc:        buildType0FontDesc(ft),
		}}, nil
	}

	return BuiltFont{}, errors.New("missing font subtype")
}

// if no font desc is given, create one from the properties
// of the font dict
func buildType3FontDesc(tf model.FontType3) model.FontDescriptor {
	if tf.FontDescriptor != nil {
		return *tf.FontDescriptor
	}
	var out model.FontDescriptor
	out.FontBBox = tf.FontBBox
	return out
}

var fallbacks = [...]*type1.Metrics{
	&standardfonts.Courier, // archetype of fixed width
	&standardfonts.Courier_Oblique,
	&standardfonts.Courier_Bold,
	&standardfonts.Courier_BoldOblique,
	&standardfonts.Helvetica, // archetype of sans serif
	&standardfonts.Helvetica_Oblique,
	&standardfonts.Helvetica_Bold,
	&standardfonts.Helvetica_BoldOblique,
	&standardfonts.Times_Roman, // archetype of serif
	&standardfonts.Times_Italic,
	&standardfonts.Times_Bold,
	&standardfonts.Times_BoldItalic,
}

// this should never be used: the font dict must specify
// the widths, but certain PDF generators
// apparently don't include widths for Arial and TimesNewRoman
func fallbackWidths(ft model.FontDescriptor) *type1.Metrics {
	var i uint8
	if ft.Flags&model.FixedPitch != 0 {
		i = 0
	} else if ft.Flags&model.Serif != 0 {
		i = 8
	} else {
		i = 4
	}

	if ft.Flags&model.ForceBold != 0 {
		i += 2
	}

	if ft.Flags&model.Italic != 0 {
		i += 1
	}

	return fallbacks[i]
}

func fetchSimpleTrueTypeKerning(file *model.FontFile, charMap map[rune]byte) (map[uint16]int, error) {
	if file == nil {
		return nil, errors.New("missing TrueType font file")
	}
	decoded, err := file.Decode()
	if err != nil {
		return nil, fmt.Errorf("can't decode embedded font file: %s", err)
	}
	font, err := truetype.Parse(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("invalid embedded font file: %s", err)
	}

	kerns, err := font.KernTable(false)
	if err != nil { // we consider kerns as optional
		log.Printf("missing kern table: %s", err)
		return nil, nil
	}

	cmap, err := font.CmapTable()
	if err != nil {
		return nil, fmt.Errorf("invalid cmap table: %s", err)
	}
	chars := cmap.Compile()

	out := make(map[uint16]int)
	for r, b := range charMap {
		left := chars[r]
		for r2, b2 := range charMap {
			right := chars[r2]
			if kern, ok := kerns.KernPair(left, right); ok {
				out[uint16(b)<<8|uint16(b2)] = int(kern)
			}
		}
	}
	return out, nil
}

func buildType0FontDesc(tf model.FontType0) model.FontDescriptor {
	out := tf.DescendantFonts.FontDescriptor
	if tf.DescendantFonts.DW != 0 {
		out.MissingWidth = tf.DescendantFonts.DW
	}
	if out.MissingWidth == 0 { // use the default from the SPEC
		out.MissingWidth = 1000
	}
	return out
}
