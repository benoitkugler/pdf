package model

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
)

const (
	ASCII85   Name = "ASCII85Decode"
	ASCIIHex  Name = "ASCIIHexDecode"
	RunLength Name = "RunLengthDecode"
	LZW       Name = "LZWDecode"
	Flate     Name = "FlateDecode"
	CCITTFax  Name = "CCITTFaxDecode"
	JBIG2     Name = "JBIG2Decode"
	DCT       Name = "DCTDecode"
	JPX       Name = "JPXDecode"
)

type Filter struct {
	Name Name
	// optional, boolean value are stored as 0 (false) or 1 (true)
	DecodeParms map[string]int
}

// Clone returns a deep copy of the filter
func (f Filter) Clone() Filter {
	out := f
	if f.DecodeParms != nil {
		out.DecodeParms = make(map[string]int, len(f.DecodeParms))
		for n, v := range f.DecodeParms {
			out.DecodeParms[n] = v
		}
	}
	return out
}

func (fi Filter) DecodeReader(r io.Reader) (io.Reader, error) {
	fil, err := filter.NewFilter(string(fi.Name), fi.DecodeParms)
	if err != nil {
		return nil, err
	}
	return fil.Decode(r)
}

// // NewFilter validate `s` and returns
// // an empty string it is not a known filter
// func NewFilter(s string) Filter {
// 	f := Filter(s)
// 	switch f {
// 	case ASCII85, ASCIIHex, RunLength, LZW,
// 		Flate, CCITTFax, JBIG2, DCT, JPX:
// 		return f
// 	default:
// 		return ""
// 	}
// }

var booleanNames = map[string]bool{
	"EndOfLine":        true,
	"EncodedByteAlign": true,
	"EndOfBlock":       true,
	"BlackIs1":         true,
}

// type DecodeParms []map[Name]int

type Filters []Filter

