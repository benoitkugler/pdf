package filters

import (
	"io"
)

type SkipperAsciiHex struct{}

const eodHexDecode = '>'

// Skip implements Skipper for an ASCIIHexDecode filter.
func (f SkipperAsciiHex) Skip(encoded io.Reader) (int, error) {
	// we make sure not to read passed EOD
	origin := newCountReader(encoded)
	r := newReacher(origin, []byte{eodHexDecode})
	_, err := io.ReadAll(r)
	return origin.totalRead, err
}
