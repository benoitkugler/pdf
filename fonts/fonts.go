// This package provides the types needed to write text using the
// fonts defined (and embedded) in a PDF file.
//
// PDF supports 4 kinds of fonts: the Simples (Type1, TrueType and Type3)
// and the Composite (Type0) and divides the text representation
// in 3 differents objects:
//
//	1- Glyph (selector): it is either a name (for Simples) or an integer called CID (for Composite)
//	2- Chars (character code): it is a slice of bytes (1 byte for Simples, 1 to 4 bytes for Composite)
//	3- Unicode (point): the Unicode point of a character, coded in Go as runes.
//
// The Glyphs are mapped to Chars (which are the bytes written in the PDF in content streams)
// by an Encoding entry (and also the 'buitlin' encoding of a font).
// Going from Chars to Glyphs is well-defined, but in general, there is no clear mapping
// from Unicode to Glyph (or Chars). Thus, to be able to write an Unicode string (such as UTF-8 strings, which are
// the default in Go), a writter need to build a mapping between Unicode and Glyph.
//
// Use the fonts/builder package to build a `BuiltFont` from a `model.FontDict`.
package fonts

import (
	"github.com/benoitkugler/pdf/model"
)

type Fl = model.Fl

// TextSpaced subtracts space after showing the text
// See 9.4.3 - Text-Showing Operators
type TextSpaced struct {
	// unescaped content. required a font to interpret the codes
	CharCodes            []byte
	SpaceSubtractedAfter int // value in thousands of text space unit
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
type Font interface {
	// GetWidth return the size, in points, needed to display the character `c`
	// using the font size `size`.
	// Note that this method can't handle kerning.
	GetWidth(c rune, size Fl) Fl

	// Encode transform a slice of unicode points to a
	// slice of bytes, conform to the font expectation.
	// See `EncodeKern` for kerning support.
	Encode(cs []rune) []byte

	// Desc return the font descriptor
	Desc() model.FontDescriptor
}
