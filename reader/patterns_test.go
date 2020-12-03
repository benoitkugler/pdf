package reader

import (
	"os"
	"testing"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/exp/errors/fmt"
)

func TestCS(t *testing.T) {
	file := "test/CMYKSpot_OP.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}

	doc, _, err := ParsePDFReader(f, Options{})
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
		for _, sh := range page.Resources.Shading {
			fmt.Printf("%T\n", sh.ShadingType)
		}
		for _, pat := range page.Resources.Pattern {
			if sh, ok := pat.(*model.PatternShading); ok {
				fmt.Printf("%T\n", sh.Shading.ShadingType)
			}
		}
	}
	fmt.Println("Total color spaces:", len(m))
}

func walkShadings(doc model.Document) (nbFreeForm, nbCoons int) {
	ffs := map[*model.ShadingDict]int{}
	coons := map[*model.ShadingDict]int{}
	analyseShading := func(sh *model.ShadingDict) {
		switch sub := sh.ShadingType.(type) {
		case model.ShadingFreeForm:
			ffs[sh]++
			fmt.Println("FreeForm:", sub.BitsPerFlag, sub.BitsPerComponent, sub.BitsPerCoordinate)
		case model.ShadingLattice:
			fmt.Println("Lattice:", sub.VerticesPerRow, sub.BitsPerComponent, sub.BitsPerCoordinate)
		case model.ShadingCoons:
			coons[sh]++
			fmt.Println("Coons:", sub.BitsPerFlag, sub.BitsPerComponent, sub.BitsPerCoordinate)
		case model.ShadingTensorProduct:
			fmt.Println("TensorProduct:", sub.BitsPerFlag, sub.BitsPerComponent, sub.BitsPerCoordinate)
		}
	}
	for _, page := range doc.Catalog.Pages.Flatten() {
		if page.Resources == nil {
			continue
		}
		for _, sh := range page.Resources.Shading {
			analyseShading(sh)
		}
		for _, pat := range page.Resources.Pattern {
			if pat, ok := pat.(*model.PatternShading); ok {
				analyseShading(pat.Shading)
			}
		}
	}
	return len(ffs), len(coons)
}

func TestShading7(t *testing.T) {
	file := "test/Shading7.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, _, err := ParsePDFReader(f, Options{})
	if err != nil {
		t.Fatal(err)
	}
	walkShadings(doc)
}

func TestShading6(t *testing.T) {
	_, nbCoons := walkShadings(pdfSpec)
	if nbCoons != 2 {
		t.Errorf("expected 2 reference to a Coons (type 6) Shading, got %d", nbCoons)
	}
}

func TestShading4(t *testing.T) {
	file := "test/Shading4.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, _, err := ParsePDFReader(f, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ff, _ := walkShadings(doc)
	// the same shading is referenced twice
	if ff != 1 {
		t.Errorf("expected 1 reference to a Free Form (type 4) Shading, got %d", ff)
	}
}

func TestWriteShadings(t *testing.T) {
	for _, file := range []string{
		"test/Shading4.pdf",
		"test/Shading7.pdf",
	} {

		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		doc, _, err := ParsePDFReader(f, Options{})
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		err = reWrite(doc, file+".pdf")
		if err != nil {
			t.Error(err)
		}
	}
}

func TestPatternTiling(t *testing.T) {
	tps := map[*model.PatternTiling]int{}
	for _, page := range pdfSpec.Catalog.Pages.Flatten() {
		if page.Resources == nil {
			continue
		}
		for _, pat := range page.Resources.Pattern {
			if pat, ok := pat.(*model.PatternTiling); ok {
				tps[pat]++
			}
		}
	}
	if len(tps) != 13 {
		t.Errorf("expected 13 tiling patterns, got %d", len(tps))
	}
}
