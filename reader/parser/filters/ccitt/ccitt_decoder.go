package ccitt

import (
	"errors"
	"fmt"
	"io"
	"math"
)

// CCITTParams holds the parameters of an encoded CCITT input.
// `DamagedRowsBeforeError` is not supported.
type CCITTParams struct {
	Encoding                                int32 // K parameter
	Columns, Rows                           int32
	EndOfBlock, EndOfLine, ByteAlign, Black bool
}

// NewReader return a ready to use Reader, decoding the CCITT
// `src`.
// The resultant byte stream is one bit per pixel (MSB first), with 1 meaning
// white and 0 meaning black. Each row in the result is byte-aligned.
//
// A zero height, means that the image height is not known in advance. In thise
// case, EndOfBlock must be true.
func NewReader(src io.ByteReader, params CCITTParams) (*CCITTDecoder, error) {
	out := CCITTDecoder{src: src, p: params}

	if out.p.Columns < 1 {
		out.p.Columns = 1
	} else if out.p.Columns > math.MaxInt32-2 {
		out.p.Columns = math.MaxInt32 - 2
	}
	out.codingLine = make([]int32, out.p.Columns+1)
	out.refLine = make([]int32, out.p.Columns+2)
	out.codingLine[0] = out.p.Columns
	out.nextLine2D = out.p.Encoding < 0

	err := out.initialize()

	return &out, err
}

type short = int16

func bad2DCode(code int32) error {
	return fmt.Errorf("bad 2D code %x in CCITT stream", code)
}

type CCITTDecoder struct {
	src io.ByteReader // the input source

	inputBits           int32 // number of bits in input buffer
	inputBuf            uint32
	outputBits          int32
	eof                 bool
	nextLine2D          bool
	refLine, codingLine []int32
	a0i                 int32 // index into codingLine
	row                 int32 // current row

	// user parameters
	p CCITTParams
}

// skip any initial zero bits and end-of-line marker, and get the 2D
// encoding tag
func (st *CCITTDecoder) initialize() error {
	code1, err := st.lookBits(12)
	if err != nil {
		return err
	}
	for code1 == 0 {
		st.eatBits(1)
		code1, err = st.lookBits(12)
		if err != nil {
			return err
		}
	}
	if code1 == 0x001 {
		st.eatBits(12)
		st.p.EndOfLine = true
	}
	if st.p.Encoding > 0 {
		b, err := st.lookBits(1)
		if err != nil {
			return err
		}
		st.nextLine2D = b == 0
		st.eatBits(1)
	}
	return nil
}

func (st *CCITTDecoder) Read(p []byte) (int, error) {
	i := 0
	for ; i < len(p); i++ {
		b, err := st.ReadByte()
		if err != nil {
			return i, err
		}
		p[i] = b
	}
	return len(p), nil
}

func (st *CCITTDecoder) ReadByte() (byte, error) {
	if st.outputBits == 0 { // read the next row
		err := st.readRow()
		if err != nil {
			return 0, err
		}
	}
	// get a byte
	out, err := st.getByte()
	return out, err
}

func (st *CCITTDecoder) eatBits(n int32) {
	st.inputBits -= n
	if st.inputBits < 0 {
		st.inputBits = 0
	}
}

// return an error only if the underlying reader error is different from EOF
// n must be <= 32
func (st *CCITTDecoder) lookBits(n int32) (short, error) {
	for st.inputBits < n {
		c, err := st.src.ReadByte()
		// first check for EOF ...
		if err == io.EOF {
			if st.inputBits == 0 {
				return codeEOF, nil
			}
			// near the end of the stream, the caller may ask for more bits
			// than are available, but there may still be a valid code in
			// however many bits are available -- we need to return correct
			// data in this case
			return short((st.inputBuf << (n - st.inputBits)) & (0xffffffff >> (32 - n))), nil
		}
		// ... then for an unexpected error
		if err != nil {
			return 0, err
		}
		st.inputBuf = (st.inputBuf << 8) + uint32(c)
		st.inputBits += 8
	}
	out := short((st.inputBuf >> (st.inputBits - n)) & (0xffffffff >> (32 - n)))
	return out, nil
}

