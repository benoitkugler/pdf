// Implements a PDF object parser, mapping a list of tokens (see the tokenizer package) into
// tree-like structure.
// Higher-level reader is neeed to decrypt a full PDF file.
package parser

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	tkn "github.com/benoitkugler/pdf/reader/parser/tokenizer"
	"github.com/pdfcpu/pdfcpu/pkg/log"
)

var (
	errArrayNotTerminated      = errors.New("pdfcpu: parse: unterminated array")
	errDictionaryCorrupt       = errors.New("pdfcpu: parse: corrupt dictionary")
	errDictionaryDuplicateKey  = errors.New("pdfcpu: parse: duplicate key")
	errDictionaryNotTerminated = errors.New("pdfcpu: parse: unterminated dictionary")
	errBufNotAvailable         = errors.New("pdfcpu: parse: no buffer available")
)

type Object = model.Object
type Name = model.Name
type Integer = model.ObjInt
type Float = model.ObjFloat
type StringLiteral = model.ObjStringLiteral
type HexLiteral = model.ObjHexLiteral
type Array = model.ObjArray
type Dict = model.ObjDict
type Bool = model.ObjBool
type Command = model.ObjCommand
type IndirectRef = model.ObjIndirectRef

// Standalone implementation of a PDF parser.
// The parser only handles chunks of PDF files
// (corresponding for example to object definitions),
// but cannot handle a full file with streams.
// An higher-level reader is needed to decode Streams and Inline Data,
// which require knowledge on the filters used.
type Parser struct {
	tokens *tkn.Tokenizer

	// If true, disallow Indirect Reference,
	// but allow Commands
	ContentStreamMode bool

	opsStack []Object // only used in content stream
}

// NewParser uses a byte slice as input.
func NewParser(data []byte) *Parser {
	return NewParserFromTokenizer(tkn.NewTokenizer(data))
}

// NewParserFromTokenizer use a tokenizer as input.
func NewParserFromTokenizer(tokens *tkn.Tokenizer) *Parser {
	return &Parser{tokens: tokens}
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
	case tkn.EOF:
		err = errBufNotAvailable
	case tkn.Name:
		value = Name(tk.Value)
		log.Parse.Printf("ParseObject: value = Name Object : %s\n", value)
	case tkn.String:
		value = StringLiteral(tk.Value)
		log.Parse.Printf("ParseObject: value = String Literal: <%s>\n", value)
	case tkn.StringHex:
		value = HexLiteral(tk.Value)
		log.Parse.Printf("ParseObject: value = Hex Literal: <%s>\n", value)
	case tkn.StartArray:
		log.Parse.Println("ParseObject: value = Array")
		arr, err := p.parseArray()
		if err != nil {
			return nil, err
		}
		log.Parse.Printf("ParseArray: returning array (len=%d): %v\n", len(arr), arr)
		value = arr
	case tkn.StartDic:
		// Hack for #252: we start by parsing according to the SPEC
		// which will be almost always successful
		save := p.tokens.CurrentPosition()
		dict, err := p.parseDict(false)
		if err != nil {
			// try relaxed
			p.tokens.SetPosition(save)
			dict, err = p.parseDict(true)
		}
		if err != nil {
			return nil, err
		}
		log.Parse.Printf("ParseDict: returning dict (len=%d): %v\n", len(dict), dict)
		value = dict
	case tkn.Float:
		// We have a Float!
		f, err := tk.Float()
		if err != nil {
			return nil, err
		}
		log.Parse.Printf("ParseObject: value = Float: %f\n", f)
		value = Float(f)
	case tkn.Other:
		value, err = p.parseOther(tk.Value)
		log.Parse.Println("parseOther: returning: %v", value)
	default:
		// Must be numeric or indirect reference:
		// int 0 r
		// int
		// float
		value, err = p.parseNumericOrIndRef(tk)
	}

	log.Parse.Printf("ParseObject returning %v\n", value)
	return value, err
}

