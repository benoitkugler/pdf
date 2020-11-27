// Code adapted from https://git.maze.io/go/unipdf/src/branch/master/internal/cmap
package cmapparser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/benoitkugler/pdf/fonts/cidfonts"
	"github.com/benoitkugler/pdf/model"
)

const (
	// Maximum number of possible bytes per code.
	maxCodeLen = 4

	// MissingCodeRune replaces runes that can't be decoded. '\ufffd' = �. Was '?'.
	MissingCodeRune = '\ufffd' // �
)

type CharCode = rune

type charRange struct {
	code0 CharCode
	code1 CharCode
}

type fbRange struct {
	code0 CharCode
	code1 CharCode
	r0    rune
}

// CMap represents a character code to unicode mapping used in PDF files.
// References:
//  https://www.adobe.com/content/dam/acom/en/devnet/acrobat/pdfs/5411.ToUnicode.pdf
//  https://github.com/adobe-type-tools/cmap-resources/releases
type CMap struct {
	*cMapParser

	cidfonts.CMap

	version       string
	usecmap       model.Name        // Base this cmap on `usecmap` if `usecmap` is not empty.
	codeToUnicode map[CharCode]rune // CID -> Unicode
}

// newCMap returns an initialized CMap.
func newCMap() *CMap {
	return &CMap{codeToUnicode: make(map[CharCode]rune)}
}

// LoadCmapFromData parses the in-memory cmap `data` and returns the resulting CMap.
//
// 9.10.3 ToUnicode CMaps (page 293).
func LoadCmapFromData(data []byte) (*CMap, error) {
	cmap := newCMap()
	cmap.cMapParser = newCMapParser(data)

	err := cmap.parse()
	if err != nil {
		return nil, err
	}
	if len(cmap.Codespaces) == 0 {
		if cmap.usecmap != "" {
			return cmap, nil
		}
		return nil, fmt.Errorf("%w: no codespaces", ErrBadCMap)
	}

	cmap.computeInverseMappings()
	return cmap, nil
}

// // LoadPredefinedCMap loads a predefined CJK CMap by name.
// // See section 9.7.5.2 "Predefined CMaps" (page 273, Table 118).
// func LoadPredefinedCMap(name model.Name) (*CMap, error) {
// 	// Load cmap.
// 	cmap, err := loadPredefinedCMap(name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if cmap.usecmap == "" {
// 		cmap.computeInverseMappings()
// 		return cmap, nil
// 	}

// 	// Load base cmap.
// 	_, err = loadPredefinedCMap(cmap.usecmap)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Add CID ranges.
// 	// for charcode, cid := range base.codeToCID {
// 	// 	if _, ok := cmap.codeToCID[charcode]; !ok {
// 	// 		cmap.codeToCID[charcode] = cid
// 	// 	}
// 	// }

// 	// Add codespaces.
// 	// for _, codespace := range base.codespaces {
// 	// 	cmap.Codespaces = append(cmap.Codespaces, codespace)
// 	// }

// 	cmap.computeInverseMappings()
// 	return cmap, nil
// }

func (cmap *CMap) computeInverseMappings() {
	// // Generate CID -> charcode map.
	// for code, cid := range cmap.codeToCID {
	// 	if c, ok := cmap.cidToCode[cid]; !ok || (ok && c > code) {
	// 		cmap.cidToCode[cid] = code
	// 	}
	// }

	// // Generate Unicode -> CID map.
	// for cid, r := range cmap.codeToUnicode {
	// 	if c, ok := cmap.unicodeToCode[r]; !ok || (ok && c > cid) {
	// 		cmap.unicodeToCode[r] = cid
	// 	}
	// }

	// Sort codespaces in order for shorter codes to be checked first.
	sort.Slice(cmap.Codespaces, func(i, j int) bool {
		return cmap.Codespaces[i].Low < cmap.Codespaces[j].Low
	})
}

// CharcodeBytesToUnicode converts a byte array of charcodes to a unicode string representation.
// It also returns a bool flag to tell if the conversion was successful.
// NOTE: This only works for ToUnicode cmaps.
func (cmap *CMap) CharcodeBytesToUnicode(data []byte) (string, int) {
	charcodes, matched := cmap.BytesToCharcodes(data)
	if !matched {
		return "", 0
	}

	var (
		parts   []rune
		missing []CharCode
	)
	for _, code := range charcodes {
		s, ok := cmap.codeToUnicode[code]
		if !ok {
			missing = append(missing, code)
			s = MissingCodeRune
		}
		parts = append(parts, s)
	}
	unicode := string(parts)
	return unicode, len(missing)
}

// CharcodeToUnicode converts a single character code `code` to a unicode string.
// If `code` is not in the unicode map, '�' is returned.
// NOTE: CharcodeBytesToUnicode is typically more efficient.
func (cmap *CMap) CharcodeToUnicode(code CharCode) (rune, bool) {
	if s, ok := cmap.codeToUnicode[code]; ok {
		return s, true
	}
	return MissingCodeRune, false
}

// RuneToCID maps the specified rune to a character identifier. If the provided
// rune has no available mapping, the second return value is false.
// func (cmap *CMap) RuneToCID(r rune) (CharCode, bool) {
// 	cid, ok := cmap.unicodeToCode[r]
// 	return cid, ok
// }

// CharcodeToCID maps the specified character code to a character identifier.
// If the provided charcode has no available mapping, the second return value
// is false. The returned CID can be mapped to a Unicode character using a
// Unicode conversion CMap.
// func (cmap *CMap) CharcodeToCID(code CharCode) (CharCode, bool) {
// 	cid, ok := cmap.codeToCID[code]
// 	return cid, ok
// }

