// Simple encodings map a subset of the unicode characters (at most 256)
// to a set of single bytes. The characters are referenced in fonts by their
// name, not their Unicode value, so both mappings are provided.
// PDF use some predefined encodings, defined in this package.
package simpleencodings

type Encoding struct {
	Names       [256]string
	Runes       map[rune]byte
	NamesToRune map[string]rune
}

// RuneToByte returns a copy of the rune to byte map
func (e Encoding) RuneToByte() map[rune]byte {
	out := make(map[rune]byte, len(e.Runes))
	for r, b := range e.Runes {
		out[r] = b
	}
	return out
}

// NameToRune returns a copy of the name to rune map
func (e Encoding) NameToRune() map[string]rune {
	out := make(map[string]rune, len(e.NamesToRune))
	for n, r := range e.NamesToRune {
		out[n] = r
	}
	return out
}

var encs = [...]*Encoding{
	&MacExpert, &MacRoman, &PdfDoc, &Standard, &Symbol, &WinAnsi, &ZapfDingbats,
}

func init() {
	for _, enc := range encs {
		enc.NamesToRune = make(map[string]rune)
		for r, b := range enc.Runes {
			name := enc.Names[b]
			enc.NamesToRune[name] = r
		}
	}
}
