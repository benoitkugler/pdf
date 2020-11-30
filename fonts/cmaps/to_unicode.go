package cmaps

import (
	"github.com/benoitkugler/pdf/model"
)

type ToUnicode interface {
	MergeTo(accu map[CharCode]rune)
}

type ToUnicodePair struct {
	From CharCode
	Dest rune
}

func (p ToUnicodePair) MergeTo(simple map[CharCode]rune) {
	simple[p.From] = p.Dest
}

// ToUnicodeArray is a compact mapping
// of [From, To] to Runes
type ToUnicodeArray struct {
	From, To CharCode
	Runes    []rune // length To - From + 1
}

func (arr ToUnicodeArray) MergeTo(simple map[CharCode]rune) {
	for code := arr.From; code <= arr.To; code++ {
		o := arr.Runes[code-arr.From]
		simple[code] = o
	}
}

// ToUnicodeTranslation is a compact mapping
// of [From,To] to [Dest,Dest+To-From].
// It can also represent a simple mapping by taking From = To
type ToUnicodeTranslation struct {
	From, To CharCode
	Dest     rune
}

func (tr ToUnicodeTranslation) MergeTo(simple map[CharCode]rune) {
	r := tr.Dest
	for code := tr.From; code <= tr.To; code++ {
		simple[code] = r
		r++
	}
}

type UnicodeCMap struct {
	UseCMap model.Name // base this cmap on `UseCMap` if `UseCMap` is not empty.

	Mappings []ToUnicode // compact representation
}

// ProperLookupTable returns convenient form of the mapping,
// without resolving potential UseCMap
func (u UnicodeCMap) ProperLookupTable() map[CharCode]rune {
	out := make(map[CharCode]rune, len(u.Mappings)) // at least
	for _, m := range u.Mappings {
		m.MergeTo(out)
	}
	return out
}
