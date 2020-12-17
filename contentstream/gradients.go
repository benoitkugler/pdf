package contentstream

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

// GradientPointRGB defines the position and the (RGB) color
// of a gradient step.
type GradientPointRGB struct {
	X, Y    Fl
	R, G, B uint8
}

func rgbToArray(r, g, b uint8) []Fl {
	return []Fl{Fl(r) / 255, Fl(g) / 255, Fl(b) / 255}
}

func newBaseGradient(from, to GradientPointRGB) model.BaseGradient {
	return model.BaseGradient{
		Function: []model.FunctionDict{
			{
				FunctionType: model.FunctionExpInterpolation{
					C0: rgbToArray(from.R, from.G, from.B),
					C1: rgbToArray(to.R, to.G, to.B),
					N:  1, // linear
				},
				Domain: []model.Range{{0, 1}},
			},
		},
		Extend: [2]bool{true, true},
	}
}

// NewLinearGradientRGB builds a linear gradient shading dictionnary,
// usuable for example as arguments of a 'sh' operator.
//
// The vector's origin and destination are specified by
// the points `from` and `to`, expressed in user space units.
// In a linear gradient, blending occurs
// perpendicularly to this vector. Color 1 is used up to the origin of the
// vector and color 2 is used beyond the vector's end point. Between the points
// the colors are gradually blended.
func NewLinearGradientRGB(from, to GradientPointRGB) *model.ShadingDict {
	return &model.ShadingDict{
		ColorSpace: model.ColorSpaceRGB,
		ShadingType: model.ShadingAxial{
			BaseGradient: newBaseGradient(from, to),
			Coords:       [4]Fl{from.X, from.Y, to.X, to.Y},
		},
	}
}

// NewRadialGradientRGB builds a radial gradient shading dictionnary,
// usuable for example as arguments of a 'sh' operator.
//
// Color 1 begins at the origin point specified by `from`. Color 2 begins at the
// circle specified by the center point `to` and `radius`.
func NewRadialGradientRGB(from, to GradientPointRGB, radius Fl) *model.ShadingDict {
	return &model.ShadingDict{
		ColorSpace: model.ColorSpaceRGB,
		ShadingType: model.ShadingRadial{
			BaseGradient: newBaseGradient(from, to),
			Coords:       [6]Fl{from.X, from.Y, 0, to.X, to.Y, radius},
		},
	}
}

// check is the shading is in the resources map or generate a new name and add the shading
func (ap Appearance) addShading(newShading *model.ShadingDict) model.ObjName {
	for name, f := range ap.resources.Shading {
		if f == newShading {
			return name
		}
	}
	// this convention must be respected so that the names are distincts
	name := model.ObjName(fmt.Sprintf("Shading%d", len(ap.resources.Shading)))
	ap.resources.Shading[name] = newShading
	return name
}

// FillShading adds the given shadings to the resources and fill the current path
// using the 'sh' operator.
func (ap *Appearance) FillShading(sh *model.ShadingDict) {
	name := ap.addShading(sh)
	ap.Ops(OpShFill{Shading: name})
}
