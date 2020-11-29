// Implements the lowest level of processing of PS/PDF files.
// The tokenizer is also usable with Type1 font files.
// See the higher level package parser to read PDF objects.
package tokenizer

// Code ported from the Java PDFTK library - BK 2020

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Fl = float64

type Kind uint8

const (
	EOF Kind = iota
	Float
	Integer
	String
	StringHex
	Name
	StartArray
	EndArray
	StartDic
	EndDic
	// Ref
	Other // include commands in content stream

	StartProc  // only valid in PostScript files
	EndProc    // idem
	CharString // PS only: binary stream, introduce by and integer and a RD or -| command
)

func (k Kind) String() string {
	switch k {
	case EOF:
		return "EOF"
	case Float:
		return "Float"
	case Integer:
		return "Integer"
	case String:
		return "String"
	case StringHex:
		return "StringHex"
	case Name:
		return "Name"
	case StartArray:
		return "StartArray"
	case EndArray:
		return "EndArray"
	case StartDic:
		return "StartDic"
	case EndDic:
		return "EndDic"
	case Other:
		return "Other"
	case StartProc:
		return "StartProc"
	case EndProc:
		return "EndProc"
	case CharString:
		return "CharString"
	default:
		return "<invalid token>"
	}
}

func isWhitespace(ch byte) bool {
	switch ch {
	case 0, 9, 10, 12, 13, 32:
		return true
	default:
		return false
	}
}

