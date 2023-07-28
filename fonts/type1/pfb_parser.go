package type1

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/benoitkugler/pdf/fonts/simpleencodings"
	tk "github.com/benoitkugler/pstokenizer"
)

const (
	// constants for encryption
	eexecKey       = 55665
	CHARSTRING_KEY = 4330

	headerT11 = "%!FontType"
	headerT12 = "%!PS-AdobeFont"

	// start marker of a segment
	startMarker = 0x80

	// marker of the ascii segment
	asciiMarker = 0x01
)

func readOneRecord(pfb *bytes.Reader, expectedMarker byte, totalSize int64) ([]byte, error) {
	var buffer [6]byte

	_, err := io.ReadFull(pfb, buffer[:])
	if err != nil {
		return nil, fmt.Errorf("invalid .pfb file: missing record marker")
	}
	if buffer[0] != startMarker {
		return nil, errors.New("invalid .pfb file: start marker missing")
	}

	if buffer[1] != expectedMarker {
		return nil, errors.New("invalid .pfb file: incorrect record type")
	}

	size := int64(binary.LittleEndian.Uint32(buffer[2:]))
	if size >= totalSize {
		return nil, errors.New("corrupted .pfb file")
	}
	out := make([]byte, size)
	_, err = io.ReadFull(pfb, out)
	if err != nil {
		return nil, fmt.Errorf("invalid .pfb file: %s", err)
	}
	return out, nil
}

// fetchs the first segment of a .pfb font file.
// see https://www.adobe.com/content/dam/acom/en/devnet/font/pdfs/5040.Download_Fonts.pdf
// IBM PC format
func openPfb(pfb *bytes.Reader) (segment1 []byte, err error) {
	totalSize, err := pfb.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	_, err = pfb.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// ascii record
	segment1, err = readOneRecord(pfb, asciiMarker, totalSize)
	if err != nil {
		// try with the brute force approach for file who have no tag
		segment1, _, err = seekMarkers(pfb)
		return segment1, err
	}

	return segment1, nil
}

// fallback when no binary marker are present:
// we look for the currentfile exec pattern, then for the cleartomark
func seekMarkers(pfb *bytes.Reader) (segment1, segment2 []byte, err error) {
	_, err = pfb.Seek(0, io.SeekStart)
	if err != nil {
		return nil, nil, err
	}

	// quickly return for invalid files
	var buffer [len(headerT12)]byte
	io.ReadFull(pfb, buffer[:])
	if h := string(buffer[:]); !(strings.HasPrefix(h, headerT11) || strings.HasPrefix(h, headerT12)) {
		return nil, nil, errors.New("not a Type1 font file")
	}

	_, err = pfb.Seek(0, io.SeekStart)
	if err != nil {
		return nil, nil, err
	}
	data, err := io.ReadAll(pfb)
	if err != nil {
		return nil, nil, err
	}
	const exec = "currentfile eexec"
	index := bytes.Index(data, []byte(exec))
	if index == -1 {
		return nil, nil, errors.New("not a Type1 font file")
	}
	segment1 = data[:index+len(exec)]
	segment2 = data[index+len(exec):]
	if len(segment2) != 0 && tk.IsAsciiWhitespace(segment2[0]) { // end of line
		segment2 = segment2[1:]
	}
	return segment1, segment2, nil
}

type parser struct {
	lexer lexer
}

type lexer struct {
	tk.Tokenizer
}

// constructs a new lexer given a header-less .pfb segment
func newLexer(data []byte) lexer {
	return lexer{*tk.NewTokenizer(data)}
}

func (l *lexer) nextToken() (tk.Token, error) {
	return l.Tokenizer.NextToken()
}

func (l lexer) peekToken() tk.Token {
	t, _ := l.Tokenizer.PeekToken()
	return t
}

