package model

import "fmt"

// ----------------------- colors spaces -----------------------

// check conformity with either Referenceable or directColorSpace

var _ Referenceable = (*ColorSpaceICCBased)(nil)
var _ directColorSpace = ColorSpaceSeparation{}
var _ directColorSpace = ColorSpaceDeviceN{}
var _ directColorSpace = ColorSpaceName("")
var _ directColorSpace = ColorSpaceCalGray{}
var _ directColorSpace = ColorSpaceCalRGB{}
var _ directColorSpace = ColorSpaceLab{}
var _ directColorSpace = ColorSpaceIndexed{}
var _ directColorSpace = ColorSpaceUncoloredPattern{}

// check conformity with the ColorSpace interface

var _ ColorSpace = (*ColorSpaceICCBased)(nil)

// ColorSpace is either a Name or a more complex object
// The three families Device, CIE-based and Special are supported.
type ColorSpace interface {
	colorSpacePDFString(pdf pdfWriter) string
}

type directColorSpace interface {
	ColorSpace
	// returns a deep copy, preserving the concrete type
	cloneCS(cloneCache) ColorSpace
}

// c may be nil
func cloneColorSpace(c ColorSpace, cache cloneCache) ColorSpace {
	if c == nil {
		return nil
	}
	if c, ok := c.(directColorSpace); ok {
		return c.cloneCS(cache)
	}
	// if it's not direct, it must be Referenceable
	refe, _ := c.(Referenceable)
	return cache.checkOrClone(refe).(ColorSpace)
}

// ------------------------ Device ------------------------

const (
	ColorSpaceRGB     ColorSpaceName = "DeviceRGB"
	ColorSpaceGray    ColorSpaceName = "DeviceGray"
	ColorSpaceCMYK    ColorSpaceName = "DeviceCMYK"
	ColorSpacePattern ColorSpaceName = "Pattern"
)

type ColorSpaceName Name

// NewNameColorSpace validate the color space
func NewNameColorSpace(cs string) (ColorSpaceName, error) {
	c := ColorSpaceName(cs)
	switch c {
	case ColorSpaceRGB, ColorSpaceGray, ColorSpaceCMYK, ColorSpacePattern:
		return c, nil
	default:
		return "", fmt.Errorf("invalid named color space %s", cs)
	}
}

func (n ColorSpaceName) colorSpacePDFString(pdf pdfWriter) string {
	return Name(n).String()
}

func (n ColorSpaceName) cloneCS(cloneCache) ColorSpace { return n }

// ---------------------- CIE-based ----------------------
type ColorSpaceCalGray struct {
	WhitePoint [3]Fl
	BlackPoint [3]Fl // optional, default to [0 0 0]
	Gamma      Fl    // optional, default to 1
}

func (c ColorSpaceCalGray) colorSpacePDFString(pdf pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]Fl{} {
		out += fmt.Sprintf("/BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Gamma != 0 {
		out += fmt.Sprintf("/Gamma %.3f", c.Gamma)
	}
	out += ">>"
	return fmt.Sprintf("[/CalGray %s]", out)
}

func (c ColorSpaceCalGray) cloneCS(cloneCache) ColorSpace { return c }

type ColorSpaceCalRGB struct {
	WhitePoint [3]Fl
	BlackPoint [3]Fl // optional, default to [0 0 0]
	Gamma      [3]Fl // optional, default to [1 1 1]
	Matrix     [9]Fl // [ X_A Y_A Z_A X_B Y_B Z_B X_C Y_C Z_C ], optional, default to identity
}

func (c ColorSpaceCalRGB) colorSpacePDFString(pdf pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]Fl{} {
		out += fmt.Sprintf("/BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Gamma != [3]Fl{} {
		out += fmt.Sprintf("/Gamma %s", writeFloatArray(c.Gamma[:]))
	}
	if c.Matrix != [9]Fl{} {
		out += fmt.Sprintf("/Matrix %s", writeFloatArray(c.Matrix[:]))
	}
	out += ">>"
	return fmt.Sprintf("[/CalRGB %s]", out)
}

func (c ColorSpaceCalRGB) cloneCS(cloneCache) ColorSpace { return c }

type ColorSpaceLab struct {
	WhitePoint [3]Fl
	BlackPoint [3]Fl // optional, default to [0 0 0]
	Range      [4]Fl // [ a_min a_max b_min b_max ], optional, default to [−100 100 −100 100 ]
}

func (c ColorSpaceLab) colorSpacePDFString(pdf pdfWriter) string {
	out := fmt.Sprintf("<</WhitePoint %s", writeFloatArray(c.WhitePoint[:]))
	if c.BlackPoint != [3]Fl{} {
		out += fmt.Sprintf("/BlackPoint %s", writeFloatArray(c.BlackPoint[:]))
	}
	if c.Range != [4]Fl{} {
		out += fmt.Sprintf("/Range %s", writeFloatArray(c.Range[:]))
	}
	out += ">>"
	return fmt.Sprintf("[/Lab %s]", out)
}

