package type1font

import (
	"errors"
	"strconv"
	"strings"

	"golang.org/x/exp/errors/fmt"
)

// token kind
const (
	_ = iota
	string_
	name
	literal
	real
	integer
	startArray
	endArray
	startProc
	endProc
	startDict
	endDict
	charstring
)

var none = token{} // null token

type kind uint8

func (k kind) String() string {
	switch k {
	case 0:
		return "NONE"
	case string_:
		return "STRING"
	case name:
		return "NAME"
	case literal:
		return "LITERAL"
	case real:
		return "REAL"
	case integer:
		return "INTEGER"
	case startArray:
		return "STARTARRAY"
	case endArray:
		return "ENDARRAY"
	case startProc:
		return "STARTPROC"
	case endProc:
		return "ENDPROC"
	case startDict:
		return "STARTDICT"
	case endDict:
		return "ENDDICT"
	case charstring:
		return "CHARSTRING"
	default:
		return "invalid token"
	}
}

type token struct {
	kind  kind
	value string
}

func (t token) floatValue() float64 {
	// some fonts have reals where integers should be, so we tolerate it
	f, _ := strconv.ParseFloat(t.value, 64)
	return f
}

func (t token) intValue() int {
	// some fonts have reals where integers should be, so we tolerate it
	return int(t.floatValue())
}

