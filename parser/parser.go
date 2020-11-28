// Implements a PDF object parser, mapping a list of tokens (see the tokenizer package) into
// tree-like structure.
// Higher-level reader is neeed to decrypt a full PDF file.
package parser

import (
	"errors"
	"fmt"

	tok "github.com/benoitkugler/pdf/parser/tokenizer"
	"github.com/pdfcpu/pdfcpu/pkg/log"
)

var (
	errArrayNotTerminated      = errors.New("pdfcpu: parse: unterminated array")
	errDictionaryCorrupt       = errors.New("pdfcpu: parse: corrupt dictionary")
	errDictionaryDuplicateKey  = errors.New("pdfcpu: parse: duplicate key")
	errDictionaryNotTerminated = errors.New("pdfcpu: parse: unterminated dictionary")
	errBufNotAvailable         = errors.New("pdfcpu: parse: no buffer available")
)

// Standalone implementation of a PDF parser.
// The parser only handles chunks of PDF files
// (corresponding for example to object definitions),
// but cannot handle a full file with streams.
// An higher-level reader is needed to decode Streams and Inline Data,
// which require knowledge on the filters used.
type Parser struct {
	tokens tok.Tokenizer

	// If true, disallow Indirect Reference,
	// but allow Commands
	ContentStreamMode bool
}

func NewParser(data []byte) *Parser {
	p := &Parser{tokens: tok.NewTokenizer(data)}
	return p
}

// ParseObject tokenizes and parses the input,
// expecting a valid PDF object.
func ParseObject(data []byte) (Object, error) {
	log.Parse.Printf("ParseObject: buf= <%s>\n", data)
	p := NewParser(data)
	return p.ParseObject()
}

// ParseObject read one of the (potentially) many objects
// in the input data (See NewParser).
func (p *Parser) ParseObject() (Object, error) {
	tk, err := p.tokens.NextToken()
	if err != nil {
		return nil, err
	}

	var value Object

	switch tk.Kind {
	case tok.EOF:
		return nil, errBufNotAvailable
	case tok.Name:
		value = Name(tk.Value)
		log.Parse.Printf("ParseObject: value = Name Object : %s\n", value)
	case tok.String:
		value = StringLiteral(tk.Value)
		log.Parse.Printf("ParseObject: value = String Literal: <%s>\n", value)
	case tok.StringHex:
		value = HexLiteral(tk.Value)
		log.Parse.Printf("ParseObject: value = Hex Literal: <%s>\n", value)
	case tok.StartArray:
		log.Parse.Println("ParseObject: value = Array")
		arr, err := p.parseArray()
		if err != nil {
			return nil, err
		}
		log.Parse.Printf("ParseArray: returning array (len=%d): %v\n", len(arr), arr)
		value = arr
	case tok.StartDic:
		dict, err := p.parseDict()
		if err != nil {
			return nil, err
		}
		log.Parse.Printf("ParseDict: returning dict (len=%d): %v\n", len(dict), dict)
		value = dict
	case tok.Float:
		// We have a Float!
		f, err := tk.Float()
		if err != nil {
			return nil, err
		}
		log.Parse.Printf("ParseObject: value = Float: %f\n", f)
		return Float(f), nil
	default:
		var ok bool
		value, ok = parseBooleanOrNull(tk.Value)
		if ok {
			log.Parse.Println("parseBooleanOrNull: returning: %v", value)
			break
		}
		if p.ContentStreamMode {
			// TODO: parse commands
		}
		// Must be numeric or indirect reference:
		// int 0 r
		// int
		// float
		value, err = p.parseNumericOrIndRef(tk)
		if err != nil {
			return nil, err
		}
	}

	log.Parse.Printf("ParseObject returning %v\n", value)
	return value, nil
}

func (p *Parser) parseArray() (Array, error) {
	a := Array{}
	tk, err := p.tokens.PeekToken()
	for ; err == nil; tk, err = p.tokens.PeekToken() {
		switch tk.Kind {
		case tok.EndArray:
			_, _ = p.tokens.NextToken() // consume it
			return a, nil
		case tok.EOF:
			return nil, errArrayNotTerminated
		default:
			obj, err := p.ParseObject()
			if err != nil {
				return nil, err
			}
			log.Parse.Printf("ParseArray: new array obj=%v\n", obj)
			a = append(a, obj)
		}
	}

	return nil, err
}

func (p *Parser) parseDict() (Dict, error) {
	d := NewDict()

	tk, err := p.tokens.PeekToken()
	for ; err == nil; tk, err = p.tokens.PeekToken() {
		switch tk.Kind {
		case tok.EndDic:
			_, _ = p.tokens.NextToken() // consume it
			return d, nil
		case tok.EOF:
			return nil, errDictionaryNotTerminated
		case tok.Name:
			key := tk.Value
			log.Parse.Printf("ParseDict: key = %s\n", key)
			_, _ = p.tokens.NextToken() // consume the key

			obj, err := p.ParseObject()
			if err != nil {
				return nil, err
			}
			// Specifying the null object as the value of a dictionary entry (7.3.7, "Dictionary Objects")
			// shall be equivalent to omitting the entry entirely.
			if obj != nil {
				log.Parse.Printf("ParseDict: dict[%s]=%v\n", key, obj)
				if _, has := d[key]; has {
					return nil, errDictionaryDuplicateKey
				} else {
					d[key] = obj
				}
			}
		default:
			return nil, errDictionaryCorrupt
		}
	}
	return nil, err
}

func parseBooleanOrNull(l string) (val Object, ok bool) {
	switch l {
	case "null": // null, absent object
		return nil, true
	case "true": // boolean true
		return Boolean(true), true
	case "false": // boolean false
		return Boolean(false), true
	default:
		return nil, false
	}
}

var tokenReference = tok.Token{Kind: tok.Other, Value: "R"}

func (p *Parser) parseNumericOrIndRef(currentToken tok.Token) (Object, error) {
	// if this object is an integer we need to check for an indirect reference eg. 1 0 R

	if currentToken.Kind != tok.Integer {
		return nil, fmt.Errorf("expected number got %v", currentToken)
	}

	i, err := currentToken.Int()
	if err != nil {
		return nil, err
	}
	// We have an Int!

	if p.ContentStreamMode {
		// in a content stream, no indirect reference is allowed:
		// return early
		return Integer(i), nil
	}

	next, err := p.tokens.PeekToken()
	if err != nil {
		return nil, err
	}

	// if not followed by whitespace return sole integer value.
	gen, err := next.Int()
	if next.Kind != tok.Integer || err != nil {
		log.Parse.Printf("parseNumericOrIndRef: value is numeric int: %d\n", i)
		return Integer(i), nil
	}

	// Must be indirect reference. (123 0 R)
	// Missing is the 2nd int and "R".

	// if only 2 token, can't be indirect reference.
	// if not followed by whitespace return sole integer value.
	if nextNext, _ := p.tokens.PeekPeekToken(); nextNext != tokenReference { // adjourn error checking
		log.Parse.Printf("parseNumericOrIndRef: 2 objects => value is numeric int: %d\n", i)
		return Integer(i), nil
	}

	// We have the 2nd int(generation number):
	// consume the tokens and return
	_, _ = p.tokens.NextToken()
	_, _ = p.tokens.NextToken()
	return IndirectRef{
		ObjectNumber:     Integer(i),
		GenerationNumber: Integer(gen),
	}, nil
}
