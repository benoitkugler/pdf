package type1font

import (
	pt "github.com/benoitkugler/pdf/parser/tokenizer"
)

var none = pt.Token{} // null token

// // reads a single token.
// func (l *lexer1) readToken(prevToken pt.Token) (pt.Token, error) {
// 	skip := true
// 	for skip {
// 		skip = false
// 		for l.hasRemaining() {
// 			c := l.getChar()
// 			switch c {
// 			// delimiters
// 			case '%':
// 				// comment
// 				l.readComment()
// 			case '(':
// 				return l.readString(), nil
// 			case ')':
// 				// not allowed outside a string context
// 				return none, errors.New("unexpected closing parenthesis")
// 			case '[':
// 				return pt.Token{Value: string(c), Kind: pt.StartArray}, nil
// 			case '{':
// 				return pt.Token{Value: string(c), Kind: pt.StartProc}, nil
// 			case ']':
// 				return pt.Token{Value: string(c), Kind: pt.EndArray}, nil
// 			case '}':
// 				return pt.Token{Value: string(c), Kind: pt.EndProc}, nil
// 			case '/':
// 				return pt.Token{Value: l.readRegular(), Kind: pt.Name}, nil
// 			case '<':
// 				c2 := l.getChar()
// 				if c2 == '<' {
// 					return pt.Token{Value: "<<", Kind: pt.StartDic}, nil
// 				} else {
// 					l.pos--
// 					return pt.Token{Value: string(c), Kind: pt.Name}, nil
// 				}
// 			case '>':
// 				c2 := l.getChar()
// 				if c2 == c {
// 					return pt.Token{Value: ">>", Kind: pt.EndDic}, nil
// 				} else {
// 					l.pos--
// 					return pt.Token{Value: string(c), Kind: name}, nil
// 				}
// 			default:
// 				if isWhitespace(c) {
// 					skip = true
// 				} else {
// 					l.pos--

// 					// regular character: try parse as number
// 					number := l.tryReadNumber()
// 					if number != none {
// 						return number, nil
// 					} else {
// 						// otherwise this must be a name
// 						name_ := l.readRegular()
// 						if name_ == "" {
// 							// the stream is corrupt
// 							return none, fmt.Errorf("could not read token at position %d", l.pos)
// 						}

// 						if name_ == "RD" || name_ == "-|" {
// 							// return the next CharString instead
// 							if prevToken.kind == integer {
// 								f := prevToken.intValue()
// 								return l.readCharString(f), nil
// 							} else {
// 								return none, errors.New("expected INTEGER before -| or RD")
// 							}
// 						} else {
// 							return pt.Token{Value: name_, Kind: name}, nil
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return none, nil
// }

// // reads a binary CharString.
// func (l *lexer1) readCharString(length int) token {
// 	l.pos++ // space
// 	maxL := l.pos + length
// 	if maxL >= len(l.data) {
// 		maxL = len(l.data)
// 	}
// 	out := pt.Token{Value: string(l.data[l.pos:maxL]), Kind: charstring}
// 	l.pos += length
// 	return out
// }
