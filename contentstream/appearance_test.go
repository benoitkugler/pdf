package contentstream

import (
	"os"
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
	a.Op(OpShowSpaceText{Texts: []fonts.TextSpaced{
		{Text: "aaaaa", SpaceSubtractedAfter: -400},
		{Text: "bbbbb", SpaceSubtractedAfter: 200},
		{Text: "ccccc", SpaceSubtractedAfter: 0},
		{Text: "ddddd"},
	}})
	a.Op(OpTextNextLine{})
	a.MoveText(0, 40)
	a.Op(OpShowSpaceText{Texts: []fonts.TextSpaced{
		{Text: "aaaaa", SpaceSubtractedAfter: -400},
		{Text: "bbbbb", SpaceSubtractedAfter: 200},
		{Text: "cccccddddd"},
	}})
	a.EndText()
	fo := a.ToXFormObject()
	var doc model.Document

	doc.Catalog.Pages.Kids = []model.PageNode{
		&model.PageObject{
			Contents: []model.ContentStream{
				fo.ContentStream,
			},
			MediaBox:  &fo.BBox,
			Resources: &fo.Resources,
		},
	}
	f, err := os.Create("test/kerning.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = doc.Write(f, nil)
	if err != nil {
		t.Fatal(err)
	}
}
