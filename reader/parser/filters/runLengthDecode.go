package filters

import (
	"bytes"
	"errors"
	"io"
)

type SkipperRunLength struct{}

const eodRunLength = 0x80

func unexpectedEOF(err error) error {
	if err == io.EOF {
		return errors.New("missing EOD marker in encoded stream")
	}
	return err
}

func (f SkipperRunLength) decode(w io.ByteWriter, src io.ByteReader) error {
	for b, err := src.ReadByte(); ; b, err = src.ReadByte() {
		// EOF is an error since we expect the EOD marker
		if err != nil {
			return unexpectedEOF(err)
		}
		if b == eodRunLength { // eod
			return nil
		}
		if b < 0x80 {
			c := int(b) + 1
			for j := 0; j < c; j++ {
				nextChar, err := src.ReadByte()
				if err != nil {
					return unexpectedEOF(err) // EOF here is an error
				}
				w.WriteByte(nextChar)
			}
			continue
		}
		c := 257 - int(b)
		nextChar, err := src.ReadByte()
		if err != nil {
			return unexpectedEOF(err) // EOF here is an error
		}
		for j := 0; j < c; j++ {
			w.WriteByte(nextChar)
		}
	}
}

// Skip implements Skipper for an RunLengthDecode filter.
func (f SkipperRunLength) Skip(encoded io.Reader) (int, error) {
	// we make sure not to read passed EOD
	r := newCountReader(encoded)
	err := f.decode(&bytes.Buffer{}, r)
	return r.totalRead, err
}
