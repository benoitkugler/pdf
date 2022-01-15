package contentstream

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

type RGB [3]uint8

func (rgb RGB) toArray() []Fl {
	return []Fl{Fl(rgb[0]) / 255, Fl(rgb[1]) / 255, Fl(rgb[2]) / 255}
}

// GradientPointRGB defines the position and the (RGB) color
// of a gradient step.
type GradientPointRGB struct {
	X, Y Fl
	RGB
}

// GradientPointGray defines the position and the gray color
// of a gradient step.
type GradientPointGray struct {
	X, Y Fl
	G    uint8
}

func newBaseGradientRGB(from, to RGB) model.BaseGradient {
	return model.BaseGradient{
		Function: []model.FunctionDict{
			{
				FunctionType: model.FunctionExpInterpolation{
					C0: from.toArray(),
					C1: to.toArray(),
					N:  1, // linear
				},
				Domain: []model.Range{{0, 1}},
			},
		},
		Extend: [2]bool{true, true},
	}
}

func newBaseGradientGray(from, to GradientPointGray) model.BaseGradient {
	return model.BaseGradient{
		Function: []model.FunctionDict{
			{
				FunctionType: model.FunctionExpInterpolation{
					C0: []Fl{Fl(from.G) / 255},
					C1: []Fl{Fl(to.G) / 255},
					N:  1, // linear
				},
				Domain: []model.Range{{0, 1}},
			},
		},
		Extend: [2]bool{true, true},
	}
}

