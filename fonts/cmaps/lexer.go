package cmaps

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	tokenizer "github.com/benoitkugler/pstokenizer"
)

type cmapObject interface {
}

type cmapOperand string

// cmapHexString represents a PostScript hex string such as <FFFF>
type cmapHexString []byte

type cmapArray = []cmapObject

type cmapDict = map[model.ObjName]cmapObject

// parseObject detects the signature at the current file position and parses the corresponding object.
// a nil object with a nil error means EOF
func (p *parser) parseObject() (cmapObject, error) {
	token, err := p.tokenizer.NextToken()
	for ; token.Kind != tokenizer.EOF && err == nil; token, err = p.tokenizer.NextToken() {
		switch token.Kind {
		case tokenizer.Name:
			return model.ObjName(token.Value), nil
		case tokenizer.String:
			return string(token.Value), nil
		case tokenizer.StringHex:
			return cmapHexString(token.Value), nil
		case tokenizer.StartArray:
			return p.parseArray()
		case tokenizer.StartDic:
			return p.parseDict()
		case tokenizer.Integer:
			v, err := token.Int()
			if err != nil {
				v = 0
			}
			return v, nil
		case tokenizer.Float:
			v, err := token.Float()
			if err != nil {
				v = 0
			}
			return v, nil
		case tokenizer.EndArray, tokenizer.EndDic: // should not happend here
			return nil, errors.New("unexpected end of container")
		case tokenizer.Other:
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
	for ; token.Kind != tokenizer.EOF && err == nil; token, err = p.tokenizer.PeekToken() {
		switch token.Kind {
		case tokenizer.EndArray:
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
	for ; token.Kind != tokenizer.EOF && err == nil; token, err = p.tokenizer.NextToken() {
		switch token.Kind {
		case tokenizer.Name: // key
			key := model.ObjName(token.Value)
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
			if token.IsOther("def") {
				_, _ = p.tokenizer.NextToken() // consume it
			}
		case tokenizer.EndDic:
			return dict, nil
		default:
			return nil, fmt.Errorf("invalid token in dict %v", token)
		}
	}
	return nil, err
}
