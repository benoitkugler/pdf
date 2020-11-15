package model

import "fmt"

const Undef = -122

type DashPattern struct {
	Array []float64
	Phase float64
}

func (d DashPattern) String() string {
	return fmt.Sprintf("[%s %.3f]", writeFloatArray(d.Array), d.Phase)
}

// Clone returns a deep copy
func (d *DashPattern) Clone() *DashPattern {
	if d == nil {
		return nil
	}
	out := *d
	out.Array = append([]float64(nil), d.Array...)
	return &out
}

type FontStyle struct {
	Font *FontDict
	Size float64
}

func (f FontStyle) pdfString(pdf pdfWriter) string {
	ref := pdf.addItem(f.Font)
	return fmt.Sprintf("[%s %.3f]", ref, f.Size)
}

func (f FontStyle) clone(cache cloneCache) FontStyle {
	out := f
	out.Font = cache.checkOrClone(f.Font).(*FontDict)
	return out
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

func (g *GraphicState) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
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

func (g *GraphicState) clone(cache cloneCache) Referencable {
	if g == nil {
		return g
	}
	out := *g // shallow copy
	out.D = g.D.Clone()
	out.Font = g.Font.clone(cache)
	return &out
}

// ----------------------- colors spaces -----------------------

// check conformity with either Referencable or directColorSpace

var _ Referencable = (*ColorSpaceICCBased)(nil)
var _ directColorSpace = ColorSpaceSeparation{}
var _ directColorSpace = ColorSpaceName("")
var _ directColorSpace = ColorSpaceCalGray{}
var _ directColorSpace = ColorSpaceCalRGB{}
var _ directColorSpace = ColorSpaceLab{}
var _ directColorSpace = ColorSpaceIndexed{}
var _ directColorSpace = ColorSpaceUncoloredPattern{}

// check conformity with the ColorSpace interface

var _ ColorSpace = (*ColorSpaceICCBased)(nil)

// ColorSpace is either a Name or a more complex object
type ColorSpace interface {
	isColorSpace()
}

type directColorSpace interface {
	ColorSpace
	// returns a deep copy, preserving the concrete type
	cloneCS(cloneCache) ColorSpace
	pdfString(pdf pdfWriter) string
}

func (*ColorSpaceICCBased) isColorSpace()        {}
func (ColorSpaceSeparation) isColorSpace()       {}
func (ColorSpaceName) isColorSpace()             {}
func (ColorSpaceCalGray) isColorSpace()          {}
func (ColorSpaceCalRGB) isColorSpace()           {}
func (ColorSpaceLab) isColorSpace()              {}
func (ColorSpaceIndexed) isColorSpace()          {}
func (ColorSpaceUncoloredPattern) isColorSpace() {}

// return either an indirect reference or a direct object
func writeColorSpace(c ColorSpace, pdf pdfWriter) string {
	if c, ok := c.(directColorSpace); ok {
		return c.pdfString(pdf)
	}
	// if it's not direct, it must be Referencable
	ca, _ := c.(Referencable)
	ref := pdf.addItem(ca)
	return ref.String()
}

// c may be nil
func cloneColorSpace(c ColorSpace, cache cloneCache) ColorSpace {
	if c == nil {
		return nil
	}
	if c, ok := c.(directColorSpace); ok {
		return c.cloneCS(cache)
	}
	// if it's not direct, it must be Referencable
	refe, _ := c.(Referencable)
	return cache.checkOrClone(refe).(ColorSpace)
}

type ColorSpaceICCBased struct {
	Stream

	N         int        // 1, 3 or 4
	Alternate ColorSpace // optional
	Range     []Range    // optional, default to [{0, 1}, ...]
}

// returns the stream object. `pdf` is used
// to write potential alternate space.
func (c *ColorSpaceICCBased) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
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

func (cs *ColorSpaceICCBased) clone(cache cloneCache) Referencable {
	if cs == nil {
		return cs
	}
	out := *cs
	out.Stream = cs.Stream.Clone()
	out.Range = append([]Range(nil), cs.Range...)
	if cs.Alternate != nil {
		out.Alternate = cloneColorSpace(cs.Alternate, cache)
	}
	return &out
}

// ColorSpaceSeparation is defined in PDF as an array
// [/Separation name alternateSpace tintTransform ]
type ColorSpaceSeparation struct {
	Name           Name
	AlternateSpace ColorSpace   // required may not be another special colour space
	TintTransform  FunctionDict // required, may be an indirect object
}

func (s ColorSpaceSeparation) pdfString(pdf pdfWriter) string {
	cs := writeColorSpace(s.AlternateSpace, pdf)
	funcRef := pdf.addObject(s.TintTransform.pdfContent(pdf))
	return fmt.Sprintf("[/Separation %s %s %s]", s.Name, cs, funcRef)
}

func (s ColorSpaceSeparation) cloneCS(cache cloneCache) ColorSpace {
	out := s
	if s.AlternateSpace != nil {
		out.AlternateSpace = cloneColorSpace(s.AlternateSpace, cache)
	}
	out.TintTransform = s.TintTransform.Clone()
	return out
}

const (
	CSDeviceRGB  ColorSpaceName = "DeviceRGB"
	CSDeviceGray ColorSpaceName = "DeviceGray"
	CSDeviceCMYK ColorSpaceName = "DeviceCMYK"
	CSPattern    ColorSpaceName = "Pattern"
	CSIndexed    ColorSpaceName = "Indexed"
	CSSeparation ColorSpaceName = "Separation"
	CSDeviceN    ColorSpaceName = "DeviceN"
)

type ColorSpaceName Name

// NewNameColorSpace validate the color space
func NewNameColorSpace(cs string) (ColorSpaceName, error) {
	c := ColorSpaceName(cs)
	switch c {
	case CSDeviceGray, CSDeviceRGB, CSDeviceCMYK, CSPattern, CSIndexed, CSSeparation, CSDeviceN:
		return c, nil
	default:
		return "", fmt.Errorf("invalid named color space %s", cs)
	}
}

func (n ColorSpaceName) pdfString(pdfWriter) string {
	return Name(n).String()
}

func (n ColorSpaceName) cloneCS(cloneCache) ColorSpace { return n }

type ColorSpaceCalGray struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Gamma      float64    // optional, default to 1
}

