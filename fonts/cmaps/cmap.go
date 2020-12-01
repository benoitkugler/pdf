// Implements a CMap parser (both for ToUnicode and CID CMaps)
package cmaps

import (
	"errors"
	"fmt"
	"sort"

	"github.com/benoitkugler/pdf/model"
)

const (
	// Maximum number of possible bytes per code.
	maxCodeLen = 4

	// MissingCodeRune replaces runes that can't be decoded. '\ufffd' = �. Was '?'.
	MissingCodeRune = '\ufffd' // �
)

// CharCode is a compact representation of 1 to 4 bytes,
// as found in PDF content streams.
type CharCode int32

// Append add 1 to 4 bytes to `bs`, in Big-Endian order.
func (c CharCode) Append(bs *[]byte) {
	if c < (1 << 8) {
		*bs = append(*bs, byte(c))
	} else if c < (1 << 16) {
		*bs = append(*bs, byte(c>>8), byte(c))
	} else if c < (1 << 24) {
		*bs = append(*bs, byte(c>>16), byte(c>>8), byte(c))
	} else {
		*bs = append(*bs, byte(c>>24), byte(c>>16), byte(c>>8), byte(c))
	}
}

// CMap map character code to CIDs.
// It is either predefined, or embedded in PDF as a stream.
type CMap struct {
	Name          model.Name
	CIDSystemInfo model.CIDSystemInfo
	Type          int
	Codespaces    []Codespace
	CIDs          []CIDRange

	UseCMap model.Name

	simple *bool // cached value of Simple
}

// Codespace represents a single codespace range used in the CMap.
type Codespace struct {
	NumBytes  int      // how many bytes should be read to match this code (between 1 and 4)
	Low, High CharCode // compact version of [4]byte
}

// newCodespaceFromBytes convert the bytes to an int character code
// Invalid ranges will be rejected with an error
func newCodespaceFromBytes(low, high []byte) (Codespace, error) {
	if len(low) != len(high) {
		return Codespace{}, errors.New("unequal number of bytes in range")
	}
	if L := len(low); L > 4 {
		return Codespace{}, fmt.Errorf("unsupported number of bytes: %d", L)
	}
	lowR := CharCode(hexToRune(low))
	highR := CharCode(hexToRune(high))
	if highR < lowR {
		return Codespace{}, errors.New("invalid caracter code range")
	}
	c := Codespace{Low: lowR, High: highR}
	c.NumBytes = numBytes(c)
	return c, nil
}

// numBytes returns how many bytes should be read to match this code.
// It will always be between 1 and 4.
func numBytes(c Codespace) int {
	if c.High < (1 << 8) {
		return 1
	} else if c.High < (1 << 16) {
		return 2
	} else if c.High < (1 << 24) {
		return 3
	} else {
		return 4
	}
}

// CIDRange is an increasing number of CIDs,
// associated from Low to High.
type CIDRange struct {
	Codespace
	CIDStart model.CID // CID code for the first character code in range
}

// Simple returns `true` if only one-byte character code are encoded
// It is cached for performance reasons, so `Codespaces` shoudn't be mutated
// after the call.
func (cm *CMap) Simple() bool {
	if cm.simple != nil {
		return *cm.simple
	}
	simple := true
	for _, space := range cm.Codespaces {
		if space.NumBytes > 1 {
			simple = false
			break
		}
	}
	cm.simple = &simple
	return simple
}

// CharCodeToCID accumulate all the CID ranges into one map
func (cm CMap) CharCodeToCID() map[CharCode]model.CID {
	out := map[CharCode]model.CID{}
	for _, v := range cm.CIDs {
		for index := CharCode(0); index <= v.High-v.Low; index++ {
			out[v.Low+index] = v.CIDStart + model.CID(index)
		}
	}
	return out
}

// BytesToCharcodes attempts to convert the entire byte array `data` to a list
// of character codes from the ranges specified by `cmap`'s codespaces.
// Returns:
//      character code sequence (if there is a match complete match)
//      matched?
// NOTE: A partial list of character codes will be returned if a complete match
//       is not possible.
// func (cmap CMap) BytesToCharcodes(data []byte) ([]CharCode, bool) {
// 	var charcodes []CharCode
// 	if cmap.CMap.Simple() {
// 		for _, b := range data {
// 			charcodes = append(charcodes, CharCode(b))
// 		}
// 		return charcodes, true
// 	}
// 	for i := 0; i < len(data); {
// 		code, n, matched := cmap.matchCode(data[i:])
// 		if !matched {
// 			return charcodes, false
// 		}
// 		charcodes = append(charcodes, code)
// 		i += n
// 	}
// 	return charcodes, true
// }