// white space + delimiters
func isDelimiter(ch byte) bool {
	switch ch {
	case 40, 41, 60, 62, 91, 93, 123, 125, 47, 37:
		return true
	default:
		return isWhitespace(ch)
	}
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// Token represents a basic piece of information.
// `Value` must be interpreted according to `Kind`,
// which is left to parsing packages.
type Token struct {
	Kind  Kind
	Value string // additional value found in the data
}

// Int returns the integer value of the token,
// also accepting float values and rouding them.
func (t Token) Int() (int, error) {
	// also accepts floats and round
	f, err := t.Float()
	return int(f), err
}

// Float returns the float value of the token.
func (t Token) Float() (Fl, error) {
	return strconv.ParseFloat(t.Value, 64)
}

// IsNumber returns `true` for integers and floats.
func (t Token) IsNumber() bool {
	return t.Kind == Integer || t.Kind == Float
}

// return true for binary stream or inline data
func (t Token) startsBinary() bool {
	return t.Kind == Other && (t.Value == "stream" || t.Value == "ID")
}

// Tokenize consume all the input, splitting it
// into tokens.
// When performance matters, you should use
// the iteration method `NextToken` of the Tokenizer type.
func Tokenize(data []byte) ([]Token, error) {
	tk := NewTokenizer(data)
	var out []Token
	t, err := tk.NextToken()
	for ; t.Kind != EOF && err == nil; t, err = tk.NextToken() {
		out = append(out, t)
	}
	return out, err
}

// Tokenizer is a PS/PDF tokenizer.
//
// It handles PS features like Procs and CharStrings:
// strict parsers should check for such tokens and return an error if needed.
//
// Comments are ignored.
//
// The tokenizer can't handle streams and inline image data on it's own:
// it will stop (by returning EOF) when reached. Processing may be resumed
// with `SetPos` method.
//
// Regarding exponential numbers: 7.3.3 Numeric Objects:
// A conforming writer shall not use the PostScript syntax for numbers
// with non-decimal radices (such as 16#FFFE) or in exponential format
// (such as 6.02E23).
// Nonetheless, we sometimes get numbers with exponential format, so
// we support it in the tokenizer (no confusion with other types, so
// no compromise).
type Tokenizer struct {
	data []byte

	// since indirect reference require
	// to read two more tokens
	// we store the two next token

	pos int // main position (end of the aaToken)

	currentPos int // end of the current token
	nextPos    int // end of the +1 token

	aToken Token // +1
	aError error // +1

	aaToken Token // +2
	aaError error // +2
}

func NewTokenizer(data []byte) Tokenizer {
	tk := Tokenizer{data: data}
	tk.initiateAt(0)
	return tk
}

// there are two cases where NextToken() is not sufficient:
// at the stat (aToken and aaToken are empty)
// end after skipping over bytes (aToken and aaToken are invalid)
// in this cases, `initiateAt` force the 2 next tokenizations
// (in the contrary, NextToken only does 1).
func (tk *Tokenizer) initiateAt(pos int) {
	tk.currentPos = pos
	tk.pos = pos
	tk.aToken, tk.aError = tk.nextToken(Token{})
	tk.nextPos = tk.pos
	tk.aaToken, tk.aaError = tk.nextToken(tk.aToken)
}

// PeekToken reads a token but does not advance the position.
// It returns a cached value, meaning it is a very cheap call.
func (pr Tokenizer) PeekToken() (Token, error) {
	return pr.aToken, pr.aError
}

// PeekPeekToken reads the token after the next but does not advance the position.
// It returns a cached value, meaning it is a very cheap call.
func (pr Tokenizer) PeekPeekToken() (Token, error) {
	return pr.aaToken, pr.aaError
}

// NextToken reads a token and advances (consuming the token).
// If EOF is reached, no error is returned, but an `EOF` token.
func (pr *Tokenizer) NextToken() (Token, error) {
	tk, err := pr.PeekToken()                     // n+1 to n
	pr.aToken, pr.aError = pr.aaToken, pr.aaError // n+2 to n+1
	pr.currentPos = pr.nextPos                    // n+1 to n
	pr.nextPos = pr.pos                           // n+2 to n

	// the tokenizer can't handle binary stream or inline data:
	// such data will be handled with a parser
	// thus, we simply stop the tokenization when we encounter them
	// to avoid useless (and maybe costly) processing
	if pr.aaToken.startsBinary() {
		pr.aaToken, pr.aaError = Token{Kind: EOF}, nil
	} else {
		pr.aaToken, pr.aaError = pr.nextToken(pr.aaToken) // read the n+3 and store it in n+2
	}
	return tk, err
}

// SkipBytes skips the next `n` bytes and return them. This method is useful
// to handle streams and inline data.
func (pr *Tokenizer) SkipBytes(n int) []byte {
	// use currentPos, which is the position 'expected' by the caller
	target := pr.currentPos + n
	if target > len(pr.data) { // truncate if needed
		target = len(pr.data)
	}
	out := pr.data[pr.currentPos:target]
	pr.initiateAt(target)
	return out
}

// Bytes return a slice of the bytes, starting
// from the current position.
func (pr Tokenizer) Bytes() []byte {
	if pr.currentPos >= len(pr.data) {
		return nil
	}
	return pr.data[pr.currentPos:]
}

// IsHexChar converts a hex character into its value and a success flag
// (see encoding/hex for details).
func IsHexChar(c byte) (uint8, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return c, false
}

// return false if EOF, true if the moved forward
func (pr *Tokenizer) read() (byte, bool) {
	if pr.pos >= len(pr.data) {
		return 0, false
	}
	ch := pr.data[pr.pos]
	pr.pos++
	return ch, true
}

// reads and advances, mutatinng `pos`
func (pr *Tokenizer) nextToken(previous Token) (Token, error) {
	ch, ok := pr.read()
	for ok && isWhitespace(ch) {
		ch, ok = pr.read()
	}
	if !ok {
		return Token{Kind: EOF}, nil
	}

	var outBuf []byte
	switch ch {
	case '[':
		return Token{Kind: StartArray}, nil
	case ']':
		return Token{Kind: EndArray}, nil
	case '{':
		return Token{Kind: StartProc}, nil
	case '}':
		return Token{Kind: EndProc}, nil
	case '/':
		for {
			ch, ok = pr.read()
			if !ok || isDelimiter(ch) {
				break
			}
			outBuf = append(outBuf, ch)
			if ch == '#' {
				h1, _ := pr.read()
				h2, _ := pr.read()
				_, err := hex.Decode([]byte{0}, []byte{h1, h2})
				if err != nil {
					return Token{}, errors.New("corrupted name object")
				}
				outBuf = append(outBuf, h1, h2)
			}
		}
		// the delimiter may be important, dont skip it
		if ok { // we moved, so its safe go back
			pr.pos--
		}
		return Token{Kind: Name, Value: string(outBuf)}, nil
	case '>':
		ch, ok = pr.read()
		if ch != '>' {
			return Token{}, errors.New("'>' not expected")
		}
		return Token{Kind: EndDic}, nil
	case '<':
		v1, ok1 := pr.read()
		if v1 == '<' {
			return Token{Kind: StartDic}, nil
		}
		var (
			v2  byte
			ok2 bool
		)
		for {
			for ok1 && isWhitespace(v1) {
				v1, ok1 = pr.read()
			}
			if v1 == '>' {
				break
			}
			v1, ok1 = IsHexChar(v1)
			if !ok1 {
				return Token{}, fmt.Errorf("invalid hex char %d (%s)", v1, string(rune(v1)))
			}
			v2, ok2 = pr.read()
			for ok2 && isWhitespace(v2) {
				v2, ok2 = pr.read()
			}
			if v2 == '>' {
				ch = v1 << 4
				outBuf = append(outBuf, ch)
				break
			}
			v2, ok2 = IsHexChar(v2)
			if !ok2 {
				return Token{}, fmt.Errorf("invalid hex char %d", v2)
			}
			ch = (v1 << 4) + v2
			outBuf = append(outBuf, ch)
			v1, ok1 = pr.read()
		}
		return Token{Kind: StringHex, Value: string(outBuf)}, nil
	case '%':
		ch, ok = pr.read()
		for ok && ch != '\r' && ch != '\n' {
			ch, ok = pr.read()
		}
		// ignore comments: go to next token
		return pr.nextToken(previous)
	case '(':
		nesting := 0
		for {
			ch, ok = pr.read()
			if !ok {
				break
			}
			if ch == '(' {
				nesting++
			} else if ch == ')' {
				nesting--
			} else if ch == '\\' {
				lineBreak := false
				ch, ok = pr.read()
				switch ch {
				case 'n':
					ch = '\n'
				case 'r':
					ch = '\r'
				case 't':
					ch = '\t'
				case 'b':
					ch = '\b'
				case 'f':
					ch = '\f'
				case '(', ')', '\\':
				case '\r':
					lineBreak = true
					ch, ok = pr.read()
					if ch != '\n' {
						pr.pos--
					}
				case '\n':
					lineBreak = true
				default:
					if ch < '0' || ch > '7' {
						break
					}
					octal := ch - '0'
					ch, ok = pr.read()
					if ch < '0' || ch > '7' {
						pr.pos--
						ch = octal
						break
					}
					octal = (octal << 3) + ch - '0'
					ch, ok = pr.read()
					if ch < '0' || ch > '7' {
						pr.pos--
						ch = octal
						break
					}
					octal = (octal << 3) + ch - '0'
					ch = octal & 0xff
					break
				}
				if lineBreak {
					continue
				}
				if !ok || ch < 0 {
					break
				}
			} else if ch == '\r' {
				ch, ok = pr.read()
				if !ok {
					break
				}
				if ch != '\n' {
					pr.pos--
					ch = '\n'
				}
			}
			if nesting == -1 {
				break
			}
			outBuf = append(outBuf, ch)
		}
		if !ok {
			return Token{}, errors.New("error reading string: unexpected EOF")
		}
		return Token{Kind: String, Value: string(outBuf)}, nil
	default:
		pr.pos-- // we need the test char
		if token, ok := pr.readNumber(); ok {
			return token, nil
		}
		ch, ok = pr.read() // we went back before parsing a number
		outBuf = append(outBuf, ch)
		ch, ok = pr.read()
		for !isDelimiter(ch) {
			outBuf = append(outBuf, ch)
			ch, ok = pr.read()
		}
		if ok {
			pr.pos--
		}
		cmd := string(outBuf)
		if cmd == "RD" || cmd == "-|" {
			// return the next CharString instead
			if previous.Kind == Integer {
				f, err := previous.Int()
				if err != nil {
					return Token{}, fmt.Errorf("invalid charstring length: %s", err)
				}
				return pr.readCharString(f), nil
			} else {
				return Token{}, errors.New("expected INTEGER before -| or RD")
			}
		}
		return Token{Kind: Other, Value: cmd}, nil
	}
}

// accept PS syntax (radix and exponents)
// return false if it is not a number
func (pr *Tokenizer) readNumber() (Token, bool) {
	markedPos := pr.pos

	sb, radix := &strings.Builder{}, &strings.Builder{}
	c, ok := pr.read() // one char is OK
	hasDigit := false
	// optional + or -
	if c == '+' || c == '-' {
		sb.WriteByte(c)
		c, _ = pr.read()
	}

	// optional digits
	for isDigit(c) {
		sb.WriteByte(c)
		c, ok = pr.read()
		hasDigit = true
	}

	// optional .
	if c == '.' {
		sb.WriteByte(c)
		c, _ = pr.read()
	} else if c == '#' {
		// PostScript radix number takes the form base#number
		radix = sb
		sb = &strings.Builder{}
		c, _ = pr.read()
	} else if sb.Len() == 0 || !hasDigit {
		// failure
		pr.pos = markedPos
		return Token{}, false
	} else if c == 'E' || c == 'e' {
		// optional minus
		sb.WriteByte(c)
		c, ok = pr.read()
		if c == '-' {
			sb.WriteByte(c)
			c, ok = pr.read()
		}
	} else {
		// integer
		if ok {
			pr.pos--
		}
		return Token{Value: sb.String(), Kind: Integer}, true
	}

	// required digit
	if isDigit(c) {
		sb.WriteByte(c)
		c, ok = pr.read()
	} else {
		// failure
		pr.pos = markedPos
		return Token{}, false
	}

	// optional digits
	for isDigit(c) {
		sb.WriteByte(c)
		c, ok = pr.read()
	}

	if ok {
		pr.pos--
	}
	if radix := radix.String(); radix != "" {
		intRadix, _ := strconv.Atoi(radix)
		valInt, _ := strconv.ParseInt(sb.String(), intRadix, 0)
		return Token{Value: strconv.Itoa(int(valInt)), Kind: Integer}, true
	}
	return Token{Value: sb.String(), Kind: Float}, true
}

// reads a binary CharString.
func (pr *Tokenizer) readCharString(length int) Token {
	pr.pos++ // space
	maxL := pr.pos + length
	if maxL >= len(pr.data) {
		maxL = len(pr.data)
	}
	out := Token{Value: string(pr.data[pr.pos:maxL]), Kind: CharString}
	pr.pos += length
	return out
}

//  public void nextValidToken() throws IOException {
// 	 int level = 0;
// 	 String n1 = null;
// 	 String n2 = null;
// 	 int ptr = 0;
// 	 while (nextToken()) {
// 		 if (pr.tokenType == tkComment)
// 			 continue;
// 		 switch (level) {
// 			 case 0:
// 			 {
// 				 if (type != tkNumber)
// 					 return;
// 				 ptr = file.getFilePointer();
// 				 n1 = stringValue;
// 				 ++level;
// 				 break;
// 			 }
// 			 case 1:
// 			 {
// 				 if (type != tkNumber) {
// 					 file.seek(ptr);
// 					 pr.tokenType = tkNumber;
// 					 stringValue = n1;
// 					 return;
// 				 }
// 				 n2 = stringValue;
// 				 ++level;
// 				 break;
// 			 }
// 			 default:
// 			 {
// 				 if (type != tkOther || !stringValue.equals("R")) {
// 					 file.seek(ptr);
// 					 pr.tokenType = tkNumber;
// 					 stringValue = n1;
// 					 return;
// 				 }
// 				 pr.tokenType = tkRef;
// 				 reference = Integer.parseInt(n1);
// 				 generation = Integer.parseInt(n2);
// 				 return;
// 			 }
// 		 }
// 	 }
// 	 // http://bugs.debian.org/cgi-bin/bugreport.cgi?bug=687669#20
// 	 if (level > 0) {
// 		 pr.tokenType = tkNumber;
// 		 file.seek(ptr);
// 		 stringValue = n1;
// 		 return;
// 	 }
// 	 throwError("Unexpected end of file");
//  }

// 	 public int intValue() {
// 		 return Integer.parseInt(stringValue);
// 	 }

// 	 public boolean readLineSegment(byte input[]) throws IOException {
// 		 int c = -1;
// 		 boolean eol = false;
// 		 int ptr = 0;
// 		 int len = input.length;

// 		 // ssteward, pdftk-1.10, 040922:
// 		 // skip initial whitespace; added this because PdfReader.rebuildXref()
// 		 // assumes that line provided by readLineSegment does not have init. whitespace;
// 		 if ( ptr < len ) {
// 			 while ( isWhitespace( (c = read()) ) );
// 		 }
// 		 while ( !eol && ptr < len ) {
// 			 switch (c) {
// 			 case -1:
// 			 case '\n':
// 				 eol = true;
// 			 break;
// 			 case '\r':
// 				 eol = true;
// 				 int cur = getFilePointer();
// 				 if ((read()) != '\n') {
// 					 seek(cur);
// 				 }
// 				 break;
// 			 default:
// 				 input[ptr++] = (byte)c;
// 				 break;
// 			 }

// 			 // break loop? do it before we read() again
// 			 if( eol || len <= ptr ) {
// 				 break;
// 			 }
// 			 else {
// 				 c = read();
// 			 }
// 		 }

// 		 if( len <= ptr  ) {
// 			 eol = false;
// 			 while (!eol) {
// 				 switch (c = read()) {
// 				 case -1:
// 				 case '\n':
// 					 eol = true;
// 				 break;
// 				 case '\r':
// 					 eol = true;
// 					 int cur = getFilePointer();
// 					 if ((read()) != '\n') {
// 						 seek(cur);
// 					 }
// 					 break;
// 				 }
// 			 }
// 		 }

// 		 if ((c == -1) && (ptr == 0)) {
// 			 return false;
// 		 }
// 		 if (ptr + 2 <= len) {
// 			 input[ptr++] = (byte)' ';
// 			 input[ptr] = (byte)'X';
// 		 }
// 		 return true;
// 	 }

// 	 public static int[] checkObjectStart(byte line[]) {
// 		 try {
// 			 PRTokeniser tk = new PRTokeniser(line);
// 			 int num = 0;
// 			 int gen = 0;
// 			 if (!tk.nextToken() || tk.getTokenType() != tkNumber)
// 				 return null;
// 			 num = tk.intValue();
// 			 if (!tk.nextToken() || tk.getTokenType() != tkNumber)
// 				 return null;
// 			 gen = tk.intValue();
// 			 if (!tk.nextToken())
// 				 return null;
// 			 if (!tk.getStringValue().equals("obj"))
// 				 return null;
// 			 return new int[]{num, gen};
// 		 }
// 		 catch (Exception ioe) {
// 			 // empty on purpose
// 		 }
// 		 return null;
// 	 }

// 	 public boolean isHexString() {
// 		 return this.HexString;
// 	 }

//  }
