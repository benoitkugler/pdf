package cmaps

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/text/encoding/unicode"
)

// CMap parser errors.
var ErrBadCMap = errors.New("bad cmap")

func hexToCID(shex cmapHexString) (model.CID, error) {
	switch len(shex) {
	case 1:
		return model.CID(shex[0]), nil
	case 2:
		return model.CID(uint16(shex[0])<<8 + uint16(shex[1])), nil
	default:
		return 0, fmt.Errorf("invalid hex litteral %v", shex)
	}
}

// hexToCharCode returns the integer that is encoded in `shex` as a big-endian hex value
func hexToCharCode(shex cmapHexString) CharCode {
	code := CharCode(0)
	for _, v := range shex {
		code <<= 8
		code |= CharCode(v)
	}
	return code
}

// hexToString decodes the UTF-16BE encoded string `shex` to unicode runes.
// 9.10.3 ToUnicode CMaps (page 293)
// • It shall use the beginbfchar, endbfchar, beginbfrange, and endbfrange operators to define the
// mapping from character codes to Unicode character sequences expressed in UTF-16BE encoding.
func hexToRunes(shex cmapHexString) ([]rune, error) {
	// hex was already decoded
	b, err := utf16Dec.Bytes(shex)
	if err != nil {
		return nil, fmt.Errorf("invalid runes hex string %v: %s", shex, err)
	}
	return []rune(string(b)), nil
}

// hexToRune is the same as hexToRunes but expects only a single rune to be decoded.
func hexToRune(shex cmapHexString) rune {
	runes, err := hexToRunes(shex)
	if err != nil {
		return MissingCodeRune
	}
	if n := len(runes); n == 0 {
		return MissingCodeRune
	}
	return runes[0]
}

var (
	utf16Dec = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	utf16Enc = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder()
)

func runesToHex(text []rune) string {
	var chunks []string
	for _, letter := range text {
		s, _ := utf16Enc.Bytes([]byte(string(letter)))
		chunks = append(chunks, hex.EncodeToString(s))
	}
	return strings.Join(chunks, "")
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
