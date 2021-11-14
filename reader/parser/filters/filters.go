// Package filters provide logic to handle binary
// data encoded with PDF filters, such as inline data images.
// Regular stream objects provide a Length information, but inline data images don't,
// which requires to detect the End of Data marker, which depends on the filter.
// This package only parse encoded content. See pdfcpu/filter for an alternative
// to also encode data.
package filters

import (
	"bufio"
	"fmt"
	"io"

	"github.com/benoitkugler/pdf/reader/parser/filters/ccitt"
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

// NewFilter wraps the given `src` to decode the according to the given filter `name`,
// or returns an error it the filter is not supported.
func NewFilter(name string, params map[string]int, src io.Reader) (io.Reader, error) {
	switch name {
	case Flate:
		params, err := processFlateParams(params)
		if err != nil {
			return nil, err
		}
		return flateDecoder(params, src)
	case LZW:
		return lzwDecoder(processLZWParams(params), src), nil
	case CCITTFax:
		return ccittDecoder(processCCITTFaxParams(params), src)
	case RunLength:
		return runLengthDecoder(src)
	default:
		return nil, fmt.Errorf("unsupported filter %s", name)
	}
}

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

// SkipperFromFilter select the right skipper.
// An error is returned if and only if the filter is not supported
func SkipperFromFilter(name string, params map[string]int) (Skipper, error) {
	var skipper Skipper
	switch name {
	case ASCII85:
		skipper = SkipperAscii85{}
	case ASCIIHex:
		skipper = SkipperAsciiHex{}
	case Flate:
		// parameters only influence post processing
		skipper = SkipperFlate{}
	case RunLength:
		skipper = SkipperRunLength{}
	case DCT:
		skipper = SkipperDCT{}
	case CCITTFax:
		skipper = SkipperCCITT{Params: processCCITTFaxParams(params)}
	case LZW:
		skipper = SkipperLZW{EarlyChange: processLZWParams(params)}
	default:
		return nil, fmt.Errorf("unsupported filter: %s", name)
	}
	return skipper, nil
}

func processLZWParams(params map[string]int) (earlyChange bool) {
	ec, ok := params["EarlyChange"]
	if !ok || ec == 1 {
		earlyChange = true
	}
	return earlyChange
}

func processCCITTFaxParams(params map[string]int) ccitt.CCITTParams {
	cols := 1728
	col, ok := params["Columns"]
	if ok {
		cols = col
	}

	endOfBlock := true
	if v, has := params["EndOfBlock"]; has && v != 1 {
		endOfBlock = false
	}
	return ccitt.CCITTParams{
		Encoding:   int32(params["K"]),
		Columns:    int32(cols),
		Rows:       int32(params["Rows"]),
		EndOfBlock: endOfBlock,
		EndOfLine:  params["EndOfLine"] == 1,
		Black:      params["BlackIs1"] == 1,
		ByteAlign:  params["EncodedByteAlign"] == 1,
	}
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
