package fonts

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf16"
)

// WriteAdobeIdentityUnicodeCMap dumps the given mapping to a Cmap ressource,
// ready to be embedded in a PDF file.
func WriteAdobeIdentityUnicodeCMap(dict map[uint32][]rune) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `/CIDInit /ProcSet findresource begin
	12 dict begin
	begincmap
	/CIDSystemInfo
	<< /Registry (Adobe)
	/Ordering (UCS)
	/Supplement 0
	>> def
	/CMapName /Adobe-Identity-UCS def
	/CMapType 2 def
	1 begincodespacerange
	<0000> <ffff>
	endcodespacerange
	%d beginbfchar
	`, len(dict))

	for glyph, text := range dict {
		fmt.Fprintf(&buf, "<%04x> <%s>\n", glyph, runesToHex(text))
	}

	buf.WriteString(`
	endbfchar
	endcmap
	CMapName currentdict /CMap defineresource pop
	end
	end`)

	return buf.Bytes()
}

// runesToHex returns the UTF-16BE hexadecimal representation of `text`,
// without any byte order mark.
func runesToHex(text []rune) string {
	var sb strings.Builder
	for _, letter := range text {
		for _, unit := range utf16.Encode([]rune{letter}) {
			fmt.Fprintf(&sb, "%04x", unit)
		}
	}
	return sb.String()
}
