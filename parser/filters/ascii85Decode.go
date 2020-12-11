package filters

import (
	"bytes"
	"io/ioutil"
)

type SkipperAscii85 struct{}

const eodASCII85 = "~>"

// Skip implements Skipper for an ASCII85Decode filter.
func (f SkipperAscii85) Skip(encoded []byte) (int, error) {
	// we make sure not to read passed EOD
	origin := bytes.NewReader(encoded)
	r := newReacher(origin, []byte(eodASCII85))
	_, err := ioutil.ReadAll(r)
	return len(encoded) - origin.Len(), err
}
