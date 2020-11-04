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

// StreamDict stores the metadata associated
// with a stream
type StreamDict struct {
	Length      int
	Filters     []Filter
	DecodeParms map[string]int
}

type ContentStream struct {
	StreamDict
	Content []byte
}

// XObject is a ContentStream with a specialized dictionnary
type XObject struct {
	BBox      Rectangle
	Matrix    *Matrix // optional, default to identity
	Resources *ResourcesDict

	Content []byte
}