// DecodeReader accumulate the filters to produce a Reader,
// decoding `r`.
func (fs Filters) DecodeReader(r io.Reader) (io.Reader, error) {
	var err error
	for _, fi := range fs {
		r, err = fi.DecodeReader(r)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Stream is a PDF stream.
//
// A new stream can be created
// by applying the filters described
// in `StreamDict.Filters` to the non-filtered data
// to obtain `Content`.
type Stream struct {
	// Length      int
	Filter Filters

	// DecodeParms DecodeParms

	Content []byte // such as read/writen, not decoded
}

// NewStream attemps to encode `content` using the given filters,
// applying them in the given order, and storing them in the returned object
// following the PDF order (that is, reversed).
// Be aware that not all PDF filters are supported (see filters.List).
func NewStream(content []byte, filters Filters) (Stream, error) {
	var r io.Reader = bytes.NewReader(content)
	L := len(filters)
	reversed := make(Filters, L)
	for i, fi := range filters {
		fil, err := filter.NewFilter(string(fi.Name), fi.DecodeParms)
		if err != nil {
			return Stream{}, err
		}
		r, err = fil.Encode(r)
		if err != nil {
			return Stream{}, err
		}
		reversed[L-1-i] = fi
	}
	var (
		out Stream
		err error
	)
	out.Content, err = ioutil.ReadAll(r)
	if err != nil {
		return out, err
	}
	out.Filter = reversed
	return out, nil
}

// Decode attemps to apply the Filters to decode its content.
// Be aware that not all PDF filters are supported (see filters.List).
func (s Stream) Decode() ([]byte, error) {
	r, err := s.Filter.DecodeReader(bytes.NewReader(s.Content))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func (c Stream) Length() int {
	return len(c.Content)
}

// Clone returns a deep copy of the stream
func (c Stream) Clone() Stream {
	var s Stream
	if c.Filter != nil { // preserve nil
		s.Filter = make([]Filter, len(c.Filter))
		for i, f := range c.Filter {
			s.Filter[i] = f.Clone()
		}
	}
	s.Content = append([]byte(nil), c.Content...)
	return s
}

// ParamsForFilter is a convenience which returns
// the additionnal arguments of the i-th filter
// func (dc DecodeParms) ParamsForFilter(index int) map[string]int {
// 	if index >= len(dc) {
// 		return nil
// 	}
// 	out := make(map[string]int)
// 	for k, v := range dc[index] {
// 		out[string(k)] = v
// 	}
// 	return out
// }

// PDFCommonArgs returns the content of the Dictionary of `s`
// without the enclosing << >>.
// It will usually be used in combination with other fields.
func (s Stream) PDFCommonFields(withLength bool) string {
	b := newBuffer()
	if withLength {
		b.fmt("/Length %d", s.Length())
	}
	if len(s.Filter) != 0 {
		fs := make([]string, len(s.Filter))
		var st strings.Builder
		for i, f := range s.Filter {
			fs[i] = Name(f.Name).String()
			if len(f.DecodeParms) == 0 {
				st.WriteString("null ")
				continue
			}
			st.WriteString("<< ")
			for n, k := range f.DecodeParms {
				var arg interface{} = k
				if booleanNames[n] {
					arg = k == 1
				}
				st.WriteString(fmt.Sprintf("/%s %v ", n, arg))
			}
			st.WriteString(" >> ")
		}
		b.fmt("/Filter [%s]/DecodeParms [ %s]", strings.Join(fs, " "), st.String())
	}
	return b.String()
}

// PDFContent return the stream object content.
// Often, additional arguments will be needed, so `PDFCommonFields`
// should be used instead.
func (s Stream) PDFContent() (string, []byte) {
	arg := s.PDFCommonFields(true)
	return fmt.Sprintf("<<%s>>", arg), s.Content
}

// Content stream is a PDF stream object whose data consists
// of a sequence of instructions describing the
// graphical elements to be painted on a page.
type ContentStream struct {
	Stream
}

func (c ContentStream) Clone() ContentStream {
	return ContentStream{Stream: c.Stream.Clone()}
}

// XObject is either an image or PDF form
type XObject interface {
	Referenceable
	isXObject()
}

func (*XObjectForm) isXObject()  {}
func (*XObjectImage) isXObject() {}

// XObjectForm is a is a self-contained description of an arbitrary sequence of
// graphics objects
type XObjectForm struct {
	ContentStream

	BBox      Rectangle
	Matrix    Matrix        // optional, default to identity
	Resources ResourcesDict // optional

	// Integer key of the form XObject’s entry in the structural parent tree.
	// At most one of the entries StructParent or StructParents shall be
	// present: a form XObject shall be either a content item in its entirety (StructParent) or
	// a container for marked-content sequences that are content items (StructParents), but
	// not both.
	// Optional
	StructParent, StructParents MaybeInt
}

// GetStructParent implements StructParentObject
func (a *XObjectForm) GetStructParent() MaybeInt {
	return a.StructParent
}

func (f *XObjectForm) pdfContent(pdf pdfWriter, ref Reference) (string, []byte) {
	args := f.ContentStream.PDFCommonFields(true)
	b := newBuffer()
	b.fmt("<</Subtype/Form %s/BBox %s", args, f.BBox.String())
	if f.Matrix != (Matrix{}) {
		b.fmt("/Matrix %s", f.Matrix)
	}
	if !f.Resources.IsEmpty() {
		b.line("/Resources %s", f.Resources.pdfString(pdf, ref))
	}
	if f.StructParent != nil {
		b.fmt("/StructParent %d", f.StructParent.(ObjInt))
	} else if f.StructParents != nil {
		b.fmt("/StructParents %d", f.StructParent.(ObjInt))
	}
	b.fmt(">>")
	return b.String(), f.Content
}

// clone returns a deep copy of the form
// (with concrete type `XObjectForm`)
func (f *XObjectForm) clone(cache cloneCache) Referenceable {
	if f == nil {
		return f
	}
	out := *f
	out.ContentStream = f.ContentStream.Clone()
	out.Resources = f.Resources.clone(cache)
	return &out
}

// ----------------------- images -----------------------

// Mask is either MaskColor or *XObjectImage
// See 8.9.6.2 - Stencil Masking
type Mask interface {
	maskPDFString(pdf pdfWriter) string
	cloneMask(cloneCache) Mask
}

// MaskColor is a range of colours to be masked
// See 8.9.6.4 - Colour Key Masking
type MaskColor [][2]int

func (m MaskColor) maskPDFString(pdfWriter) string {
	out := make([]int, 2*len(m))
	for i, v := range m {
		out[2*i], out[2*i+1] = v[0], v[1]
	}
	return writeIntArray(out)
}

func (m MaskColor) cloneMask(cloneCache) Mask {
	return append(MaskColor(nil), m...)
}

func (m *XObjectImage) maskPDFString(p pdfWriter) string {
	ref := p.addItem(m)
	return ref.String()
}

func (m *XObjectImage) cloneMask(cache cloneCache) Mask {
	return m.cloneMask(cache).(*XObjectImage)
}

// Image are shared between inline images and XForm images.
// The ColorSpace is not included since inline images have additional restrictions.
type Image struct {
	Stream

	BitsPerComponent uint8 // 1, 2, 4, 8, or  16.

	Width, Height int
	// optional. Array of length : number of color component required by color space.
	// Special case for Mask image where [1 0] is also allowed
	Decode      [][2]Fl
	ImageMask   bool // optional
	Intent      Name // optional
	Interpolate bool // optional
}

// PDFFields return a succession of key/value pairs,
// describing the image characteristics.
func (img Image) PDFFields(inline bool) string {
	b := newBuffer()
	base := img.PDFCommonFields(!inline)
	b.line("%s /Width %d /Height %d /BitsPerComponent %d",
		base, img.Width, img.Height, img.BitsPerComponent)
	b.fmt("/ImageMask %v", img.ImageMask)
	if img.Intent != "" {
		b.fmt("/Intent %s", img.Intent)
	}
	if len(img.Decode) != 0 {
		b.fmt("/Decode %s", writePointsArray(img.Decode))
	}
	b.line("/Interpolate %v", img.Interpolate)
	return b.String()
}

// Clone returns a deep copy
func (img Image) Clone() Image {
	out := img
	out.Stream = img.Stream.Clone()
	out.Decode = append([][2]Fl(nil), img.Decode...)
	return out
}

// ImageSMask is a soft image mask
// See 11.6.5.3 - Soft-Mask Images
type ImageSMask struct {
	Image      // ImageMask and Intent are ignored
	Matte []Fl // optional, length: number of color components in the parent image
}

func (f *ImageSMask) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	base := f.Image.PDFFields(false)
	s := "<</Subtype/Image/ColorSpace" + Name(ColorSpaceGray).String() + base + ">>"
	return s, f.Content
}

func (img *ImageSMask) clone(cache cloneCache) Referenceable {
	if img == nil {
		return img
	}
	out := *img
	out.Image = img.Image.Clone()
	out.Matte = append([]Fl(nil), img.Matte...)
	return &out
}

// XObjectImage represents a sampled visual image such as a photograph
type XObjectImage struct {
	Image

	ColorSpace ColorSpace // optional, any type of colour space except Pattern

	Alternates   []AlternateImage // optional
	Mask         Mask             // optional
	SMask        *ImageSMask      // optional
	SMaskInData  uint8            // optional, 0, 1 or 2
	StructParent MaybeInt         // required if the image is a structural content item
}

// GetStructParent implements StructParentObject
func (img *XObjectImage) GetStructParent() MaybeInt {
	return img.StructParent
}

func (f *XObjectImage) pdfContent(pdf pdfWriter, ref Reference) (string, []byte) {
	b := newBuffer()
	base := f.Image.PDFFields(false)
	b.fmt("<</Subtype/Image" + base)

	if f.ColorSpace != nil {
		b.fmt("/ColorSpace %s", f.ColorSpace.colorSpaceWrite(pdf, ref))
	}

	if f.Mask != nil {
		b.WriteString("/Mask " + f.Mask.maskPDFString(pdf))
	}

	if len(f.Alternates) != 0 {
		chunks := make([]string, len(f.Alternates))
		for i, alt := range f.Alternates {
			chunks[i] = alt.pdfString(pdf)
		}
		b.line("/Alternates [%s]", strings.Join(chunks, " "))
	}
	b.fmt("/SMaskInData %d", f.SMaskInData)
	if f.SMask != nil {
		ref := pdf.addItem(f.SMask)
		b.fmt("/SMask %s", ref)
	}
	if f.StructParent != nil {
		b.fmt("/StructParent %d", f.StructParent.(ObjInt))
	}
	b.WriteString(">>")
	return b.String(), f.Content
}

func (img *XObjectImage) clone(cache cloneCache) Referenceable {
	if img == nil {
		return img
	}
	out := *img
	out.Stream = img.Stream.Clone()
	out.ColorSpace = cloneColorSpace(img.ColorSpace, cache)
	if img.Mask != nil {
		out.Mask = img.Mask.cloneMask(cache)
	}
	out.Decode = append([][2]Fl(nil), img.Decode...)
	if img.Alternates != nil {
		out.Alternates = make([]AlternateImage, len(img.Alternates))
	}
	for i, alt := range img.Alternates {
		out.Alternates[i] = alt.clone(cache)
	}
	out.SMask = cache.checkOrClone(img.SMask).(*ImageSMask)
	return &out
}

type AlternateImage struct {
	Image              *XObjectImage
	DefaultForPrinting bool // optional
}

func (alt AlternateImage) pdfString(pdf pdfWriter) string {
	imgRef := pdf.addItem(alt.Image)
	return fmt.Sprintf("<</DefaultForPrinting %v/Image %s", alt.DefaultForPrinting, imgRef)
}

func (a AlternateImage) clone(cache cloneCache) AlternateImage {
	out := a
	a.Image = cache.checkOrClone(a.Image).(*XObjectImage)
	return out
}
