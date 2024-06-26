// Implements a PDF object parser, mapping a list of tokens (see the tokenizer package) into
// tree-like structure.
// Higher-level reader is neeed to decrypt a full PDF file.
package parser

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	tkn "github.com/benoitkugler/pstokenizer"
)

var (
	errArrayNotTerminated      = errors.New("parse: unterminated array")
	errDictionaryCorrupt       = errors.New("parse: corrupted dictionary")
	errDictionaryDuplicateKey  = errors.New("parse: duplicate key")
	errDictionaryNotTerminated = errors.New("parse: unterminated dictionary")
	errBufNotAvailable         = errors.New("parse: no buffer available")
)

type (
	Object        = model.Object
	Name          = model.Name
	Integer       = model.ObjInt
	Float         = model.ObjFloat
	StringLiteral = model.ObjStringLiteral
	HexLiteral    = model.ObjHexLiteral
	Array         = model.ObjArray
	Dict          = model.ObjDict
	Bool          = model.ObjBool
	Command       = model.ObjCommand
	IndirectRef   = model.ObjIndirectRef
)

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
	case tkn.String:
		value = StringLiteral(tk.Value)
	case tkn.StringHex:
		value = HexLiteral(tk.Value)
	case tkn.StartArray:
		arr, err := p.parseArray()
		if err != nil {
			return nil, err
		}
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
		value = dict
	case tkn.Float:
		// We have a Float!
		f, err := tk.Float()
		if err != nil {
			return nil, err
		}
		value = Float(f)
	case tkn.Other:
		value, err = p.parseOther(tk.Value)
	default:
		// Must be numeric or indirect reference:
		// int 0 r
		// int
		// float
		value, err = p.parseNumericOrIndRef(tk)
	}

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

func (p Parser) parseOther(l []byte) (Object, error) {
	switch string(l) {
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
		return Integer(i), nil
	}

	// Must be indirect reference. (123 0 R)
	// Missing is the 2nd int and "R".

	// if only 2 token, can't be indirect reference.
	// if not followed by whitespace return sole integer value.
	if nextNext, _ := p.tokens.PeekPeekToken(); !nextNext.IsOther("R") { // adjourn error checking
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
	if !tok.IsOther("obj") {
		return 0, 0, nil, errors.New("pdfcpu: ParseObjectDefinition: can't find \"obj\"")
	}

	if headerOnly {
		return objNr, genNr, nil, nil
	}

	pr := Parser{tokens: tokens}
	obj, err := pr.ParseObject()
	return objNr, genNr, obj, err
}
