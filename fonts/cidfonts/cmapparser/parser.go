package cmapparser

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"github.com/benoitkugler/pdf/model"
)

// cMapParser parses CMap character to unicode mapping files.
type cMapParser struct {
	reader *bufio.Reader
}

// cMapParser creates a new instance of the PDF CMap parser from input data.
func newCMapParser(content []byte) *cMapParser {
	parser := cMapParser{}

	buffer := bytes.NewBuffer(content)
	parser.reader = bufio.NewReader(buffer)

	return &parser
}

// isDecimalDigit checks if the character is a part of a decimal number string.
func isDecimalDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

// parseObject detects the signature at the current file position and parses the corresponding object.
func (p *cMapParser) parseObject() (cmapObject, error) {
	p.skipSpaces()
	for {
		bb, err := p.reader.Peek(2)
		if err != nil {
			return nil, err
		}

		if bb[0] == '%' {
			p.parseComment()
			p.skipSpaces()
			continue
		} else if bb[0] == '/' {
			name, err := p.parseName()
			return name, err
		} else if bb[0] == '(' {
			str, err := p.parseString()
			return str, err
		} else if bb[0] == '[' {
			arr, err := p.parseArray()
			return arr, err
		} else if (bb[0] == '<') && (bb[1] == '<') {
			dict, err := p.parseDict()
			return dict, err
		} else if bb[0] == '<' {
			shex, err := p.parseHexString()
			return shex, err
		} else if isDecimalDigit(bb[0]) || (bb[0] == '-' && isDecimalDigit(bb[1])) {
			number, err := p.parseNumber()
			if err != nil {
				return nil, err
			}
			return number, nil
		} else {
			// Operand?
			operand, err := p.parseOperand()
			if err != nil {
				return nil, err
			}

			return operand, nil
		}
	}
}

// isWhiteSpace checks if byte represents a white space character.
func isWhiteSpace(ch byte) bool {
	// Table 1 white-space characters (7.2.2 Character Set)
	// spaceCharacters := string([]byte{0x00, 0x09, 0x0A, 0x0C, 0x0D, 0x20})
	switch ch {
	case 0x00, 0x09, 0x0A, 0x0C, 0x0D, 0x20:
		return true
	default:
		return false
	}
}

// skipSpaces skips over any spaces.  Returns the number of spaces skipped and an error if any.
func (p *cMapParser) skipSpaces() (int, error) {
	cnt := 0
	for {
		bb, err := p.reader.Peek(1)
		if err != nil {
			return 0, err
		}
		if isWhiteSpace(bb[0]) {
			p.reader.ReadByte()
			cnt++
		} else {
			break
		}
	}

	return cnt, nil
}

// parseComment reads a comment line starting with '%'.
func (p *cMapParser) parseComment() (string, error) {
	var r bytes.Buffer

	_, err := p.skipSpaces()
	if err != nil {
		return r.String(), err
	}

	isFirst := true
	for {
		bb, err := p.reader.Peek(1)
		if err != nil {
			return r.String(), err
		}
		if isFirst && bb[0] != '%' {
			return r.String(), ErrBadCMapComment
		}
		isFirst = false
		if (bb[0] != '\r') && (bb[0] != '\n') {
			b, _ := p.reader.ReadByte()
			r.WriteByte(b)
		} else {
			break
		}
	}
	return r.String(), nil
}

// parseName parses a name starting with '/'.
func (p *cMapParser) parseName() (model.Name, error) {
	name := ""
	nameStarted := false
	for {
		bb, err := p.reader.Peek(1)
		if err == io.EOF {
			break // Can happen when loading from object stream.
		}
		if err != nil {
			return model.Name(name), err
		}

		if !nameStarted {
			// Should always start with '/', otherwise not valid.
			if bb[0] == '/' {
				nameStarted = true
				p.reader.ReadByte()
			} else {
				return "", fmt.Errorf("invalid name: (%c)", bb[0])
			}
		} else {
			if isWhiteSpace(bb[0]) {
				break
			} else if (bb[0] == '/') || (bb[0] == '[') || (bb[0] == '(') || (bb[0] == ']') || (bb[0] == '<') || (bb[0] == '>') {
				break // Looks like start of next statement.
			} else if bb[0] == '#' {
				hexcode, err := p.reader.Peek(3)
				if err != nil {
					return model.Name(name), err
				}
				p.reader.Discard(3)

				code, err := hex.DecodeString(string(hexcode[1:3]))
				if err != nil {
					return model.Name(name), err
				}
				name += string(code)
			} else {
				b, _ := p.reader.ReadByte()
				name += string(b)
			}
		}
	}

	return model.Name(name), nil
}

