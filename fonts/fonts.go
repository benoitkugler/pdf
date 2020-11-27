package fonts

import (
	"log"

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

type simpleFont struct {
	desc      model.FontDescriptor
	firstChar byte
	widths    []int
	charMap   map[rune]byte
}

func (ft simpleFont) GetWidth(c rune, size Fl) Fl {
	by := ft.charMap[c] // = FirstChar + i
	if ft.firstChar > by {
		by = ft.firstChar
	}
	index := by - ft.firstChar
	return Fl(ft.widths[index]) * 0.001 * size
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

// BuildFont compiles an existing FontDictionary, as found in a PDF,
// to a usefull font metrics. When needed the font builtin encoding is parsed
// and used.
// For fonts who used glyph names which are not referenced, `BuildFontWithCharMap`
// provides a way of specifying custom names.
// TODO: New font should be created from a font file using `NewFont`
func BuildFont(f *model.FontDict) BuiltFont {
	return BuildFontWithCharMap(f, nil)
}

// Type1 fonts describe their character by name and not by Unicode point.
// In several cases, we can use predefined encodings to establish this mapping.
// In particular, all 14 Standard fonts are covered, as well as all the Nonsymbolic fonts
// (described by the Flags field of the FontDescriptor).
// However, an arbitrary font may use unknown character names, which can't be mapped implicitly
// to Unicode point. For this edge case, a user-defined map from the
// character names to their Unicode value must used. This map will be merged to
// a registry of common names.
func BuildFontWithCharMap(f *model.FontDict, userCharMap map[string]rune) BuiltFont {
	switch ft := f.Subtype.(type) {
	case model.FontType1:
		charMap := resolveCharMapType1(ft, userCharMap)
		return BuiltFont{Meta: f, Font: simpleFont{
			desc:      ft.FontDescriptor,
			charMap:   charMap,
			firstChar: ft.FirstChar,
			widths:    ft.Widths,
		}}
	case model.FontTrueType:
		charMap := resolveCharMapTrueType(ft, userCharMap)
		return BuiltFont{Meta: f, Font: simpleFont{
			desc:      ft.FontDescriptor,
			charMap:   charMap,
			firstChar: ft.FirstChar,
			widths:    ft.Widths,
		}}
	default:
		//TODO: support the other type of font
		panic("not implemented")
	}
	// return BuiltFont{}
}
