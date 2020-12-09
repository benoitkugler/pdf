// Simple encodings map a subset of the unicode characters (at most 256)
// to a set of single bytes. The characters are referenced in fonts by their
// name, not their Unicode value, so both mappings are provided.
// PDF use some predefined encodings, defined in this package.
package simpleencodings

import (
	"github.com/benoitkugler/pdf/fonts/glyphsnames"
	"github.com/benoitkugler/pdf/model"
)

type Encoding [256]string

// RuneToByte returns a rune to byte map
func (e Encoding) RuneToByte() map[rune]byte {
	out := make(map[rune]byte)
	for b, name := range e {
		if name == "" {
			continue
		}
		r, _ := glyphsnames.GlyphToRune(name)
		// TestRuneNames assert that each name is referenced
		out[r] = byte(b)
	}
	return out
}

// ByteToRune returns the reverse byte -> rune mapping,
// using a common name registry.
func (e Encoding) ByteToRune() map[byte]rune {
	out := make(map[byte]rune)
	for b, name := range e {
		if name == "" {
			continue
		}
		r, _ := glyphsnames.GlyphToRune(name)
		// TestRuneNames assert that each name is referenced
		out[byte(b)] = r
	}
	return out
}

// NameToRune returns a name to rune map
func (e Encoding) NameToRune() map[string]rune {
	out := make(map[string]rune)
	for _, name := range e {
		if name == "" {
			continue
		}
		r, _ := glyphsnames.GlyphToRune(name)
		// TestRuneNames assert that each name is referenced
		out[name] = r
	}
	return out
}

var PredefinedEncodings = map[model.SimpleEncodingPredefined]*Encoding{
	model.MacExpertEncoding: &MacExpert,
	model.MacRomanEncoding:  &MacRoman,
	model.WinAnsiEncoding:   &WinAnsi,
}
