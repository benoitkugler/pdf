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
	DecodeParams map[Name]int
}

// Clone returns a deep copy of the filter
func (f Filter) Clone() Filter {
	out := f
	if f.DecodeParams != nil {
		out.DecodeParams = make(map[Name]int, len(f.DecodeParams))
		for n, v := range f.DecodeParams {
			out.DecodeParams[n] = v
		}
	}
	return out
}

func (f Filter) params() map[string]int {
	out := make(map[string]int, len(f.DecodeParams))
	for k, v := range f.DecodeParams {
		out[string(k)] = v
	}
	return out
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

var booleanNames = map[Name]bool{
	"EndOfLine":        true,
	"EncodedByteAlign": true,
	"EndOfBlock":       true,
	"BlackIs1":         true,
}

// type DecodeParams []map[Name]int

// Stream is a PDF stream.
// New Stream must be created
// by applying the filters described
// in `StreamDict.Filters` to the non-filtered data
// to obtain `Content`. See NewStream for a convenient
// way to encode stream.
type Stream struct {
	// Length      int
	Filter []Filter

	// DecodeParms DecodeParams

	Content []byte // such as read/writen, not decoded
}

// NewStream attemps to encode `content` using the given filters,
// applying them in the given order, and storing them in the returned object
// following the PDF order (that is, reversed).
// Be aware that not all PDF filters are supported (see filters.List).
func NewStream(content []byte, filters []Filter) (Stream, error) {
	var r io.Reader = bytes.NewReader(content)
	L := len(filters)
	reversed := make([]Filter, L)
	for i, fi := range filters {
		fil, err := filter.NewFilter(string(fi.Name), fi.params())
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
	var r io.Reader = bytes.NewReader(s.Content)
	for _, fi := range s.Filter {
		fil, err := filter.NewFilter(string(fi.Name), fi.params())
		if err != nil {
			return nil, err
		}
		r, err = fil.Decode(r)
		if err != nil {
			return nil, err
		}
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
// func (dc DecodeParams) ParamsForFilter(index int) map[string]int {
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
func (s Stream) PDFCommonFields() string {
	b := newBuffer()
	b.fmt("/Length %d", s.Length())
	if len(s.Filter) != 0 {
		fs := make([]string, len(s.Filter))
		var st strings.Builder
		for i, f := range s.Filter {
			fs[i] = Name(f.Name).String()
			if len(f.DecodeParams) == 0 {
				st.WriteString("null ")
				continue
			}
			st.WriteString("<< ")
			for n, k := range f.DecodeParams {
				var arg interface{} = k
				if booleanNames[n] {
					arg = k == 1
				}
				st.WriteString(fmt.Sprintf("%s %v ", n, arg))
			}
			st.WriteString(" >> ")
		}
		b.fmt("/Filter [%s]/DecodeParams [ %s]", strings.Join(fs, " "), st.String())
	}
	return b.String()
}

// PDFContent return the stream object content.
// Often, additional arguments will be needed, so `PDFCommonFields`
// should be used instead.
func (s Stream) PDFContent() (string, []byte) {
	arg := s.PDFCommonFields()
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
	Matrix    Matrix         // optional, default to identity
	Resources *ResourcesDict // optional

	// Integer key of the form XObject’s entry in the structural parent tree.
	// At most one of the entries StructParent or StructParents shall be
	// present: a form XObject shall be either a content item in its entirety (StructParent) or
	// a container for marked-content sequences that are content items (StructParents), but
	// not both.
	// Optional
	StructParent, StructParents MaybeInt
}

func (f *XObjectForm) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	args := f.ContentStream.PDFCommonFields()
	b := newBuffer()
	b.fmt("<</Subtype/Form %s/BBox %s", args, f.BBox.String())
	if f.Matrix != (Matrix{}) {
		b.fmt("/Matrix %s", f.Matrix)
	}
	if f.Resources != nil {
		b.line("/Resources %s", f.Resources.pdfString(pdf))
	}
	if f.StructParent != nil {
		b.fmt("/StructParent %d", f.StructParent.(Int))
	} else if f.StructParents != nil {
		b.fmt("/StructParents %d", f.StructParent.(Int))
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
	if f.Resources != nil {
		res := f.Resources.clone(cache)
		out.Resources = &res
	}
	return &out
}

// ----------------------- images -----------------------

// TODO:
type Mask interface {
	isMask()
	Clone() Mask
}

// XObjectImage represents a sampled visual image such as a photograph
type XObjectImage struct {
	Stream

	Width, Height    int
	ColorSpace       ColorSpace // optional, any type of colour space except Pattern
	BitsPerComponent uint8      // 1, 2, 4, 8, or  16.
	Intent           Name       // optional
	ImageMask        bool       // optional
	Mask             Mask       // optional

	// optional. Array of length : number of color component required by color space.
	// Special case for Mask image where [1 0] is also allowed (despite not having 1 <= 0)
	Decode       []Range
	Interpolate  bool             // optional
	Alternates   []AlternateImage // optional
	SMask        *XObjectImage    // optional
	SMaskInData  uint8            // optional, 0, 1 or 2
	StructParent MaybeInt         // required if the image is a structural content item
}

func (f *XObjectImage) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	b := newBuffer()
	base := f.PDFCommonFields()
	b.line("<</Subtype/Image %s/Width %d/Height %d/BitsPerComponent %d",
		base, f.Width, f.Height, f.BitsPerComponent)
	b.fmt("/ImageMask %v", f.ImageMask)
	if f.ColorSpace != nil {
		cs := writeColorSpace(f.ColorSpace, pdf)
		b.fmt("/ColorSpace %s", cs)
	}
	if f.Intent != "" {
		b.fmt("/Intent %s", f.Intent)
	}
	//TODO: mask
	if len(f.Decode) != 0 {
		b.fmt("/Decode %s", writeRangeArray(f.Decode))
	}
	b.line("/Interpolate %v", f.Interpolate)
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
		b.fmt("/StructParent %d", f.StructParent.(Int))
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
		out.Mask = img.Mask.Clone()
	}
	out.Decode = append([]Range(nil), img.Decode...)
	if img.Alternates != nil {
		out.Alternates = make([]AlternateImage, len(img.Alternates))
	}
	for i, alt := range img.Alternates {
		out.Alternates[i] = alt.clone(cache)
	}
	out.SMask = cache.checkOrClone(img.SMask).(*XObjectImage)
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
