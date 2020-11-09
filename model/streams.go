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

// ContentStream is a PDF stream.
// New ContentStream must be created
// by applying the filters described
// in `StreamDict.Filters` to the non-filtered data
// to obtain `Content`
type ContentStream struct {
	// Length      int
	Filters []Filter
	// nil, or same length than Filters.
	// boolean value are stored as 0 (false) or 1 (true)
	DecodeParms []map[Name]int

	Content []byte // such as read/writen, not decoded
}

func (c ContentStream) Length() int {
	return len(c.Content)
}

// ParamsForFilter is a convenience which returns
// the additionnal arguments of the i-th filter
func (s ContentStream) ParamsForFilter(index int) map[Name]int {
	if len(s.DecodeParms) == 0 {
		return nil
	}
	return s.DecodeParms[index]
}

// PDFCommonArgs returns the content of the dictionnary of `s`
// without the enclosing << >>.
// It will usually be used in combination with other fields.
func (s ContentStream) PDFCommonFields() string {
	fs := make([]string, len(s.Filters))
	for i, f := range s.Filters {
		fs[i] = Name(f).PDFString()
	}
	decode := ""
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
				st.WriteString(n.PDFString() + fmt.Sprintf(" %v ", arg))
			}
			st.WriteString(" >> ")
		}
		decode = fmt.Sprintf("/DecodeParams [ %s]", st.String())
	}
	return fmt.Sprintf("/Length %d /Filters [%s] %s", s.Length(), strings.Join(fs, " "), decode)
}

// XObject is either an image or PDF form
type XObject interface {
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

// ----------------------- images -----------------------

// TODO:
type Mask interface {
	isMask()
}

// XObjectImage represents a sampled visual image such as a photograph
type XObjectImage struct {
	ContentStream

	Width, Height    int
	ColorSpace       ColorSpace // any type of colour space except Pattern
	BitsPerComponent uint8      // 1, 2, 4, 8, or  16.
	Intent           Name       // optional
	ImageMask        bool       // optional
	Mask             Mask       // optional
	// optional.  length : number of color component required by color space.
	// Special case for Mask image where [1 0] is also allowed (despite not having 1 <= 0)
	Decode      []Range
	Interpolate bool             // optional
	Alternates  []AlternateImage // optional
	SMask       *XObjectImage    // optional
	SMaskInData uint8            // optional, 0, 1 or 2
}

type AlternateImage struct {
	Image              *XObjectImage
	DefaultForPrinting bool // optional
}
