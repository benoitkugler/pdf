// Simple encodings map a subset of the unicode characters (at most 256)
// to a set of single bytes. The characters are referenced in fonts by their
// name, not their Unicode value, so both mappings are provided.
// PDF use some predefined encodings, defined in this package.
package simpleencodings

var encs = [...][256]string{
	MacExpertNames, MacRomanNames, PdfDocNames, StandardNames, SymbolNames, WinAnsiNames, ZapfDingbatsNames,
}

var encsRunes = [...]map[rune]byte{
	MacExpert, MacRoman, PdfDoc, Standard, Symbol, WinAnsi, ZapfDingbats,
}

var encsNameRunes = [...]map[string]rune{
	MacExpertRunes, MacRomanRunes, PdfDocRunes, StandardRunes, SymbolRunes, WinAnsiRunes, ZapfDingbatsRunes,
}

var (
	MacExpertRunes    = map[string]rune{}
	MacRomanRunes     = map[string]rune{}
	PdfDocRunes       = map[string]rune{}
	StandardRunes     = map[string]rune{}
	SymbolRunes       = map[string]rune{}
	WinAnsiRunes      = map[string]rune{}
	ZapfDingbatsRunes = map[string]rune{}
)

func init() {
	for i, enc := range encsRunes {
		encBytes := encs[i]
		for r, b := range enc {
			name := encBytes[b]
			encsNameRunes[i][name] = r
		}
	}
}