func (st *CCITTDecoder) getTwoDimCode() (int32, error) {
	var (
		code short
		err  error
	)

	if st.p.EndOfBlock {
		code, err = st.lookBits(7)
		if err != nil {
			return 0, err
		}
		if code != codeEOF {
			p := twoDimTab1[code]
			if p.bits > 0 {
				st.eatBits(int32(p.bits))
				return int32(p.n), nil
			}
		}
	} else {
		for n := int32(1); n <= 7; n++ {
			code, err = st.lookBits(n)
			if err != nil {
				return 0, err
			}
			if code == codeEOF {
				break
			}
			if n < 7 {
				code <<= 7 - n
			}
			p := twoDimTab1[code]
			if int32(p.bits) == n {
				st.eatBits(n)
				return int32(p.n), nil
			}
		}
	}
	return codeEOF, bad2DCode(int32(code))
}

func (st *CCITTDecoder) getWhiteCode() (short, error) {
	var (
		code short
		err  error
	)
	if st.p.EndOfBlock {
		code, err = st.lookBits(12)
		if err != nil {
			return 0, err
		}
		if code == codeEOF {
			return 1, nil
		}
		var p ccittCode
		if (code >> 5) == 0 {
			p = whiteTab1[code]
		} else {
			p = whiteTab2[code>>3]
		}
		if p.bits > 0 {
			st.eatBits(int32(p.bits))
			return p.n, nil
		}
	} else {
		for n := int32(1); n <= 9; n++ {
			code, err = st.lookBits(n)
			if err != nil {
				return 0, err
			}
			if code == codeEOF {
				return 1, nil
			}
			if n < 9 {
				code <<= 9 - n
			}
			p := whiteTab2[code]
			if int32(p.bits) == n {
				st.eatBits(n)
				return p.n, nil
			}
		}
		for n := int32(11); n <= 12; n++ {
			code, err = st.lookBits(n)
			if err != nil {
				return 0, err
			}
			if code == codeEOF {
				return 1, nil
			}
			if n < 12 {
				code <<= 12 - n
			}
			p := whiteTab1[code]
			if int32(p.bits) == n {
				st.eatBits(n)
				return p.n, nil
			}
		}
	}
	// eat a bit and return a positive number so that the caller doesn't
	// go into an infinite loop
	st.eatBits(1)
	return 1, fmt.Errorf("bad white code (%x) in CCITTFax stream", code)
}

func (st *CCITTDecoder) getBlackCode() (short, error) {
	var (
		code short
		err  error
	)
	if st.p.EndOfBlock {
		code, err = st.lookBits(13)
		if err != nil {
			return 0, err
		}
		if code == codeEOF {
			return 1, nil
		}
		var p ccittCode
		if (code >> 7) == 0 {
			p = blackTab1[code]
		} else if (code>>9) == 0 && (code>>7) != 0 {
			p = blackTab2[(code>>1)-64]
		} else {
			p = blackTab3[code>>7]
		}
		if p.bits > 0 {
			st.eatBits(int32(p.bits))
			return p.n, nil
		}
	} else {
		for n := int32(2); n <= 6; n++ {
			code, err = st.lookBits(n)
			if err != nil {
				return 0, err
			}
			if code == codeEOF {
				return 1, nil
			}

			if n < 6 {
				code <<= 6 - n
			}
			p := blackTab3[code]
			if int32(p.bits) == n {
				st.eatBits(n)
				return p.n, nil
			}
		}
		for n := int32(7); n <= 12; n++ {
			code, err = st.lookBits(n)
			if err != nil {
				return 0, err
			}
			if code == codeEOF {
				return 1, nil
			}

			if n < 12 {
				code <<= 12 - n
			}
			if code >= 64 {
				p := blackTab2[code-64]
				if int32(p.bits) == n {
					st.eatBits(n)
					return p.n, nil
				}
			}
		}
		for n := int32(10); n <= 13; n++ {
			code, err = st.lookBits(n)
			if err != nil {
				return 0, err
			}
			if code == codeEOF {
				return 1, nil
			}

			if n < 13 {
				code <<= 13 - n
			}
			p := blackTab1[code]
			if int32(p.bits) == n {
				st.eatBits(n)
				return p.n, nil
			}
		}
	}
	// eat a bit and return a positive number so that the caller doesn't
	// go into an infinite loop
	st.eatBits(1)
	return 1, fmt.Errorf("bad black code (%x) in CCITTFax stream", code)
}

func (st *CCITTDecoder) addPixels(a1, blackPixels int32) error {
	if a1 > st.codingLine[st.a0i] {
		if a1 > st.p.Columns {
			return fmt.Errorf("CCITTFax row is wrong length (%d)", a1)
		}
		if (st.a0i&1)^blackPixels != 0 {
			st.a0i++
		}
		st.codingLine[st.a0i] = a1
	}
	return nil
}

