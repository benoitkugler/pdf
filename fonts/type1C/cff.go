// Package type1c provides a parser for the CFF font format
// defined at https://www.adobe.com/content/dam/acom/en/devnet/font/pdfs/5176.CFF.pdf.
// It can be used to read standalone CFF font files, but is mainly used
// through the truetype package to read embedded CFF glyph tables.
package type1c

import (
	"bytes"
	"errors"
	"io"

	"github.com/benoitkugler/pdf/fonts/simpleencodings"
)

// Parse parse a .cff font file, extracting its encoding
// Although CFF enables multiple font or CIDFont programs to be bundled together in a
// single file, embedded CFF font file in PDF or in TrueType/OpenType fonts
// shall consist of exactly one font or CIDFont. Thus, this function
// returns an error if the file contains more than one font.
func ParseEncoding(file *bytes.Reader) (*simpleencodings.Encoding, error) {
	fonts, err := parse(file)
	if err != nil {
		return nil, err
	}
	if len(fonts) != 1 {
		return nil, errors.New("only one CFF font is allowed in embedded files")
	}
	return fonts[0], nil
}

func parse(file *bytes.Reader) ([]*simpleencodings.Encoding, error) {
	// read 4 bytes to check if its a supported CFF file
	var buf [4]byte
	file.Read(buf[:])
	if buf[0] != 1 || buf[1] != 0 || buf[2] != 4 {
		return nil, errUnsupportedCFFVersion
	}
	file.Seek(0, io.SeekStart)

	// if this is really needed, we can modify the parser to directly use `file`
	// without reading all in memory
	input, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	p := cffParser{src: input}
	p.skip(4)
	out, err := p.parse()
	if err != nil {
		return nil, err
	}
	return out, nil
}