func (c ColorSpaceCalGray) pdfString(pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]float64{} {
		out += fmt.Sprintf("/BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Gamma != 0 {
		out += fmt.Sprintf("/Gamma %.3f", c.Gamma)
	}
	out += ">>"
	return out
}

func (c ColorSpaceCalGray) cloneCS(cloneCache) ColorSpace { return c }

type ColorSpaceCalRGB struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Gamma      [3]float64 // optional, default to [1 1 1]
	Matrix     [9]float64 // [ X_A Y_A Z_A X_B Y_B Z_B X_C Y_C Z_C ], optional, default to identity
}

func (c ColorSpaceCalRGB) pdfString(pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]float64{} {
		out += fmt.Sprintf("/BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Gamma != [3]float64{} {
		out += fmt.Sprintf("/Gamma %s", writeFloatArray(c.Gamma[:]))
	}
	if c.Matrix != [9]float64{} {
		out += fmt.Sprintf("/Matrix %s", writeFloatArray(c.Matrix[:]))
	}
	out += ">>"
	return out
}

func (c ColorSpaceCalRGB) cloneCS(cloneCache) ColorSpace { return c }

type ColorSpaceLab struct {
	WhitePoint [3]float64
	BlackPoint [3]float64 // optional, default to [0 0 0]
	Range      [4]float64 // [ a_min a_max b_min b_max ], optional, default to [−100 100 −100 100 ]
}

func (c ColorSpaceLab) pdfString(pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]float64{} {
		out += fmt.Sprintf("/BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Range != [4]float64{} {
		out += fmt.Sprintf("/Range %s", writeFloatArray(c.Range[:]))
	}
	out += ">>"
	return out
}

func (c ColorSpaceLab) cloneCS(cloneCache) ColorSpace { return c }

// ColorSpaceIndexed is written in PDF as
// [/Indexed base hival lookup ]
type ColorSpaceIndexed struct {
	Base   ColorSpace // required
	Hival  uint8
	Lookup ColorTable
}

