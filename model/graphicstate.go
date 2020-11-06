package model

type DashPattern struct {
	Array []uint
	Phase uint
}

type FontStyle struct {
	Font *Font
	Size float64
}

type GraphicState struct {
	LW   float64
	LC   int
	LJ   int
	ML   float64
	D    DashPattern
	RI   Name
	Font FontStyle // font and size
	CA   float64
	Ca   float64 // non-stroking
	AIS  bool
	SM   float64
	SA   bool
}

type SeparationColor struct {
	Name           Name
	AlternateSpace Name
	TintTransform  Function
}

type NameColorSpace string

const (
	DeviceRGB  NameColorSpace = "DeviceRGB"
	DeviceGray NameColorSpace = "DeviceGray"
	DeviceCMYK NameColorSpace = "DeviceCMYK"
)

// ColorSpace is either a Name or a more complex object
type ColorSpace interface {
	isColorSpace()
}

func (NameColorSpace) isColorSpace()  {}
func (SeparationColor) isColorSpace() {}

// ----------------------- Patterns -----------------------

// Pattern is either a Tiling or a Shading pattern
type Pattern interface {
	isPattern()
}

func (Tiling) isPattern()  {}
func (Shading) isPattern() {}

// Tiling is a type 1 pattern
type Tiling struct {
	// TODO:
}

// ShadingType may FunctionBased, Axial, Radial, FreeForm,
// Lattice, Coons, TensorProduct
type ShadingType interface {
	isShading()
}

func (FunctionBased) isShading() {}
func (Axial) isShading()         {}
func (Radial) isShading()        {}
func (FreeForm) isShading()      {}
func (Lattice) isShading()       {}
func (Coons) isShading()         {}
func (TensorProduct) isShading() {}

type FunctionBased struct {
	Domain   *[4]float64 // optional
	Matrix   *Matrix     // optional
	Function Function
}
type Axial struct {
	Coords   [4]float64  // x0, y0, x1, y1
	Domain   *[2]float64 // optional, default to [0 1]
	Function Function    // 1 -> n (m=1)
	Extend   [2]bool     // optional, default to [false, false]
}
type Radial struct {
	Coords   [6]float64  // x0, y0, r0, x1, y1, r1
	Domain   *[2]float64 // optional, default to [0 1]
	Function Function    // 1 -> n (m=1)
	Extend   [2]bool     // optional, default to [false, false]
}
type FreeForm struct{}      // TODO:
type Lattice struct{}       // TODO:
type Coons struct{}         // TODO:
type TensorProduct struct{} // TODO:

// ShadingDict is either a plain dict, or is a stream (+ dict)
type ShadingDict struct {
	ShadingType ShadingType
	ColorSpace  ColorSpace
	Background  []float64  // colour components appropriate to the colour space
	BBox        *Rectangle // in shading’s target coordinate space
	AntiAlias   bool
}

// Shading is a type2 pattern
type Shading struct {
	Shading   ShadingDict
	Matrix    Matrix        // optionnal, default to Identity
	ExtGState *GraphicState // optionnal
}
