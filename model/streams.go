package model

import (
	"fmt"
	"strings"
)

const (
	ASCII85   Filter = "ASCII85Decode"
	ASCIIHex  Filter = "ASCIIHexDecode"
	RunLength Filter = "RunLengthDecode"
	LZW       Filter = "LZWDecode"
	Flate     Filter = "FlateDecode"
	CCITTFax  Filter = "CCITTFaxDecode"
	JBIG2     Filter = "JBIG2Decode"
	DCT       Filter = "DCTDecode"
	JPX       Filter = "JPXDecode"
)

type Filter Name

// NewFilter validate `s` and returns
// an empty string it is not a known filter
func NewFilter(s string) Filter {
	f := Filter(s)
	switch f {
	case ASCII85, ASCIIHex, RunLength, LZW,
		Flate, CCITTFax, JBIG2, DCT, JPX:
		return f
	default:
		return ""
	}
}

var booleanNames = map[Name]bool{
	"EndOfLine":        true,
	"EncodedByteAlign": true,
	"EndOfBlock":       true,
	"BlackIs1":         true,
}

// Stream is a PDF stream.
// New Stream must be created
// by applying the filters described
// in `StreamDict.Filters` to the non-filtered data
// to obtain `Content`
type Stream struct {
	// Length      int
	Filter []Filter
	// nil, or same length than Filters.
	// boolean value are stored as 0 (false) or 1 (true)
	DecodeParms []map[Name]int

	Content []byte // such as read/writen, not decoded
}

func (c Stream) Length() int {
	return len(c.Content)
}

// ParamsForFilter is a convenience which returns
// the additionnal arguments of the i-th filter
func (s Stream) ParamsForFilter(index int) map[Name]int {
	if len(s.DecodeParms) == 0 {
		return nil
	}
	return s.DecodeParms[index]
}

// PDFCommonArgs returns the content of the Dictionary of `s`
// without the enclosing << >>.
// It will usually be used in combination with other fields.
func (s Stream) PDFCommonFields() string {
	b := newBuffer()
	b.fmt("/Length %d", s.Length())
	if len(s.Filter) != 0 {
		fs := make([]string, len(s.Filter))
		for i, f := range s.Filter {
			fs[i] = Name(f).String()
		}
		b.fmt("/Filter [%s]", strings.Join(fs, " "))
	}
	if len(s.DecodeParms) != 0 {
		var st strings.Builder
		for _, v := range s.DecodeParms {
			if len(v) == 0 {
				st.WriteString("null ")
				continue
			}
			st.WriteString("<< ")
			for n, k := range v {
				var arg interface{} = k
				if booleanNames[n] {
					arg = k == 1
				}
				st.WriteString(fmt.Sprintf("%s %v ", n, arg))
			}
			st.WriteString(" >> ")
		}
		b.fmt("/DecodeParams [ %s]", st.String())
	}
	return b.String()
}

// PDFContent return the stream object content
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

// XObject is either an image or PDF form
type XObject interface {
	cachable
	isXObject()
}

func (*XObjectForm) isXObject()  {}
func (*XObjectImage) isXObject() {}

// XObjectForm is a is a self-contained description of an arbitrary sequence of
// graphics objects
type XObjectForm struct {
	ContentStream

	BBox      Rectangle
	Matrix    *Matrix // optional, default to identity
	Resources *ResourcesDict
}

func (f *XObjectForm) pdfContent(pdf pdfWriter, _ Reference) (string, []byte) {
	args := f.ContentStream.PDFCommonFields()
	b := newBuffer()
	b.fmt("<</Subtype/Form %s/BBox %s", args, f.BBox.String())
	if f.Matrix != nil {
		b.fmt("/Matrix %s", *f.Matrix)
	}
	if f.Resources != nil {
		b.line("/Resources %s", pdf.addItem(f.Resources))
	}
	b.fmt(">>")
	return b.String(), f.Content
}

// ----------------------- images -----------------------

// TODO:
type Mask interface {
	isMask()
}

// XObjectImage represents a sampled visual image such as a photograph
type XObjectImage struct {
	Stream

	Width, Height    int
	ColorSpace       ColorSpace // any type of colour space except Pattern
	BitsPerComponent uint8      // 1, 2, 4, 8, or  16.
	Intent           Name       // optional
	ImageMask        bool       // optional
	Mask             Mask       // optional

	// optional. Array of length : number of color component required by color space.
	// Special case for Mask image where [1 0] is also allowed (despite not having 1 <= 0)
	Decode      []Range
	Interpolate bool             // optional
	Alternates  []AlternateImage // optional
	SMask       *XObjectImage    // optional
	SMaskInData uint8            // optional, 0, 1 or 2
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
	b.WriteString(">>")
	return b.String(), f.Content
}

type AlternateImage struct {
	Image              *XObjectImage
	DefaultForPrinting bool // optional
}

func (alt AlternateImage) pdfString(pdf pdfWriter) string {
	imgRef := pdf.addItem(alt.Image)
	return fmt.Sprintf("<</DefaultForPrinting %v/Image %s", alt.DefaultForPrinting, imgRef)
}
