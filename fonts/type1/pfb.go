package type1

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

const (
	// the pdf header length.
	// (start-marker (1 byte), ascii-/binary-marker (1 byte), size (4 byte))
	// 3*6 == 18
	pfbHeaderLength = 18

	// the start marker.
	startMarker = 0x80

	// the ascii marker.
	asciiMarker = 0x01

	// the binary marker.
	binaryMarker = 0x02
)

// The record types in the pfb-file.
var pfbRecords = [...]int{asciiMarker, binaryMarker, asciiMarker}

// PFBFont exposes the content of a .pfb file.
// The main field, regarding PDF processing, is the Encoding
// entry, which defines the "builtin encoding" of the font.
type PFBFont struct {
	Encoding Encoding

	FontName    string
	PaintType   int
	FontType    int
	UniqueID    int
	StrokeWidth Fl
	FontID      string
	FontMatrix  []Fl
	FontBBox    []Fl

	Version            string
	Notice             string
	FullName           string
	FamilyName         string
	Weight             string
	ItalicAngle        Fl
	UnderlinePosition  Fl
	UnderlineThickness Fl
	IsFixedPitch       bool
}

type stream struct {
	*bytes.Reader
}

func (s stream) read() int {
	c, err := s.Reader.ReadByte()
	if err != nil {
		return -1
	}
	return int(c)
}

// OpenPfb fetch the segments of a .pfb font file.
func OpenPfb(pfb []byte) (segment1, segment2 []byte, err error) {
	in := stream{bytes.NewReader(pfb)}
	pfbdata := make([]byte, len(pfb)-pfbHeaderLength)
	var lengths [len(pfbRecords)]int
	pointer := 0
	for records := 0; records < len(pfbRecords); records++ {
		if in.read() != startMarker {
			return nil, nil, errors.New("Start marker missing")
		}

		if in.read() != pfbRecords[records] {
			return nil, nil, errors.New("Incorrect record type")
		}

		size := in.read()
		size += in.read() << 8
		size += in.read() << 16
		size += in.read() << 24
		lengths[records] = size
		if pointer >= len(pfbdata) {
			return nil, nil, errors.New("attempted to read past EOF")
		}
		inL := io.LimitedReader{R: in, N: int64(size)}
		got, err := inL.Read(pfbdata[pointer:])
		if err != nil {
			return nil, nil, err
		}
		pointer += got
	}

	return pfbdata[0:lengths[0]], pfbdata[lengths[0] : lengths[0]+lengths[1]], nil
}

// ParsePFBFile is a convenience wrapper, reading and
// parsing a .pfb font file.
func ParsePFBFile(pfb []byte) (PFBFont, error) {
	seg1, _, err := OpenPfb(pfb)
	if err != nil {
		return PFBFont{}, fmt.Errorf("invalid .pfb font file: %s", err)
	}
	info, err := Parse(seg1)
	if err != nil {
		return PFBFont{}, fmt.Errorf("invalid .pfb font file: %s", err)
	}
	return info, nil
}
