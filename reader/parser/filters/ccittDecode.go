package filters

import (
	"bytes"
	"io/ioutil"

	"github.com/benoitkugler/pdf/reader/parser/filters/ccitt"
)

type SkipperCCITT struct {
	Params ccitt.CCITTParams
}

// Skip implements Skipper for a CCITT filter.
func (f SkipperCCITT) Skip(encoded []byte) (int, error) {
	r := bytes.NewReader(encoded)
	rc, err := ccitt.NewReader(r, f.Params)
	if err != nil {
		return 0, err
	}
	_, err = ioutil.ReadAll(rc)
	return len(encoded) - r.Len(), err
}
