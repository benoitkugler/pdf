package fonts

import (
	"log"

	"github.com/benoitkugler/pdf/fonts/glyphsnames"
	"github.com/benoitkugler/pdf/fonts/simpleencodings"
	"github.com/benoitkugler/pdf/fonts/type1font"
	"github.com/benoitkugler/pdf/model"
)

// build the definitive font encoding, expressed in term
// of Unicode codepoint to byte
func (t type1) resolveCharMap(userCharMap map[string]rune) map[rune]byte {
	if enc, ok := t.Encoding.(model.SimpleEncodingPredefined); ok {
		// the font dict overide the font builtin encoding
		return runeMap(enc)
	}
	var (
		base    [256]string
		runeMap map[string]rune
		diffs   model.Differences
	)

	if enc, ok := t.Encoding.(*model.SimpleEncodingDict); ok { // the font modifies an encoding
		// resolve the base encoding
		if enc.BaseEncoding != "" {
			base, runeMap = baseCharMap(enc.BaseEncoding), namesMap(enc.BaseEncoding)
		} else { // try and fetch the embedded font information
			base, runeMap = builtinEncoding(t.FontDescriptor, t), simpleencodings.StandardRunes
		}
		diffs = enc.Differences
	} else { // the font use its builtin encoding (or Standard if none is found)
		base, runeMap = builtinEncoding(t.FontDescriptor, t), simpleencodings.StandardRunes
	}

	// add an eventual user name mapping
	for name, r := range userCharMap {
		runeMap[name] = r
	}

	out := make(map[rune]byte)
	for by, name := range base {
		// add the potential difference
		if diff, ok := diffs[byte(by)]; ok {
			name = string(diff)
		}

		if name == "" {
			continue // not encoded
		}
		// resolve the rune from the name: first try with the
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
func baseCharMap(enc model.SimpleEncodingPredefined) [256]string {
	switch enc {
	case model.MacExpertEncoding:
		return simpleencodings.MacExpertNames
	case model.MacRomanEncoding:
		return simpleencodings.MacRomanNames
	case model.WinAnsiEncoding:
		return simpleencodings.WinAnsiNames
	default:
		panic("invalid simple encoding")
	}
}

// returns the mapping from unicode code points to byte
func runeMap(enc model.SimpleEncodingPredefined) map[rune]byte {
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

// returns the mapping from names to unicode code points
func namesMap(enc model.SimpleEncodingPredefined) map[string]rune {
	switch enc {
	case model.MacExpertEncoding:
		return simpleencodings.MacExpertRunes
	case model.MacRomanEncoding:
		return simpleencodings.MacRomanRunes
	case model.WinAnsiEncoding:
		return simpleencodings.WinAnsiRunes
	default:
		panic("invalid simple encoding")
	}
}

// try to read the embedded font file and return the font builtin
// encoding. If f is nil or an error occur, default to Standard
// fontType is needed to select the correct font parser.
func builtinEncoding(desc model.FontDescriptor, fontType model.Font) [256]string {
	// special case for two standard fonts where we dont need to read the font file
	if desc.FontName == "ZapfDingbats" {
		return simpleencodings.ZapfDingbatsNames
	} else if desc.FontName == "Symbol" {
		return simpleencodings.SymbolNames
	}

	if desc.FontFile == nil {
		return simpleencodings.StandardNames
	}
	content, err := desc.FontFile.Decode()
	if err != nil {
		log.Printf("unable to decode embedded font file: %s\n", err)
		return simpleencodings.StandardNames
	}
	switch fontType.(type) {
	case model.FontType1:
		info, err := type1font.ParsePfbContent(content)
		if err != nil {
			log.Printf("invalid embedded font file: %s\n", err)
			return simpleencodings.StandardNames
		}
		if info.Encoding.Standard {
			return simpleencodings.StandardNames
		}
		return info.Encoding.Custom
	default: // TODO:
		return simpleencodings.StandardNames
	}
}