func (p *Parser) parseArray() (Array, error) {
	a := Array{}
	tk, err := p.tokens.PeekToken()
	for ; err == nil; tk, err = p.tokens.PeekToken() {
		switch tk.Kind {
		case tkn.EndArray:
			_, _ = p.tokens.NextToken() // consume it
			return a, nil
		case tkn.EOF:
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

func (p *Parser) parseDict(relaxed bool) (Dict, error) {
	d := Dict{}

	tk, err := p.tokens.PeekToken()
	for ; err == nil; tk, err = p.tokens.PeekToken() {
		switch tk.Kind {
		case tkn.EndDic:
			_, _ = p.tokens.NextToken() // consume it
			return d, nil
		case tkn.EOF:
			return nil, errDictionaryNotTerminated
		case tkn.Name:
			key := tk.Value
			log.Parse.Printf("ParseDict: key = %s\n", key)
			_, _ = p.tokens.NextToken() // consume the key

			var obj Object

			// A friendly 🤢 to the devs of the Kdan Pocket Scanner for the iPad.
			// Hack for #252:
			// For dicts with kv pairs terminated by eol we accept a missing value as an empty string.
			if relaxed && p.tokens.HasEOLBeforeToken() {
				obj = StringLiteral("")
			} else {
				obj, err = p.ParseObject()
				if err != nil {
					return nil, err
				}
			}

			// Specifying the null object as the value of a dictionary entry (7.3.7, "Dictionary Objects")
			// shall be equivalent to omitting the entry entirely.
			if obj != nil {
				log.Parse.Printf("ParseDict: dict[%s]=%v\n", key, obj)
				if _, has := d[Name(key)]; has {
					return nil, errDictionaryDuplicateKey
				} else {
					d[Name(key)] = obj
				}
			}
		default:
			return nil, errDictionaryCorrupt
		}
	}
	return nil, err
}

func (p Parser) parseOther(l string) (Object, error) {
	switch l {
	case "null": // null, absent object
		return model.ObjNull{}, nil
	case "true": // boolean true
		return Bool(true), nil
	case "false": // boolean false
		return Bool(false), nil
	default:
		if p.ContentStreamMode {
			return Command(l), nil
		} else {
			return nil, fmt.Errorf("unexpected command %s outside of Content Stream", l)
		}
	}
}

var tokenReference = tkn.Token{Kind: tkn.Other, Value: "R"}

func (p *Parser) parseNumericOrIndRef(currentToken tkn.Token) (Object, error) {
	// if this object is an integer we need to check for an indirect reference eg. 1 0 R

	if currentToken.Kind != tkn.Integer {
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
	if next.Kind != tkn.Integer || err != nil {
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
	return IndirectRef{ObjectNumber: i, GenerationNumber: gen}, nil
}

// ParseObjectDefinition parses an object definition.
// If `headerOnly`, stops after the X X obj header and return a nil object.
func ParseObjectDefinition(line []byte, headerOnly bool) (objectNumber int, generationNumber int, o Object, err error) {
	log.Parse.Printf("ParseObjectDefinition: buf=<%s>\n", line)

	tokens := tkn.NewTokenizer(line)

	// object number
	tok, err := tokens.NextToken()
	if err != nil {
		return 0, 0, nil, err
	}
	objNr, err := tok.Int()
	if tok.Kind != tkn.Integer || err != nil {
		return 0, 0, nil, errors.New("pdfcpu: ParseObjectDefinition: can't find object number")
	}

	// generation number
	tok, err = tokens.NextToken()
	if err != nil {
		return 0, 0, nil, err
	}
	genNr, err := tok.Int()
	if tok.Kind != tkn.Integer || err != nil {
		return 0, 0, nil, errors.New("pdfcpu: ParseObjectDefinition: can't find generation number")
	}

	tok, err = tokens.NextToken()
	if err != nil {
		return 0, 0, nil, errors.New("pdfcpu: ParseObjectDefinition: can't find \"obj\"")
	}
	if tok != (tkn.Token{Kind: tkn.Other, Value: "obj"}) {
		return 0, 0, nil, errors.New("pdfcpu: ParseObjectDefinition: can't find \"obj\"")
	}

	if headerOnly {
		return objNr, genNr, nil, nil
	}

	pr := Parser{tokens: tokens}
	obj, err := pr.ParseObject()
	return objNr, genNr, obj, err
}
