package model

import "fmt"

const Undef = -1

type DashPattern struct {
	Array []float64
	Phase float64
}

func (d DashPattern) String() string {
	return fmt.Sprintf("[%s %.3f]", writeFloatArray(d.Array), d.Phase)
}

type FontStyle struct {
	Font *Font
	Size float64
}

func (f FontStyle) pdfString(pdf pdfWriter) string {
	ref := pdf.addItem(f.Font)
	return fmt.Sprintf("[%s %.3f]", ref, f.Size)
}

type GraphicState struct {
	LW   float64
	LC   int // optional, >= 0, Undef for not specified
	LJ   int // optional, >= 0, Undef for not specified
	ML   float64
	D    *DashPattern // optional
	RI   Name
	Font FontStyle // font and size
	CA   float64   // optional, >= 0, Undef for not specified
	Ca   float64   // non-stroking, optional, >= 0, Undef for not specified
	AIS  bool
	SM   float64
	SA   bool
}

func (g *GraphicState) pdfContent(pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	b.WriteString("<<")
	if g.LW != 0 {
		b.fmt("/LW %.3f", g.LW)
	}
	if g.LC != Undef {
		b.fmt("/LC %d", g.LC)
	}
	if g.LJ != Undef {
		b.fmt("/LJ %d", g.LJ)
	}
	if g.ML != 0 {
		b.fmt("/ML %.3f", g.ML)
	}
	if g.D != nil {
		b.fmt("/D %s", *g.D)
	}
	if g.RI != "" {
		b.fmt("/RI %s", g.RI)
	}
	if g.Font.Font != nil {
		b.fmt("/Font %s", g.Font.pdfString(pdf))
	}
	if g.CA != Undef {
		b.fmt("/CA %.3f", g.CA)
	}
	if g.Ca != Undef {
		b.fmt("/ca %.3f", g.Ca)
	}
	b.fmt("/AIS %v", g.AIS)
	if g.SM != Undef {
		b.fmt("/SM %.3f", g.SM)
	}
	b.fmt("/SA %v", g.SA)
	b.WriteString(">>")
	return b.String(), nil
}

// ----------------------- colors spaces -----------------------

// check conformity with either cachable or directColorSpace

var _ cachable = (*ICCBasedColorSpace)(nil)
var _ directColorSpace = SeparationColorSpace{}
var _ directColorSpace = NameColorSpace("")
var _ directColorSpace = CalGrayColorSpace{}
var _ directColorSpace = CalRGBColorSpace{}
var _ directColorSpace = LabColorSpace{}
var _ directColorSpace = IndexedColorSpace{}
var _ directColorSpace = UncoloredTilingPattern{}

// ColorSpace is either a Name or a more complex object
type ColorSpace interface {
	isColorSpace()
}

type directColorSpace interface {
	pdfString(pdf pdfWriter) string
}

func (*ICCBasedColorSpace) isColorSpace()    {}
func (SeparationColorSpace) isColorSpace()   {}
func (NameColorSpace) isColorSpace()         {}
func (CalGrayColorSpace) isColorSpace()      {}
func (CalRGBColorSpace) isColorSpace()       {}
func (LabColorSpace) isColorSpace()          {}
func (IndexedColorSpace) isColorSpace()      {}
func (UncoloredTilingPattern) isColorSpace() {}

// return either an indirect reference or a direct object
func writeColorSpace(c ColorSpace, pdf pdfWriter) string {
	if c, ok := c.(directColorSpace); ok {
		return c.pdfString(pdf)
	}
	// if it's not direct, it must be cachable
	ca, _ := c.(cachable)
	ref := pdf.addItem(ca)
	return ref.String()
}

type ICCBasedColorSpace struct {
	ContentStream

	N         int        // 1, 3 or 4
	Alternate ColorSpace // optional
	Range     []Range    // optional, default to [{0, 1}, ...]
}

// returns the stream object. `pdf` is used
// to write potential alternate space.
func (c *ICCBasedColorSpace) pdfContent(pdf pdfWriter) (string, []byte) {
	baseArgs := c.PDFCommonFields()
	b := newBuffer()
	b.fmt("<</N %d %s", c.N, baseArgs)
	if c.Alternate != nil {
		alt := writeColorSpace(c.Alternate, pdf)
		b.fmt("/Alternate %s", alt)
	}
	if len(c.Range) != 0 {
		b.fmt("/Range %s", writeRangeArray(c.Range))
	}
	b.fmt(">>")
	return b.String(), c.Content
}

