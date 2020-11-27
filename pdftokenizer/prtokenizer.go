// Implements the lowest level of processing of PS/PDF files.
package pdftokenizer

// code ported from the Java PDFTK library - BK 2020

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Kind uint8

const (
	EOF Kind = iota
	Float
	Integer
	String
	StringHex
	Name
	Comment
	StartArray
	EndArray
	StartDic
	EndDic
	// Ref
	StartProc // only valid in PostScript files
	EndProc   // idem
	Other     // include commands in content stream
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
	case Comment:
		return "Comment"
	case StartArray:
		return "StartArray"
	case EndArray:
		return "EndArray"
	case StartDic:
		return "StartDic"
	case EndDic:
		return "EndDic"
	case StartProc:
		return "StartProc"
	case EndProc:
		return "EndProc"
	case Other:
		return "Other"
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
func (t Token) Float() (float64, error) {
	return strconv.ParseFloat(t.Value, 64)
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

type Tokenizer struct {
	data []byte
	pos  int

	// we store the next token to have a cheap
	// PeekToken method
	aheadToken Token
	aheadError error
}

func NewTokenizer(data []byte) Tokenizer {
	tk := Tokenizer{data: data}
	tk.aheadToken, tk.aheadError = tk.nextToken()
	return tk
}

// PeekToken reads a token but does not advance the position.
func (pr Tokenizer) PeekToken() (Token, error) {
	return pr.aheadToken, pr.aheadError
}

// NextToken reads a token and advances (consuming the token).
// If EOF is reached, no error is returned, but a Endoffile token.
//
// Regarding exponential numbers: 7.3.3 Numeric Objects:
// A conforming writer shall not use the PostScript syntax for numbers
// with non-decimal radices (such as 16#FFFE) or in exponential format
// (such as 6.02E23).
// Nonetheless, we sometimes get numbers with exponential format, so
// we will support it in the reader (no confusion with other types, so
// no compromise).
func (pr *Tokenizer) NextToken() (Token, error) {
	tk, err := pr.PeekToken()
	pr.aheadToken, pr.aheadError = pr.nextToken()
	return tk, err
}

// fromHexChar converts a hex character into its value and a success flag.
// see encoding/hex
func fromHexChar(c byte) (byte, bool) {
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

func (pr *Tokenizer) nextToken() (Token, error) {
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
			v1, ok1 = fromHexChar(v1)
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
			v2, ok2 = fromHexChar(v2)
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
		// ignore comments
		return Token{Kind: Comment}, nil
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
		// fmt.Println("before", pr.pos)
		pr.pos-- // we need the test char
		if token, ok := pr.readNumber(); ok {
			// fmt.Println(token, pr.pos)
			return token, nil
		}
		// fmt.Println("after", pr.pos)
		// if ch == '-' || ch == '+' || ch == '.' || (ch >= '0' && ch <= '9') {
		// 	tokenType = Integer
		// 	outBuf = append(outBuf, ch)
		// 	ch, ok = pr.read()
		// 	for ok {
		// 		if ch >= '0' && ch <= '9' { // decimal
		// 		} else if ch == '.' { // float
		// 			tokenType = Float
		// 		} else if ch == 'e' || ch == 'E' { // we accept Postscript notation
		// 			tokenType = Float
		// 		} else {
		// 			break
		// 		}
		// 		outBuf = append(outBuf, ch)
		// 		ch, ok = pr.read()
		// 	}
		// } else {
		ch, ok = pr.read() // we went back before parsing a number
		outBuf = append(outBuf, ch)
		ch, ok = pr.read()
		for !isDelimiter(ch) {
			outBuf = append(outBuf, ch)
			ch, ok = pr.read()
			// }
		}
		if ok {
			pr.pos--
		}
		return Token{Kind: Other, Value: string(outBuf)}, nil
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
