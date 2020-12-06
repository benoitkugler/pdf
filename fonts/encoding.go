package fonts

import (
	"bytes"
	"log"

	"github.com/benoitkugler/font/sfnt"
	"github.com/benoitkugler/pdf/fonts/cmaps"
	"github.com/benoitkugler/pdf/fonts/glyphsnames"
	"github.com/benoitkugler/pdf/fonts/simpleencodings"
	"github.com/benoitkugler/pdf/fonts/standardcmaps"
	"github.com/benoitkugler/pdf/fonts/type1font"
	"github.com/benoitkugler/pdf/model"
)

// build the definitive font encoding, expressed in term
// of Unicode codepoint to byte
func resolveCharMapType1(t model.FontType1, userCharMap map[string]rune) map[rune]byte {
	if enc, ok := t.Encoding.(model.SimpleEncodingPredefined); ok {
		// the font dict overide the font builtin encoding
		return simpleencodings.PredefinedEncodings[enc].RuneToByte()
	}
	var (
		base  *simpleencodings.Encoding
		diffs model.Differences
	)

	if enc, ok := t.Encoding.(*model.SimpleEncodingDict); ok { // the font modifies an encoding
		// resolve the base encoding
		if enc.BaseEncoding != "" {
			base = simpleencodings.PredefinedEncodings[enc.BaseEncoding]
		} else { // try and fetch the embedded font information
			base = builtinType1Encoding(t.FontDescriptor)
		}
		diffs = enc.Differences
	} else { // the font use its builtin encoding (or Standard if none is found)
		base = builtinType1Encoding(t.FontDescriptor)
	}

	return applyDifferences(diffs, userCharMap, base)
}

func applyDifferences(diffs model.Differences, userCharMap map[string]rune, baseEnc *simpleencodings.Encoding) map[rune]byte {
	runeMap := baseEnc.NameToRune()
	// add an eventual user name mapping
	for name, r := range userCharMap {
		runeMap[name] = r
	}

	// add the potential difference
	withDiffs := diffs.Apply(baseEnc.Names)

	out := make(map[rune]byte)

	for by, name := range withDiffs {
		if name == "" {
			continue // not encoded
		}
		// resolve the rune from the name: first try with the
		// encoding names
		r := runeMap[name]
		if r == 0 {
			// try a global name registry
			r, _ = glyphsnames.GlyphToRune(name)
		}
		if r == 0 {
			log.Printf("font encoding: the name <%s> has no matching rune\n", name)
		} else {
			out[r] = byte(by)
		}
	}
	return out
}

// try to read the embedded font file and return the font builtin
// encoding. If f is nil or an error occur, default to Standard
// fontType is needed to select the correct font parser.
func builtinType1Encoding(desc model.FontDescriptor) *simpleencodings.Encoding {
	// special case for two standard fonts where we dont need to read the font file
	if desc.FontName == "ZapfDingbats" {
		return &simpleencodings.ZapfDingbats
	} else if desc.FontName == "Symbol" {
		return &simpleencodings.Symbol
	}

	if desc.FontFile == nil {
		return &simpleencodings.Standard
	}
	content, err := desc.FontFile.Decode()
	if err != nil {
		log.Printf("unable to decode embedded font file: %s\n", err)
		return &simpleencodings.Standard
	}

	info, err := type1font.ParsePFBFile(content)
	if err != nil {
		log.Printf("invalid Type1 embedded font file: %s\n", err)
		return &simpleencodings.Standard
	}
	if info.Encoding.Standard {
		return &simpleencodings.Standard
	}
	return &simpleencodings.Encoding{Names: info.Encoding.Custom, Runes: simpleencodings.Standard.Runes}
}

func resolveCharMapTrueType(f model.FontTrueType, userCharMap map[string]rune) map[rune]byte {
	// 9.6.6.3 - when the font has no Encoding entry, or the font descriptor’s Symbolic flag is set
	// (in which case the Encoding entry is ignored)
	if (f.FontDescriptor.Flags&model.Symbolic) != 0 || f.Encoding == nil {
		return builtinTrueTypeEncoding(f.FontDescriptor).Runes
	}

	// 9.6.6.3 - if the font has a named Encoding entry of either MacRomanEncoding or WinAnsiEncoding,
	// or if the font descriptor’s Nonsymbolic flag (see Table 123) is set
	if (f.FontDescriptor.Flags&model.Nonsymbolic) != 0 || f.Encoding == model.MacRomanEncoding || f.Encoding == model.WinAnsiEncoding {
		if f.Encoding == model.MacRomanEncoding {
			return simpleencodings.MacRoman.Runes
		} else if f.Encoding == model.WinAnsiEncoding {
			return simpleencodings.WinAnsi.Runes
		} else if dict, ok := f.Encoding.(*model.SimpleEncodingDict); ok {
			var base *simpleencodings.Encoding
			if dict.BaseEncoding != "" {
				base = simpleencodings.PredefinedEncodings[dict.BaseEncoding]
			} else {
				base = builtinTrueTypeEncoding(f.FontDescriptor)
			}
			out := applyDifferences(dict.Differences, userCharMap, base)
			// Finally, any undefined entries in the table shall be filled using StandardEncoding.
			for r, bStd := range simpleencodings.Standard.Runes {
				if _, ok := out[r]; !ok { // missing rune
					out[r] = bStd
				}
			}
			return out
		}
	}
	// default value
	return simpleencodings.Standard.Runes
}

