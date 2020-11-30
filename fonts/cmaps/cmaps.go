package cmaps

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

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
