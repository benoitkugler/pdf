package model

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

type Filter string

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

// StreamDict stores the metadata associated
// with a stream
type StreamDict struct {
	// Length      int
	Filters []Filter
	// nil, or same length than Filters
	// boolean value are stored as 0 (false) or 1 (true)
	DecodeParms []map[Name]int
}

// ContentStream is a PDF stream.
// New ContentStream must be created
// by applying the filters described
// in `StreamDict.Filters` to the non-filtered data
// to obtain `Content`
type ContentStream struct {
	StreamDict
	Content []byte // such as read/writen, not decoded
}

func (c ContentStream) Length() int {
	return len(c.Content)
}

// XObject is a ContentStream with a specialized dictionnary
type XObject struct {
	BBox      Rectangle
	Matrix    *Matrix // optional, default to identity
	Resources *ResourcesDict

	Content []byte
}
