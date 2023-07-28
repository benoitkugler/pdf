package cmaps

import (
	"bytes"
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

type ToUnicode interface {
	MergeTo(accu map[model.CID][]rune)
}

type ToUnicodePair struct {
	From model.CID
	Dest []rune
}

func (p ToUnicodePair) MergeTo(simple map[model.CID][]rune) {
	simple[p.From] = p.Dest
}

// ToUnicodeArray is a compact mapping
// of [From, To] to Runes
type ToUnicodeArray struct {
	Runes    [][]rune // length To - From + 1
	From, To model.CID
}

func (arr ToUnicodeArray) MergeTo(simple map[model.CID][]rune) {
	for code := arr.From; code <= arr.To; code++ {
		o := arr.Runes[code-arr.From]
		simple[code] = o
	}
}

// ToUnicodeTranslation is a compact mapping
// of [From,To] to [Dest,Dest+To-From].
// It can also represent a simple mapping by taking From = To
type ToUnicodeTranslation struct {
	From, To model.CID
	Dest     rune
}

func (tr ToUnicodeTranslation) MergeTo(simple map[model.CID][]rune) {
	r := tr.Dest
	for code := tr.From; code <= tr.To; code++ {
		simple[code] = []rune{r}
		r++
	}
}

// UnicodeCMap maps from CID to Unicode points.
// Note that it differs from CID Cmap in the sense
// that the origin of the mapping are CID and not CharCode.
type UnicodeCMap struct {
	UseCMap model.ObjName // base this cmap on `UseCMap` if `UseCMap` is not empty.

	Mappings []ToUnicode // compact representation
}

// ProperLookupTable returns a convenient form of the mapping,
// without resolving a potential UseCMap.
func (u UnicodeCMap) ProperLookupTable() map[model.CID][]rune {
	out := make(map[model.CID][]rune, len(u.Mappings)) // at least
	for _, m := range u.Mappings {
		m.MergeTo(out)
	}
	return out
}

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
