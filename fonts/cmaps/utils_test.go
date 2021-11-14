package cmaps

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestHexToRune(t *testing.T) {
	runes := [][]rune{
		{0x2003e},
		{0x0066, 0x0066},
		{0x0066, 0x0066, 0x006c},
	}
	strings := []string{
		"d840dc3e",
		"00660066",
		"00660066006c",
	}
	for i, rs := range runes {
		dec, _ := hex.DecodeString(strings[i])
		st := cmapHexString(dec)
		got, err := hexToRunes(st)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, rs) {
			t.Errorf("expected %v, got %v", rs, got)
		}

		if got := runesToHex(rs); got != strings[i] {
			t.Errorf("expected %v, got %v", strings[i], got)
		}
	}
}
