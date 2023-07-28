package filters

import (
	"io"

	"github.com/hhrutter/lzw"
)

type SkipperLZW struct {
	EarlyChange bool // In PDF, written as an integer. Default value: 1 (true).
}

// Skip implements Skipper for an LZWDecode filter.
func (f SkipperLZW) Skip(encoded io.Reader) (int, error) {
	r := newCountReader(encoded)

	rc := lzwDecoder(f.EarlyChange, r)
	_, err := io.ReadAll(rc)
	if err != nil {
		return 0, err
	}
	err = rc.Close()

	return r.totalRead, err
}

func lzwDecoder(earlyChange bool, src io.Reader) io.ReadCloser {
	return lzw.NewReader(src, earlyChange)
}
