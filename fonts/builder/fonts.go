// Package builder provides tooling for exploiting the
// fonts defined (and embedded) in a PDF file.
//
// It builds a `fonts.BuiltFont`, which may then be used to write content
// streams. Since this requires parsing font programs and encodings,
// it is kept out of the write-only `fonts` package.
package builder

import (
	"errors"
	"log"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/fonts/cmaps"
	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
)

type Fl = model.Fl

// Type1, TrueType, Type3
type simpleFont struct {
	desc      model.FontDescriptor
	firstChar byte
	widths    []int
	charMap   map[rune]byte
}

func (ft simpleFont) GetWidth(c rune, size Fl) Fl {
	by := ft.charMap[c] // = FirstChar + i
	index := int(by) - int(ft.firstChar)
	var w int
	if index < 0 || index >= len(ft.widths) { // not encoded
		w = ft.desc.MissingWidth
	} else {
		w = ft.widths[index]
	}
	return Fl(w) * 0.001 * size
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

// type0
type compositeFont struct {
	desc        model.FontDescriptor
	fromUnicode map[rune]model.CID
	widths      map[model.CID]int

	// the special case of the Identity CMap
	// is handled by setting cmap to nil
	reversedCMap map[model.CID]cmaps.CharCode
}

func (ft compositeFont) Encode(cs []rune) []byte {
	out := make([]byte, 0, len(cs)) // at least
	for _, r := range cs {
		cid, ok := ft.fromUnicode[r]
		if !ok {
			log.Printf("unsupported rune %s %d", string(r), r)
		}
		var charCode cmaps.CharCode
		if ft.reversedCMap == nil { // identity
			charCode = cmaps.CharCode(cid)
		} else {
			charCode, ok = ft.reversedCMap[cid]
			if !ok {
				log.Printf("unsupported CID %d", cid)
			}
		}
		charCode.Append(&out)
	}
	return out
}

func (ft compositeFont) GetWidth(c rune, size Fl) Fl {
	cid := ft.fromUnicode[c]
	w, ok := ft.widths[cid]
	if !ok {
		w = ft.desc.MissingWidth
	}
	return Fl(w) * 0.001 * size
}

func (ct compositeFont) Desc() model.FontDescriptor { return ct.desc }

// BuildFont compiles an existing FontDictionary, as found in a PDF,
// to a usefull font metrics. When needed the font builtin encoding is parsed
// and used.
func BuildFont(f *model.FontDict) (fonts.BuiltFont, error) {
	// 9.10.2 - Mapping Character Codes to Unicode Values
	var (
		toUnicode map[model.CID][]rune
		err       error
	)
	if f.ToUnicode != nil {
		toUnicode, err = resolveToUnicode(*f.ToUnicode)
		if err != nil {
			return fonts.BuiltFont{}, err
		}
	}

	if ft, ok := f.Subtype.(model.FontSimple); ok {
		enc := ResolveSimpleEncoding(ft)
		simpleCharMap := buildSimpleFromUnicode(&enc, toUnicode)
		var out simpleFont
		switch ft := f.Subtype.(type) {
		case model.FontType1:
			out = simpleFont{
				desc:      ft.FontDescriptor,
				charMap:   simpleCharMap,
				firstChar: ft.FirstChar,
				widths:    ft.Widths,
			}
		case model.FontTrueType:
			out = simpleFont{
				desc:      ft.FontDescriptor,
				charMap:   simpleCharMap,
				firstChar: ft.FirstChar,
				widths:    ft.Widths,
			}
		case model.FontType3:
			out = simpleFont{
				desc:      buildType3FontDesc(ft),
				charMap:   simpleCharMap,
				firstChar: ft.FirstChar,
				widths:    ft.Widths,
			}
		default:
			panic("should be an exhaustive switch")
		}
		if len(out.widths) == 0 {
			out.firstChar, out.widths = fallbackWidths(out.desc).WidthsWithEncoding(enc)
		}
		return fonts.BuiltFont{Meta: f, Font: out}, nil
	}

	// TODO:
	if ft, ok := f.Subtype.(model.FontType0); ok {
		var fromUnicode map[rune]model.CID
		if toUnicode != nil {
			fromUnicode = reverseToUnicode(toUnicode)
		} else {
			resolveCharMapType0(ft)
		}
		return fonts.BuiltFont{Meta: f, Font: compositeFont{
			fromUnicode: fromUnicode,
			desc:        buildType0FontDesc(ft),
		}}, nil
	}

	return fonts.BuiltFont{}, errors.New("missing font subtype")
}

// if no font desc is given, create one from the properties
// of the font dict
func buildType3FontDesc(tf model.FontType3) model.FontDescriptor {
	if tf.FontDescriptor != nil {
		return *tf.FontDescriptor
	}
	var out model.FontDescriptor
	out.FontBBox = tf.FontBBox
	return out
}

var fallbacks = [...]*standardfonts.Metrics{
	&standardfonts.Courier, // archetype of fixed width
	&standardfonts.Courier_Oblique,
	&standardfonts.Courier_Bold,
	&standardfonts.Courier_BoldOblique,
	&standardfonts.Helvetica, // archetype of sans serif
	&standardfonts.Helvetica_Oblique,
	&standardfonts.Helvetica_Bold,
	&standardfonts.Helvetica_BoldOblique,
	&standardfonts.Times_Roman, // archetype of serif
	&standardfonts.Times_Italic,
	&standardfonts.Times_Bold,
	&standardfonts.Times_BoldItalic,
}

// this should never be used: the font dict must specify
// the widths, but certain PDF generators
// apparently don't include widths for Arial and TimesNewRoman
func fallbackWidths(ft model.FontDescriptor) *standardfonts.Metrics {
	var i uint8
	if ft.Flags&model.FixedPitch != 0 {
		i = 0
	} else if ft.Flags&model.Serif != 0 {
		i = 8
	} else {
		i = 4
	}

	if ft.Flags&model.ForceBold != 0 {
		i += 2
	}

	if ft.Flags&model.Italic != 0 {
		i += 1
	}

	return fallbacks[i]
}

func buildType0FontDesc(tf model.FontType0) model.FontDescriptor {
	out := tf.DescendantFonts.FontDescriptor
	if tf.DescendantFonts.DW != 0 {
		out.MissingWidth = tf.DescendantFonts.DW
	}
	if out.MissingWidth == 0 { // use the default from the SPEC
		out.MissingWidth = 1000
	}
	return out
}
