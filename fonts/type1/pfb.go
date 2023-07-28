package type1

import (
	"bytes"
	"fmt"

	"github.com/benoitkugler/pdf/fonts/simpleencodings"
)

// Parse parses an Adobe Type 1 (.pfb) font file, extracting its builtin
// encoding.
func ParseEncoding(pfb *bytes.Reader) (*simpleencodings.Encoding, error) {
	seg1, err := openPfb(pfb)
	if err != nil {
		return nil, fmt.Errorf("invalid .pfb font file: %s", err)
	}

	p := parser{}
	out, err := p.parseASCII(seg1)
	if err != nil {
		return nil, fmt.Errorf("invalid .pfb font file: %s", err)
	}

	return out, nil
}