func (c ColorSpaceIndexed) pdfString(pdf pdfWriter) string {
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

func (c ColorSpaceIndexed) cloneCS(cache cloneCache) ColorSpace {
	out := c
	out.Base = cloneColorSpace(c.Base, cache)
	switch l := c.Lookup.(type) {
	case ColorTableBytes:
		out.Lookup = append(ColorTableBytes(nil), l...)
	case *ColorTableStream:
		out.Lookup = cache.checkOrClone(l).(*ColorTableStream)
	}
	return out
}

// ColorTable is either a content stream or a simple byte string
type ColorTable interface {
	isColorTable()
}

func (*ColorTableStream) isColorTable() {}
func (ColorTableBytes) isColorTable()   {}

type ColorTableStream Stream

// pdfContent return the content of the stream.
func (table *ColorTableStream) pdfContent(pdfWriter, Reference) (string, []byte) {
	return (*Stream)(table).PDFContent()
}

func (table *ColorTableStream) clone(cloneCache) Referencable {
	if table == nil {
		return table
	}
	out := ColorTableStream((*Stream)(table).Clone())
	return &out
}

type ColorTableBytes []byte

// ColorSpaceUncoloredPattern is written in PDF
// [/Pattern underlyingColorSpace ]
type ColorSpaceUncoloredPattern struct {
	UnderlyingColorSpace ColorSpace // required
}

func (c ColorSpaceUncoloredPattern) pdfString(pdf pdfWriter) string {
	under := writeColorSpace(c.UnderlyingColorSpace, pdf)
	return fmt.Sprintf("[/Pattern %s]", under)
}

func (c ColorSpaceUncoloredPattern) cloneCS(cache cloneCache) ColorSpace {
	return ColorSpaceUncoloredPattern{UnderlyingColorSpace: cloneColorSpace(c.UnderlyingColorSpace, cache)}
}

// ----------------------- Patterns -----------------------

// Pattern is either a Tiling or a Shading pattern
type Pattern interface {
	isPattern()
	Referencable
}

func (*PaternTiling) isPattern()  {}
func (*PaternShading) isPattern() {}

// PaternTiling is a type 1 pattern
type PaternTiling struct {
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
func (t *PaternTiling) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	return "<<>>", nil
}

func (t *PaternTiling) clone(cache cloneCache) Referencable {
	if t == nil {
		return t
	}
	out := *t
	out.ContentStream = t.ContentStream.Clone()
	out.Resources = t.Resources.clone(cache)
	return &out
}

// Shading may FunctionBased, Axial, Radial, FreeForm,
// Lattice, Coons, TensorProduct
type Shading interface {
	isShading()
	pdfContent(commonFields string, pdf pdfWriter) (string, []byte)
	Clone() Shading
}

func (ShadingFunctionBased) isShading() {}
func (ShadingAxial) isShading()         {}
func (ShadingRadial) isShading()        {}
func (ShadingFreeForm) isShading()      {}
func (ShadingLattice) isShading()       {}
func (ShadingCoons) isShading()         {}
func (ShadingTensorProduct) isShading() {}

type ShadingFunctionBased struct {
	Domain   [4]float64     // optional, default to [0 1 0 1]
	Matrix   Matrix         // optional, default to identity
	Function []FunctionDict // either one 2 -> n function, or n 2 -> 1 functions
}

func (s ShadingFunctionBased) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	fns := pdf.writeFunctions(s.Function)
	b.fmt("<</ShadingType 1 %s/Function %s", commonFields, writeRefArray(fns))
	if s.Domain != [4]float64{} {
		b.fmt("/Domain %s", writeFloatArray(s.Domain[:]))
	}
	if (s.Matrix != Matrix{}) {
		b.fmt("/Matrix %s", s.Matrix)
	}
	b.fmt(">>")
	return b.String(), nil
}

// Clone returns a deep copy with concrete type `FunctionBased`
func (f ShadingFunctionBased) Clone() Shading {
	out := f
	out.Function = append([]FunctionDict(nil), f.Function...)
	return out
}

type BaseGradient struct {
	Domain   [2]float64     // optional, default to [0 1]
	Function []FunctionDict // either one 1 -> n function, or n 1->1 functions
	Extend   [2]bool        // optional, default to [false, false]
}

//	return the inner fields, without << >>
// `pdf` is used to write the functions
func (g BaseGradient) pdfString(pdf pdfWriter) string {
	fns := pdf.writeFunctions(g.Function)
	out := fmt.Sprintf("/Function %s", writeRefArray(fns))
	if g.Domain != [2]float64{} {
		out += fmt.Sprintf("/Domain %s", writeFloatArray(g.Domain[:]))
	}
	if g.Extend != [2]bool{} {
		out += fmt.Sprintf("/Extend [%v %v]", g.Extend[0], g.Extend[1])
	}
	return out
}

// Clone returns a deep copy
func (b BaseGradient) Clone() BaseGradient {
	out := b
	if b.Function != nil { // to preserve reflect.DeepEqual
		out.Function = make([]FunctionDict, len(b.Function))
	}
	for i, f := range b.Function {
		out.Function[i] = f.Clone()
	}
	return out
}

type ShadingAxial struct {
	BaseGradient
	Coords [4]float64 // x0, y0, x1, y1
}

func (s ShadingAxial) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	gradArgs := s.BaseGradient.pdfString(pdf)
	out := fmt.Sprintf("<</ShadingType 2 %s %s/Coords %s>>",
		commonFields, gradArgs, writeFloatArray(s.Coords[:]))
	return out, nil
}