// SeparationColorSpace is defined in PDF as an array
// [ /Separation name alternateSpace tintTransform ]
type SeparationColorSpace struct {
	Name           Name
	AlternateSpace ColorSpace // may not be another special colour space
	TintTransform  Function   // required, may be an indirect object
}

func (s SeparationColorSpace) pdfString(pdf pdfWriter) string {
	cs := writeColorSpace(s.AlternateSpace, pdf)
	funcRef := pdf.addObject(s.TintTransform.pdfContent(pdf))
	return fmt.Sprintf("[/Separation %s %s %s]", s.Name, cs, funcRef)
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

func (n NameColorSpace) pdfString(pdfWriter) string {
	return Name(n).String()
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

func (c CalGrayColorSpace) pdfString(pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]float64{} {
		out += fmt.Sprintf(" /BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Gamma != 0 {
		out += fmt.Sprintf(" /Gamma %.3f", c.Gamma)
	}
	out += ">>"
	return out
}

type CalRGBColorSpace struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Gamma      [3]float64 // optional, default to [1 1 1]
	Matrix     [9]float64 // [ X_A Y_A Z_A X_B Y_B Z_B X_C Y_C Z_C ], optional, default to identity
}

func (c CalRGBColorSpace) pdfString(pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]float64{} {
		out += fmt.Sprintf(" /BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Gamma != [3]float64{} {
		out += fmt.Sprintf(" /Gamma %s", writeFloatArray(c.Gamma[:]))
	}
	if c.Matrix != [9]float64{} {
		out += fmt.Sprintf(" /Matrix %s", writeFloatArray(c.Matrix[:]))
	}
	out += ">>"
	return out
}

type LabColorSpace struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Range      [4]float64 // [ a_min a_max b_min b_max ], optional, default to [−100 100 −100 100 ]
}

func (c LabColorSpace) pdfString(pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]float64{} {
		out += fmt.Sprintf(" /BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Range != [4]float64{} {
		out += fmt.Sprintf(" /Range %s", writeFloatArray(c.Range[:]))
	}
	out += ">>"
	return out
}

// IndexedColorSpace is written in PDF as
// [ /Indexed base hival lookup ]
type IndexedColorSpace struct {
	Base   ColorSpace
	Hival  uint8
	Lookup ColorTable
}

func (c IndexedColorSpace) pdfString(pdf pdfWriter) string {
	base := writeColorSpace(c.Base, pdf)
	var tableString string
	switch table := c.Lookup.(type) {
	case *ColorTableStream:
		ref := pdf.addItem(table)
		tableString = ref.String()
	case ColorTableBytes:
		tableString = string(table)
	}
	return fmt.Sprintf("[/Indexed %s %d %s]", base, c.Hival, tableString)
}

// ColorTable is either a content stream or a simple byte string
type ColorTable interface {
	isColorTable()
}

func (*ColorTableStream) isColorTable() {}
func (ColorTableBytes) isColorTable()   {}

type ColorTableStream ContentStream

// PDFBytes return the content of the stream.
// PDFWriter is not used
func (table *ColorTableStream) pdfContent(pdfWriter) (string, []byte) {
	return (*ContentStream)(table).PDFContent()
}

type ColorTableBytes []byte

// UncoloredTilingPattern is written in PDF
// [ /Pattern underlyingColorSpace ]
type UncoloredTilingPattern struct {
	UnderlyingColorSpace ColorSpace
}

func (c UncoloredTilingPattern) pdfString(pdf pdfWriter) string {
	under := writeColorSpace(c.UnderlyingColorSpace, pdf)
	return fmt.Sprintf("[/Pattern %s]", under)
}

// ----------------------- Patterns -----------------------

