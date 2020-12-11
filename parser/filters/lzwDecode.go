package filters

import (
	"bytes"
	"io/ioutil"

	"github.com/hhrutter/lzw"
)

// TODO: check parameters
type SkipperLZW struct {
	EarlyChange bool
}

// Skip implements decoding for an LZWDecode filter.
func (f SkipperLZW) Skip(encoded []byte) (int, error) {
	r := bytes.NewReader(encoded)

	rc := lzw.NewReader(r, f.EarlyChange)
	_, err := ioutil.ReadAll(rc)
	if err != nil {
		return 0, err
	}
	err = rc.Close()

	return len(encoded) - r.Len(), err
}
