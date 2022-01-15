package contentstream

import (
	"testing"

	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
)

var defaultFont = &model.FontDict{
	Subtype: standardfonts.Helvetica.WesternType1Font(),
}

func newAp(width, height Fl) GraphicStream {
	return NewGraphicStream(model.Rectangle{Llx: 0, Lly: 0, Urx: width, Ury: height})
}

func TestGradient(t *testing.T) {
	var doc model.Document

	a := newAp(600, 600)

	a.Ops(OpSave{})
	a.Ops(OpRectangle{20, 20, 200, 200})
	a.Ops(OpClip{})
	a.Ops(OpEndPath{})
	a.FillLinearGradientRGB(GradientPointRGB{
		25, 25, RGB{100, 10, 200},
	}, GradientPointRGB{
		120, 200, RGB{10, 200, 10},
	})
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
	a.FillRadialGradientRGB(GradientPointRGB{
		40, 300, RGB{100, 100, 200},
	}, GradientPointRGB{
		25, 300, RGB{230, 10, 50},
	}, 100)
	a.Ops(OpRestore{})

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.toPageObject())

	err := doc.WriteFile("test/gradients.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGradientTransform(t *testing.T) {
	var doc model.Document

	from, to := GradientPointRGB{80, 80, RGB{100, 10, 200}}, GradientPointRGB{90, 90, RGB{10, 200, 10}}
	var radius Fl = 60.
	sh := &model.ShadingDict{
		ColorSpace: model.ColorSpaceRGB,
		ShadingType: model.ShadingRadial{
			BaseGradient: newBaseGradientRGB(from.RGB, to.RGB),
			Coords:       [6]Fl{from.X, from.X, 0, to.X, to.X, radius},
		},
	}
	pat1 := &model.PatternShading{
		Shading: sh,
	}
	// pat2 := &model.PatternShading{
	// 	Shading: sh,
	// 	// Matrix:  model.Matrix{2, 0, 0, 1, 250, 250},
	// }
	a := newAp(600, 600)

	patName1 := a.AddPattern(pat1)
	// patName2 := a.AddPattern(pat2)

	a.Ops(
		OpSave{},
		OpRectangle{20, 20, 200, 200},
		OpSetFillColorSpace{ColorSpace: model.ColorSpacePattern},
		OpSetFillColorN{Pattern: patName1},
		OpFill{},
		OpRestore{},
	)

	rectXO := a.ToXFormObject(false)
	rectXOName := a.addXobject(rectXO)

	a.Ops(
		OpSave{},
		OpConcat{Matrix: model.Matrix{1, 0, 0, 2, 200, 300}},
		OpXObject{XObject: rectXOName},
		OpRestore{},
	)

	a.Ops(
		OpConcat{Matrix: model.Matrix{1, 0, 0, 1, 0, 250}},
		OpRectangle{20, 20, 200, 200},
		OpClip{},
		OpEndPath{},
	)
	a.FillRadialGradientRGB(from, to, radius)

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.toPageObject())

	err := doc.WriteFile("test/gradient_transform.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultiGradient(t *testing.T) {
	p := GradientComplex{
		Direction: GradientLinear{20, 20, 200, 200},
		Colors: [][4]Fl{
			{1, 0.40, 0.50, 1},
			{0.200, 0.200, 0.50, 1},
			{0.200, 0.40, 0.200, 1},
			{0.0, 0.200, 1, 1},
		},
		Offsets: []Fl{0, 0.2, 0.4, 1},
	}
	sh, alpha := p.BuildShadings()

	if alpha != nil {
		t.Fatal("expected nil alpha")
	}

	var doc model.Document

	a := newAp(600, 600)
	shName := a.AddShading(sh)

	a.Ops(
		OpSave{},
		OpRectangle{20, 20, 200, 200},
		OpClip{},
		OpEndPath{},
		OpShFill{Shading: shName},
		OpRestore{},
	)

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.toPageObject())

	err := doc.WriteFile("test/gradient_multi.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGradientOpacity(t *testing.T) {
	p := GradientComplex{
		Direction: GradientLinear{20, 20, 200, 200},
		Colors: [][4]Fl{
			{1, 0.40, 0.50, 1},
			{0.200, 0.200, 0.50, 0.8},
			{0.200, 0.40, 0.200, 0.4},
			{0.0, 0.200, 1, 0},
		},
		Offsets: []Fl{0, 0.2, 0.4, 1},
	}
	sh, alpha := p.BuildShadings()

	if alpha == nil {
		t.Fatal("expected alpha")
	}

	pOpaque := GradientComplex{
		Direction: GradientLinear{20, 20, 200, 200},
		Colors: [][4]Fl{
			{1, 0.40, 0.50, 1},
			{0.200, 0.200, 0.50, 1},
			{0.200, 0.40, 0.200, 1},
			{0.0, 0.200, 1, 1},
		},
		Offsets: []Fl{0, 0.2, 0.4, 1},
	}
	shOpaque, _ := pOpaque.BuildShadings()
	var doc model.Document

	a := newAp(600, 600)
	shName := a.AddShading(sh)
	shOpaqueName := a.AddShading(shOpaque)

	apAlpha := newAp(600, 600)
	apAlpha.Shading(alpha)
	transparency := apAlpha.ToXFormObject(false)

	a.Ops(
		OpSave{},
		OpRectangle{20, 20, 200, 200},
		OpClip{},
		OpEndPath{},
	)
	a.DrawMask(transparency)
	a.Ops(
		OpShFill{Shading: shName},
		OpRestore{},
	)

	a.Transform(model.Matrix{1, 0, 0, 1, 200, 0})
	a.Ops(
		OpRectangle{20, 20, 200, 200},
		OpClip{},
		OpEndPath{},
		OpShFill{Shading: shOpaqueName},
	)

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.toPageObject())

	err := doc.WriteFile("test/gradient_opacity.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpacity(t *testing.T) {
	var doc model.Document

	a := newAp(600, 600)
	a.SetFillAlpha(0.5)

	a.Ops(
		OpRectangle{20, 20, 200, 200},
		OpFill{},
	)

	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.toPageObject())

	err := doc.WriteFile("/tmp/opa.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
}