func (st *CCITTDecoder) addPixelsNeg(a1, blackPixels int32) error {
	if a1 > st.codingLine[st.a0i] {
		if a1 > st.p.Columns {
			return fmt.Errorf("CCITTFax row is wrong length (%d)", a1)
		}
		if (st.a0i&1)^blackPixels != 0 {
			st.a0i++
		}
		st.codingLine[st.a0i] = a1
	} else if a1 < st.codingLine[st.a0i] {
		if a1 < 0 {
			return errors.New("invalid CCITTFax code")
		}
		for st.a0i > 0 && a1 <= st.codingLine[st.a0i-1] {
			st.a0i--
		}
		st.codingLine[st.a0i] = a1
	}
	return nil
}

func (st *CCITTDecoder) readRow() error {
	var (
		gotEOL bool
		err    error
	)

	// if at eof just return EOF
	if st.eof {
		return io.EOF
	}

	// 2-D encoding
	if st.nextLine2D {
		err = st.encoding2D()
	} else { // 1-D encoding
		err = st.encoding1D()
	}
	if err != nil {
		return err
	}

	// check for end-of-line marker, skipping over any extra zero bits
	// (if EncodedByteAlign is true and EndOfLine is false, there can
	// be "false" EOL markers -- i.e., if the last n unused bits in
	// row i are set to zero, and the first 11-n bits in row i+1
	// happen to be zero -- so we don't look for EOL markers in this
	// case)
	gotEOL = false
	if !st.p.EndOfBlock && st.row == st.p.Rows-1 {
		st.eof = true
	} else if st.p.EndOfLine || !st.p.ByteAlign {
		code1, err := st.lookBits(12)
		if err != nil {
			return err
		}
		if st.p.EndOfLine {
			for code1 != codeEOF && code1 != 0x001 {
				st.eatBits(1)
				code1, err = st.lookBits(12)
				if err != nil {
					return err
				}
			}
		} else {
			for code1 == 0 {
				st.eatBits(1)
				code1, err = st.lookBits(12)
				if err != nil {
					return err
				}
			}
		}
		if code1 == 0x001 {
			st.eatBits(12)
			gotEOL = true
		}
	}

	// byte-align the row
	// (Adobe apparently doesn't do byte alignment after EOL markers
	// -- I've seen CCITT image data streams in two different formats,
	// both with the byteAlign flag set:
	//   1. xx:x0:01:yy:yy
	//   2. xx:00:1y:yy:yy
	// where xx is the previous line, yy is the next line, and colons
	// separate bytes.)
	if st.p.ByteAlign && !gotEOL {
		st.inputBits &= ^7
	}

	// check for end of stream
	code, err := st.lookBits(1)
	if err != nil {
		return err
	}
	if code == codeEOF {
		st.eof = true
	}

	// get 2D encoding tag
	if !st.eof && st.p.Encoding > 0 {
		b, err := st.lookBits(1)
		if err != nil {
			return err
		}
		st.nextLine2D = b != 0
		st.eatBits(1)
	}

	// check for end-of-block marker
	if st.p.EndOfBlock && !st.p.EndOfLine && st.p.ByteAlign {
		// in this case, we didn't check for an EOL code above, so we
		// need to check here
		code1, err := st.lookBits(24)
		if err != nil {
			return err
		}
		if code1 == 0x001001 {
			st.eatBits(12)
			gotEOL = true
		}
	}
	if st.p.EndOfBlock && gotEOL {
		code1, err := st.lookBits(12)
		if err != nil {
			return err
		}
		if code1 == 0x001 {
			st.eatBits(12)
			if st.p.Encoding > 0 {
				_, err := st.lookBits(1)
				if err != nil {
					return err
				}
				st.eatBits(1)
			}
			if st.p.Encoding >= 0 {
				for i := 0; i < 4; i++ {
					code1, err := st.lookBits(12)
					if err != nil {
						return err
					}
					if code1 != 0x001 {
						return errors.New("bad RTC code in CCITTFax stream")
					}
					st.eatBits(12)
					if st.p.Encoding > 0 {
						_, err := st.lookBits(1)
						if err != nil {
							return err
						}
						st.eatBits(1)
					}
				}
			}
			st.eof = true
		}
	}

	// set up for output
	if st.codingLine[0] > 0 {
		st.a0i = 0
		st.outputBits = st.codingLine[st.a0i]
	} else {
		st.a0i = 1
		st.outputBits = st.codingLine[st.a0i]
	}

	st.row++
	return nil
}

