package pdftokenizer

import (
	"encoding/hex"
	"errors"
)

type Kind uint8

const (
	EOF Kind = iota
	Number
	String
	StringHex
	Name
	Comment
	StartArray
	EndArray
	StartDic
	EndDic
	// Ref
	Other // include commands in content stream
)

var delims = [...]bool{
	true, false, false, false, false, false, false, false, false,
	true, true, false, true, true, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, true, false, false, false, false, true, false,
	false, true, true, false, false, false, false, false, true, false,
	false, false, false, false, false, false, false, false, false, false,
	false, true, false, true, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, true, false, true, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false}

type Token struct {
	Kind  Kind
	Value string // additional value found in the data
}

type Tokenizer struct {
	data []byte
	pos  int
}

func NewTokenizer(data []byte) *Tokenizer {
	return &Tokenizer{data: data}
}

func isWhitespace(ch byte) bool {
	return (ch == 0 || ch == 9 || ch == 10 || ch == 12 || ch == 13 || ch == 32)
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
	return 0, false
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

// NextToken read a token and advance.
// If EOF is reached, no error is returned, but a Endoffile token.
func (pr *Tokenizer) NextToken() (Token, error) {
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
	case '/':
		for {
			ch, ok = pr.read()
			if !ok || delims[ch] {
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
				return Token{}, errors.New("invalid hex char")
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
				return Token{}, errors.New("invalid hex char")
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
		var tokenType Kind
		if ch == '-' || ch == '+' || ch == '.' || (ch >= '0' && ch <= '9') {
			tokenType = Number
			outBuf = append(outBuf, ch)
			ch, ok = pr.read()
			for ok && ((ch >= '0' && ch <= '9') || ch == '.') {
				outBuf = append(outBuf, ch)
				ch, ok = pr.read()
			}
		} else {
			tokenType = Other
			outBuf = append(outBuf, ch)
			ch, ok = pr.read()
			for !delims[ch] {
				outBuf = append(outBuf, ch)
				ch, ok = pr.read()
			}
		}
		if ok {
			pr.pos--
		}
		return Token{Kind: tokenType, Value: string(outBuf)}, nil
	}
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
