package filters

import (
	"bytes"
	"io/ioutil"
)

type SkipperAsciiHex struct{}

const eodHexDecode = '>'

// Skip implements Skipper for an ASCIIHexDecode filter.
func (f SkipperAsciiHex) Skip(encoded []byte) (int, error) {
	// we make sure not to read passed EOD
	origin := bytes.NewReader(encoded)
	r := newReacher(origin, []byte{eodHexDecode})
	_, err := ioutil.ReadAll(r)
	return len(encoded) - origin.Len(), err
}
