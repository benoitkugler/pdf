package fonts

import (
	"log"

	"github.com/benoitkugler/pdf/fonts/glyphsnames"
	"github.com/benoitkugler/pdf/fonts/sfnt"
	"github.com/benoitkugler/pdf/fonts/simpleencodings"
	"github.com/benoitkugler/pdf/fonts/type1font"
	"github.com/benoitkugler/pdf/model"
)

// build the definitive font encoding, expressed in term
// of Unicode codepoint to byte
func resolveCharMapType1(t model.FontType1, userCharMap map[string]rune) map[rune]byte {
	if enc, ok := t.Encoding.(model.SimpleEncodingPredefined); ok {
		// the font dict overide the font builtin encoding
		return baseEnc(enc).RuneToByte()
	}
	var (
		base  simpleencodings.Encoding
		diffs model.Differences
	)

	if enc, ok := t.Encoding.(*model.SimpleEncodingDict); ok { // the font modifies an encoding
		// resolve the base encoding
		if enc.BaseEncoding != "" {
			base = baseEnc(enc.BaseEncoding)
		} else { // try and fetch the embedded font information
			base = builtinType1Encoding(t.FontDescriptor)
		}
		diffs = enc.Differences
	} else { // the font use its builtin encoding (or Standard if none is found)
		base = builtinType1Encoding(t.FontDescriptor)
	}

	return applyDifferences(diffs, userCharMap, base)
}

func applyDifferences(diffs model.Differences, userCharMap map[string]rune, baseEnc simpleencodings.Encoding) map[rune]byte {
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
			log.Printf("font encoding: missing rune for the name <%s>\n", name)
		} else {
			out[r] = byte(by)
		}
	}
	return out
}

// baseCharMap returns the mapping from byte to character name.
// it is only useful as a base for differences
func baseEnc(enc model.SimpleEncodingPredefined) simpleencodings.Encoding {
	switch enc {
	case model.MacExpertEncoding:
		return simpleencodings.MacExpert
	case model.MacRomanEncoding:
		return simpleencodings.MacRoman
	case model.WinAnsiEncoding:
		return simpleencodings.WinAnsi
	default:
		panic("invalid simple encoding")
	}
}

// try to read the embedded font file and return the font builtin
// encoding. If f is nil or an error occur, default to Standard
// fontType is needed to select the correct font parser.
func builtinType1Encoding(desc model.FontDescriptor) simpleencodings.Encoding {
	// special case for two standard fonts where we dont need to read the font file
	if desc.FontName == "ZapfDingbats" {
		return simpleencodings.ZapfDingbats
	} else if desc.FontName == "Symbol" {
		return simpleencodings.Symbol
	}

	if desc.FontFile == nil {
		return simpleencodings.Standard
	}
	content, err := desc.FontFile.Decode()
	if err != nil {
		log.Printf("unable to decode embedded font file: %s\n", err)
		return simpleencodings.Standard
	}

	info, err := type1font.ParsePfbContent(content)
	if err != nil {
		log.Printf("invalid Type1 embedded font file: %s\n", err)
		return simpleencodings.Standard
	}
	if info.Encoding.Standard {
		return simpleencodings.Standard
	}
	return simpleencodings.Encoding{Names: info.Encoding.Custom, Runes: simpleencodings.Standard.Runes}
}

func resolveCharMapTrueType(f model.FontTrueType, userCharMap map[string]rune) map[rune]byte {
	// 9.6.6.3 - when the font has no Encoding entry, or the font descriptor’s Symbolic flag is set
	// (in which case the Encoding entry is ignored)
	if (f.FontDescriptor.Flags&model.Symbolic) != 0 || f.Encoding == nil {
		return builtinTrueTypeEncoding(f.FontDescriptor)
	}

	// 9.6.6.3 - if the font has a named Encoding entry of either MacRomanEncoding or WinAnsiEncoding,
	// or if the font descriptor’s Nonsymbolic flag (see Table 123) is set
	if (f.FontDescriptor.Flags&model.Nonsymbolic) != 0 || f.Encoding == model.MacRomanEncoding || f.Encoding == model.WinAnsiEncoding {
		if f.Encoding == model.MacRomanEncoding {
			return simpleencodings.MacRoman.Runes
		} else if f.Encoding == model.WinAnsiEncoding {
			return simpleencodings.WinAnsi.Runes
		} else if dict, ok := f.Encoding.(*model.SimpleEncodingDict); ok {
			base := baseEnc(dict.BaseEncoding)
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

func builtinTrueTypeEncoding(desc model.FontDescriptor) map[rune]byte {
	if desc.FontFile == nil { // we choose an arbitrary encoding
		return simpleencodings.Standard.Runes
	}
	content, err := desc.FontFile.Decode()
	if err != nil {
		log.Printf("unable to decode embedded font file: %s\n", err)
		return simpleencodings.Standard.Runes
	}
	font, err := sfnt.Parse(content)
	if err != nil {
		log.Printf("invalid TrueType embedded font file: %s\n", err)
		return simpleencodings.Standard.Runes
	}
	fontChars, err := font.Chars()
	if err != nil {
		log.Printf("invalid encoding in TrueType embedded font file: %s\n", err)
		return simpleencodings.Standard.Runes
	}
	out := make(map[rune]byte, len(fontChars))
	for r, index := range fontChars {
		out[r] = byte(index) // keep the lower order byte
	}
	return out
}
