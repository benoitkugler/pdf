package model

import "fmt"

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
	LC   MaybeInt // optional, >= 0
	LJ   MaybeInt // optional, >= 0
	ML   float64
	D    *DashPattern // optional
	RI   Name
	Font FontStyle  // font and size
	CA   MaybeFloat // optional, >= 0
	Ca   MaybeFloat // non-stroking, optional, >= 0
	AIS  bool
	SM   MaybeFloat // optional
	SA   bool
}

func (g *GraphicState) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	b := newBuffer()
	b.WriteString("<<")
	if g.LW != 0 {
		b.fmt("/LW %.3f", g.LW)
	}
	if g.LC != nil {
		b.fmt("/LC %d", g.LC.(Int))
	}
	if g.LJ != nil {
		b.fmt("/LJ %d", g.LJ.(Int))
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
	if g.CA != nil {
		b.fmt("/CA %.3f", g.CA.(Float))
	}
	if g.Ca != nil {
		b.fmt("/ca %.3f", g.Ca.(Float))
	}
	b.fmt("/AIS %v", g.AIS)
	if g.SM != nil {
		b.fmt("/SM %.3f", g.SM.(Float))
	}
	b.fmt("/SA %v", g.SA)
	b.WriteString(">>")
	return b.String(), nil
}

func (g *GraphicState) clone(cache cloneCache) Referenceable {
	if g == nil {
		return g
	}
	out := *g // shallow copy
	out.D = g.D.Clone()
	out.Font = g.Font.clone(cache)
	return &out
}

// ----------------------- Patterns -----------------------

// Pattern is either a Tiling or a Shading pattern
type Pattern interface {
	isPattern()
	Referenceable
}

func (*PatternTiling) isPattern()  {}
func (*PatternShading) isPattern() {}

// PatternTiling is a type 1 pattern
type PatternTiling struct {
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
func (t *PatternTiling) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	return "<<>>", nil
}

func (t *PatternTiling) clone(cache cloneCache) Referenceable {
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
	shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte)
	Clone() Shading
}

type ShadingFunctionBased struct {
	Domain   [4]float64     // optional, default to [0 1 0 1]
	Matrix   Matrix         // optional, default to identity
	Function []FunctionDict // either one 2 -> n function, or n 2 -> 1 functions
}

func (s ShadingFunctionBased) shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	fns := pdf.writeFunctions(s.Function)
	b.fmt("<</ShadingType 1 %s/Function %s", shadingFields, writeRefArray(fns))
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

func (s ShadingAxial) shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte) {
	gradArgs := s.BaseGradient.pdfString(pdf)
	out := fmt.Sprintf("<</ShadingType 2 %s %s/Coords %s>>",
		shadingFields, gradArgs, writeFloatArray(s.Coords[:]))
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

func (s ShadingRadial) shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte) {
	gradArgs := s.BaseGradient.pdfString(pdf)
	out := fmt.Sprintf("<</ShadingType 3 %s %s/Coords %s>>",
		shadingFields, gradArgs, writeFloatArray(s.Coords[:]))
	return out, nil
}

// Clone returns a deep copy with concrete type `Radial`
func (f ShadingRadial) Clone() Shading {
	out := f
	out.BaseGradient = f.BaseGradient.Clone()
	return out
}

// ShadingStream is the base type shared by 4 to 7 shadings type
type ShadingStream struct {
	Stream

	BitsPerCoordinate uint8 // 1, 2, 4, 8, 12, 16, 24, or 32
	BitsPerComponent  uint8 // 1, 2, 4, 8, 12, or 16
	Decode            [][2]float64
	Function          []FunctionDict // optional, one 1->n function or n 1->1 functions (n is the number of colour components)

}

// Clone returns a deep copy
func (ss ShadingStream) Clone() ShadingStream {
	out := ss
	out.Stream = ss.Stream.Clone()
	out.Decode = append([][2]float64(nil), ss.Decode...)
	if ss.Function != nil { // to preserve reflect.DeepEqual
		out.Function = make([]FunctionDict, len(ss.Function))
	}
	for i, f := range ss.Function {
		out.Function[i] = f.Clone()
	}
	return out
}