// Pattern is either a Tiling or a Shading pattern
type Pattern interface {
	isPattern()
	cachable
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

// TODO: tiling patern
func (t *TilingPatern) pdfContent(pdf pdfWriter) (string, []byte) {
	return "<<>>", nil
}

// ShadingType may FunctionBased, Axial, Radial, FreeForm,
// Lattice, Coons, TensorProduct
type ShadingType interface {
	isShading()
	pdfContent(commonFields string, pdf pdfWriter) (string, []byte)
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

func (s FunctionBased) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	fns := pdf.writeFunctions(s.Function)
	b.fmt("<</ShadingType 1 %s /Function %s", commonFields, writeRefArray(fns))
	if s.Domain != [4]float64{} {
		b.fmt("/Domain %s", writeFloatArray(s.Domain[:]))
	}
	if (s.Matrix != Matrix{}) {
		b.fmt("/Matrix %s", s.Matrix)
	}
	b.fmt(">>")
	return b.String(), nil
}

type BaseGradient struct {
	Domain   [2]float64 // optional, default to [0 1]
	Function []Function // either one 1 -> n function, or n 1->1 functions
	Extend   [2]bool    // optional, default to [false, false]
}

//	return the inner fields, without << >>
// `pdf` is used to write the functions
func (g BaseGradient) pdfString(pdf pdfWriter) string {
	fns := pdf.writeFunctions(g.Function)
	out := fmt.Sprintf("/Function %s", writeRefArray(fns))
	if g.Domain != [2]float64{} {
		out += fmt.Sprintf(" /Domain %s", writeFloatArray(g.Domain[:]))
	}
	if g.Extend != [2]bool{} {
		out += fmt.Sprintf(" /Extend [%v %v]", g.Extend[0], g.Extend[1])
	}
	return out
}

type Axial struct {
	BaseGradient
	Coords [4]float64 // x0, y0, x1, y1
}

func (s Axial) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	gradArgs := s.BaseGradient.pdfString(pdf)
	out := fmt.Sprintf("<</ShadingType 2 %s %s /Coords %s>>",
		commonFields, gradArgs, writeFloatArray(s.Coords[:]))
	return out, nil
}

type Radial struct {
	BaseGradient
	Coords [6]float64 // x0, y0, r0, x1, y1, r1
}

func (s Radial) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	gradArgs := s.BaseGradient.pdfString(pdf)
	out := fmt.Sprintf("<</ShadingType 3 %s %s /Coords %s>>",
		commonFields, gradArgs, writeFloatArray(s.Coords[:]))
	return out, nil
}

type FreeForm struct{} // TODO:
type Lattice struct{}  // TODO:
type Coons struct {
	ContentStream

	BitsPerCoordinate uint8 // 1, 2, 4, 8, 12, 16, 24, or 32
	BitsPerComponent  uint8 // 1, 2, 4, 8, 12, or 16
	BitsPerFlag       uint8 // 2, 4, or 8
	Decode            []Range
	Function          []Function // optional, one 1->n function or n 1->1 functions (n is the number of colour components)
}

func (c Coons) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	args := c.PDFCommonFields()
	b := newBuffer()
	b.fmt("<</ShadingType 6 %s %s /BitsPerCoordinate %d /BitsPerComponent %d /BitsPerFlag %d /Decode %s",
		commonFields, args, c.BitsPerCoordinate, c.BitsPerComponent, c.BitsPerFlag, writeRangeArray(c.Decode))
	if len(c.Function) != 0 {
		fns := pdf.writeFunctions(c.Function)
		b.fmt("/Function %s", writeRefArray(fns))
	}
	b.fmt(">>")
	return b.String(), nil
}

type TensorProduct struct{} // TODO:

// ShadingDict is either a plain dict, or is a stream (+ dict)
type ShadingDict struct {
	ShadingType ShadingType
	ColorSpace  ColorSpace
	// colour components appropriate to the colour space
	// only applied when part of a (shading) pattern
	Background []float64  // optional
	BBox       *Rectangle // optional in shading’s target coordinate space
	AntiAlias  bool       // optional, default to false
}

func (s *ShadingDict) pdfContent(pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	cs := writeColorSpace(s.ColorSpace, pdf)
	b.fmt("/ColorSpace %s", cs)
	if len(s.Background) != 0 {
		b.fmt("/Background %s", writeFloatArray(s.Background))
	}
	if s.BBox != nil {
		b.fmt("/BBox %s", s.BBox.PDFstring())
	}
	b.fmt("/AntiAlias %v", s.AntiAlias)
	return s.ShadingType.pdfContent(b.String(), pdf)
}

// ShadingPatern is a type2 pattern
type ShadingPatern struct {
	Shading   *ShadingDict  // required
	Matrix    Matrix        // optionnal, default to Identity
	ExtGState *GraphicState // optionnal
}

func (s *ShadingPatern) pdfContent(pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	shadingRef := pdf.addItem(s.Shading)
	b.fmt("<</PatternType 2 /Shading %s", shadingRef)
	if s.Matrix != (Matrix{}) {
		b.fmt("/Matrix %s", s.Matrix)
	}
	if s.ExtGState != nil {
		stateRef := pdf.addItem(s.ExtGState)
		b.fmt("/ExtGState %s", stateRef)
	}
	b.fmt(">>")
	return b.String(), nil
}
