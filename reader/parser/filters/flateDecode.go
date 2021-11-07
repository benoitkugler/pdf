package filters

import (
	"compress/zlib"
	"io"
	"io/ioutil"
)

type SkipperFlate struct{}

// Skip implements Skipper for a Flate filter.
func (f SkipperFlate) Skip(encoded io.Reader) (int, error) {
	r := newCountReader(encoded)
	rc, err := zlib.NewReader(r)
	if err != nil {
		return 0, err
	}
	_, err = ioutil.ReadAll(rc)
	if err != nil {
		return 0, err
	}
	err = rc.Close()
	return r.totalRead, err
}
