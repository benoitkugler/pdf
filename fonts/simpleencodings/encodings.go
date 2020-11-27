// Simple encodings map a subset of the unicode characters (at most 256)
// to a set of single bytes. The characters are referenced in fonts by their
// name, not their Unicode value, so both mappings are provided.
// PDF use some predefined encodings, defined in this package.
package simpleencodings

import "github.com/benoitkugler/pdf/model"

type Encoding struct {
	Names [256]string
	Runes map[rune]byte
}

// RuneToByte returns a copy of the rune to byte map
func (e Encoding) RuneToByte() map[rune]byte {
	out := make(map[rune]byte, len(e.Runes))
	for r, b := range e.Runes {
		out[r] = b
	}
	return out
}

// NameToRune combines the informations of `Names` and `Runes`
// to return a name to rune map
func (e Encoding) NameToRune() map[string]rune {
	out := make(map[string]rune, len(e.Runes))
	for r, b := range e.Runes {
		out[e.Names[b]] = r
	}
	return out
}

var PredefinedEncodings = map[model.SimpleEncodingPredefined]*Encoding{
	model.MacExpertEncoding: &MacExpert,
	model.MacRomanEncoding:  &MacRoman,
	model.WinAnsiEncoding:   &WinAnsi,
}
