package model

import "fmt"

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

// ----------------------- colors spaces -----------------------

// ColorSpace is either a Name or a more complex object
type ColorSpace interface {
	isColorSpace()
}

func (NameColorSpace) isColorSpace()         {}
func (CalGrayColorSpace) isColorSpace()      {}
func (CalRGBColorSpace) isColorSpace()       {}
func (LabColorSpace) isColorSpace()          {}
func (*ICCBasedColorSpace) isColorSpace()    {}
func (SeparationColorSpace) isColorSpace()   {}
func (IndexedColorSpace) isColorSpace()      {}
func (UncoloredTilingPattern) isColorSpace() {}

// SeparationColorSpace is defined in PDF as an array
// [ /Separation name alternateSpace tintTransform ]
type SeparationColorSpace struct {
	Name           Name
	AlternateSpace ColorSpace // may not be another special colour space
	TintTransform  Function   // required, may be an indirect object
}

type NameColorSpace Name

// NewNameColorSpace validate the color space
func NewNameColorSpace(cs string) (NameColorSpace, error) {
	c := NameColorSpace(cs)
	switch c {
	case CSDeviceGray, CSDeviceRGB, CSDeviceCMYK, CSPattern, CSIndexed, CSSeparation, CSDeviceN:
		return c, nil
	default:
		return "", fmt.Errorf("invalid named color space %s", cs)
	}
}

const (
	CSDeviceRGB  NameColorSpace = "DeviceRGB"
	CSDeviceGray NameColorSpace = "DeviceGray"
	CSDeviceCMYK NameColorSpace = "DeviceCMYK"
	CSPattern    NameColorSpace = "Pattern"
	CSIndexed    NameColorSpace = "Indexed"
	CSSeparation NameColorSpace = "Separation"
	CSDeviceN    NameColorSpace = "DeviceN"
)

type CalGrayColorSpace struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Gamma      float64    // optional, default to 1
}

type CalRGBColorSpace struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Gamma      [3]float64 // optional, default to [1 1 1]
	Matrix     [9]float64 // [ X_A Y_A Z_A X_B Y_B Z_B X_C Y_C Z_C ], optional, default to identity
}

type LabColorSpace struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Range      [4]float64 // [ a min a max b min b max ], optional, default to [−100 100 −100 100 ]
}

type ICCBasedColorSpace struct {
	ContentStream

	N         int        // 1, 3 or 4
	Alternate ColorSpace // optional
	Range     []Range    // optional, default to [{0, 1}, ...]
}

type IndexedColorSpace struct {
	Base   ColorSpace
	Hival  uint8
	Lookup ColorTable
}

// ColorTable is either a content stream or a simple byte string
type ColorTable interface {
	isColorTable()
}

func (ColorTableStream) isColorTable() {}
func (ColorTableBytes) isColorTable()  {}

type ColorTableStream ContentStream

type ColorTableBytes []byte

type UncoloredTilingPattern struct {
	UnderlyingColorSpace ColorSpace
}

// ----------------------- Patterns -----------------------

// Pattern is either a Tiling or a Shading pattern
type Pattern interface {
	isPattern()
}

func (*TilingPatern) isPattern()  {}
func (*ShadingPatern) isPattern() {}

// TilingPatern is a type 1 pattern
type TilingPatern struct {
	ContentStream

	PaintType  uint8 // 1 for coloured; 2 for uncoloured
	TilingType uint8 // 1, 2, 3
	BBox       Rectangle
	XStep      float64
	YStep      float64
	Resources  ResourcesDict
	Matrix     Matrix // optional, default to identity
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
	Domain   [4]float64 // optional, default to [0 1 0 1]
	Matrix   Matrix     // optional, default to identity
	Function []Function // either one 2 -> n function, or n 2 -> 1 functions
}
type BaseGradient struct {
	Domain   [2]float64 // optional, default to [0 1]
	Function []Function // either one 1 -> n function, or n 1->1 functions
	Extend   [2]bool    // optional, default to [false, false]
}
type Axial struct {
	BaseGradient
	Coords [4]float64 // x0, y0, x1, y1
}
type Radial struct {
	BaseGradient
	Coords [6]float64 // x0, y0, r0, x1, y1, r1
}
type FreeForm struct{} // TODO:
type Lattice struct{}  // TODO:
type Coons struct {
	ContentStream

	BitsPerCoordinate uint8 // 1, 2, 4, 8, 12, 16, 24, or 32
	BitsPerComponent  uint8 // 1, 2, 4, 8, 12, or 16
	BitsPerFlag       uint8 // 2, 4, or 8
	Decode            []Range
	Function          []Function // a 1->n function or n 1->1 functions (n is the number of colour components)
}
type TensorProduct struct{} // TODO:

// ShadingDict is either a plain dict, or is a stream (+ dict)
type ShadingDict struct {
	ShadingType ShadingType
	ColorSpace  ColorSpace
	// colour components appropriate to the colour space
	// only applied when part of a (shading) pattern
	Background []float64
	BBox       *Rectangle // in shading’s target coordinate space
	AntiAlias  bool
}

// ShadingPatern is a type2 pattern
type ShadingPatern struct {
	Shading   ShadingDict
	Matrix    Matrix        // optionnal, default to Identity
	ExtGState *GraphicState // optionnal
}