func (st *CCITTDecoder) getByte() (byte, error) {
	var buf byte
	if st.outputBits >= 8 {
		buf = 0xff
		if st.a0i&1 != 0 {
			buf = 0x00
		}
		st.outputBits -= 8
		if st.outputBits == 0 && st.codingLine[st.a0i] < st.p.Columns {
			st.a0i++
			st.outputBits = st.codingLine[st.a0i] - st.codingLine[st.a0i-1]
		}
	} else {
		bits := int32(8)
		buf = 0
		var err error
		bits, buf, err = st.getOneByteLoopBody(bits, buf)
		if err != nil {
			return 0, err
		}
		for bits != 0 {
			bits, buf, err = st.getOneByteLoopBody(bits, buf)
			if err != nil {
				return 0, err
			}
		}
	}
	if st.p.Black {
		buf = ^buf
	}
	return buf, nil
}

func (st *CCITTDecoder) encoding2D() error {
	var i, b1i, blackPixels int32
	for i = 0; i < st.p.Columns && st.codingLine[i] < st.p.Columns; i++ {
		st.refLine[i] = st.codingLine[i]
	}
	for ; i < st.p.Columns+2; i++ {
		st.refLine[i] = st.p.Columns
	}
	st.codingLine[0] = 0
	st.a0i = 0
	// invariant:
	// st.refLine[b1i-1] <= st.codingLine[st.a0i] < st.refLine[b1i] < st.refLine[b1i+1]
	//                                                             <= st.columns
	// exception at left edge:
	//   st.codingLine[st.a0i = 0] = st.refLine[b1i = 0] = 0 is possible
	// exception at right edge:
	//   st.refLine[b1i] = st.refLine[b1i+1] = st.columns is possible
	for st.codingLine[st.a0i] < st.p.Columns {
		code1, err := st.getTwoDimCode()
		if err != nil {
			return err
		}
		switch code1 {
		case twoDimPass:
			if b1i+1 < st.p.Columns+2 {
				err = st.addPixels(st.refLine[b1i+1], blackPixels)
				if err != nil {
					return err
				}
				if st.refLine[b1i+1] < st.p.Columns {
					b1i += 2
				}
			}
		case twoDimHoriz:
			var code1, code2 int32
			if blackPixels != 0 {
				code3, err := st.getBlackCode()
				if err != nil {
					return err
				}
				code1 += int32(code3)
				for code3 >= 64 {
					code3, err = st.getBlackCode()
					if err != nil {
						return err
					}
					code1 += int32(code3)
				}
				code3, err = st.getWhiteCode()
				if err != nil {
					return err
				}
				code2 += int32(code3)
				for code3 >= 64 {
					code3, err = st.getWhiteCode()
					if err != nil {
						return err
					}
					code2 += int32(code3)
				}
			} else {
				code3, err := st.getWhiteCode()
				if err != nil {
					return err
				}
				code1 += int32(code3)
				for code3 >= 64 {
					code3, err = st.getWhiteCode()
					if err != nil {
						return err
					}
					code1 += int32(code3)
				}

				code3, err = st.getBlackCode()
				if err != nil {
					return err
				}
				code2 += int32(code3)
				for code3 >= 64 {
					code3, err = st.getBlackCode()
					if err != nil {
						return err
					}
					code2 += int32(code3)
				}
			}
			err = st.addPixels(st.codingLine[st.a0i]+code1, blackPixels)
			if err != nil {
				return err
			}
			if st.codingLine[st.a0i] < st.p.Columns {
				err = st.addPixels(st.codingLine[st.a0i]+code2, blackPixels^1)
				if err != nil {
					return err
				}
			}
			for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
				b1i += 2
				if b1i > st.p.Columns+1 {
					return bad2DCode(code1)
				}
			}
		case twoDimVertR3:
			if b1i > st.p.Columns+1 {
				return bad2DCode(code1)
			}
			err = st.addPixels(st.refLine[b1i]+3, blackPixels)
			if err != nil {
				return err
			}
			blackPixels ^= 1
			if st.codingLine[st.a0i] < st.p.Columns {
				b1i++
				for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
					b1i += 2
					if b1i > st.p.Columns+1 {
						return bad2DCode(code1)
					}
				}
			}
		case twoDimVertR2:
			if b1i > st.p.Columns+1 {
				return bad2DCode(code1)
			}
			err = st.addPixels(st.refLine[b1i]+2, blackPixels)
			if err != nil {
				return err
			}
			blackPixels ^= 1
			if st.codingLine[st.a0i] < st.p.Columns {
				b1i++
				for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
					b1i += 2
					if b1i > st.p.Columns+1 {
						return bad2DCode(code1)
					}
				}
			}
		case twoDimVertR1:
			if b1i > st.p.Columns+1 {
				return bad2DCode(code1)
			}
			err = st.addPixels(st.refLine[b1i]+1, blackPixels)
			if err != nil {
				return err
			}
			blackPixels ^= 1
			if st.codingLine[st.a0i] < st.p.Columns {
				b1i++
				for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
					b1i += 2
					if b1i > st.p.Columns+1 {
						return bad2DCode(code1)
					}
				}
			}
		case twoDimVert0:
			if b1i > st.p.Columns+1 {
				return bad2DCode(code1)
			}
			st.addPixels(st.refLine[b1i], blackPixels)
			blackPixels ^= 1
			if st.codingLine[st.a0i] < st.p.Columns {
				b1i++
				for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
					b1i += 2
					if b1i > st.p.Columns+1 {
						return bad2DCode(code1)
					}
				}
			}
		case twoDimVertL3:
			if b1i > st.p.Columns+1 {
				return bad2DCode(code1)
			}
			err = st.addPixelsNeg(st.refLine[b1i]-3, blackPixels)
			if err != nil {
				return err
			}
			blackPixels ^= 1
			if st.codingLine[st.a0i] < st.p.Columns {
				if b1i > 0 {
					b1i--
				} else {
					b1i++
				}
				for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
					b1i += 2
					if b1i > st.p.Columns+1 {
						return bad2DCode(code1)
					}
				}
			}
		case twoDimVertL2:
			if b1i > st.p.Columns+1 {
				return bad2DCode(code1)
			}
			st.addPixelsNeg(st.refLine[b1i]-2, blackPixels)
			blackPixels ^= 1
			if st.codingLine[st.a0i] < st.p.Columns {
				if b1i > 0 {
					b1i--
				} else {
					b1i++
				}
				for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
					b1i += 2
					if b1i > st.p.Columns+1 {
						return bad2DCode(code1)
					}
				}
			}
		case twoDimVertL1:
			if b1i > st.p.Columns+1 {
				return bad2DCode(code1)
			}
			st.addPixelsNeg(st.refLine[b1i]-1, blackPixels)
			blackPixels ^= 1
			if st.codingLine[st.a0i] < st.p.Columns {
				if b1i > 0 {
					b1i--
				} else {
					b1i++
				}
				for st.refLine[b1i] <= st.codingLine[st.a0i] && st.refLine[b1i] < st.p.Columns {
					b1i += 2
					if b1i > st.p.Columns+1 {
						return bad2DCode(code1)
					}
				}
			}
		case codeEOF:
			err = st.addPixels(st.p.Columns, 0)
			if err != nil {
				return err
			}
			st.eof = true
		default:
			return bad2DCode(code1)
		}
	}
	return nil
}

