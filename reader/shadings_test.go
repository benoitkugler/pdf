package reader

import (
	"os"
	"testing"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/exp/errors/fmt"
)

func TestCS(t *testing.T) {
	file := "datatest/CMYKSpot_OP.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}

	doc, _, err := ParsePDF(f, "")
	if err != nil {
		t.Fatal(err)
	}

	m := map[*model.ColorSpace]int{}
	var walkCs func(cs model.ColorSpace)
	walkCs = func(cs model.ColorSpace) {
		m[&cs]++
		switch cs := cs.(type) {
		case model.ColorSpaceSeparation:
			walkCs(cs.AlternateSpace)
		case *model.ColorSpaceICCBased:
			walkCs(cs.Alternate)
		case model.ColorSpaceDeviceN:
			walkCs(cs.AlternateSpace)
			if cs.Attributes != nil {
				for _, col := range cs.Attributes.Colorants {
					walkCs(col)
				}
				walkCs(cs.Attributes.Process.ColorSpace)
			}
		case model.ColorSpaceUncoloredPattern:
			walkCs(cs.UnderlyingColorSpace)
		}
	}
	for _, page := range doc.Catalog.Pages.Flatten() {
		if page.Resources == nil {
			continue
		}
		for _, cs := range page.Resources.ColorSpace {
			walkCs(cs)
		}
	}
	fmt.Println("Total color spaces:", len(m))
}
