package model

import (
	"fmt"
	"strconv"
)

type DashPattern struct {
	Array []Fl
	Phase Fl
}

// String returns a description as a PDF array.
func (d DashPattern) String() string {
	return fmt.Sprintf("[%s %s]", writeFloatArray(d.Array), FmtFloat(d.Phase))
}

// Clone returns a deep copy
func (d *DashPattern) Clone() *DashPattern {
	if d == nil {
		return nil
	}
	out := *d
	out.Array = append([]Fl(nil), d.Array...)
	return &out
}

type FontStyle struct {
	Font *FontDict
	Size Fl
}

func (f FontStyle) pdfString(pdf pdfWriter) string {
	ref := pdf.addItem(f.Font)
	return fmt.Sprintf("[%s %s]", ref, FmtFloat(f.Size))
}

func (f FontStyle) clone(cache cloneCache) FontStyle {
	out := f
	out.Font = cache.checkOrClone(f.Font).(*FontDict)
	return out
}

var _ XObject = (*XObjectTransparencyGroup)(nil)

// XObjectTransparencyGroup is a sequence of consecutive objects in a transparency stack that shall be collected
// together and composited to produce a single colour, shape, and opacity at each point. The result shall then be
// treated as if it were a single object for subsequent compositing operations. Groups may be nested within other
// groups to form a tree-structured group hierarchy.
// See Table 147 – Additional entries specific to a transparency group attributes dictionary
type XObjectTransparencyGroup struct {
	XObjectForm

	// the followings are written in PDF under a /Group dict.
	CS ColorSpace
	I  bool // optional, default value: false
	K  bool // optional, default value: false
}

func (tg *XObjectTransparencyGroup) pdfContent(pdf pdfWriter, ref Reference) (StreamHeader, string, []byte) {
	base := tg.XObjectForm.commonFields(pdf, ref)
	gDict := "<</Type/Group /S/Transparency"
	if tg.CS != nil {
		gDict += "/CS " + tg.CS.colorSpaceWrite(pdf, ref)
	}
	if tg.I {
		gDict += "/I true"
	}
	if tg.K {
		gDict += "/K true"
	}
	gDict += ">>"
	base.Fields["Group"] = gDict
	return base, "", tg.Content
}

func (tg *XObjectTransparencyGroup) clone(cache cloneCache) Referenceable {
	if tg == nil {
		return tg
	}
	out := *tg
	out.XObjectForm = *(tg.XObjectForm.clone(cache).(*XObjectForm))
	out.CS = cloneColorSpace(tg.CS, cache)
	return &out
}

// FunctionTransfer is either the name /Identity
// or FunctionDict (1 -> 1)
type FunctionTransfer interface {
	functionTransferPDFString(pdf pdfWriter) string
	cloneFunctionTransfer(cache cloneCache) FunctionTransfer
}

func (n Name) functionTransferPDFString(pdf pdfWriter) string    { return n.String() }
func (n Name) cloneFunctionTransfer(cloneCache) FunctionTransfer { return n }

func (f *FunctionDict) functionTransferPDFString(pdf pdfWriter) string {
	ref := pdf.addItem(f)
	return ref.String()
}

func (f *FunctionDict) cloneFunctionTransfer(cache cloneCache) FunctionTransfer {
	return cache.checkOrClone(f).(*FunctionDict)
}

// SoftMaskDict
// See Table 144 – Entries in a soft-mask dictionary
// In addition, we use the following convention:
//	- S == "" means 'nil' (not specified)
//	- S == /None means the name None
//	- other value means normal dictionary
type SoftMaskDict struct {
	S  Name                      // required
	G  *XObjectTransparencyGroup // required
	BC []Fl                      // optional, length: number of color components
	TR FunctionTransfer          // optional
}

