package reader

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
)

func TestDifferences(t *testing.T) {
	expected := model.Differences{24: "breve", 25: "caron", 39: "quotesingle", 96: "grave", 128: "bullet", 129: "emdash"}
	ar := model.ObjArray{
		model.ObjInt(39),
		model.ObjName("quotesingle"),
		model.ObjInt(24),
		model.ObjName("breve"),
		model.ObjName("caron"),
		model.ObjInt(96),
		model.ObjName("grave"),
		model.ObjInt(128),
		model.ObjName("bullet"),
		model.ObjName("emdash"),
	}
	var r resolver
	diff := r.parseDiffArray(ar)
	if !reflect.DeepEqual(diff, expected) {
		t.Errorf("expected %v, got %v", expected, diff)
	}
}

func TestGradients(t *testing.T) {
	doc, _, err := ParsePDFFile("test/gradients.pdf", Options{})
	if err != nil {
		t.Fatal(err)
	}

	var processRes func(model.ResourcesDict)
	processRes = func(r model.ResourcesDict) {
		for name, pattern := range r.Pattern {
			if sh, ok := pattern.(*model.PatternShading); ok {
				fmt.Printf("Pattern shading %s: %T\n", name, sh.Shading.ShadingType)
				fmt.Println("Pattern state", sh.ExtGState, "pattern cm", sh.Matrix)
			}
		}
		for name, state := range r.ExtGState {
			if g := state.SMask.G; g != nil {
				b, _ := g.Decode()
				fmt.Println("SMask  for", name, ":", string(b))
				fmt.Println("BBox for soft max Xobject", g.BBox)
				processRes(g.Resources)
			}
		}
		for _, obj := range r.XObject {
			if f, ok := obj.(*model.XObjectForm); ok {
				processRes(f.Resources)
			}
		}
		for name, obj := range r.Shading {
			fmt.Printf("Shading %s: %T\n", name, obj.ShadingType)
		}
	}
	for _, page := range doc.Catalog.Pages.Flatten() {
		if page.Resources != nil {
			processRes(*page.Resources)
		}
	}
}

func TestFontType1C(t *testing.T) {
	file := "test/corpus/UTF-32.pdf"
	// file := "test/type1C.pdf"
	ti := time.Now()
	doc, _, err := ParsePDFFile(file, Options{})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("processed in", time.Since(ti))
	seen := map[*model.FontDict]bool{} // avoid processing the same font twice
	for _, page := range doc.Catalog.Pages.Flatten() {
		if r := page.Resources; r != nil {
			for _, font := range r.Font {
				if seen[font] {
					continue
				}
				seen[font] = true
				var fontFile *model.FontFile
				if ttf, ok := font.Subtype.(model.FontType1); ok {
					if ft := ttf.FontDescriptor.FontFile; ft != nil && ft.Subtype == "Type1C" {
						fontFile = ft
					}
				} else if type0, ok := font.Subtype.(model.FontType0); ok {
					if ft := type0.DescendantFonts.FontDescriptor.FontFile; ft != nil && ft.Subtype == "CIDFontType0C" {
						fontFile = ft
					}
				}
				if fontFile == nil {
					continue
				}
				b, err := fontFile.Decode()
				if err != nil {
					t.Fatal(err)
				}
				fmt.Println("skipping Type1C font with length", len(b))
				// err = ioutil.WriteFile(string(font.Subtype.FontName())+".cff", b, os.ModePerm)
				// if err != nil {
				// 	t.Error(err)
				// }
				// ft, err := sfnt.Parse(bytes.NewReader(b))
				// // ft, err := sfnt.Parse(b)
				// if err != nil {
				// 	t.Fatal(err)
				// }

				// fmt.Println(ft.HheaTable())
				// fmt.Println(ft.OS2Table())
				// fmt.Println(ft.GposTable())
				// fmt.Println(ft.CmapTable())
				// ft.Kern(&b, sfnt.GlyphIndex(b1), sfnt.GlyphIndex(b2))
			}
		}
	}
}