func (st *CCITTDecoder) encoding1D() error {
	st.codingLine[0] = 0
	st.a0i = 0
	var blackPixels int32
	for st.codingLine[st.a0i] < st.p.Columns {
		var code1 int32
		if blackPixels != 0 {
			code3, err := st.getBlackCode()
			if err != nil {
				return err
			}
			code1 += int32(code3)
			for code3 >= 64 {
				code3, err = st.getBlackCode()
				if err != nil {
					return err
				}
				code1 += int32(code3)
			}
		} else {
			code3, err := st.getWhiteCode()
			if err != nil {
				return err
			}
			code1 += int32(code3)
			for code3 >= 64 {
				code3, err = st.getWhiteCode()
				if err != nil {
					return err
				}
				code1 += int32(code3)
			}
		}
		err := st.addPixels(st.codingLine[st.a0i]+code1, blackPixels)
		if err != nil {
			return err
		}
		blackPixels ^= 1
	}
	return nil
}

func (st *CCITTDecoder) getOneByteLoopBody(bits int32, buf byte) (int32, byte, error) {
	if st.outputBits > bits {
		buf = buf << bits
		if (st.a0i & 1) == 0 {
			buf |= 0xff >> (8 - bits)
		}
		st.outputBits -= bits
		bits = 0
	} else {
		buf <<= st.outputBits
		if (st.a0i & 1) == 0 {
			buf |= 0xff >> (8 - st.outputBits)
		}
		bits -= st.outputBits
		st.outputBits = 0
		if st.codingLine[st.a0i] < st.p.Columns {
			st.a0i++
			if st.a0i > st.p.Columns {
				return 0, 0, fmt.Errorf("bad bits %x in CCITTFax stream", bits)
			}
			st.outputBits = st.codingLine[st.a0i] - st.codingLine[st.a0i-1]
		} else if bits > 0 {
			buf <<= bits
			bits = 0
		}
	}
	return bits, buf, nil
}