// CIDToCharcode maps the specified character identified to a character code. If
// the provided CID has no available mapping, the second return value is false.
// func (cmap *CMap) CIDToCharcode(cid CharCode) (CharCode, bool) {
// 	code, ok := cmap.cidToCode[cid]
// 	return code, ok
// }

// BytesToCharcodes attempts to convert the entire byte array `data` to a list
// of character codes from the ranges specified by `cmap`'s codespaces.
// Returns:
//      character code sequence (if there is a match complete match)
//      matched?
// NOTE: A partial list of character codes will be returned if a complete match
//       is not possible.
func (cmap *CMap) BytesToCharcodes(data []byte) ([]CharCode, bool) {
	var charcodes []CharCode
	if cmap.CMap.Simple() {
		for _, b := range data {
			charcodes = append(charcodes, CharCode(b))
		}
		return charcodes, true
	}
	for i := 0; i < len(data); {
		code, n, matched := cmap.matchCode(data[i:])
		if !matched {
			return charcodes, false
		}
		charcodes = append(charcodes, code)
		i += n
	}
	return charcodes, true
}

// Bytes returns the raw bytes of a PDF CMap corresponding to `cmap`.
func (cmap *CMap) Bytes() []byte {
	body := cmap.toBfData()
	whole := strings.Join([]string{cmapHeader, body, cmapTrailer}, "\n")
	return []byte(whole)
}

// matchCode attempts to match the byte array `data` with a character code in `cmap`'s codespaces.
// Returns:
//      character code (if there is a match) of
//      number of bytes read (if there is a match)
//      matched?
func (cmap *CMap) matchCode(data []byte) (code CharCode, n int, matched bool) {
	for j := 0; j < maxCodeLen; j++ {
		if j < len(data) {
			code = code<<8 | CharCode(data[j])
			n++
		}
		matched = cmap.inCodespace(code, j+1)
		if matched {
			return code, n, true
		}
	}
	// No codespace matched data. This is a serious problem.
	return 0, 0, false
}

// inCodespace returns true if `code` is in the `numBytes` byte codespace.
func (cmap *CMap) inCodespace(code CharCode, numBytes int) bool {
	for _, cs := range cmap.Codespaces {
		if cs.Low <= code && code <= cs.High && numBytes == cs.NumBytes {
			return true
		}
	}
	return false
}

// toBfData returns the bfchar and bfrange sections of a CMap text file.
// Both sections are computed from cmap.codeToUnicode.
func (cmap *CMap) toBfData() string {
	if len(cmap.codeToUnicode) == 0 {
		return ""
	}

	// codes is a sorted list of the codeToUnicode keys.
	var codes []CharCode
	for code := range cmap.codeToUnicode {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })

	// charRanges is a list of the contiguous character code ranges in `codes`.
	var charRanges []charRange
	c0, c1 := codes[0], codes[0]+1
	for _, c := range codes[1:] {
		if c != c1 {
			charRanges = append(charRanges, charRange{c0, c1})
			c0 = c
		}
		c1 = c + 1
	}
	if c1 > c0 {
		charRanges = append(charRanges, charRange{c0, c1})
	}

	// fbChars is a list of single character ranges. fbRanges is a list of multiple character ranges.
	var fbChars []CharCode
	var fbRanges []fbRange
	for _, cr := range charRanges {
		if cr.code0+1 == cr.code1 {
			fbChars = append(fbChars, cr.code0)
		} else {
			fbRanges = append(fbRanges, fbRange{
				code0: cr.code0,
				code1: cr.code1,
				r0:    cmap.codeToUnicode[cr.code0],
			})
		}
	}

	var lines []string
	if len(fbChars) > 0 {
		numRanges := (len(fbChars) + maxBfEntries - 1) / maxBfEntries
		for i := 0; i < numRanges; i++ {
			n := min(len(fbChars)-i*maxBfEntries, maxBfEntries)
			lines = append(lines, fmt.Sprintf("%d beginbfchar", n))
			for j := 0; j < n; j++ {
				code := fbChars[i*maxBfEntries+j]
				r := cmap.codeToUnicode[code]
				lines = append(lines, fmt.Sprintf("<%04x> <%04x>", code, r))
			}
			lines = append(lines, "endbfchar")
		}
	}
	if len(fbRanges) > 0 {
		numRanges := (len(fbRanges) + maxBfEntries - 1) / maxBfEntries
		for i := 0; i < numRanges; i++ {
			n := min(len(fbRanges)-i*maxBfEntries, maxBfEntries)
			lines = append(lines, fmt.Sprintf("%d beginbfrange", n))
			for j := 0; j < n; j++ {
				rng := fbRanges[i*maxBfEntries+j]
				r := rng.r0
				lines = append(lines, fmt.Sprintf("<%04x><%04x> <%04x>", rng.code0, rng.code1-1, r))
			}
			lines = append(lines, "endbfrange")
		}
	}
	return strings.Join(lines, "\n")
}

const (
	maxBfEntries = 100 // Maximum number of entries in a bfchar or bfrange section.
	cmapHeader   = `
/CIDInit/ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo <</Registry (Adobe)/Ordering (UCS)/Supplement 0 >> def
/CMapName/Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
`
	cmapTrailer = `endcmap
CMapName currentdict/CMap defineresource pop
end
end
`
)
