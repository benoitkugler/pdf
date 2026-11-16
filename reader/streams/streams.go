// Package streams provides decoding of PDF stream objects.
//
// PDF streams may be encoded with a chain of filters; this package
// decodes them. Since decoding is only needed when reading a PDF,
// it is kept out of the model package, which is also used to write files.
package streams

import (
	"bytes"
	"io"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/parser/filters"
)

// DecodeFilter returns a Reader decoding `r` according to the filter `fi`.
func DecodeFilter(fi model.Filter, r io.Reader) (io.Reader, error) {
	// special case for Crypt filters
	if fi.Name == "Crypt" {
		return r, nil
	}

	return filters.NewFilter(string(fi.Name), fi.DecodeParms, r)
}

// DecodeFilters accumulates the filters to produce a Reader,
// decoding `r`.
func DecodeFilters(fs model.Filters, r io.Reader) (io.Reader, error) {
	var err error
	for _, fi := range fs {
		r, err = DecodeFilter(fi, r)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

// DecodeStream attempts to apply the Filters to decode its content.
// Be aware that not all PDF filters are supported (see filters.List).
func DecodeStream(s model.Stream) ([]byte, error) {
	r, err := DecodeFilters(s.Filter, bytes.NewReader(s.Content))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// DecodeContentStream decodes the underlying stream of `ct`.
func DecodeContentStream(ct model.ContentStream) ([]byte, error) {
	return DecodeStream(ct.Stream)
}

// DecodePageContents reads each content stream of `p` and returns the
// aggregated one.
func DecodePageContents(p *model.PageObject) ([]byte, error) {
	var totalPageContent []byte
	for _, ct := range p.Contents {
		ctContent, err := DecodeContentStream(ct)
		if err != nil {
			return nil, err
		}
		totalPageContent = append(totalPageContent, ctContent...)
	}

	return totalPageContent, nil
}