func (c ColorSpaceLab) cloneCS(cloneCache) ColorSpace { return c }

type ColorSpaceICCBased struct {
	Stream

	N         int        // 1, 3 or 4
	Alternate ColorSpace // optional
	Range     [][2]Fl    // optional, default to [{0, 1}, ...]
}

// returns the stream object. `pdf` is used
// to write potential alternate space.
func (c *ColorSpaceICCBased) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	baseArgs := c.PDFCommonFields()
	b := newBuffer()
	b.fmt("<</N %d %s", c.N, baseArgs)
	if c.Alternate != nil {
		b.fmt("/Alternate %s", c.Alternate.colorSpacePDFString(pdf))
	}
	if len(c.Range) != 0 {
		b.fmt("/Range %s", writePointsArray(c.Range))
	}
	b.fmt(">>")
	return b.String(), c.Content
}

func (cs *ColorSpaceICCBased) colorSpacePDFString(pdf pdfWriter) string {
	ref := pdf.addItem(cs) // get or write the content stream (see ColorSpaceICCBased.pdfContent)
	return fmt.Sprintf("[/ICCBased %s]", ref)
}

func (cs *ColorSpaceICCBased) clone(cache cloneCache) Referenceable {
	if cs == nil {
		return cs
	}
	out := *cs
	out.Stream = cs.Stream.Clone()
	out.Range = append([][2]Fl(nil), cs.Range...)
	if cs.Alternate != nil {
		out.Alternate = cloneColorSpace(cs.Alternate, cache)
	}
	return &out
}

// ----------------------------------- Special -----------------------------------

// ColorSpaceIndexed is written in PDF as
// [/Indexed base hival lookup ]
type ColorSpaceIndexed struct {
	Base   ColorSpace // required
	Hival  uint8
	Lookup ColorTable
}

