package contentstream

import (
	"fmt"
	"testing"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
)

// toPageObject returns a page object.
func (ap Appearance) toPageObject() *model.PageObject {
	var page model.PageObject
	ap.ApplyToPageObject(&page, false)
	return &page
}

func TestKerning(t *testing.T) {
	fontDict := model.FontDict{Subtype: standardfonts.Helvetica.WesternType1Font()}
	font, err := fonts.BuildFont(&fontDict)
	if err != nil {
		t.Fatal(err)
	}

	enc := func(s string) []byte { return font.Encode([]rune(s)) }
	a := newAp(400, 400)
	a.SetFontAndSize(font, 12)
	a.BeginText()
	// a.ShowText("ceci est un test avec des è)=àéà=é")
	a.Ops(OpShowSpaceText{Texts: []fonts.TextSpaced{
		{CharCodes: enc("aaaaa"), SpaceSubtractedAfter: -400},
		{CharCodes: enc("bbbbb"), SpaceSubtractedAfter: 200},
		{CharCodes: enc("ccccc"), SpaceSubtractedAfter: 0},
		{CharCodes: enc("ddddd")},
	}})
	a.Ops(OpTextNextLine{})
	a.MoveText(0, 40)
	a.Ops(OpShowSpaceText{Texts: []fonts.TextSpaced{
		{CharCodes: enc("éaaaa"), SpaceSubtractedAfter: -400},
		{CharCodes: enc("bbbbb"), SpaceSubtractedAfter: 200},
		{CharCodes: enc("cccccddddd")},
	}})
	a.EndText()
	var doc model.Document

	fmt.Println(font.Encode([]rune("éaaaa")))

	doc.Catalog.Pages.Kids = []model.PageNode{
		a.toPageObject(),
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
		a := newAp(600, 600)
		// default dpi
		w, h := RenderingDims{}.EffectiveSize(img)
		a.AddXObjectDims(img, 50, 50, w, h)
		w2, h2 := RenderingDims{Width: RenderingLength(200)}.EffectiveSize(img)
		a.AddXObjectDims(img, 300, 300, w2, h2)

		// one image per page
		doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.toPageObject())
	}

	err := doc.WriteFile("test/images.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRoundedRect(t *testing.T) {
	var doc model.Document

	a := newAp(600, 600)

	a.Ops(RoundedRectPath(20, 20, 200, 200, 10, 20, 0, 60)...)
	a.Ops(
		OpSetStrokeGray{G: 0.5},
		OpSetLineWidth{W: 3},
		OpSetFillRGBColor{R: 0.9, G: 0.9, B: 0.1},
		OpFillStroke{},
	)

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.toPageObject())

	err := doc.WriteFile("test/rectangles.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}
