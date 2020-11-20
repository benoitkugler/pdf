package reader

import (
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func TestDestinations(t *testing.T) {
	pages := pdfSpec.Catalog.Pages.Flatten()
	lookup := map[*model.PageObject]int{}
	for i, pa := range pages {
		lookup[pa] = i
	}
	for _, dest := range pdfSpec.Catalog.Names.Dests.LookupTable() {
		switch d := dest.(type) {
		case model.DestinationExplicitExtern: // ignored
		case model.DestinationExplicitIntern:
			if _, isIn := lookup[d.Page]; !isIn {
				t.Errorf("missing pages pointer %v", d.Page)
			}
		}
	}
}