func (s SoftMaskDict) pdfString(pdf pdfWriter) string {
	if s.S == "None" {
		return s.S.String()
	}
	out := "<</S" + s.S.String()
	if s.G != nil {
		ref := pdf.addItem(s.G)
		out += "/G " + ref.String()
	}
	if len(s.BC) != 0 {
		out += "/BC " + writeFloatArray(s.BC)
	}
	if s.TR != nil {
		out += "/TR " + s.TR.functionTransferPDFString(pdf)
	}
	out += ">>"
	return out
}

func (s SoftMaskDict) clone(cache cloneCache) SoftMaskDict {
	out := s
	out.G = cache.checkOrClone(s.G).(*XObjectTransparencyGroup)
	out.BC = append([]Fl(nil), s.BC...)
	if s.TR != nil {
		out.TR = s.TR.cloneFunctionTransfer(cache)
	}
	return out
}

// GraphicState precises parameters in the graphics state.
// See Table 58 – Entries in a Graphics State Parameter Dictionary
// TODO: The following entries are not yet supported:
//	- OP, op, OPM
//	- BG, BG2, UCR, UCR2, TR, TR2
//	- HT
//	- FL
//	- TK
type GraphicState struct {
	LW   Fl
	LC   MaybeInt // optional, >= 0
	LJ   MaybeInt // optional, >= 0
	ML   Fl
	D    *DashPattern // optional
	RI   Name
	Font FontStyle  // font and size
	SM   MaybeFloat // optional
	SA   bool
	// Blend mode
	// See Table 136 – Standard separable blend modes
	// and Table 137 – Standard nonseparable blend modes
	BM    []Name       // 1-element array are written in PDF as a singlename
	SMask SoftMaskDict // optional
	CA    MaybeFloat   // stroking, optional, >= 0
	Ca    MaybeFloat   // non-stroking, optional, >= 0
	AIS   bool
}

func (g *GraphicState) pdfContent(pdf pdfWriter, _ Reference) (StreamHeader, string, []byte) {
	b := newBuffer()
	b.WriteString("<<")
	if g.LW != 0 {
		b.fmt("/LW %s", FmtFloat(g.LW))
	}
	if g.LC != nil {
		b.fmt("/LC %d", g.LC.(ObjInt))
	}
	if g.LJ != nil {
		b.fmt("/LJ %d", g.LJ.(ObjInt))
	}
	if g.ML != 0 {
		b.fmt("/ML %s", FmtFloat(g.ML))
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
	if g.SM != nil {
		b.fmt("/SM %s", FmtFloat(Fl(g.SM.(ObjFloat))))
	}
	if g.SA {
		b.fmt("/SA %v", g.SA)
	}
	if len(g.BM) == 1 {
		b.WriteString("/BM " + g.BM[0].String())
	} else if len(g.BM) > 1 {
		b.WriteString("/BM " + writeNameArray(g.BM))
	}
	if g.SMask.S != "" {
		b.WriteString("/SMask " + g.SMask.pdfString(pdf))
	}
	if g.CA != nil {
		b.fmt("/CA %s", FmtFloat(Fl(g.CA.(ObjFloat))))
	}
	if g.Ca != nil {
		b.fmt("/ca %s", FmtFloat(Fl(g.Ca.(ObjFloat))))
	}
	if g.AIS {
		b.fmt("/AIS %v", g.AIS)
	}
	b.WriteString(">>")
	return StreamHeader{}, b.String(), nil
}

func (g *GraphicState) clone(cache cloneCache) Referenceable {
	if g == nil {
		return g
	}
	out := *g // shallow copy
	out.D = g.D.Clone()
	out.Font = g.Font.clone(cache)
	out.BM = append([]Name(nil), g.BM...)
	out.SMask = g.SMask.clone(cache)
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
	XStep      Fl
	YStep      Fl
	Resources  ResourcesDict
	Matrix     Matrix // optional, default to identity
}

func (t *PatternTiling) pdfContent(pdf pdfWriter, ref Reference) (StreamHeader, string, []byte) {
	out := t.ContentStream.PDFCommonFields(true)
	out.Fields["PatternType"] = "1"
	out.Fields["PaintType"] = strconv.Itoa(int(t.PaintType))
	out.Fields["TilingType"] = strconv.Itoa(int(t.TilingType))
	out.Fields["BBox"] = t.BBox.String()
	out.Fields["XStep"] = ObjFloat(t.XStep).Write(nil, 0)
	out.Fields["YStep"] = ObjFloat(t.YStep).Write(nil, 0)
	out.Fields["Resources"] = t.Resources.pdfString(pdf, ref)
	if t.Matrix != (Matrix{}) {
		out.Fields["Matrix"] = t.Matrix.String()
	}
	return out, "", t.Content
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

// Shading is the kind of shading and
// may be FunctionBased, Axial, Radial, FreeForm,
// Lattice, Coons, TensorProduct
type Shading interface {
	// update `shadingFields` and returns the content stream
	shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte)
	Clone() Shading
}

type ShadingFunctionBased struct {
	Domain   [4]Fl          // optional, default to [0 1 0 1]
	Matrix   Matrix         // optional, default to identity
	Function []FunctionDict // either one 2 -> n function, or n 2 -> 1 functions
}

func (s ShadingFunctionBased) shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte) {
	fns := pdf.writeFunctions(s.Function)
	shadingFields["ShadingType"] = "1"
	shadingFields["Function"] = writeRefArray(fns)
	if s.Domain != [4]Fl{} {
		shadingFields["Domain"] = writeFloatArray(s.Domain[:])
	}
	if (s.Matrix != Matrix{}) {
		shadingFields["Matrix"] = s.Matrix.String()
	}
	return StreamHeader{Fields: shadingFields}, nil
}

