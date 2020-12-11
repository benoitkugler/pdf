package filters

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
)

type SkipperFlate struct{}

// Skip implements Skipper for a Flate filter.
func (f SkipperFlate) Skip(encoded []byte) (int, error) {
	r := bytes.NewReader(encoded)
	rc, err := zlib.NewReader(r)
	if err != nil {
		return 0, err
	}
	_, err = ioutil.ReadAll(rc)
	if err != nil {
		return 0, err
	}
	err = rc.Close()
	return len(encoded) - r.Len(), err
}
