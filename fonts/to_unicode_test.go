package fonts

import (
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/fonts/cmaps"
)

func TestToUnicode(t *testing.T) {
	data := map[uint32][]rune{
		2:  {12, 4, 789},
		4:  {12, 4, 789},
		5:  {78},
		45: {0x0c078, 0x0c356},
	}
	b := WriteAdobeIdentityUnicodeCMap(data)

	out, err := cmaps.ParseUnicodeCMap(b)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range out.Mappings {
		m, ok := m.(cmaps.ToUnicodePair)
		if !ok {
			t.Fatal()
		}
		expected := data[uint32(m.From)]

		if !reflect.DeepEqual(expected, m.Dest) {
			t.Fatalf("expected %v, got  %v", expected, m.Dest)
		}
	}
}
