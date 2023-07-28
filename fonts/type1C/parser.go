package type1c

// code is adapted from golang.org/x/image/font/sfnt

import (
	"encoding/binary"
	"errors"
	"fmt"

	ps "github.com/benoitkugler/pdf/fonts/psinterpreter"
	"github.com/benoitkugler/pdf/fonts/simpleencodings"
)

var (
	errUnsupportedCFFVersion       = errors.New("unsupported CFF version")

	be = binary.BigEndian
)

const (
	// since SID = 0 means .notdef, we use a reserved value
	// to mean unset
	unsetSID = uint16(0xFFFF)
)

type userStrings [][]byte

// return either the predefined string or the user defined one
func (u userStrings) getString(sid uint16) (string, error) {
	if sid == unsetSID {
		return "", nil
	}
	if sid < 391 {
		return stdStrings[sid], nil
	}
	sid -= 391
	if int(sid) >= len(u) {
		return "", fmt.Errorf("invalid glyph index %d", sid)
	}
	return string(u[sid]), nil
}

// Compact Font Format (CFF) fonts are written in PostScript, a stack-based
// programming language.
//
// A fundamental concept is a DICT, or a key-value map, expressed in reverse
// Polish notation. For example, this sequence of operations:
//   - push the number 379
//   - version operator
//   - push the number 392
//   - Notice operator
//   - etc
//   - push the number 100
//   - push the number 0
//   - push the number 500
//   - push the number 800
//   - FontBBox operator
//   - etc
//
// defines a DICT that maps "version" to the String ID (SID) 379, "Notice" to
// the SID 392, "FontBBox" to the four numbers [100, 0, 500, 800], etc.
//
// The first 391 String IDs (starting at 0) are predefined as per the CFF spec
// Appendix A, in 5176.CFF.pdf referenced below. For example, 379 means
// "001.000". String ID 392 is not predefined, and is mapped by a separate
// structure, the "String INDEX", inside the CFF data. (String ID 391 is also
// not predefined. Specifically for ../testdata/CFFTest.otf, 391 means
// "uni4E2D", as this font contains a glyph for U+4E2D).
//
// The actual glyph vectors are similarly encoded (in PostScript), in a format
// called Type 2 Charstrings. The wire encoding is similar to but not exactly
// the same as CFF's. For example, the byte 0x05 means FontBBox for CFF DICTs,
// but means rlineto (relative line-to) for Type 2 Charstrings. See
// 5176.CFF.pdf Appendix H and 5177.Type2.pdf Appendix A in the PDF files
// referenced below.
//
// The relevant specifications are:
//   - http://wwwimages.adobe.com/content/dam/Adobe/en/devnet/font/pdfs/5176.CFF.pdf
//   - http://wwwimages.adobe.com/content/dam/Adobe/en/devnet/font/pdfs/5177.Type2.pdf
type cffParser struct {
	src    []byte // whole input
	offset int    // current position
}

