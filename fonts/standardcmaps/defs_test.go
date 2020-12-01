package standardcmaps

import (
	"io/ioutil"
	"testing"

	"github.com/benoitkugler/pdf/fonts/cmaps"
)

func TestUnicodeMapping(t *testing.T) {
	b, err := ioutil.ReadFile("../cmaps/test/Adobe-CNS1-3.cmap")
	if err != nil {
		t.Fatal(err)
	}
	cmap, err := cmaps.ParseCIDCMap(b)
	if err != nil {
		t.Fatal(err)
	}

	toU := Adobe_CNS1_UCS2.ProperLookupTable()

	cids := cmap.CharCodeToCID()
	for _, cid := range cids {
		if _, ok := toU[cid]; !ok {
			t.Errorf("missing cid %d", cid)
		}
	}
}