// return the shared dict attributes
func (ss ShadingStream) pdfFields(shadingFields string, pdf pdfWriter, type_ uint8) (string, []byte) {
	args := ss.PDFCommonFields()
	b := newBuffer()
	b.fmt("/ShadingType %d %s %s/BitsPerCoordinate %d/BitsPerComponent %d/Decode %s",
		type_, shadingFields, args, ss.BitsPerCoordinate,
		ss.BitsPerComponent, writePointsArray(ss.Decode))
	if len(ss.Function) != 0 {
		fns := pdf.writeFunctions(ss.Function)
		b.fmt("/Function %s", writeRefArray(fns))
	}
	return b.String(), ss.Content
}

type ShadingFreeForm struct {
	ShadingStream
	BitsPerFlag uint8 // 2, 4, or 8
}

// method shared with ShadingCoons and ShadingTensorProduct
// type_ is 4, 6 or 7
func (c ShadingFreeForm) pdfContentExt(shadingFields string, pdf pdfWriter, type_ uint8) (string, []byte) {
	sharedField, content := c.ShadingStream.pdfFields(shadingFields, pdf, type_)
	return fmt.Sprintf("<<%s /BitsPerFlag %d>>", sharedField, c.BitsPerFlag), content
}

func (c ShadingFreeForm) shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte) {
	return c.pdfContentExt(shadingFields, pdf, 4)
}

// Clone returns a deep copy with concrete type `ShadingFreeForm`
func (co ShadingFreeForm) Clone() Shading {
	out := co
	out.ShadingStream = co.ShadingStream.Clone()
	return out
}

type ShadingLattice struct {
	ShadingStream
	VerticesPerRow int // required
}

func (c ShadingLattice) shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte) {
	sharedField, content := c.ShadingStream.pdfFields(shadingFields, pdf, 5)
	return fmt.Sprintf("<<%s /VerticesPerRow %d>>", sharedField, c.VerticesPerRow), content
}

// Clone returns a deep copy with concrete type `ShadingLattice`
func (co ShadingLattice) Clone() Shading {
	out := co
	out.ShadingStream = co.ShadingStream.Clone()
	return out
}

type ShadingCoons ShadingFreeForm

func (c ShadingCoons) shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte) {
	return ShadingFreeForm(c).pdfContentExt(shadingFields, pdf, 6)
}

// Clone returns a deep copy with concrete type `ShadingCoons`
func (co ShadingCoons) Clone() Shading {
	return ShadingCoons(ShadingFreeForm(co).Clone().(ShadingFreeForm))
}

type ShadingTensorProduct ShadingFreeForm

func (c ShadingTensorProduct) shadingPDFContent(shadingFields string, pdf pdfWriter) (string, []byte) {
	return ShadingFreeForm(c).pdfContentExt(shadingFields, pdf, 7)
}

// Clone returns a deep copy with concrete type `ShadingTensorProduct`
func (co ShadingTensorProduct) Clone() Shading {
	return ShadingTensorProduct(ShadingFreeForm(co).Clone().(ShadingFreeForm))
}

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
	if s.ColorSpace != nil {
		b.fmt("/ColorSpace %s", s.ColorSpace.colorSpacePDFString(pdf))
	}
	if len(s.Background) != 0 {
		b.fmt("/Background %s", writeFloatArray(s.Background))
	}
	if s.BBox != nil {
		b.fmt("/BBox %s", s.BBox.String())
	}
	b.fmt("/AntiAlias %v", s.AntiAlias)
	return s.ShadingType.shadingPDFContent(b.String(), pdf)
}

func (s *ShadingDict) clone(cache cloneCache) Referenceable {
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

// PatternShading is a type2 pattern
type PatternShading struct {
	Shading   *ShadingDict  // required
	Matrix    Matrix        // optionnal, default to Identity
	ExtGState *GraphicState // optionnal
}

func (s *PatternShading) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
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

func (s *PatternShading) clone(cache cloneCache) Referenceable {
	if s == nil {
		return s
	}
	out := *s
	out.Shading = cache.checkOrClone(s.Shading).(*ShadingDict)
	out.ExtGState = cache.checkOrClone(s.ExtGState).(*GraphicState)
	return &out
}