// Clone returns a deep copy with concrete type `FunctionBased`
func (f ShadingFunctionBased) Clone() Shading {
	out := f
	out.Function = append([]FunctionDict(nil), f.Function...)
	return out
}

// BaseGradient stores attributes common to linear and radial gradients.
type BaseGradient struct {
	Domain [2]Fl // optional, default to [0 1]
	// either one 1 -> n function, or n 1->1 functions,
	//  where n is the number of color components
	Function []FunctionDict
	Extend   [2]bool // optional, default to [false, false]
}

//	update out
// `pdf` is used to write the functions
func (g BaseGradient) pdfString(pdf pdfWriter, out map[Name]string) {
	fns := pdf.writeFunctions(g.Function)
	// we have to differentiate one-length arrays
	if len(fns) == 1 {
		out["Function"] = fns[0].String()
	} else {
		out["Function"] = writeRefArray(fns)
	}
	if g.Domain != [2]Fl{} {
		out["Domain"] = writeFloatArray(g.Domain[:])
	}
	if g.Extend != [2]bool{} {
		out["Extend"] = fmt.Sprintf("[%v %v]", g.Extend[0], g.Extend[1])
	}
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
	Coords [4]Fl // x0, y0, x1, y1
}

func (s ShadingAxial) shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte) {
	s.BaseGradient.pdfString(pdf, shadingFields)
	shadingFields["ShadingType"] = "2"
	shadingFields["Coords"] = writeFloatArray(s.Coords[:])
	return StreamHeader{Fields: shadingFields}, nil
}

// Clone returns a deep copy with concrete type `Axial`
func (f ShadingAxial) Clone() Shading {
	out := f
	out.BaseGradient = f.BaseGradient.Clone()
	return out
}

type ShadingRadial struct {
	BaseGradient
	Coords [6]Fl // x0, y0, r0, x1, y1, r1
}

func (s ShadingRadial) shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte) {
	s.BaseGradient.pdfString(pdf, shadingFields)
	shadingFields["ShadingType"] = "3"
	shadingFields["Coords"] = writeFloatArray(s.Coords[:])
	return StreamHeader{Fields: shadingFields}, nil
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
	Decode            [][2]Fl
	Function          []FunctionDict // optional, one 1->n function or n 1->1 functions (n is the number of colour components)

}