func isWhitespace(ch byte) bool {
	switch ch {
	case 0, 9, 10, 12, 13, 32:
		return true
	default:
		return false
	}
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

/*
Lexer for the ASCII portions of an Adobe Type 1 font.

The PostScript language, of which Type 1 fonts are a subset, has a
somewhat awkward lexical structure. It is neither regular nor
context-free, and the execution of the program can modify the
the behaviour of the lexer/parser.

Nevertheless, this type represents an attempt to artificially seperate
the PostScript parsing process into separate lexing and parsing phases
in order to reduce the complexity of the parsing phase.

See "PostScript Language Reference 3rd ed, Adobe Systems (1999)"

Ported from the code from John Hewson.
*/
type lexer struct {
	data []byte
	pos  int

	// aheadToken token
	openParens int

	// store one token in advance, so that
	// peeking token is immediate
	aheadToken token
}

func (l lexer) hasRemaining() bool {
	return l.pos < len(l.data)
}

// constructs a new lexer given a header-less .pfb segment
func newLexer(data []byte) (lexer, error) {
	l := lexer{data: data}
	var err error
	l.aheadToken, err = l.readToken(none)
	return l, err
}

// returns the next token and consumes it.
func (l *lexer) nextToken() (token, error) {
	curToken := l.aheadToken
	var err error
	l.aheadToken, err = l.readToken(curToken)
	return curToken, err
}

// Returns the next token without consuming it.
func (l lexer) peekToken() token {
	return l.aheadToken
}

// reads an ASCII char from the buffer and advance
func (l *lexer) getChar() byte {
	c := l.data[l.pos]
	l.pos++
	return c
}

// reads a single token.
func (l *lexer) readToken(prevToken token) (token, error) {
	skip := true
	for skip {
		skip = false
		for l.hasRemaining() {
			c := l.getChar()
			switch c {
			// delimiters
			case '%':
				// comment
				l.readComment()
			case '(':
				return l.readString(), nil
			case ')':
				// not allowed outside a string context
				return token{}, errors.New("unexpected closing parenthesis")
			case '[':
				return token{value: string(c), kind: startArray}, nil
			case '{':
				return token{value: string(c), kind: startProc}, nil
			case ']':
				return token{value: string(c), kind: endArray}, nil
			case '}':
				return token{value: string(c), kind: endProc}, nil
			case '/':
				return token{value: l.readRegular(), kind: literal}, nil
			case '<':
				c2 := l.getChar()
				if c2 == c {
					return token{value: "<<", kind: startDict}, nil
				} else {
					l.pos--
					return token{value: string(c), kind: name}, nil
				}
			case '>':
				c2 := l.getChar()
				if c2 == c {
					return token{value: ">>", kind: endDict}, nil
				} else {
					l.pos--
					return token{value: string(c), kind: name}, nil
				}
			default:
				if isWhitespace(c) {
					skip = true
				} else {
					l.pos--

					// regular character: try parse as number
					number := l.tryReadNumber()
					if number != none {
						return number, nil
					} else {
						// otherwise this must be a name
						name_ := l.readRegular()
						if name_ == "" {
							// the stream is corrupt
							return token{}, fmt.Errorf("could not read token at position %d", l.pos)
						}

						if name_ == "RD" || name_ == "-|" {
							// return the next CharString instead
							if prevToken.kind == integer {
								f := prevToken.intValue()
								return l.readCharString(f), nil
							} else {
								return none, errors.New("expected INTEGER before -| or RD")
							}
						} else {
							return token{value: name_, kind: name}, nil
						}
					}
				}
			}
		}
	}
	return none, nil
}

// Reads a number or returns empty.
func (l *lexer) tryReadNumber() token {
	markedPos := l.pos

	var sb, radix strings.Builder
	c := l.getChar()
	hasDigit := false

	// optional + or -
	if c == '+' || c == '-' {
		sb.WriteByte(c)
		c = l.getChar()
	}

	// optional digits
	for isDigit(c) {
		sb.WriteByte(c)
		c = l.getChar()
		hasDigit = true
	}

	// optional .
	if c == '.' {
		sb.WriteByte(c)
		c = l.getChar()
	} else if c == '#' {
		// PostScript radix number takes the form base#number
		radix = sb
		sb = strings.Builder{}
		c = l.getChar()
	} else if sb.Len() == 0 || !hasDigit {
		// failure
		l.pos = markedPos
		return token{}
	} else {
		// integer
		l.pos--
		return token{value: sb.String(), kind: integer}
	}

	// required digit
	if isDigit(c) {
		sb.WriteByte(c)
		c = l.getChar()
	} else {
		// failure
		l.pos = markedPos
		return none
	}

	// optional digits
	for isDigit(c) {
		sb.WriteByte(c)
		c = l.getChar()
	}

	// optional E
	if c == 'E' {
		sb.WriteByte(c)
		c = l.getChar()

		// optional minus
		if c == '-' {
			sb.WriteByte(c)
			c = l.getChar()
		}

		// required digit
		if isDigit(c) {
			sb.WriteByte(c)
			c = l.getChar()
		} else {
			// failure
			l.pos = markedPos
			return none
		}

		// optional digits
		for isDigit(c) {
			sb.WriteByte(c)
			c = l.getChar()
		}
	}

	l.pos--
	if radix := radix.String(); radix != "" {
		intRadix, _ := strconv.Atoi(radix)
		valInt, _ := strconv.ParseInt(sb.String(), intRadix, 0)
		return token{value: strconv.Itoa(int(valInt)), kind: integer}
	}
	return token{value: sb.String(), kind: real}
}

// Reads a sequence of regular characters, i.e. not delimiters
// or whitespace
func (l *lexer) readRegular() string {
	var sb strings.Builder
	for l.hasRemaining() {
		markedPos := l.pos
		c := l.getChar()
		if isWhitespace(c) ||
			c == '(' || c == ')' ||
			c == '<' || c == '>' ||
			c == '[' || c == ']' ||
			c == '{' || c == '}' ||
			c == '/' || c == '%' {
			l.pos = markedPos
			break
		} else {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// Reads a line comment.
func (l *lexer) readComment() string {
	var sb strings.Builder
	for l.hasRemaining() {
		switch c := l.getChar(); c {
		case '\r', '\n':
			return sb.String()
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// Reads a (string).
func (l *lexer) readString() token {
	var sb strings.Builder
	for l.hasRemaining() {
		c := l.getChar()

		// string context
		switch c {
		case '(':
			l.openParens++
			sb.WriteByte('(')
		case ')':
			if l.openParens == 0 {
				// end of string
				return token{value: sb.String(), kind: string_}
			}
			sb.WriteByte(')')
			l.openParens--
		case '\\':
			// escapes: \n \r \t \b \f \\ \( \)
			c1 := l.getChar()
			switch c1 {
			case 'n', 'r':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case '\\':
				sb.WriteByte('\\')
			case '(':
				sb.WriteByte('(')
			case ')':
				sb.WriteByte(')')
			}
			// octal \ddd
			if isDigit(c1) {
				num := string([]byte{c1, l.getChar(), l.getChar()})
				code, _ := strconv.ParseInt(num, 8, 0)
				sb.WriteRune(rune(code))
			}
		case '\r', '\n':
			sb.WriteByte('\n')
		default:
			sb.WriteByte(c)
		}
	}
	return token{}
}

// reads a binary CharString.
func (l *lexer) readCharString(length int) token {
	l.pos++ // space
	maxL := l.pos + length
	if maxL >= len(l.data) {
		maxL = len(l.data)
	}
	out := token{value: string(l.data[l.pos:maxL]), kind: charstring}
	l.pos += length
	return out
}