func builtinTrueTypeEncoding(desc model.FontDescriptor) *simpleencodings.Encoding {
	if desc.FontFile == nil { // we choose an arbitrary encoding
		return &simpleencodings.Standard
	}
	content, err := desc.FontFile.Decode()
	if err != nil {
		log.Printf("unable to decode embedded font file: %s\n", err)
		return &simpleencodings.Standard
	}
	font, err := sfnt.Parse(bytes.NewReader(content))
	if err != nil {
		log.Printf("invalid TrueType embedded font file: %s\n", err)
		return &simpleencodings.Standard
	}
	cmap, err := font.CmapTable()
	if err != nil {
		log.Printf("invalid encoding in TrueType embedded font file: %s\n", err)
		return &simpleencodings.Standard
	}
	fontChars := cmap.Compile()

	// TODO: use builtin glyph names
	// var glyphNames sfnt.GlyphNames
	// if postTable, err := font.PostTable(); err == nil && postTable.Names != nil {
	// 	glyphNames = postTable.Names
	// }

	runes := make(map[rune]byte, len(fontChars))
	var names [256]string
	for r, index := range fontChars {
		if index > 0xFF {
			log.Printf("overflow for glyph index %d in TrueType font", index)
		}
		runes[r] = byte(index) // keep the lower order byte

		// TODO:
		// name, err := font.GlyphName(&b, index)
		// if err != nil {
		// 	log.Printf("glyph index %d without name: %s\n", index, err)
		// } else {
		// 	names[runes[r]] = name
		// }
	}
	return &simpleencodings.Encoding{Names: names, Runes: runes}
}

func resolveCharMapType3(f model.FontType3, userCharMap map[string]rune) map[rune]byte {
	switch enc := f.Encoding.(type) {
	case model.SimpleEncodingPredefined:
		return simpleencodings.PredefinedEncodings[enc].Runes
	case *model.SimpleEncodingDict:
		base := &simpleencodings.Standard
		if enc.BaseEncoding != "" {
			base = simpleencodings.PredefinedEncodings[enc.BaseEncoding]
		}
		return applyDifferences(enc.Differences, userCharMap, base)
	default: // should not happen according to the spec
		return simpleencodings.Standard.Runes
	}
}

// parse the CMap and resolve the chain of UseCMap if needed
func resolveToUnicode(cmap model.UnicodeCMap) (map[model.CID]rune, error) {
	content, err := cmap.Decode()
	if err != nil {
		return nil, err
	}
	inner, err := cmaps.ParseUnicodeCMap(content)
	if err != nil {
		return nil, err
	}
	out := inner.ProperLookupTable()

	var used map[model.CID]rune
	switch use := cmap.UseCMap.(type) {
	case model.UnicodeCMap:
		used, err = resolveToUnicode(use)
		if err != nil {
			return nil, err
		}
	case model.UnicodeCMapBasePredefined:
		predef, ok := standardcmaps.ToUnicodeCMaps[model.ObjName(use)]
		if !ok {
			log.Printf("unknown predefined UnicodeCMap %s", use)
		}
		used = predef.ProperLookupTable()
	}
	// merged the data from the UseCMap entry
	for k, v := range used {
		out[k] = v
	}
	return out, nil
}

func resolveCharMapType0(ft model.FontType0) {
	// 9.10.2 - Mapping Character Codes to Unicode Values
	ft.DescendantFonts.CIDSystemInfo.ToUnicodeCMapName()
}

// build the reverse mapping, assuming a simple font
func reverseToUnicodeSimple(m map[model.CID]rune) map[rune]byte {
	out := make(map[rune]byte, len(m))
	for k, v := range m {
		out[v] = byte(k)
	}
	return out
}

// build the reverse mapping
func reverseToUnicode(m map[model.CID]rune) map[rune]model.CID {
	out := make(map[rune]model.CID, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}
