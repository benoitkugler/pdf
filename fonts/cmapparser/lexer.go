package parser

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/pdftokenizer"
)

type cmapObject interface {
}

type cmapOperand string

// cmapHexString represents a PostScript hex string such as <FFFF>
type cmapHexString []byte

type cmapArray = []cmapObject

type cmapDict = map[model.Name]cmapObject

// parseObject detects the signature at the current file position and parses the corresponding object.
// a nil object with a nil error means EOF
func (p *parser) parseObject() (cmapObject, error) {
	token, err := p.tokenizer.NextToken()
	for ; token.Kind != pdftokenizer.EOF && err == nil; token, err = p.tokenizer.NextToken() {
		switch token.Kind {
		case pdftokenizer.Name:
			return model.Name(token.Value), nil
		case pdftokenizer.String:
			return token.Value, nil
		case pdftokenizer.StringHex:
			return cmapHexString(token.Value), nil
		case pdftokenizer.StartArray:
			return p.parseArray()
		case pdftokenizer.StartDic:
			return p.parseDict()
		case pdftokenizer.Integer:
			v, err := token.Int()
			if err != nil {
				v = 0
			}
			return v, nil
		case pdftokenizer.Float:
			v, err := token.Float()
			if err != nil {
				v = 0
			}
			return v, nil
		case pdftokenizer.EndArray, pdftokenizer.EndDic: // should not happend here
			return nil, errors.New("unexpected end of container")
		case pdftokenizer.Other:
			return cmapOperand(token.Value), nil
		}
		// default: continue
	}
	return nil, err
}

// parseArray parses a PDF array, which starts with '[', ends with ']'and can contain any kinds of
// direct objects.
func (p *parser) parseArray() (cmapArray, error) {
	var arr cmapArray
	token, err := p.tokenizer.PeekToken()
	for ; token.Kind != pdftokenizer.EOF && err == nil; token, err = p.tokenizer.PeekToken() {
		switch token.Kind {
		case pdftokenizer.EndArray:
			// consume
			_, _ = p.tokenizer.NextToken()
			return arr, nil
		default:
			obj, err := p.parseObject()
			if err != nil {
				return nil, err
			}
			arr = append(arr, obj)
		}
	}
	return nil, err
}

// parseDict parses a PDF dictionary object, which starts with with '<<' and ends with '>>'.
func (p *parser) parseDict() (cmapDict, error) {
	dict := cmapDict{}
	token, err := p.tokenizer.NextToken()
	for ; token.Kind != pdftokenizer.EOF && err == nil; token, err = p.tokenizer.NextToken() {
		switch token.Kind {
		case pdftokenizer.Name: // key
			key := model.Name(token.Value)
			value, err := p.parseObject()
			if err != nil {
				return nil, err
			}
			dict[key] = value

			// Skip "def" which optionally follows key value dict definitions in CMaps.
			token, err = p.tokenizer.PeekToken()
			if err != nil {
				return nil, err
			}
			if token.Kind == pdftokenizer.Other && token.Value == "def" {
				_, _ = p.tokenizer.NextToken() // consume it
			}
		case pdftokenizer.EndDic:
			return dict, nil
		default:
			return nil, fmt.Errorf("invalid token in dict %v", token)
		}
	}
	return nil, err
}
