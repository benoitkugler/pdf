package cmaps

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/benoitkugler/textlayout/fonts"
)

func TestToUnicode(t *testing.T) {
	data := map[fonts.GID][]rune{
		2:  {12, 4, 789},
		4:  {12, 4, 789},
		5:  {78},
		45: {0x0c078, 0x0c356},
	}
	b := WriteAdobeIdentityUnicodeCMap(data)

	out, err := ParseUnicodeCMap(b)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range out.Mappings {
		m, ok := m.(ToUnicodePair)
		if !ok {
			t.Fatal()
		}
		expected := data[fonts.GID(m.From)]

		if !reflect.DeepEqual(expected, m.Dest) {
			t.Fatalf("expected %v, got  %v", expected, m.Dest)
		}
	}
}

func TestEncoding(t *testing.T) {
	b, _ := utf16Enc.Bytes([]byte(string(rune(0x2003e))))

	fmt.Println(hex.EncodeToString(b))
}