// Parses the ASCII portion of a Type 1 font.
func (p *parser) parseASCII(bytes []byte) (*simpleencodings.Encoding, error) {
	if len(bytes) == 0 {
		return nil, errors.New("bytes is empty")
	}

	// %!FontType1-1.0
	// %!PS-AdobeFont-1.0
	if len(bytes) < 2 || (bytes[0] != '%' && bytes[1] != '!') {
		return nil, errors.New("invalid start of ASCII segment")
	}

	p.lexer = newLexer(bytes)

	// (corrupt?) synthetic font
	if string(p.lexer.peekToken().Value) == "FontDirectory" {
		if err := p.readWithName(tk.Other, "FontDirectory"); err != nil {
			return nil, err
		}
		if _, err := p.read(tk.Name); err != nil { // font name;
			return nil, err
		}
		if err := p.readWithName(tk.Other, "known"); err != nil {
			return nil, err
		}
		if _, err := p.read(tk.StartProc); err != nil {
			return nil, err
		}
		if err := p.readProc(); err != nil {
			return nil, err
		}
		if _, err := p.read(tk.StartProc); err != nil {
			return nil, err
		}
		if err := p.readProc(); err != nil {
			return nil, err
		}
		if err := p.readWithName(tk.Other, "ifelse"); err != nil {
			return nil, err
		}
	}

	// font dict
	lengthT, err := p.read(tk.Integer)
	if err != nil {
		return nil, err
	}
	length, _ := lengthT.Int()
	if err := p.readWithName(tk.Other, "dict"); err != nil {
		return nil, err
	}
	// found in some TeX fonts
	if _, err := p.readMaybe(tk.Other, "dup"); err != nil {
		return nil, err
	}
	// if present, the "currentdict" is not required
	if err := p.readWithName(tk.Other, "begin"); err != nil {
		return nil, err
	}

	var out *simpleencodings.Encoding
	for i := 0; i < length; i++ {
		token := p.lexer.peekToken()
		if token.Kind == 0 { // premature end
			break
		}
		if token.IsOther("currentdict") || token.IsOther("end") {
			break
		}

		// key/value
		keyT, err := p.read(tk.Name)
		if err != nil {
			return nil, err
		}
		switch key := string(keyT.Value); key {
		case "FontInfo", "Fontinfo":
			err = p.readSimpleDict()
		case "Metrics":
			err = p.readSimpleDict()
		case "Encoding":
			out, err = p.readEncoding()
		default:
			err = p.readDictValue()
		}
		if err != nil {
			return nil, err
		}
	}

	// do not bother with the remaining bytes
	return out, nil
}