// Clone returns a deep copy with concrete type `Axial`
func (f ShadingAxial) Clone() Shading {
	out := f
	out.BaseGradient = f.BaseGradient.Clone()
	return out
}

type ShadingRadial struct {
	BaseGradient
	Coords [6]float64 // x0, y0, r0, x1, y1, r1
}

func (s ShadingRadial) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	gradArgs := s.BaseGradient.pdfString(pdf)
	out := fmt.Sprintf("<</ShadingType 3 %s %s/Coords %s>>",
		commonFields, gradArgs, writeFloatArray(s.Coords[:]))
	return out, nil
}

// Clone returns a deep copy with concrete type `Radial`
func (f ShadingRadial) Clone() Shading {
	out := f
	out.BaseGradient = f.BaseGradient.Clone()
	return out
}

type ShadingFreeForm struct{} // TODO:
type ShadingLattice struct{}  // TODO:

type ShadingCoons struct {
	Stream

	BitsPerCoordinate uint8 // 1, 2, 4, 8, 12, 16, 24, or 32
	BitsPerComponent  uint8 // 1, 2, 4, 8, 12, or 16
	BitsPerFlag       uint8 // 2, 4, or 8
	Decode            []Range
	Function          []FunctionDict // optional, one 1->n function or n 1->1 functions (n is the number of colour components)
}

func (c ShadingCoons) pdfContent(commonFields string, pdf pdfWriter) (string, []byte) {
	args := c.PDFCommonFields()
	b := newBuffer()
	b.fmt("<</ShadingType 6 %s %s/BitsPerCoordinate %d/BitsPerComponent %d/BitsPerFlag %d/Decode %s",
		commonFields, args, c.BitsPerCoordinate, c.BitsPerComponent, c.BitsPerFlag, writeRangeArray(c.Decode))
	if len(c.Function) != 0 {
		fns := pdf.writeFunctions(c.Function)
		b.fmt("/Function %s", writeRefArray(fns))
	}
	b.fmt(">>")
	return b.String(), nil
}

// Clone returns a deep copy with concrete type `Coons`
func (co ShadingCoons) Clone() Shading {
	out := co
	out.Stream = co.Stream.Clone()
	out.Decode = append([]Range(nil), co.Decode...)
	if co.Function != nil { // to preserve reflect.DeepEqual
		out.Function = make([]FunctionDict, len(co.Function))
	}
	for i, f := range co.Function {
		out.Function[i] = f.Clone()
	}
	return out
}

type ShadingTensorProduct struct{} // TODO:

// ShadingDict is either a plain dict, or is a stream (+ dict)
type ShadingDict struct {
	ShadingType Shading
	ColorSpace  ColorSpace // required
	// colour components appropriate to the colour space
	// only applied when part of a (shading) pattern
	Background []float64  // optional
	BBox       *Rectangle // optional in shading’s target coordinate space
	AntiAlias  bool       // optional, default to false
}

func (s *ShadingDict) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	b := newBuffer()
	cs := writeColorSpace(s.ColorSpace, pdf)
	b.fmt("/ColorSpace %s", cs)
	if len(s.Background) != 0 {
		b.fmt("/Background %s", writeFloatArray(s.Background))
	}
	if s.BBox != nil {
		b.fmt("/BBox %s", s.BBox.String())
	}
	b.fmt("/AntiAlias %v", s.AntiAlias)
	return s.ShadingType.pdfContent(b.String(), pdf)
}

func (s *ShadingDict) clone(cache cloneCache) Referencable {
	if s == nil {
		return s
	}
	out := *s // shallow copy
	if s.ShadingType != nil {
		out.ShadingType = s.ShadingType.Clone()
	}
	out.ColorSpace = cloneColorSpace(s.ColorSpace, cache)
	out.Background = append([]float64(nil), s.Background...)
	if s.BBox != nil {
		bbox := *s.BBox
		out.BBox = &bbox
	}
	return &out
}

// PaternShading is a type2 pattern
type PaternShading struct {
	Shading   *ShadingDict  // required
	Matrix    Matrix        // optionnal, default to Identity
	ExtGState *GraphicState // optionnal
}

func (s *PaternShading) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	b := newBuffer()
	shadingRef := pdf.addItem(s.Shading)
	b.fmt("<</PatternType 2/Shading %s", shadingRef)
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

func (s *PaternShading) clone(cache cloneCache) Referencable {
	if s == nil {
		return s
	}
	out := *s
	out.Shading = cache.checkOrClone(s.Shading).(*ShadingDict)
	out.ExtGState = cache.checkOrClone(s.ExtGState).(*GraphicState)
	return &out
}
