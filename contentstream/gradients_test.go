package contentstream

import (
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func TestGradient(t *testing.T) {
	var doc model.Document

	sh1 := NewLinearGradientRGB(GradientPointRGB{
		25, 25, 100, 10, 200,
	}, GradientPointRGB{
		120, 200, 10, 200, 10,
	})

	sh2 := NewRadialGradientRGB(GradientPointRGB{
		40, 300, 100, 100, 200,
	}, GradientPointRGB{
		25, 300, 230, 10, 50,
	}, 100)

	a := NewAppearance(600, 600)

	a.Ops(OpSave{})
	a.Ops(OpRectangle{20, 20, 200, 200})
	a.Ops(OpClip{})
	a.Ops(OpEndPath{})
	a.FillShading(sh1)
	a.Ops(OpRestore{})

	a.Ops(OpMoveTo{25, 25})
	a.Ops(OpLineTo{120, 200})
	a.Ops(OpSetStrokeGray{G: 0.1})
	a.Ops(OpSetLineWidth{W: 2})
	a.Ops(OpStroke{})

	a.Ops(OpSave{})
	a.Ops(OpRectangle{20, 250, 200, 200})
	a.Ops(OpClip{})
	a.Ops(OpEndPath{})
	a.FillShading(sh2)
	a.Ops(OpRestore{})

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.ToPageObject())

	err := doc.WriteFile("test/gradients.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}