// AddShading checks is the shading is in the resources map
// or generates a new name and adds the shading.
func (ap GraphicStream) AddShading(newShading *model.ShadingDict) model.ObjName {
	for name, f := range ap.resources.Shading {
		if f == newShading {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.ObjName(fmt.Sprintf("SH%d", len(ap.resources.Shading)))
	ap.resources.Shading[name] = newShading
	return name
}

// AddLinearGradientRGB builds a linear gradient shading dictionnary,
// and use it to fill the current path.
//
// The vector's origin and destination are specified by
// the points `from` and `to`, expressed in user space units.
// In a linear gradient, blending occurs
// perpendicularly to this vector. Color 1 is used up to the origin of the
// vector and color 2 is used beyond the vector's end point. Between the points
// the colors are gradually blended.
func (ap *GraphicStream) FillLinearGradientRGB(from, to GradientPointRGB) {
	sh := &model.ShadingDict{
		ColorSpace: model.ColorSpaceRGB,
		ShadingType: model.ShadingAxial{
			BaseGradient: newBaseGradientRGB(from.RGB, to.RGB),
			Coords:       [4]Fl{from.X, from.Y, to.X, to.Y},
		},
	}
	ap.Shading(sh)
}

// AddRadialGradientRGB builds a radial gradient shading dictionnary,
// and use it to fill the current path.
//
// Color 1 begins at the origin point specified by `from`. Color 2 begins at the
// circle specified by the center point `to` and `radius`. Colors are
// gradually blended from the origin to the circle. The origin and the circle's
// center do not necessarily have to coincide, but the origin must be within
// the circle to avoid rendering problems.
func (ap *GraphicStream) FillRadialGradientRGB(from, to GradientPointRGB, radius Fl) {
	sh := &model.ShadingDict{
		ColorSpace: model.ColorSpaceRGB,
		ShadingType: model.ShadingRadial{
			BaseGradient: newBaseGradientRGB(from.RGB, to.RGB),
			Coords:       [6]Fl{from.X, from.Y, 0, to.X, to.Y, radius},
		},
	}
	ap.Shading(sh)
}

// AddExtGState checks if the graphic state is in the resources map or
// generate a new name and adds the graphic state to the resources.
func (ap GraphicStream) AddExtGState(newExtGState *model.GraphicState) model.ObjName {
	for name, f := range ap.resources.ExtGState {
		if f == newExtGState {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.ObjName(fmt.Sprintf("GS%d", len(ap.resources.ExtGState)))
	ap.resources.ExtGState[name] = newExtGState
	return name
}

// AddPattern checks if the pattern is in the resources map or
// generate a new name and adds the pattern.
func (ap *GraphicStream) AddPattern(newPattern model.Pattern) model.ObjName {
	for name, f := range ap.resources.Pattern {
		if f == newPattern {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.ObjName(fmt.Sprintf("PA%d", len(ap.resources.Pattern)))
	ap.resources.Pattern[name] = newPattern
	return name
}

// GradientComplex supports multiple stops and opacities.
type GradientComplex struct {
	Direction  GradientDirection // required
	Offsets    []Fl              // between 0 and 1, should contain at least 2 elements
	Colors     [][4]Fl           // RGBA values, between 0 and 1
	Reapeating bool
}

// needAlpha is false if we should avoid including an alpha stream
func (gr GradientComplex) buildBaseGradients() (color, alpha model.BaseGradient, needOpacity bool) {
	alphas := make([]Fl, len(gr.Colors))
	for i, c := range gr.Colors {
		alphas[i] = c[3]
		needOpacity = needOpacity || c[3] != 1
	}

	alphaCouples := make([][2]Fl, len(alphas)-1)
	colorCouples := make([][2][3]Fl, len(alphas)-1)
	exponents := make([]int, len(alphas)-1)
	for i := range alphaCouples {
		alphaCouples[i] = [2]Fl{alphas[i], alphas[i+1]}
		colorCouples[i] = [2][3]Fl{
			{gr.Colors[i][0], gr.Colors[i][1], gr.Colors[i][2]},
			{gr.Colors[i+1][0], gr.Colors[i+1][1], gr.Colors[i+1][2]},
		}
		exponents[i] = 1
	}

	// Premultiply colors
	for i, alpha := range alphas {
		if alpha == 0 {
			if i > 0 {
				colorCouples[i-1][1] = colorCouples[i-1][0]
			}
			if i < len(gr.Colors)-1 {
				colorCouples[i][0] = colorCouples[i][1]
			}
		}
	}
	for i, v := range alphaCouples {
		a0, a1 := v[0], v[1]
		if a0 != 0 && a1 != 0 && v != ([2]Fl{1, 1}) {
			exponents[i] = int(a0 / a1)
		}
	}

	var functions, alphaFunctions []model.FunctionDict
	for i, v := range colorCouples {
		c0, c1 := v[0], v[1]
		n := exponents[i]
		fn := model.FunctionDict{
			Domain: []model.Range{{0, 1}},
			FunctionType: model.FunctionExpInterpolation{
				C0: c0[:],
				C1: c1[:],
				N:  n,
			},
		}
		functions = append(functions, fn)

		alphaFn := fn
		a0, a1 := alphaCouples[i][0], alphaCouples[i][1]
		alphaFn.FunctionType = model.FunctionExpInterpolation{
			C0: []model.Fl{a0},
			C1: []model.Fl{a1},
			N:  1,
		}
		alphaFunctions = append(alphaFunctions, alphaFn)
	}

	stitching := model.FunctionStitching{
		Functions: functions,
		Bounds:    gr.Offsets[1 : len(gr.Offsets)-1],
		Encode:    model.FunctionEncodeRepeat(len(gr.Colors) - 1),
	}
	stitchingAlpha := stitching
	stitchingAlpha.Functions = alphaFunctions

	bg := model.BaseGradient{
		Domain: [2]Fl{gr.Offsets[0], gr.Offsets[len(gr.Offsets)-1]},
		Function: []model.FunctionDict{{
			Domain:       []model.Range{{gr.Offsets[0], gr.Offsets[len(gr.Offsets)-1]}},
			FunctionType: stitching,
		}},
	}

	if !gr.Reapeating {
		bg.Extend = [2]bool{true, true}
	}

	// alpha stream is similar
	alphaBg := bg.Clone()
	alphaBg.Function[0].FunctionType = stitchingAlpha

	return bg, alphaBg, needOpacity
}

// GradientDirection is either GradientRadial or GradientLinear
type GradientDirection interface {
	isRadial() bool
}

// x1, y1, x2, y2
type GradientLinear [4]Fl

func (GradientLinear) isRadial() bool { return false }

// cx, cy, fx, fy, r, fr
type GradientRadial [6]Fl

func (GradientRadial) isRadial() bool { return true }

// BuildShadings returns the shadings objects to use in a stream
// `alpha` may be nil if no opacity channel is needed.
func (gr GradientComplex) BuildShadings() (color, alpha *model.ShadingDict) {
	colorBase, alphaBase, needOpacity := gr.buildBaseGradients()

	var type_, alphaType model.Shading
	switch dir := gr.Direction.(type) {
	case GradientLinear:
		type_ = model.ShadingAxial{
			BaseGradient: colorBase,
			Coords:       dir,
		}
		alphaType = model.ShadingAxial{
			BaseGradient: alphaBase,
			Coords:       dir,
		}
	case GradientRadial:
		type_ = model.ShadingRadial{
			BaseGradient: colorBase,
			Coords:       dir,
		}
		alphaType = model.ShadingRadial{
			BaseGradient: alphaBase,
			Coords:       dir,
		}
	}

	color = &model.ShadingDict{
		ColorSpace:  model.ColorSpaceRGB,
		ShadingType: type_,
	}

	if needOpacity {
		alpha = &model.ShadingDict{
			ColorSpace:  model.ColorSpaceGray,
			ShadingType: alphaType,
		}
	}

	return color, alpha
}

// DrawMask adds the given `transparency` content as an alpha mask.
func (ap *GraphicStream) DrawMask(transparency *model.XObjectForm) {
	alphaState := model.GraphicState{
		SMask: model.SoftMaskDict{
			S: model.ObjName("Luminosity"),
			G: &model.XObjectTransparencyGroup{XObjectForm: *transparency},
		},
		Ca:  model.ObjFloat(1),
		AIS: false,
	}

	ap.SetGraphicState(&alphaState)
}