func (c ColorSpaceIndexed) colorSpacePDFString(pdf pdfWriter) string {
	base := "null"
	if c.Base != nil {
		base = c.Base.colorSpacePDFString(pdf)
	}
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

func (table *ColorTableStream) clone(cloneCache) Referenceable {
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

func (c ColorSpaceUncoloredPattern) colorSpacePDFString(pdf pdfWriter) string {
	under := "null"
	if c.UnderlyingColorSpace != nil {
		under = c.UnderlyingColorSpace.colorSpacePDFString(pdf)
	}
	return fmt.Sprintf("[/Pattern %s]", under)
}

func (c ColorSpaceUncoloredPattern) cloneCS(cache cloneCache) ColorSpace {
	return ColorSpaceUncoloredPattern{UnderlyingColorSpace: cloneColorSpace(c.UnderlyingColorSpace, cache)}
}

// ColorSpaceSeparation is defined in PDF as an array
// [/Separation name alternateSpace tintTransform ]
type ColorSpaceSeparation struct {
	Name           Name
	AlternateSpace ColorSpace   // required may not be another special colour space
	TintTransform  FunctionDict // required, may be an indirect object
}

func (s ColorSpaceSeparation) colorSpacePDFString(pdf pdfWriter) string {
	cs := "null"
	if s.AlternateSpace != nil {
		cs = s.AlternateSpace.colorSpacePDFString(pdf)
	}
	funcRef := pdf.addObject(s.TintTransform.pdfContent(pdf))
	return fmt.Sprintf("[/Separation %s %s %s]", s.Name, cs, funcRef)
}

// return a ColorSpaceSeparation
func (s ColorSpaceSeparation) cloneCS(cache cloneCache) ColorSpace {
	out := s
	if s.AlternateSpace != nil {
		out.AlternateSpace = cloneColorSpace(s.AlternateSpace, cache)
	}
	out.TintTransform = s.TintTransform.Clone()
	return out
}

// ColorSpaceDeviceNAttributes contains additional information about the components
// See Table 71 – Entries in a DeviceN Colour Space Attributes Dictionary
type ColorSpaceDeviceNAttributes struct {
	Subtype     Name                          // optional, DeviceN or NChannel
	Colorants   map[Name]ColorSpaceSeparation // required if Subtype is NChannel
	Process     ColorSpaceDeviceNProcess      // optional
	MixingHints *ColorSpaceDeviceNMixingHints // optional
}

func (c ColorSpaceDeviceNAttributes) pdfString(pdf pdfWriter) string {
	out := newBuffer()
	out.WriteString("<<")
	if c.Subtype != "" {
		out.WriteString("/Subtype" + c.Subtype.String())
	}
	out.WriteString("/Colorants <<")
	for name, cs := range c.Colorants {
		out.WriteString(name.String() + " " + cs.colorSpacePDFString(pdf))
	}
	out.WriteString(">>")
	if c.Process.ColorSpace != nil {
		out.fmt("/Process %s", c.Process.pdfString(pdf))
	}
	if c.MixingHints != nil {
		out.fmt("/MixingHints %s", c.MixingHints.pdfString(pdf))
	}
	out.WriteString(">>")
	return out.String()
}

func (c *ColorSpaceDeviceNAttributes) clone(cache cloneCache) *ColorSpaceDeviceNAttributes {
	if c == nil {
		return nil
	}
	out := *c
	if c.Colorants != nil {
		out.Colorants = make(map[Name]ColorSpaceSeparation, len(c.Colorants))
		for n, cs := range c.Colorants {
			out.Colorants[n] = cs.cloneCS(cache).(ColorSpaceSeparation)
		}
	}
	out.Process = c.Process.clone(cache)
	out.MixingHints = c.MixingHints.Clone()
	return &out
}

// Table 72 – Entries in a DeviceN Process Dictionary
type ColorSpaceDeviceNProcess struct {
	ColorSpace ColorSpace // required
	Components []Name     // required
}

func (c ColorSpaceDeviceNProcess) pdfString(pdf pdfWriter) string {
	cs := "null"
	if c.ColorSpace != nil {
		cs = c.ColorSpace.colorSpacePDFString(pdf)
	}
	return fmt.Sprintf("<</ColorSpace %s/Components %s>>",
		cs, writeNameArray(c.Components))
}

func (c ColorSpaceDeviceNProcess) clone(cache cloneCache) ColorSpaceDeviceNProcess {
	out := c
	out.ColorSpace = cloneColorSpace(c.ColorSpace, cache)
	out.Components = append([]Name(nil), c.Components...)
	return out
}

// Table 73 – Entries in a DeviceN Mixing Hints Dictionary
type ColorSpaceDeviceNMixingHints struct {
	Solidities    map[Name]Fl           // optional
	PrintingOrder []Name                // optional
	DotGain       map[Name]FunctionDict // optional
}

func (c ColorSpaceDeviceNMixingHints) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.WriteString("<<")
	if len(c.Solidities) != 0 {
		b.WriteString("/Solidities <<")
		for name, f := range c.Solidities {
			b.fmt("%s %.3f", name, f)
		}
		b.WriteString(">>")
	}
	b.fmt("/PrintingOrder %s", writeNameArray(c.PrintingOrder))
	if len(c.DotGain) != 0 {
		b.WriteString("/DotGain <<")
		for name, f := range c.DotGain {
			ref := pdf.addObject(f.pdfContent(pdf))
			b.fmt("%s %s", name, ref)
		}
		b.WriteString(">>")
	}
	b.WriteString(">>")
	return b.String()
}

func (c *ColorSpaceDeviceNMixingHints) Clone() *ColorSpaceDeviceNMixingHints {
	if c == nil {
		return nil
	}
	out := *c
	if c.Solidities != nil {
		out.Solidities = make(map[Name]Fl, len(c.Solidities))
		for n, s := range c.Solidities {
			out.Solidities[n] = s
		}
	}
	out.PrintingOrder = append([]Name(nil), c.PrintingOrder...)
	if c.DotGain != nil {
		out.DotGain = make(map[Name]FunctionDict, len(c.DotGain))
		for n, f := range c.DotGain {
			out.DotGain[n] = f.Clone()
		}
	}
	return &out
}

// ColorSpaceDeviceN is defined in PDF as an array
// [ /DeviceN names alternateSpace tintTransform attributes ]
// (attributes is optional)
type ColorSpaceDeviceN struct {
	Names          []Name
	AlternateSpace ColorSpace                   // required may not be another special colour space
	TintTransform  FunctionDict                 // required, may be an indirect object
	Attributes     *ColorSpaceDeviceNAttributes // optional
}

func (n ColorSpaceDeviceN) colorSpacePDFString(pdf pdfWriter) string {
	names := writeNameArray(n.Names)
	alt := "null"
	if n.AlternateSpace != nil {
		alt = n.AlternateSpace.colorSpacePDFString(pdf)
	}
	tint := pdf.addObject(n.TintTransform.pdfContent(pdf))
	attr := ""
	if n.Attributes != nil {
		attr = n.Attributes.pdfString(pdf)
	}
	return fmt.Sprintf("[/DeviceN %s %s %s %s]", names, alt, tint, attr)
}

func (n ColorSpaceDeviceN) cloneCS(cloneCache) ColorSpace { return n }
