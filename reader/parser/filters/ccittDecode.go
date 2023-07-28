package filters

import (
	"bufio"
	"io"

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
	_, err = io.ReadAll(rc)
	return r.totalRead, err
}

func ccittDecoder(params ccitt.CCITTParams, src io.Reader) (io.Reader, error) {
	return ccitt.NewReader(bufio.NewReader(src), params)
}
