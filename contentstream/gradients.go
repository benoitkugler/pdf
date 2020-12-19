package contentstream

import (
	"fmt"
	"image/color"
	"sort"

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
func (ap Appearance) AddShading(newShading *model.ShadingDict) model.ObjName {
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
func (ap *Appearance) FillLinearGradientRGB(from, to GradientPointRGB) {
	sh := &model.ShadingDict{
		ColorSpace: model.ColorSpaceRGB,
		ShadingType: model.ShadingAxial{
			BaseGradient: newBaseGradientRGB(from.RGB, to.RGB),
			Coords:       [4]Fl{from.X, from.Y, to.X, to.Y},
		},
	}
	name := ap.AddShading(sh)
	ap.Ops(OpShFill{Shading: name})
}

// AddRadialGradientRGB builds a radial gradient shading dictionnary,
// and use it to fill the current path.
//
// Color 1 begins at the origin point specified by `from`. Color 2 begins at the
// circle specified by the center point `to` and `radius`. Colors are
// gradually blended from the origin to the circle. The origin and the circle's
// center do not necessarily have to coincide, but the origin must be within
// the circle to avoid rendering problems.
func (ap *Appearance) FillRadialGradientRGB(from, to GradientPointRGB, radius Fl) {
	sh := &model.ShadingDict{
		ColorSpace: model.ColorSpaceRGB,
		ShadingType: model.ShadingRadial{
			BaseGradient: newBaseGradientRGB(from.RGB, to.RGB),
			Coords:       [6]Fl{from.X, from.Y, 0, to.X, to.Y, radius},
		},
	}
	name := ap.AddShading(sh)
	ap.Ops(OpShFill{Shading: name})
}

// AddExtGState checks if the graphic state is in the resources map or
// generate a new name and adds the graphic state to the resources.
func (ap Appearance) AddExtGState(newExtGState *model.GraphicState) model.ObjName {
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
func (ap Appearance) AddPattern(newPattern model.Pattern) model.ObjName {
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

// GradientStop defines one step in a gradient
type GradientStop struct {
	Color   color.Color
	Offset  Fl // between 0 and 1
	Opacity Fl // multiplied with the StopColor
}

func (gs GradientStop) nativeColor() ([]Fl, model.ColorSpaceName) {
	out := colorToArray(gs.Color)
	switch len(out) {
	case 1:
		return out, model.ColorSpaceGray
	default:
		return out, model.ColorSpaceRGB
	case 4:
		return out, model.ColorSpaceCMYK
	}
}

func (gs GradientStop) rgbColor() [3]Fl {
	r, g, b := colorRGB(gs.Color)
	return [3]Fl{r, g, b}
}

// GradientComplex supports multiple stops, opacity,
// and Gray, RGB or CMYK color spaces.
type GradientComplex struct {
	Direction GradientDirection // required
	Stops     []GradientStop    // should contains at least 2 elements
	Matrix    model.Matrix
}

// SetSpread enables a reflect or repeat spread. PDF only natively
// supports pad spread, meaning that the target rectangle for the gradient
// is needed.
// TODO: This is not currently supported.
func (g *GradientComplex) SetSpread(reflect bool, extent model.Rectangle) {
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

// SetGradientStroke buids a shading pattern for the given gradient,
// and adds it for futur stroking operations.
func (ap *Appearance) SetGradientStroke(params GradientComplex) {}

func (params GradientComplex) buildShading() *model.ShadingDict {
	// guard against degenerate cases
	switch len(params.Stops) {
	case 0:
		params.Stops = []GradientStop{{Color: color.Black}, {Color: color.Black, Offset: 1}}
	case 1:
		params.Stops = []GradientStop{{Color: params.Stops[0].Color}, {Color: params.Stops[0].Color, Offset: 1}}
	}

	// clamp invalid offset to produce a clean PDF output
	for i, s := range params.Stops {
		if s.Offset < 0 {
			params.Stops[i].Offset = 0
		}
		if s.Offset > 1 {
			params.Stops[i].Offset = 1
		}
	}

	// sort by offset in ascending order
	sort.SliceStable(params.Stops, func(i, j int) bool {
		return params.Stops[i].Offset < params.Stops[j].Offset
	})

	// resolve color space:
	//	- we do a first pass using the "native colors"
	//	- if the color space is not constant, we do a second pass to convert to RGB
	colors := make([][]Fl, len(params.Stops))
	_, cs := params.Stops[0].nativeColor()
	for i, s := range params.Stops {
		var csi model.ColorSpaceName
		colors[i], csi = s.nativeColor()
		if csi != cs {
			cs = ""
			break
		}
	}
	if cs == "" { // switch to RGB for all
		cs = model.ColorSpaceRGB
		for i, s := range params.Stops {
			col := s.rgbColor()
			colors[i] = col[:]
		}
	}

	bounds := make([]Fl, len(params.Stops)-2) // here len(params.Stop) >= 2
	for i := range bounds {
		bounds[i] = params.Stops[i+1].Offset
	}
	// we define each subfonction on [0,1]
	encode := make([][2]Fl, len(params.Stops)-1)
	subfunctions := make([]model.FunctionDict, len(params.Stops)-1)
	for i := range encode {
		encode[i] = [2]Fl{0, 1}
		subfunctions[i] = model.FunctionDict{
			Domain: []model.Range{{0, 1}},
			FunctionType: model.FunctionExpInterpolation{
				C0: colors[i],
				C1: colors[i+1],
				N:  1,
			},
		}
	}
	f := model.FunctionDict{
		Domain: []model.Range{{0, 1}},
		FunctionType: model.FunctionStitching{
			Functions: subfunctions,
			Bounds:    bounds,
			Encode:    encode,
		},
	}
	bg := model.BaseGradient{
		Function: []model.FunctionDict{f},
		Extend:   [2]bool{true, true},
	}
	out := &model.ShadingDict{
		ShadingType: model.ShadingAxial{},
		ColorSpace:  cs,
	}
	switch dir := params.Direction.(type) {
	case GradientLinear:
		out.ShadingType = model.ShadingAxial{
			BaseGradient: bg,
			Coords:       dir,
		}
	case GradientRadial:
		out.ShadingType = model.ShadingRadial{
			BaseGradient: bg,
			Coords:       dir,
		}
	}
	return out
}
