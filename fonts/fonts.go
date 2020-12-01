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
	"errors"
	"log"

	"github.com/benoitkugler/pdf/fonts/cmaps"
	"github.com/benoitkugler/pdf/model"
)

type Fl = model.Fl

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
type Font interface {
	// GetWidth return the size, in points, needed to display the character `c`
	// using the font size `size`.
	GetWidth(c rune, size Fl) Fl
	// Encode transform a slice of unicode points to a
	// slice of bytes, conform to the font expectation.
	// Only Font0 type are able to represent all the unicode range,
	// so for other fonts, a substitution byte is used.
	Encode(cs []rune) []byte
	Desc() model.FontDescriptor
}

// Type1, TrueType, Type3
type simpleFont struct {
	desc      model.FontDescriptor
	firstChar byte
	widths    []int
	charMap   map[rune]byte
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
// For fonts who used glyph names which are not referenced, `BuildFontWithCharMap`
// provides a way of specifying custom names.
// TODO: New font should be created from a font file using `NewFont`
func BuildFont(f *model.FontDict) (BuiltFont, error) {
	return BuildFontWithCharMap(f, nil)
}

// Type1, TrueType and Type3 fonts describe their character by name and not by Unicode point.
// In several cases, we can use predefined encodings to establish this mapping.
// In particular, all 14 Standard fonts are covered, as well as all the Nonsymbolic fonts
// (described by the Flags field of the FontDescriptor).
// However, an arbitrary font may use unknown character names, which can't be mapped implicitly
// to Unicode point. For this edge case, a user-defined map from the
// character names to their Unicode value must used. This map will be merged to
// a registry of common names.
func BuildFontWithCharMap(f *model.FontDict, userCharMap map[string]rune) (BuiltFont, error) {
	// 9.10.2 - Mapping Character Codes to Unicode Values
	var (
		toUnicode     map[model.CID]rune
		simpleCharMap map[rune]byte
	)
	if f.ToUnicode != nil {
		var err error
		toUnicode, err = resolveToUnicode(*f.ToUnicode)
		if err != nil {
			return BuiltFont{}, err
		}
	}
	switch ft := f.Subtype.(type) {
	case model.FontType1:
		if toUnicode != nil {
			simpleCharMap = reverseToUnicodeSimple(toUnicode)
		} else {
			simpleCharMap = resolveCharMapType1(ft, userCharMap)
		}
		return BuiltFont{Meta: f, Font: simpleFont{
			desc:      ft.FontDescriptor,
			charMap:   simpleCharMap,
			firstChar: ft.FirstChar,
			widths:    ft.Widths,
		}}, nil
	case model.FontTrueType:
		if toUnicode != nil {
			simpleCharMap = reverseToUnicodeSimple(toUnicode)
		} else {
			simpleCharMap = resolveCharMapTrueType(ft, userCharMap)
		}
		return BuiltFont{Meta: f, Font: simpleFont{
			desc:      ft.FontDescriptor,
			charMap:   simpleCharMap,
			firstChar: ft.FirstChar,
			widths:    ft.Widths,
		}}, nil
	case model.FontType3:
		if toUnicode != nil {
			simpleCharMap = reverseToUnicodeSimple(toUnicode)
		} else {
			simpleCharMap = resolveCharMapType3(ft, userCharMap)
		}
		return BuiltFont{Meta: f, Font: simpleFont{
			desc:      buildType3FontDesc(ft),
			charMap:   simpleCharMap,
			firstChar: ft.FirstChar,
			widths:    ft.Widths,
		}}, nil
	case model.FontType0:
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
	case nil:
		return BuiltFont{}, errors.New("missing font subtype")
	default:
		panic("should be an exhaustive switch")
	}
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