type charRange struct {
	code0 CharCode
	code1 CharCode
}

type fbRange struct {
	code0 CharCode
	code1 CharCode
	r0    rune
}

// ParseUnicodeCMap parses the cmap `data` and returns the resulting CMap.
// See 9.10.3 ToUnicode CMaps
func ParseUnicodeCMap(data []byte) (UnicodeCMap, error) {
	cmap := newparser(data)

	err := cmap.parse()
	if err != nil {
		return UnicodeCMap{}, err
	}

	cmap.computeInverseMappings()
	return cmap.unicode, nil
}

// ParseCIDCMap parses the in-memory cmap `data` and returns the resulting CMap.
// See 9.7.5.3 Embedded CMap Files
func ParseCIDCMap(data []byte) (CMap, error) {
	cmap := newparser(data)

	err := cmap.parse()
	if err != nil {
		return CMap{}, err
	}
	if len(cmap.cids.Codespaces) == 0 {
		if cmap.cids.UseCMap != "" {
			return cmap.cids, nil
		}
		return CMap{}, fmt.Errorf("%w: no codespaces", ErrBadCMap)
	}

	cmap.computeInverseMappings()
	return cmap.cids, nil
}

// // LoadPredefinedCMap loads a predefined CJK CMap by name.
// // See section 9.7.5.2 "Predefined CMaps" (page 273, Table 118).
// func LoadPredefinedCMap(name model.Name) (*CMappparser, error) {
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

func (cmap *parser) computeInverseMappings() {
	// // Generate CID -> charcode map.
	// for code, cid := range cmap.codeToCID {
	// 	if c, ok := cmap.cidToCode[cid]; !ok || (ok && c > code) {
	// 		cmap.cidToCode[cid] = code
	// 	}
	// }

	// // Generate Unicode -> CID map.
	// for cid, r := range cmap.unicode.CodeToUnicode {
	// 	if c, ok := cmap.unicodeToCode[r]; !ok || (ok && c > cid) {
	// 		cmap.unicodeToCode[r] = cid
	// 	}
	// }

	// Sort codespaces in order for shorter codes to be checked first.
	sort.Slice(cmap.cids.Codespaces, func(i, j int) bool {
		return cmap.cids.Codespaces[i].Low < cmap.cids.Codespaces[j].Low
	})
}

// // CharcodeBytesToUnicode converts a byte array of charcodes to a unicode string representation.
// // It also returns a bool flag to tell if the conversion was successful.
// // NOTE: This only works for ToUnicode cmaps.
// func (cmap *parser) CharcodeBytesToUnicode(data []byte) (string, int) {
// 	charcodes, matched := cmap.BytesToCharcodes(data)
// 	if !matched {
// 		return "", 0
// 	}

// 	var (
// 		parts   []rune
// 		missing []CharCode
// 	)
// 	lt := cmap.unicode.ProperLookupTable()
// 	for _, code := range charcodes {
// 		s, ok := lt[code]
// 		if !ok {
// 			missing = append(missing, code)
// 			s = MissingCodeRune
// 		}
// 		parts = append(parts, s)
// 	}
// 	unicode := string(parts)
// 	return unicode, len(missing)
// }

// RuneToCID maps the specified rune to a character identifier. If the provided
// rune has no available mapping, the second return value is false.
// func (cmap *CMappparser) RuneToCID(r rune) (CharCode, bool) {
// 	cid, ok := cmap.unicodeToCode[r]
// 	return cid, ok
// }

// CharcodeToCID maps the specified character code to a character identifier.
// If the provided charcode has no available mapping, the second return value
// is false. The returned CID can be mapped to a Unicode character using a
// Unicode conversion CMap.
// func (cmap *CMappparser) CharcodeToCID(code CharCode) (CharCode, bool) {
// 	cid, ok := cmap.codeToCID[code]
// 	return cid, ok
// }

