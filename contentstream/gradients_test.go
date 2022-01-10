package contentstream

import (
	"image/color"
	"testing"

	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
)

var defaultFont = &model.FontDict{
	Subtype: standardfonts.Helvetica.WesternType1Font(),
}

func TestGradient(t *testing.T) {
	var doc model.Document

	a := NewAppearance(600, 600)

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
	a := NewAppearance(600, 600)

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

func TestComplexGradient(t *testing.T) {
	paramss := []GradientComplex{
		{
			Direction: GradientLinear{},
			Stops: []GradientStop{
				{Color: color.Black},
				{Color: color.Gray{Y: 45}},
			},
		},
		{
			Direction: GradientLinear{},
			Stops: []GradientStop{
				{Color: color.RGBA{200, 40, 50, 255}},
				{Color: color.RGBA64{456, 7984, 456, 7984}},
				{Color: color.NRGBA{200, 40, 5, 0}},
				{Color: color.NRGBA64{456, 7984, 456, 0}},
			},
		},
		{
			Direction: GradientLinear{},
			Stops: []GradientStop{
				{Color: color.CMYK{45, 78, 45, 89}},
				{Color: color.CMYK{45, 78, 45, 89}},
			},
		},
		{
			Direction: GradientLinear{},
			Stops: []GradientStop{
				{Color: &color.Black},
				{Color: color.Gray{Y: 45}},
			},
		},
	}
	css := []model.ColorSpaceName{model.ColorSpaceGray, model.ColorSpaceRGB, model.ColorSpaceCMYK, model.ColorSpaceRGB}

	for i, params := range paramss {
		exp := css[i]
		if cs := params.buildShading().ColorSpace; cs != exp {
			t.Errorf("expected %s got %v", exp, cs)
		}
	}
}

func TestMultiGradient(t *testing.T) {
	p := GradientComplex{
		Direction: GradientLinear{20, 20, 200, 200},
		Stops: []GradientStop{
			{Color: color.NRGBA{200, 40, 50, 255}},
			{Color: color.NRGBA{200, 200, 50, 255}, Offset: 0.2},
			{Color: color.NRGBA{200, 40, 200, 255}, Offset: 0.4},
			{Color: color.NRGBA{0, 200, 50, 255}, Offset: 1},
			// {Color: color.NRGBA{200, 40, 50, 255}},
		},
	}
	sh := p.buildShading()

	var doc model.Document

	a := NewAppearance(600, 600)
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

// func TestGradientOpacity(t *testing.T) {
// 	var doc model.Document

// 	sh1 := NewLinearGradientRGB(GradientPointRGB{
// 		25, 25, 100, 10, 200,
// 	}, GradientPointRGB{
// 		120, 200, 10, 200, 10,
// 	})

// 	gray := newBaseGradientGray(GradientPointGray{
// 		25, 25, 20,
// 	}, GradientPointGray{
// 		120, 200, 200,
// 	})
// 	shGray := &model.ShadingDict{
// 		ColorSpace: model.ColorSpaceGray,
// 		ShadingType: model.ShadingAxial{
// 			BaseGradient: gray,
// 			Coords:       [4]Fl{25, 25, 120, 200},
// 		},
// 	}

// 	softMask := NewAppearance(600, 600)
// 	softMask.FillShading(shGray)

// 	state := model.GraphicState{
// 		SMask: model.SoftMaskDict{
// 			S: "Luminosity",
// 			G: &model.XObjectTransparencyGroup{
// 				XObjectForm: *softMask.ToXFormObject(),
// 			},
// 		},
// 	}
// 	a := NewAppearance(600, 600)

// 	a.addExtGState(&state)
// 	a.addExtGState(&model.GraphicState{
// 		Ca: model.ObjFloat(0.2),
// 	})

// 	a.Ops(
// 		OpSave{},
// 		OpRectangle{40, 40, 150, 150},
// 		OpSetFillGray{G: 0.5},
// 		OpSetExtGState{Dict: "GS1"},
// 		OpFill{},
// 		OpRestore{},
// 	)

// 	a.Ops(
// 		OpSave{},
// 		OpRectangle{20, 20, 200, 200},
// 		OpClip{},
// 		OpEndPath{},
// 		OpSetExtGState{Dict: "GS0"},
// 	)
// 	a.FillShading(sh1)
// 	a.Ops(OpRestore{})

// 	a.Ops(
// 		OpConcat{Matrix: model.Matrix{1, 0, 0, 1, 300, 300}},
// 		OpRectangle{20, 20, 200, 200},
// 		OpClip{},
// 		OpEndPath{},
// 	)
// 	a.FillShading(sh1)

// 	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.ToPageObject())

// 	err := doc.WriteFile("test/gradients2.pdf", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestGradientOpacity2(t *testing.T) {
// 	var doc model.Document

// 	sh1 := NewLinearGradientRGB(GradientPointRGB{
// 		25, 25, 100, 10, 200,
// 	}, GradientPointRGB{
// 		120, 200, 10, 200, 10,
// 	})

// 	// shGray := &model.ShadingDict{
// 	// 	ColorSpace: model.ColorSpaceGray,
// 	// 	ShadingType: model.ShadingAxial{
// 	// 		BaseGradient: newBaseGradientGray(GradientPointGray{
// 	// 			25, 25, 20,
// 	// 		}, GradientPointGray{
// 	// 			120, 200, 200,
// 	// 		}),
// 	// 		Coords: [4]Fl{25, 25, 120, 200},
// 	// 	},
// 	// }
// 	// softMask := NewAppearance(600, 600)
// 	// softMask.FillShading(shGray)
// 	// state := model.GraphicState{
// 	// 	SMask: model.SoftMaskDict{
// 	// 		S: "Luminosity",
// 	// 		G: &model.XObjectTransparencyGroup{
// 	// 			XObjectForm: *softMask.ToXFormObject(),
// 	// 		},
// 	// 	},
// 	// }

// 	pattern := &model.PatternShading{
// 		Shading: sh1,
// 		// ExtGState: &state,
// 		ExtGState: &model.GraphicState{
// 			Ca: model.ObjFloat(0.5),
// 		},
// 	}

// 	a := NewAppearance(600, 600)

// 	pattName := a.addPattern(pattern)

// 	// opaState := a.addExtGState(&state)
// 	grayState := a.addExtGState(&model.GraphicState{
// 		Ca: model.ObjFloat(0.5),
// 	})

// 	// control gray rectangle
// 	a.Ops(
// 		OpSave{},
// 		OpRectangle{40, 40, 150, 150},
// 		OpSetFillGray{G: 0.5},
// 		OpSetExtGState{Dict: grayState},
// 		OpFill{},
// 		OpRestore{},
// 	)

// 	fo, err := fonts.BuildFont(defaultFont)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	a.Ops(OpBeginText{})
// 	a.SetFontAndSize(fo, 42)
// 	a.Ops(
// 		// OpRectangle{20, 20, 200, 200},
// 		// OpClip{},
// 		// OpEndPath{},
// 		OpSetFillColorSpace{ColorSpace: "Pattern"},
// 		OpSetFillColorN{Pattern: pattName},
// 		// OpSetExtGState{Dict: grayState},
// 		// OpSetExtGState{Dict: opaState},
// 		// OpFill{},
// 		OpShowText{Text: "dsd:s!d;s!d!s:d!s:d;;:!sd;:!sd;s:d!s"},
// 		OpEndText{},
// 	)
// 	// a.FillShading(sh1)

// 	// simple gradient
// 	a.Ops(
// 		OpConcat{Matrix: model.Matrix{1, 0, 0, 1, 300, 300}},
// 		OpRectangle{20, 20, 200, 200},
// 		OpClip{},
// 		OpEndPath{},
// 	)
// 	a.FillShading(sh1)

// 	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, a.ToPageObject())

// 	err = doc.WriteFile("test/gradients2alt.pdf", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