func (p *cffParser) parse() ([]*simpleencodings.Encoding, error) {
	// header was checked prior to this call

	// Parse the Name INDEX.
	fontNames, err := p.parseNames()
	if err != nil {
		return nil, err
	}

	topDicts, err := p.parseTopDicts()
	if err != nil {
		return nil, err
	}
	// 5176.CFF.pdf section 8 "Top DICT INDEX" says that the count here
	// should match the count of the Name INDEX
	if len(topDicts) != len(fontNames) {
		return nil, fmt.Errorf("top DICT length doest not match Names (%d, %d)", len(topDicts),
			len(fontNames))
	}

	// parse the String INDEX.
	strs, err := p.parseUserStrings()
	if err != nil {
		return nil, err
	}

	out := make([]*simpleencodings.Encoding, len(topDicts))

	// Parse the Global Subrs [Subroutines] INDEX,
	// shared among all fonts.
	_, err = p.parseIndex()
	if err != nil {
		return nil, err
	}

	for i, topDict := range topDicts {
		// Parse the CharStrings INDEX, whose location was found in the Top DICT.
		if err = p.seek(topDict.charStringsOffset); err != nil {
			return nil, err
		}
		charstrings, err := p.parseIndex()
		if err != nil {
			return nil, err
		}
		numGlyphs := uint16(len(charstrings))

		charset, err := p.parseCharset(topDict.charsetOffset, numGlyphs)
		if err != nil {
			return nil, err
		}

		out[i], err = p.parseEncoding(topDict.encodingOffset, numGlyphs, charset, strs)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (p *cffParser) parseTopDicts() ([]topDictData, error) {
	// Parse the Top DICT INDEX.
	instructions, err := p.parseIndex()
	if err != nil {
		return nil, err
	}

	out := make([]topDictData, len(instructions)) // guarded by uint16 max size
	var psi ps.Machine
	for i, buf := range instructions {
		topDict := &out[i]
		if err = psi.Run(buf, nil, nil, topDict); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// parse the general form of an index
func (p *cffParser) parseIndex() ([][]byte, error) {
	count, offSize, err := p.parseIndexHeader()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}

	out := make([][]byte, count)

	stringsLocations := make([]uint32, int(count)+1)
	if err := p.parseIndexLocations(stringsLocations, offSize); err != nil {
		return nil, err
	}

	for i := range out {
		length := stringsLocations[i+1] - stringsLocations[i]
		buf, err := p.read(int(length))
		if err != nil {
			return nil, err
		}
		out[i] = buf
	}
	return out, nil
}

// parse the Name INDEX
func (p *cffParser) parseNames() ([][]byte, error) {
	return p.parseIndex()
}

// parse the String INDEX
func (p *cffParser) parseUserStrings() (userStrings, error) {
	index, err := p.parseIndex()
	return userStrings(index), err
}

// Parse the charset data, whose location was found in the Top DICT.
func (p *cffParser) parseCharset(charsetOffset int32, numGlyphs uint16) ([]uint16, error) {
	// Predefined charset may have offset of 0 to 2 // Table 22
	var charset []uint16
	switch charsetOffset {
	case 0: // ISOAdobe
		charset = charsetISOAdobe[:]
	case 1: // Expert
		charset = charsetExpert[:]
	case 2: // ExpertSubset
		charset = charsetExpertSubset[:]
	default: // custom
		if err := p.seek(charsetOffset); err != nil {
			return nil, err
		}
		buf, err := p.read(1)
		if err != nil {
			return nil, err
		}
		charset = make([]uint16, numGlyphs)
		switch buf[0] { // format
		case 0:
			buf, err = p.read(2 * (int(numGlyphs) - 1)) // ".notdef" is omited, and has an implicit SID of 0
			if err != nil {
				return nil, err
			}
			for i := uint16(1); i < numGlyphs; i++ {
				charset[i] = be.Uint16(buf[2*i-2:])
			}
		case 1:
			for i := uint16(1); i < numGlyphs; {
				buf, err = p.read(3)
				if err != nil {
					return nil, err
				}
				first, nLeft := be.Uint16(buf), uint16(buf[2])
				for j := uint16(0); j <= nLeft && i < numGlyphs; j++ {
					charset[i] = first + j
					i++
				}
			}
		case 2:
			for i := uint16(1); i < numGlyphs; {
				buf, err = p.read(4)
				if err != nil {
					return nil, err
				}
				first, nLeft := be.Uint16(buf), be.Uint16(buf[2:])
				for j := uint16(0); j <= nLeft && i < numGlyphs; j++ {
					charset[i] = first + j
					i++
				}
			}
		default:
			return nil, fmt.Errorf("invalid custom charset format %d", buf[0])
		}
	}
	return charset, nil
}

// Parse the encoding data, whose location was found in the Top DICT.
func (p *cffParser) parseEncoding(encodingOffset int32, numGlyphs uint16, charset []uint16, strs userStrings) (*simpleencodings.Encoding, error) {
	// Predefined encoding may have offset of 0 to 1 // Table 16
	switch encodingOffset {
	case 0: // Standard
		return &simpleencodings.AdobeStandard, nil
	case 1: // Expert
		return &expertEncoding, nil
	default: // custom
		var encoding simpleencodings.Encoding
		if err := p.seek(encodingOffset); err != nil {
			return nil, err
		}
		buf, err := p.read(2)
		if err != nil {
			return nil, err
		}
		format, size, charL := buf[0], int32(buf[1]), int32(len(charset))
		// high order bit may be set for supplemental encoding data
		switch f := format & 0xf; f { // 0111_1111
		case 0:
			if size > int32(numGlyphs) { // truncate
				size = int32(numGlyphs)
			}
			buf, err = p.read(int(size))
			if err != nil {
				return nil, err
			}
			for i := int32(1); i < size && i < charL; i++ {
				c := buf[i-1]
				encoding[c], err = strs.getString(charset[i])
				if err != nil {
					return nil, err
				}
			}
		case 1:
			buf, err = p.read(2 * int(size))
			if err != nil {
				return nil, err
			}
			nCodes := int32(1)
			for i := int32(0); i < size; i++ {
				c, nLeft := buf[2*i], buf[2*i+1]
				for j := byte(0); j < nLeft && nCodes < int32(numGlyphs) && nCodes < charL; j++ {
					encoding[c], err = strs.getString(charset[nCodes])
					if err != nil {
						return nil, err
					}
					nCodes++
					c++
				}
			}
		default:
			return nil, fmt.Errorf("invalid custom encoding format %d", f)
		}

		if format&0x80 != 0 { // 1000_000
			nSupsBuf, err := p.read(1)
			if err != nil {
				return nil, err
			}
			buf, err := p.read(3 * int(nSupsBuf[0]))
			if err != nil {
				return nil, err
			}
			for i := byte(0); i < nSupsBuf[0]; i++ {
				code, sid := buf[3*i], be.Uint16(buf[3*i+1:])
				encoding[code], err = strs.getString(sid)
				if err != nil {
					return nil, err
				}
			}
		}
		return &encoding, nil
	}
}

// read returns the n bytes from p.offset and advances p.offset by n.
func (p *cffParser) read(n int) ([]byte, error) {
	if n < 0 || len(p.src) < p.offset+n {
		return nil, errors.New("invalid CFF font file (EOF)")
	}
	out := p.src[p.offset : p.offset+n]
	p.offset += n
	return out, nil
}

// skip advances p.offset by n.
func (p *cffParser) skip(n int) error {
	if len(p.src) < p.offset+n {
		return errors.New("invalid CFF font file (EOF)")
	}
	p.offset += n
	return nil
}

func (p *cffParser) seek(offset int32) error {
	if offset < 0 || len(p.src) < int(offset) {
		return errors.New("invalid CFF font file (EOF)")
	}
	p.offset = int(offset)
	return nil
}

func bigEndian(b []byte) uint32 {
	switch len(b) {
	case 1:
		return uint32(b[0])
	case 2:
		return uint32(b[0])<<8 | uint32(b[1])
	case 3:
		return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
	case 4:
		return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	}
	panic("unreachable")
}

func (p *cffParser) parseIndexHeader() (count uint16, offSize int32, err error) {
	buf, err := p.read(2)
	if err != nil {
		return 0, 0, err
	}
	count = be.Uint16(buf)
	// 5176.CFF.pdf section 5 "INDEX Data" says that "An empty INDEX is
	// represented by a count field with a 0 value and no additional fields.
	// Thus, the total size of an empty INDEX is 2 bytes".
	if count == 0 {
		return count, 0, nil
	}
	buf, err = p.read(1)
	if err != nil {
		return 0, 0, err
	}
	offSize = int32(buf[0])
	if offSize < 1 || 4 < offSize {
		return 0, 0, fmt.Errorf("invalid offset size %d", offSize)
	}
	return count, offSize, nil
}

func (p *cffParser) parseIndexLocations(dst []uint32, offSize int32) error {
	if len(dst) == 0 {
		return nil
	}
	buf, err := p.read(len(dst) * int(offSize))
	if err != nil {
		return err
	}

	prev := uint32(0)
	for i := range dst {
		loc := bigEndian(buf[:offSize])
		buf = buf[offSize:]

		// Locations are off by 1 byte. 5176.CFF.pdf section 5 "INDEX Data"
		// says that "Offsets in the offset array are relative to the byte that
		// precedes the object data... This ensures that every object has a
		// corresponding offset which is always nonzero".
		if loc == 0 {
			return errors.New("invalid CFF index locations (0)")
		}
		loc--

		// In the same paragraph, "Therefore the first element of the offset
		// array is always 1" before correcting for the off-by-1.
		if i == 0 {
			if loc != 0 {
				return errors.New("invalid CFF index locations (not 0)")
			}
		} else if loc < prev { // Check that locations are increasing
			return errors.New("invalid CFF index locations (not increasing)")
		}

		// Check that locations are in bounds.
		if uint32(len(p.src)-p.offset) < loc {
			return errors.New("invalid CFF index locations (out of bounds)")
		}

		dst[i] = uint32(p.offset) + loc
		prev = loc
	}
	return nil
}

// topDictData contains fields specific to the Top DICT context.
type topDictData struct {
	charsetOffset     int32
	encodingOffset    int32
	charStringsOffset int32

	privateDictOffset int32
	privateDictLength int32
}

func (topDict *topDictData) Context() ps.PsContext { return ps.TopDict }

func (topDict *topDictData) Apply(op ps.PsOperator, state *ps.Machine) error {
	ops := topDictOperators[0]
	if op.IsEscaped {
		ops = topDictOperators[1]
	}
	if int(op.Operator) >= len(ops) {
		return fmt.Errorf("invalid operator %s in Top Dict", op)
	}
	opFunc := ops[op.Operator]
	if opFunc.run == nil {
		return fmt.Errorf("invalid operator %s in Top Dict", op)
	}
	if state.ArgStack.Top < opFunc.numPop {
		return fmt.Errorf("invalid number of arguments for operator %s in Top Dict", op)
	}
	err := opFunc.run(topDict, state)
	if err != nil {
		return err
	}
	err = state.ArgStack.PopN(opFunc.numPop)
	return err
}

// The Top DICT operators are defined by 5176.CFF.pdf Table 9 "Top DICT
// Operator Entries" and Table 10 "CIDFont Operator Extensions".
type topDictOperator struct {
	// run is the function that implements the operator. Nil means that we
	// ignore the operator, other than popping its arguments off the stack.
	run func(*topDictData, *ps.Machine) error

	// numPop is the number of stack values to pop. -1 means "array" and -2
	// means "delta" as per 5176.CFF.pdf Table 6 "Operand Types".
	numPop int32
}

func topDictNoOp(*topDictData, *ps.Machine) error { return nil }

var topDictOperators = [2][]topDictOperator{
	// 1-byte operators.
	{
		0:  {topDictNoOp, +1 /*version*/},
		1:  {topDictNoOp, +1 /*Notice*/},
		2:  {topDictNoOp, +1 /*FullName*/},
		3:  {topDictNoOp, +1 /*FamilyName*/},
		4:  {topDictNoOp, +1 /*Weight*/},
		5:  {topDictNoOp, -1 /*FontBBox*/},
		13: {topDictNoOp, +1 /*UniqueID*/},
		14: {topDictNoOp, -1 /*XUID*/},
		15: {func(t *topDictData, s *ps.Machine) error {
			t.charsetOffset = s.ArgStack.Vals[s.ArgStack.Top-1]
			return nil
		}, +1 /*charset*/},
		16: {func(t *topDictData, s *ps.Machine) error {
			t.encodingOffset = s.ArgStack.Vals[s.ArgStack.Top-1]
			return nil
		}, +1 /*Encoding*/},
		17: {func(t *topDictData, s *ps.Machine) error {
			t.charStringsOffset = s.ArgStack.Vals[s.ArgStack.Top-1]
			return nil
		}, +1 /*CharStrings*/},
		18: {func(t *topDictData, s *ps.Machine) error {
			t.privateDictLength = s.ArgStack.Vals[s.ArgStack.Top-2]
			t.privateDictOffset = s.ArgStack.Vals[s.ArgStack.Top-1]
			return nil
		}, +2 /*Private*/},
	},
	// 2-byte operators. The first byte is the escape byte.
	{
		0: {topDictNoOp, +1 /*Copyright*/},
		1: {topDictNoOp, +1 /*isFixedPitch*/},
		2: {topDictNoOp, +1 /*ItalicAngle*/},
		3: {topDictNoOp, +1 /*UnderlinePosition*/},
		4: {topDictNoOp, +1 /*UnderlineThickness*/},
		5: {topDictNoOp, +1 /*PaintType*/},
		6: {func(_ *topDictData, i *ps.Machine) error {
			if version := i.ArgStack.Vals[i.ArgStack.Top-1]; version != 2 {
				return fmt.Errorf("charstring type %d not supported", version)
			}
			return nil
		}, +1 /*CharstringType*/},
		7:  {topDictNoOp, -1 /*FontMatrix*/},
		8:  {topDictNoOp, +1 /*StrokeWidth*/},
		20: {topDictNoOp, +1 /*SyntheticBase*/},
		21: {topDictNoOp, +1 /*PostScript*/},
		22: {topDictNoOp, +1 /*BaseFontName*/},
		23: {topDictNoOp, -2 /*BaseFontBlend*/},
		30: {topDictNoOp, +3 /*ROS*/},
		31: {topDictNoOp, +1 /*CIDFontVersion*/},
		32: {topDictNoOp, +1 /*CIDFontRevision*/},
		33: {topDictNoOp, +1 /*CIDFontType*/},
		34: {topDictNoOp, +1 /*CIDCount*/},
		35: {topDictNoOp, +1 /*UIDBase*/},
		36: {topDictNoOp, +1 /*FDArray*/},
		37: {topDictNoOp, +1 /*FDSelect*/},
		38: {topDictNoOp, +1 /*FontName*/},
	},
}