// isOctalDigit checks if a character can be part of an octal digit string.
func isOctalDigit(c byte) bool {
	return '0' <= c && c <= '7'
}

// parseString parses a string starts with '(' and ends with ')'.
func (p *cMapParser) parseString() (string, error) {
	p.reader.ReadByte()

	buf := bytes.Buffer{}

	count := 1
	for {
		bb, err := p.reader.Peek(1)
		if err != nil {
			return "", err
		}

		if bb[0] == '\\' { // Escape sequence.
			p.reader.ReadByte() // Skip the escape \ byte.
			b, err := p.reader.ReadByte()
			if err != nil {
				return "", err
			}

			// Octal '\ddd' number (base 8).
			if isOctalDigit(b) {
				bb, err := p.reader.Peek(2)
				if err != nil {
					return "", err
				}

				var numeric []byte
				numeric = append(numeric, b)
				for _, val := range bb {
					if isOctalDigit(val) {
						numeric = append(numeric, val)
					} else {
						break
					}
				}
				p.reader.Discard(len(numeric) - 1)

				code, err := strconv.ParseUint(string(numeric), 8, 32)
				if err != nil {
					return "", err
				}
				buf.WriteByte(byte(code))
				continue
			}

			switch b {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case '(':
				buf.WriteByte('(')
			case ')':
				buf.WriteByte(')')
			case '\\':
				buf.WriteByte('\\')
			}

			continue
		} else if bb[0] == '(' {
			count++
		} else if bb[0] == ')' {
			count--
			if count == 0 {
				p.reader.ReadByte()
				break
			}
		}

		b, _ := p.reader.ReadByte()
		buf.WriteByte(b)
	}

	return buf.String(), nil
}

// parseHexString parses a PostScript hex string.
// Hex strings start with '<' ends with '>'.
// Currently not converting the hex codes to characters.
func (p *cMapParser) parseHexString() (cmapHexString, error) {
	p.reader.ReadByte()

	hextable := []byte("0123456789abcdefABCDEF")

	buf := bytes.Buffer{}

	for {
		p.skipSpaces()

		bb, err := p.reader.Peek(1)
		if err != nil {
			return cmapHexString{}, err
		}

		if bb[0] == '>' {
			p.reader.ReadByte()
			break
		}

		b, _ := p.reader.ReadByte()
		if bytes.IndexByte(hextable, b) >= 0 {
			buf.WriteByte(b)
		}
	}

	if buf.Len()%2 == 1 {
		buf.WriteByte('0')
	}

	hexb, _ := hex.DecodeString(buf.String())
	return cmapHexString(hexb), nil
}

// parseArray parses a PDF array, which starts with '[', ends with ']'and can contain any kinds of
// direct objects.
func (p *cMapParser) parseArray() (cmapArray, error) {
	arr := cmapArray{}

	p.reader.ReadByte()

	for {
		p.skipSpaces()

		bb, err := p.reader.Peek(1)
		if err != nil {
			return arr, err
		}

		if bb[0] == ']' {
			p.reader.ReadByte()
			break
		}

		obj, err := p.parseObject()
		if err != nil {
			return arr, err
		}
		arr = append(arr, obj)
	}

	return arr, nil
}