func (p *parser) readEncoding() (*simpleencodings.Encoding, error) {
	var out *simpleencodings.Encoding
	if p.lexer.peekToken().Kind == tk.Other {
		nameT, err := p.lexer.nextToken()
		if err != nil {
			return nil, err
		}
		name_ := string(nameT.Value)
		if name_ == "StandardEncoding" {
			out = &simpleencodings.AdobeStandard
		} else {
			return nil, errors.New("Unknown encoding: " + name_)
		}
		if _, err := p.readMaybe(tk.Other, "readonly"); err != nil {
			return nil, err
		}
		if err := p.readWithName(tk.Other, "def"); err != nil {
			return nil, err
		}
	} else {
		if _, err := p.read(tk.Integer); err != nil {
			return nil, err
		}
		if _, err := p.readMaybe(tk.Other, "array"); err != nil {
			return nil, err
		}

		// 0 1 255 {1 index exch /.notdef put } for
		// we have to check "readonly" and "def" too
		// as some fonts don't provide any dup-values, see PDFBOX-2134
		for {
			n := p.lexer.peekToken()
			if n.IsOther("dup") || n.IsOther("readonly") || n.IsOther("def") {
				break
			}
			_, err := p.lexer.nextToken()
			if err != nil {
				return nil, err
			}
		}

		out = new(simpleencodings.Encoding)
		for p.lexer.peekToken().IsOther("dup") {
			if err := p.readWithName(tk.Other, "dup"); err != nil {
				return nil, err
			}
			codeT, err := p.read(tk.Integer)
			if err != nil {
				return nil, err
			}
			code, _ := codeT.Int()
			nameT, err := p.read(tk.Name)
			if err != nil {
				return nil, err
			}
			if err := p.readWithName(tk.Other, "put"); err != nil {
				return nil, err
			}
			out[byte(code)] = string(nameT.Value)
		}
		if _, err := p.readMaybe(tk.Other, "readonly"); err != nil {
			return nil, err
		}
		if err := p.readWithName(tk.Other, "def"); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Reads a dictionary whose values are simple, i.e., do not contain nested dictionaries.
func (p *parser) readSimpleDict() error {
	lengthT, err := p.read(tk.Integer)
	if err != nil {
		return err
	}
	length, _ := lengthT.Int()
	if err := p.readWithName(tk.Other, "dict"); err != nil {
		return err
	}
	if _, err := p.readMaybe(tk.Other, "dup"); err != nil {
		return err
	}
	if err := p.readWithName(tk.Other, "begin"); err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		if p.lexer.peekToken().Kind == 0 {
			break
		}
		if p.lexer.peekToken().Kind == tk.Other &&
			!(string(p.lexer.peekToken().Value) == "end") {
			if _, err := p.read(tk.Other); err != nil {
				return err
			}
		}
		// premature end
		if p.lexer.peekToken().Kind == 0 {
			break
		}
		if p.lexer.peekToken().IsOther("end") {
			break
		}

		// simple value
		_, err := p.read(tk.Name)
		if err != nil {
			return err
		}
		err = p.readDictValue()
		if err != nil {
			return err
		}
	}

	if err := p.readWithName(tk.Other, "end"); err != nil {
		return err
	}
	if _, err := p.readMaybe(tk.Other, "readonly"); err != nil {
		return err
	}
	if err := p.readWithName(tk.Other, "def"); err != nil {
		return err
	}

	return nil
}

// Reads a simple value from a dictionary.
func (p *parser) readDictValue() error {
	err := p.readValue()
	if err != nil {
		return err
	}
	err = p.readDef()
	return err
}

// Reads a simple value. This is either a number, a string,
// a name, a literal name, an array, a procedure, or a charstring.
// This method does not support reading nested dictionaries unless they're empty.
func (p *parser) readValue() error {
	token, err := p.lexer.nextToken()
	if err != nil {
		return err
	}
	if p.lexer.peekToken().Kind == 0 {
		return nil
	}

	switch token.Kind {
	case tk.StartArray:
		openArray := 1
		for {
			if p.lexer.peekToken().Kind == 0 {
				return nil
			}
			if p.lexer.peekToken().Kind == tk.StartArray {
				openArray++
			}

			token, err = p.lexer.nextToken()
			if err != nil {
				return err
			}

			if token.Kind == tk.EndArray {
				openArray--
				if openArray == 0 {
					break
				}
			}
		}
	case tk.StartProc:
		err := p.readProc()
		if err != nil {
			return err
		}
	case tk.StartDic:
		// skip "/GlyphNames2HostCode << >> def"
		if _, err = p.read(tk.EndDic); err != nil {
			return err
		}
		return nil
	}
	err = p.readPostScriptWrapper()
	return err
}

func (p *parser) readPostScriptWrapper() error {
	// postscript wrapper (not in the Type 1 spec)
	if string(p.lexer.peekToken().Value) != "systemdict" {
		return nil
	}
	if err := p.readWithName(tk.Other, "systemdict"); err != nil {
		return err
	}
	if err := p.readWithName(tk.Name, "internaldict"); err != nil {
		return err
	}
	if err := p.readWithName(tk.Other, "known"); err != nil {
		return err
	}

	if _, err := p.read(tk.StartProc); err != nil {
		return err
	}
	if err := p.readProc(); err != nil {
		return err
	}

	if _, err := p.read(tk.StartProc); err != nil {
		return err
	}
	if err := p.readProc(); err != nil {
		return err
	}

	if err := p.readWithName(tk.Other, "ifelse"); err != nil {
		return err
	}

	// replace value
	if _, err := p.read(tk.StartProc); err != nil {
		return err
	}
	if err := p.readWithName(tk.Other, "pop"); err != nil {
		return err
	}
	err := p.readValue()
	if err != nil {
		return err
	}
	if _, err := p.read(tk.EndProc); err != nil {
		return err
	}

	if err := p.readWithName(tk.Other, "if"); err != nil {
		return err
	}
	return nil
}

// Reads a procedure.
func (p *parser) readProc() error {
	openProc := 1
	for {
		if p.lexer.peekToken().Kind == tk.StartProc {
			openProc++
		}

		token, err := p.lexer.nextToken()
		if err != nil {
			return err
		}

		if token.Kind == tk.EndProc {
			openProc--
			if openProc == 0 {
				break
			}
		}
	}
	_, err := p.readMaybe(tk.Other, "executeonly")
	return err
}

// Reads the sequence "noaccess def" or equivalent.
func (p *parser) readDef() error {
	if _, err := p.readMaybe(tk.Other, "readonly"); err != nil {
		return err
	}
	// allows "noaccess ND" (not in the Type 1 spec)
	if _, err := p.readMaybe(tk.Other, "noaccess"); err != nil {
		return err
	}

	token, err := p.read(tk.Other)
	if err != nil {
		return err
	}
	switch string(token.Value) {
	case "ND", "|-":
		return nil
	case "noaccess":
		token, err = p.read(tk.Other)
		if err != nil {
			return err
		}
	}
	if string(token.Value) == "def" {
		return nil
	}
	return fmt.Errorf("found %s but expected ND", token.Value)
}

// / Reads the next token and throws an error if it is not of the given kind.
func (p *parser) read(kind tk.Kind) (tk.Token, error) {
	token, err := p.lexer.nextToken()
	if err != nil {
		return tk.Token{}, err
	}
	if token.Kind != kind {
		return tk.Token{}, fmt.Errorf("found token %s (%s) but expected token %s", token.Kind, token.Value, kind)
	}
	return token, nil
}

// Reads the next token and throws an error if it is not of the given kind
// and does not have the given value.
func (p *parser) readWithName(kind tk.Kind, name string) error {
	token, err := p.read(kind)
	if err != nil {
		return err
	}
	if string(token.Value) != name {
		return fmt.Errorf("found %s but expected %s", token.Value, name)
	}
	return nil
}

// Reads the next token if and only if it is of the given kind and
// has the given value.
func (p *parser) readMaybe(kind tk.Kind, name string) (tk.Token, error) {
	token := p.lexer.peekToken()
	if token.Kind == kind && string(token.Value) == name {
		return p.lexer.nextToken()
	}
	return tk.Token{}, nil
}
