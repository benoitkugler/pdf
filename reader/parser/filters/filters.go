// Package filters provide logic to handle binary
// data encoded with PDF filters, such as inline data images.
// Regular stream objects provide a Length information, but inline data images don't,
// which requires to detect the End of Data marker, which depends on the filter.
// This package only parse encoded content. See pdfcpu/filter for an alternative
// to also encode data.
package filters

import (
	"bufio"
	"io"
)

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

// Skipper is able to detect the end of a filtered content.
// Since some filters take additional parameters, skippers should
// be directly created by their concrete types, but this interface is exposed as a
// convenience.
type Skipper interface {
	// Skip reads the input data and look for an EOD marker.
	// It returns the number of bytes read to go right after EOD.
	// Note that, due to buferring, the given reader may actually have read a bit more.
	Skip(io.Reader) (int, error)
}

type countReader struct {
	src       bufio.Reader
	totalRead int
}

func newCountReader(src io.Reader) *countReader {
	return &countReader{src: *bufio.NewReader(src)}
}

func (c *countReader) Read(p []byte) (n int, err error) {
	n, err = c.src.Read(p)
	c.totalRead += n
	return n, err
}

func (c *countReader) ReadByte() (byte, error) {
	b, err := c.src.ReadByte()
	c.totalRead += 1
	return b, err
}
