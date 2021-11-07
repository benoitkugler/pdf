package filters

import (
	"io"
	"io/ioutil"
)

type SkipperAscii85 struct{}

const eodASCII85 = "~>"

// Skip implements Skipper for an ASCII85Decode filter.
func (f SkipperAscii85) Skip(encoded io.Reader) (int, error) {
	// we make sure not to read passed EOD
	origin := newCountReader(encoded)
	r := newReacher(origin, []byte(eodASCII85))
	_, err := ioutil.ReadAll(r)
	return origin.totalRead, err
}
