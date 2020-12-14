// Package filters provide logic to handle binary
// data encoded with PDF filters, such as inline data images.
// Regular stream objects provide a Length information, but inline data images don't,
// which requires to detect the End of Data marker, which depends on the filter.
// This package only parse encoded content. See pdfcpu/filter for an alternative
// to also encode data.
package filters

// PDF defines the following filters. See also 7.4 in the PDF spec,
// and 8.9.7 - Inline Images
const (
	ASCII85   = "ASCII85Decode"
	ASCIIHex  = "ASCIIHexDecode"
	RunLength = "RunLengthDecode"
	LZW       = "LZWDecode"
	Flate     = "FlateDecode"
	DCT       = "DCTDecode"
	CCITTFax  = "CCITTFaxDecode"
)

// Skipper reads the input data and stop exactly after
// the EOD marker. It returns the number of bytes read (including EOD).
// Since some filters take additional parameters, skippers should
// be directly created by their concrete types, but this interface is exposed as a
// convenience.
type Skipper interface {
	Skip([]byte) (int, error)
}