// Clone returns a deep copy
func (ss ShadingStream) Clone() ShadingStream {
	out := ss
	out.Stream = ss.Stream.Clone()
	out.Decode = append([][2]Fl(nil), ss.Decode...)
	if ss.Function != nil { // to preserve reflect.DeepEqual
		out.Function = make([]FunctionDict, len(ss.Function))
	}
	for i, f := range ss.Function {
		out.Function[i] = f.Clone()
	}
	return out
}

// return the shared dict attributes
func (ss ShadingStream) pdfFields(shadingFields map[Name]string, pdf pdfWriter, type_ uint8) (StreamHeader, []byte) {
	args := ss.PDFCommonFields(true)
	args.updateWith(shadingFields)
	args.Fields["ShadingType"] = strconv.Itoa(int(type_))
	args.Fields["BitsPerCoordinate"] = strconv.Itoa(int(ss.BitsPerCoordinate))
	args.Fields["BitsPerComponent"] = strconv.Itoa(int(ss.BitsPerComponent))
	args.Fields["Decode"] = writePointsArray(ss.Decode)
	if len(ss.Function) != 0 {
		fns := pdf.writeFunctions(ss.Function)
		args.Fields["Function"] = writeRefArray(fns)
	}
	return args, ss.Content
}

type ShadingFreeForm struct {
	ShadingStream
	BitsPerFlag uint8 // 2, 4, or 8
}

// method shared with ShadingCoons and ShadingTensorProduct
// type_ is 4, 6 or 7
func (c ShadingFreeForm) pdfContentExt(shadingFields map[Name]string, pdf pdfWriter, type_ uint8) (StreamHeader, []byte) {
	out, content := c.ShadingStream.pdfFields(shadingFields, pdf, type_)
	out.Fields["BitsPerFlag"] = strconv.Itoa(int(c.BitsPerFlag))
	return out, content
}

func (c ShadingFreeForm) shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte) {
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

func (c ShadingLattice) shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte) {
	out, content := c.ShadingStream.pdfFields(shadingFields, pdf, 5)
	out.Fields["VerticesPerRow"] = strconv.Itoa(c.VerticesPerRow)
	return out, content
}

// Clone returns a deep copy with concrete type `ShadingLattice`
func (co ShadingLattice) Clone() Shading {
	out := co
	out.ShadingStream = co.ShadingStream.Clone()
	return out
}

type ShadingCoons ShadingFreeForm

func (c ShadingCoons) shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte) {
	return ShadingFreeForm(c).pdfContentExt(shadingFields, pdf, 6)
}

// Clone returns a deep copy with concrete type `ShadingCoons`
func (co ShadingCoons) Clone() Shading {
	return ShadingCoons(ShadingFreeForm(co).Clone().(ShadingFreeForm))
}

type ShadingTensorProduct ShadingFreeForm

func (c ShadingTensorProduct) shadingPDFContent(shadingFields map[Name]string, pdf pdfWriter) (StreamHeader, []byte) {
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
	Background []Fl       // optional
	BBox       *Rectangle // optional in shading’s target coordinate space
	AntiAlias  bool       // optional, default to false
}

func (s *ShadingDict) pdfContent(pdf pdfWriter, ref Reference) (StreamHeader, string, []byte) {
	b := make(map[Name]string)
	if s.ColorSpace != nil {
		b["ColorSpace"] = s.ColorSpace.colorSpaceWrite(pdf, ref)
	}
	if len(s.Background) != 0 {
		b["Background"] = writeFloatArray(s.Background)
	}
	if s.BBox != nil {
		b["BBox"] = s.BBox.String()
	}
	b["AntiAlias"] = strconv.FormatBool(s.AntiAlias)
	out, by := s.ShadingType.shadingPDFContent(b, pdf)
	return out, "", by
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
	out.Background = append([]Fl(nil), s.Background...)
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

func (s *PatternShading) pdfContent(pdf pdfWriter, _ Reference) (StreamHeader, string, []byte) {
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
	return StreamHeader{}, b.String(), nil
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