// parseDict parses a PDF dictionary object, which starts with with '<<' and ends with '>>'.
func (p *cMapParser) parseDict() (map[model.Name]cmapObject, error) {

	dict := map[model.Name]cmapObject{}

	// Pass the '<<'
	c, _ := p.reader.ReadByte()
	if c != '<' {
		return dict, ErrBadCMapDict
	}
	c, _ = p.reader.ReadByte()
	if c != '<' {
		return dict, ErrBadCMapDict
	}

	for {
		p.skipSpaces()

		bb, err := p.reader.Peek(2)
		if err != nil {
			return dict, err
		}

		if (bb[0] == '>') && (bb[1] == '>') {
			p.reader.ReadByte()
			p.reader.ReadByte()
			break
		}

		key, err := p.parseName()
		if err != nil {
			return dict, err
		}

		p.skipSpaces()

		val, err := p.parseObject()
		if err != nil {
			return dict, err
		}
		dict[key] = val

		// Skip "def" which optionally follows key value dict definitions in CMaps.
		p.skipSpaces()
		bb, err = p.reader.Peek(3)
		if err != nil {
			return dict, err
		}
		if string(bb) == "def" {
			p.reader.Discard(3)
		}

	}

	return dict, nil
}

// parseNumber parses a numeric objects from a buffered stream.
// Section 7.3.3.
// Integer or Float.
//
// An integer shall be written as one or more decimal digits optionally
// preceded by a sign. The value shall be interpreted as a signed
// decimal integer and shall be converted to an integer object.
//
// A real value shall be written as one or more decimal digits with an
// optional sign and a leading, trailing, or embedded PERIOD (2Eh)
// (decimal point). The value shall be interpreted as a real number
// and shall be converted to a real object.
//
// Regarding exponential numbers: 7.3.3 Numeric Objects:
// A conforming writer shall not use the PostScript syntax for numbers
// with non-decimal radices (such as 16#FFFE) or in exponential format
// (such as 6.02E23).
// Nonetheless, we sometimes get numbers with exponential format, so
// we will support it in the reader (no confusion with other types, so
// no compromise).
func (p *cMapParser) parseNumber() (cmapObject, error) {
	isFloat := false
	allowSigns := true
	var r bytes.Buffer
	for {
		bb, err := p.reader.Peek(1)
		if err == io.EOF {
			// GH: EOF handling.  Handle EOF like end of line.  Can happen with
			// encoded object streams that the object is at the end.
			// In other cases, we will get the EOF error elsewhere at any rate.
			break // Handle like EOF
		}
		if err != nil {
			return nil, err
		}
		if allowSigns && (bb[0] == '-' || bb[0] == '+') {
			// Only appear in the beginning, otherwise serves as a delimiter.
			b, _ := p.reader.ReadByte()
			r.WriteByte(b)
			allowSigns = false // Only allowed in beginning, and after e (exponential).
		} else if isDecimalDigit(bb[0]) {
			b, _ := p.reader.ReadByte()
			r.WriteByte(b)
		} else if bb[0] == '.' {
			b, _ := p.reader.ReadByte()
			r.WriteByte(b)
			isFloat = true
		} else if bb[0] == 'e' || bb[0] == 'E' {
			// Exponential number format.
			b, _ := p.reader.ReadByte()
			r.WriteByte(b)
			isFloat = true
			allowSigns = true
		} else {
			break
		}
	}

	if isFloat {
		fVal, err := strconv.ParseFloat(r.String(), 64)
		if err != nil {
			fVal = 0.0
		}
		return fVal, nil
	} else {
		intVal, err := strconv.ParseInt(r.String(), 10, 64)
		if err != nil {
			intVal = 0
		}

		return int(intVal), nil
	}
}

// isDelimiter checks if a character represents a delimiter.
func isDelimiter(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	default:
		return false
	}
}

// parseOperand parses an operand, which is a text command represented by a word.
func (p *cMapParser) parseOperand() (cmapOperand, error) {
	buf := bytes.Buffer{}
	for {
		bb, err := p.reader.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if isDelimiter(bb[0]) {
			break
		}
		if isWhiteSpace(bb[0]) {
			break
		}

		b, _ := p.reader.ReadByte()
		buf.WriteByte(b)
	}

	if buf.Len() == 0 {
		return "", fmt.Errorf("invalid operand (empty)")
	}

	return cmapOperand(buf.String()), nil
}
