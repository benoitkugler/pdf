package contentstream

import (
	"testing"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
)

func TestKerning(t *testing.T) {
	fontDict := model.FontDict{Subtype: standardfonts.Helvetica.WesternType1Font()}
	font, err := fonts.BuildFont(&fontDict)
	if err != nil {
		t.Fatal(err)
	}
	a := NewAppearance(400, 400)
	a.SetFontAndSize(font, 12)
	a.BeginText()
	// a.ShowText("ceci est un test avec des è)=àéà=é")
	a.Ops(OpShowSpaceText{Texts: []fonts.TextSpaced{
		{Text: "aaaaa", SpaceSubtractedAfter: -400},
		{Text: "bbbbb", SpaceSubtractedAfter: 200},
		{Text: "ccccc", SpaceSubtractedAfter: 0},
		{Text: "ddddd"},
	}})
	a.Ops(OpTextNextLine{})
	a.MoveText(0, 40)
	a.Ops(OpShowSpaceText{Texts: []fonts.TextSpaced{
		{Text: "aaaaa", SpaceSubtractedAfter: -400},
		{Text: "bbbbb", SpaceSubtractedAfter: 200},
		{Text: "cccccddddd"},
	}})
	a.EndText()
	var doc model.Document

	doc.Catalog.Pages.Kids = []model.PageNode{
		a.ToPageObject(false),
	}
	err = doc.WriteFile("test/kerning.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestImages(t *testing.T) {
	var doc model.Document

	for _, file := range imagesFiles {
		img, _, err := ParseImageFile(file)
		if err != nil {
			t.Fatal(err)
		}
		a := NewAppearance(600, 600)
		// default dpi
		w, h := RenderingDims{}.EffectiveSize(img)
		a.AddXObject(img, 50, 50, w, h)
		w2, h2 := RenderingDims{Width: RenderingLength(200)}.EffectiveSize(img)
		a.AddXObject(img, 300, 300, w2, h2)

		// one image per page
		doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.ToPageObject(false))
	}

	err := doc.WriteFile("test/images.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRoundedRect(t *testing.T) {
	var doc model.Document

	a := NewAppearance(600, 600)

	a.Ops(RoundedRectPath(20, 20, 200, 200, 10, 20, 0, 60)...)
	a.Ops(
		OpSetStrokeGray{G: 0.5},
		OpSetLineWidth{W: 3},
		OpSetFillRGBColor{R: 0.9, G: 0.9, B: 0.1},
		OpFillStroke{},
	)

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.ToPageObject(false))

	err := doc.WriteFile("test/rectangles.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}