// CIDToCharcode maps the specified character identified to a character code. If
// the provided CID has no available mapping, the second return value is false.
// func (cmap *CMappparser) CIDToCharcode(cid CharCode) (CharCode, bool) {
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
	if cmap.Simple() {
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

// // Bytes returns the raw bytes of a PDF CMap corresponding to `cmap`.
// func (cmap *parser) Bytes() []byte {
// 	body := cmap.toBfData()
// 	whole := strings.Join([]string{cmapHeader, body, cmapTrailer}, "\n")
// 	return []byte(whole)
// }

// matchCode attempts to match the byte array `data` with a character code in `cmap`'s codespaces.
// Returns:
//      character code (if there is a match) of
//      number of bytes read (if there is a match)
//      matched?
func (cmap CMap) matchCode(data []byte) (code CharCode, n int, matched bool) {
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
func (cmap CMap) inCodespace(code CharCode, numBytes int) bool {
	for _, cs := range cmap.Codespaces {
		if cs.Low <= code && code <= cs.High && numBytes == cs.NumBytes {
			return true
		}
	}
	return false
}

// // toBfData returns the bfchar and bfrange sections of a CMap text file.
// // Both sections are computed from cmap.unicode.CodeToUnicode.
// func (cmap *parser) toBfData() string {
// 	if len(cmap.unicode.CodeToUnicode) == 0 {
// 		return ""
// 	}

// 	// codes is a sorted list of the codeToUnicode keys.
// 	var codes []CharCode
// 	for code := range cmap.unicode.CodeToUnicode {
// 		codes = append(codes, code)
// 	}
// 	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })

// 	// charRanges is a list of the contiguous character code ranges in `codes`.
// 	var charRanges []charRange
// 	c0, c1 := codes[0], codes[0]+1
// 	for _, c := range codes[1:] {
// 		if c != c1 {
// 			charRanges = append(charRanges, charRange{c0, c1})
// 			c0 = c
// 		}
// 		c1 = c + 1
// 	}
// 	if c1 > c0 {
// 		charRanges = append(charRanges, charRange{c0, c1})
// 	}

// 	// fbChars is a list of single character ranges. fbRanges is a list of multiple character ranges.
// 	var fbChars []CharCode
// 	var fbRanges []fbRange
// 	for _, cr := range charRanges {
// 		if cr.code0+1 == cr.code1 {
// 			fbChars = append(fbChars, cr.code0)
// 		} else {
// 			fbRanges = append(fbRanges, fbRange{
// 				code0: cr.code0,
// 				code1: cr.code1,
// 				r0:    cmap.unicode.CodeToUnicode[cr.code0],
// 			})
// 		}
// 	}

// 	var lines []string
// 	if len(fbChars) > 0 {
// 		numRanges := (len(fbChars) + maxBfEntries - 1) / maxBfEntries
// 		for i := 0; i < numRanges; i++ {
// 			n := min(len(fbChars)-i*maxBfEntries, maxBfEntries)
// 			lines = append(lines, fmt.Sprintf("%d beginbfchar", n))
// 			for j := 0; j < n; j++ {
// 				code := fbChars[i*maxBfEntries+j]
// 				r := cmap.unicode.CodeToUnicode[code]
// 				lines = append(lines, fmt.Sprintf("<%04x> <%04x>", code, r))
// 			}
// 			lines = append(lines, "endbfchar")
// 		}
// 	}
// 	if len(fbRanges) > 0 {
// 		numRanges := (len(fbRanges) + maxBfEntries - 1) / maxBfEntries
// 		for i := 0; i < numRanges; i++ {
// 			n := min(len(fbRanges)-i*maxBfEntries, maxBfEntries)
// 			lines = append(lines, fmt.Sprintf("%d beginbfrange", n))
// 			for j := 0; j < n; j++ {
// 				rng := fbRanges[i*maxBfEntries+j]
// 				r := rng.r0
// 				lines = append(lines, fmt.Sprintf("<%04x><%04x> <%04x>", rng.code0, rng.code1-1, r))
// 			}
// 			lines = append(lines, "endbfrange")
// 		}
// 	}
// 	return strings.Join(lines, "\n")
// }

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
