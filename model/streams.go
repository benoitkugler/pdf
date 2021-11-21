package model

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/benoitkugler/pdf/reader/parser/filters"
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
	// special case for Crypt filters
	if fi.Name == "Crypt" {
		return r, nil
	}

	return filters.NewFilter(string(fi.Name), fi.DecodeParms, r)
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
// in `Stream.Filters` to the non-filtered data
// to obtain `Content`.
type Stream struct {
	Filter Filters

	Content []byte // such as read/writen, not decoded
}

// NewStream attemps to encode `content` using the given filters,
// applying them in the given order, and storing them in the returned object
// following the PDF order (that is, reversed).
// Be aware that not all PDF filters are supported.
func NewStream(content []byte, fs ...Filter) (Stream, error) {
	var (
		r   io.Reader = bytes.NewReader(content)
		err error
	)
	L := len(fs)
	reversed := make(Filters, L)
	for i, fi := range fs {
		r, err = filters.NewFilter(string(fi.Name), fi.DecodeParms, r)
		if err != nil {
			return Stream{}, err
		}
		reversed[L-1-i] = fi
	}
	var out Stream
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

func (c Stream) Length() int { return len(c.Content) }

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

// returns `true` unless the "Identity" crypt filter is used
func (s Stream) bypassEncrypt() bool {
	return len(s.Filter) == 1 && s.Filter[0].Name == "Crypt"
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
func (s Stream) PDFCommonFields(withLength bool) StreamHeader {
	b := make(map[Name]string)
	if withLength {
		b["Length"] = strconv.Itoa(s.Length())
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
		b["Filter"] = fmt.Sprintf("[%s]", strings.Join(fs, " "))
		b["DecodeParms"] = fmt.Sprintf("[ %s]", st.String())
	}
	return StreamHeader{Fields: b, BypassCrypt: s.bypassEncrypt()}
}

// PDFContent return the stream object content.
// Often, additional arguments will be needed, so `PDFCommonFields`
// should be used instead.
func (s Stream) PDFContent() (StreamHeader, []byte) {
	return s.PDFCommonFields(true), s.Content
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

// XObject is either an image or PDF form (*XObjectForm or *XObjectTransparencyGroup)
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

// inner dict fields
func (f XObjectForm) commonFields(pdf pdfWriter, ref Reference) StreamHeader {
	args := f.ContentStream.PDFCommonFields(true)
	args.Fields["Subtype"] = "/Form"
	args.Fields["BBox"] = f.BBox.String()
	if f.Matrix != (Matrix{}) {
		args.Fields["Matrix"] = f.Matrix.String()
	}
	if !f.Resources.IsEmpty() {
		args.Fields["Resources"] = f.Resources.pdfString(pdf, ref)
	}
	if f.StructParent != nil {
		args.Fields["StructParent"] = f.StructParent.(ObjInt).Write(nil, 0)
	} else if f.StructParents != nil {
		args.Fields["StructParents"] = f.StructParent.(ObjInt).Write(nil, 0)
	}
	return args
}

func (f *XObjectForm) pdfContent(pdf pdfWriter, ref Reference) (StreamHeader, string, []byte) {
	base := f.commonFields(pdf, ref)
	return base, "", f.Content
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

// PDFFields return the image characteristics.
func (img Image) PDFFields(inline bool) StreamHeader {
	base := img.PDFCommonFields(!inline)
	base.Fields["Width"] = strconv.Itoa(img.Width)
	base.Fields["Height"] = strconv.Itoa(img.Height)
	base.Fields["BitsPerComponent"] = strconv.Itoa(int(img.BitsPerComponent))
	if img.ImageMask {
		base.Fields["ImageMask"] = strconv.FormatBool(img.ImageMask)
	}
	if img.Intent != "" {
		base.Fields["Intent"] = img.Intent.String()
	}
	if len(img.Decode) != 0 {
		base.Fields["Decode"] = writePointsArray(img.Decode)
	}
	base.Fields["Interpolate"] = strconv.FormatBool(img.Interpolate)
	return base
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

func (f *ImageSMask) pdfContent(pdf pdfWriter, _ Reference) (StreamHeader, string, []byte) {
	base := f.Image.PDFFields(false)
	base.Fields["Subtype"] = "/Image"
	base.Fields["ColorSpace"] = Name(ColorSpaceGray).String()
	base.Fields["Matte"] = writeFloatArray(f.Matte)
	return base, "", f.Content
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

func (f *XObjectImage) pdfContent(pdf pdfWriter, ref Reference) (StreamHeader, string, []byte) {
	base := f.Image.PDFFields(false)
	base.Fields["Subtype"] = "/Image"

	if f.ColorSpace != nil {
		base.Fields["ColorSpace"] = f.ColorSpace.colorSpaceWrite(pdf, ref)
	}

	if f.Mask != nil {
		base.Fields["Mask"] = f.Mask.maskPDFString(pdf)
	}

	if len(f.Alternates) != 0 {
		chunks := make([]string, len(f.Alternates))
		for i, alt := range f.Alternates {
			chunks[i] = alt.pdfString(pdf)
		}
		base.Fields["Alternates"] = fmt.Sprintf("[%s]", strings.Join(chunks, " "))
	}
	base.Fields["SMaskInData"] = strconv.Itoa(int(f.SMaskInData))
	if f.SMask != nil {
		ref := pdf.addItem(f.SMask)
		base.Fields["SMask"] = ref.String()
	}
	if f.StructParent != nil {
		base.Fields["StructParent"] = f.StructParent.(ObjInt).Write(nil, 0)
	}
	return base, "", f.Content
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
