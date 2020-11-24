package fonts

import "github.com/benoitkugler/pdf/model"

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

type type1 struct {
	model.FontType1

	charMap map[rune]byte
}

func (ft type1) GetWidth(c rune, size Fl) Fl {
	by := ft.charMap[c] // = FirstChar + i
	if ft.FirstChar > by {
		by = ft.FirstChar
	}
	index := by - ft.FirstChar
	return Fl(ft.Widths[index]) * 0.001 * size
}

// simple font: use a simple map algorithm
// unsuported runes are mapped to the byte '.'
func (ft type1) Encode(cs []rune) []byte {
	out := make([]byte, len(cs))
	for i, c := range cs {
		b, ok := ft.charMap[c]
		if !ok {
			b = '.'
		}
		out[i] = b
	}
	return out
}

func (ft type1) Desc() model.FontDescriptor {
	return ft.FontType1.FontDescriptor
}

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
		out := type1{FontType1: ft}
		out.charMap = out.resolveCharMap(userCharMap)
		return BuiltFont{Meta: f, Font: out}
		//TODO: support the other type of font
	}
	return BuiltFont{}
}
