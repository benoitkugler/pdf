package filters

import (
	"io"
	"io/ioutil"

	"github.com/benoitkugler/pdf/reader/parser/filters/ccitt"
)

type SkipperCCITT struct {
	Params ccitt.CCITTParams
}

// Skip implements Skipper for a CCITT filter.
func (f SkipperCCITT) Skip(encoded io.Reader) (int, error) {
	r := newCountReader(encoded)
	rc, err := ccitt.NewReader(r, f.Params)
	if err != nil {
		return 0, err
	}
	_, err = ioutil.ReadAll(rc)
	return r.totalRead, err
}
