package model

import (
	"errors"
	"fmt"
)

// CMap map character code to CIDs.
// It is either predefined, or embedded in PDF as a stream.
type CMap struct {
	Name          Name
	CIDSystemInfo CIDSystemInfo
	Type          int
	Codespaces    []Codespace
	CIDs          []CIDRange

	simple *bool // cached value of Simple
}

// CID is a character code that correspond to one glyph
// It will be obtained (from the bytes of a string) through a CMap, and will be use
// as index in a CIDFont object
type CID int

// Codespace represents a single codespace range used in the CMap.
type Codespace struct {
	NumBytes  int  // how many bytes should be read to match this code (between 1 and 4)
	Low, High rune // compact version of [4]byte
}

// NewCodespaceFromBytes convert the bytes to an int character code
// Invalid ranges will be rejected with an error
func NewCodespaceFromBytes(low, high []byte) (Codespace, error) {
	if len(low) != len(high) {
		return Codespace{}, errors.New("unequal number of bytes in range")
	}
	if L := len(low); L > 4 {
		return Codespace{}, fmt.Errorf("unsupported number of bytes: %d", L)
	}
	lowR := hexToRune(low)
	highR := hexToRune(high)
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

// hexToRune returns the integer that is encoded in `shex` as a big-endian hex value
func hexToRune(shex []byte) rune {
	var code rune
	for _, v := range shex {
		code <<= 8
		code |= rune(v)
	}
	return code
}

// CIDRange is an increasing number of CIDs,
// associated from Low to High.
type CIDRange struct {
	Codespace
	CIDStart CID // CID code for the first character code in range
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

// RuneToCID accumulate all the CID ranges to one map
func (cm CMap) RuneToCID() map[rune]CID {
	out := map[rune]CID{}
	for _, v := range cm.CIDs {
		for index := rune(0); index <= v.High-v.Low; index++ {
			out[v.Low+index] = v.CIDStart + CID(index)
		}
	}
	return out
}
