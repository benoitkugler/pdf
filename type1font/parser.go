package type1font

import (
	"errors"
	"fmt"
)

// constants for encryption
const (
	EEXEC_KEY      = 55665
	CHARSTRING_KEY = 4330
)

type parser struct {
	// state
	lexer lexer
	font  Font
}

// Encoding is either the standard encoding, or defined by the font
type Encoding struct {
	Standard bool
	Custom   map[byte]string
}

type Font struct {
	FontName    string
	PaintType   int
	FontType    int
	UniqueID    int
	StrokeWidth float64
	FontID      string
	FontMatrix  []float64
	FontBBox    []float64
	Encoding    Encoding
}

/*
 Parses an Adobe Type 1 (.pfb) font, composed of `segment1` (ASCII) and `segment2` (Binary).
 It is used exclusively in Type1 font.

 The Type 1 font format is a free-text format which is somewhat difficult
 to parse. This is made worse by the fact that many Type 1 font files do
 not conform to the specification, especially those embedded in PDFs. This
 parser therefore tries to be as forgiving as possible.

 See "Adobe Type 1 Font Format, Adobe Systems (1999)"

 Ported from the code from John Hewson
*/
func Parse(segment1, segment2 []byte) (Font, error) {

	p := parser{}
	err := p.parseASCII(segment1)
	if err != nil {
		return Font{}, err
	}
	if len(segment2) > 0 { // TODO:
		// parser.parseBinary(segment2)
	}
	return p.font, nil
}

// Parses the ASCII portion of a Type 1 font.
func (p *parser) parseASCII(bytes []byte) error {
	if len(bytes) == 0 {
		return errors.New("bytes is empty")
	}

	// %!FontType1-1.0
	// %!PS-AdobeFont-1.0
	if len(bytes) < 2 || (bytes[0] != '%' && bytes[1] != '!') {
		return errors.New("Invalid start of ASCII segment")
	}

	var err error
	p.lexer, err = newLexer(bytes)
	if err != nil {
		return err
	}

	// (corrupt?) synthetic font
	if p.lexer.peekToken().value == "FontDirectory" {
		if err := p.readWithName(name, "FontDirectory"); err != nil {
			return err
		}
		if _, err := p.read(literal); err != nil { // font name;
			return err
		}
		if err := p.readWithName(name, "known"); err != nil {
			return err
		}
		if _, err := p.read(startProc); err != nil {
			return err
		}
		if _, err := p.readProc(); err != nil {
			return err
		}
		if _, err := p.read(startProc); err != nil {
			return err
		}
		if _, err := p.readProc(); err != nil {
			return err
		}
		if err := p.readWithName(name, "ifelse"); err != nil {
			return err
		}
	}

	// font dict
	lengthT, err := p.read(integer)
	if err != nil {
		return err
	}
	length := lengthT.intValue()
	if err := p.readWithName(name, "dict"); err != nil {
		return err
	}
	// found in some TeX fonts
	if _, err := p.readMaybe(name, "dup"); err != nil {
		return err
	}
	// if present, the "currentdict" is not required
	if err := p.readWithName(name, "begin"); err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		// premature end
		token := p.lexer.peekToken()
		if token == none {
			break
		}
		if token.kind == name && ("currentdict" == token.value || "end" == token.value) {
			break
		}

		// key/value
		keyT, err := p.read(literal)
		if err != nil {
			return err
		}
		switch key := keyT.value; key {
		case "FontInfo", "Fontinfo":
			dict, err := p.readSimpleDict()
			if err != nil {
				return err
			}
			p.readFontInfo(dict)
		case "Metrics":
			_, err = p.readSimpleDict()
		case "Encoding":
			err = p.readEncoding()
		default:
			err = p.readSimpleValue(key)
		}
		if err != nil {
			return err
		}
	}

	if _, err := p.readMaybe(name, "currentdict"); err != nil {
		return err
	}
	if err := p.readWithName(name, "end"); err != nil {
		return err
	}
	if err := p.readWithName(name, "currentfile"); err != nil {
		return err
	}
	if err := p.readWithName(name, "eexec"); err != nil {
		return err
	}
	return nil
}

func (p *parser) readSimpleValue(key string) error {
	value, err := p.readDictValue()
	if err != nil {
		return err
	}
	switch key {
	case "FontName", "PaintType", "FontType", "UniqueID", "StrokeWidth", "FID":
		if len(value) == 0 {
			return errors.New("missing value")
		}
	}
	switch key {
	case "FontName":
		p.font.FontName = value[0].value
	case "PaintType":
		p.font.PaintType = value[0].intValue()
	case "FontType":
		p.font.FontType = value[0].intValue()
	case "UniqueID":
		p.font.UniqueID = value[0].intValue()
	case "StrokeWidth":
		p.font.StrokeWidth = value[0].floatValue()
	case "FID":
		p.font.FontID = value[0].value
	case "FontMatrix":
		p.font.FontMatrix, err = p.arrayToNumbers(value)
	case "FontBBox":
		p.font.FontBBox, err = p.arrayToNumbers(value)
	}
	return err
}

func (p *parser) readEncoding() error {
	if p.lexer.peekToken().kind == name {
		nameT, err := p.lexer.nextToken()
		if err != nil {
			return err
		}
		name_ := nameT.value
		if name_ == "StandardEncoding" {
			p.font.Encoding.Standard = true
		} else {
			return errors.New("Unknown encoding: " + name_)
		}
		if _, err := p.readMaybe(name, "readonly"); err != nil {
			return err
		}
		if err := p.readWithName(name, "def"); err != nil {
			return err
		}
	} else {
		if _, err := p.read(integer); err != nil {
			return err
		}
		if _, err := p.readMaybe(name, "array"); err != nil {
			return err
		}

		// 0 1 255 {1 index exch /.notdef put } for
		// we have to check "readonly" and "def" too
		// as some fonts don't provide any dup-values, see PDFBOX-2134
		for !(p.lexer.peekToken().kind == name &&
			(p.lexer.peekToken().value == "dup" ||
				p.lexer.peekToken().value == "readonly" ||
				p.lexer.peekToken().value == "def")) {
			_, err := p.lexer.nextToken()
			if err != nil {
				return err
			}
		}

		codeToName := map[byte]string{}
		for p.lexer.peekToken().kind == name &&
			p.lexer.peekToken().value == "dup" {
			if err := p.readWithName(name, "dup"); err != nil {
				return err
			}
			codeT, err := p.read(integer)
			if err != nil {
				return err
			}
			code := codeT.intValue()
			nameT, err := p.read(literal)
			if err != nil {
				return err
			}
			if err := p.readWithName(name, "put"); err != nil {
				return err
			}
			codeToName[byte(code)] = nameT.value
		}
		p.font.Encoding.Custom = codeToName
		if _, err := p.readMaybe(name, "readonly"); err != nil {
			return err
		}
		if err := p.readWithName(name, "def"); err != nil {
			return err
		}
	}
	return nil
}

// Extracts values from an array as numbers.
func (p *parser) arrayToNumbers(value []token) ([]float64, error) {
	var numbers []float64
	for i, size := 1, len(value)-1; i < size; i++ {
		token := value[i]
		if token.kind == real || token.kind == integer {
			numbers = append(numbers, token.floatValue())
		} else {
			return nil, fmt.Errorf("Expected INTEGER or REAL but got %s", token.kind)
		}
	}
	return numbers, nil
}

// Extracts values from the /FontInfo dictionary.
// TODO: complete
func (p *parser) readFontInfo(fontInfo map[string][]token) {
	// for key, value := range fontInfo {
	// 	switch key {
	// 	case "version":
	// 		font.version = value[0].value
	// 	case "Notice":
	// 		font.notice = value[0].value
	// 	case "FullName":
	// 		font.fullName = value[0].value
	// 	case "FamilyName":
	// 		font.familyName = value[0].value
	// 	case "Weight":
	// 		font.weight = value[0].value
	// 	case "ItalicAngle":
	// 		font.italicAngle = value[0].floatValue()
	// 	case "isFixedPitch":
	// 		font.isFixedPitch = value[0].booleanValue()
	// 	case "UnderlinePosition":
	// 		font.underlinePosition = value[0].floatValue()
	// 	case "UnderlineThickness":
	// 		font.underlineThickness = value[0].floatValue()
	// 	}
	// }
}

// Reads a dictionary whose values are simple, i.e., do not contain nested dictionaries.
func (p *parser) readSimpleDict() (map[string][]token, error) {
	dict := map[string][]token{}

	lengthT, err := p.read(integer)
	if err != nil {
		return nil, err
	}
	length := lengthT.intValue()
	if err := p.readWithName(name, "dict"); err != nil {
		return nil, err
	}
	if _, err := p.readMaybe(name, "dup"); err != nil {
		return nil, err
	}
	if err := p.readWithName(name, "begin"); err != nil {
		return nil, err
	}

	for i := 0; i < length; i++ {
		if p.lexer.peekToken() == none {
			break
		}
		if p.lexer.peekToken().kind == name &&
			!(p.lexer.peekToken().value == "end") {
			if _, err := p.read(name); err != nil {
				return nil, err
			}
		}
		// premature end
		if p.lexer.peekToken() == none {
			break
		}
		if p.lexer.peekToken().kind == name &&
			p.lexer.peekToken().value == "end" {
			break
		}

		// simple value
		keyT, err := p.read(literal)
		if err != nil {
			return nil, err
		}
		value, err := p.readDictValue()
		if err != nil {
			return nil, err
		}
		dict[keyT.value] = value
	}

	if err := p.readWithName(name, "end"); err != nil {
		return nil, err
	}
	if _, err := p.readMaybe(name, "readonly"); err != nil {
		return nil, err
	}
	if err := p.readWithName(name, "def"); err != nil {
		return nil, err
	}

	return dict, nil
}

// Reads a simple value from a dictionary.
func (p *parser) readDictValue() ([]token, error) {
	value, err := p.readValue()
	if err != nil {
		return nil, err
	}
	err = p.readDef()
	return value, err
}

// Reads a simple value. This is either a number, a string,
// a name, a literal name, an array, a procedure, or a charstring.
// This method does not support reading nested dictionaries unless they're empty.
func (p *parser) readValue() ([]token, error) {
	var value []token
	token, err := p.lexer.nextToken()
	if err != nil {
		return nil, err
	}
	if p.lexer.peekToken() == none {
		return value, nil
	}
	value = append(value, token)

	switch token.kind {
	case startArray:
		openArray := 1
		for {
			if p.lexer.peekToken() == none {
				return value, nil
			}
			if p.lexer.peekToken().kind == startArray {
				openArray++
			}

			token, err = p.lexer.nextToken()
			if err != nil {
				return nil, err
			}
			value = append(value, token)

			if token.kind == endArray {
				openArray--
				if openArray == 0 {
					break
				}
			}
		}
	case startProc:
		proc, err := p.readProc()
		if err != nil {
			return nil, err
		}
		value = append(value, proc...)
	case startDict:
		// skip "/GlyphNames2HostCode << >> def"
		if _, err := p.read(endDict); err != nil {
			return nil, err
		}
		return value, nil
	}
	err = p.readPostScriptWrapper(value)
	return value, err
}

func (p *parser) readPostScriptWrapper(value []token) error {
	// postscript wrapper (not in the Type 1 spec)
	if p.lexer.peekToken().value != "systemdict" {
		return nil
	}
	if err := p.readWithName(name, "systemdict"); err != nil {
		return err
	}
	if err := p.readWithName(literal, "internaldict"); err != nil {
		return err
	}
	if err := p.readWithName(name, "known"); err != nil {
		return err
	}

	if _, err := p.read(startProc); err != nil {
		return err
	}
	if _, err := p.readProc(); err != nil {
		return err
	}

	if _, err := p.read(startProc); err != nil {
		return err
	}
	if _, err := p.readProc(); err != nil {
		return err
	}

	if err := p.readWithName(name, "ifelse"); err != nil {
		return err
	}

	// replace value
	if _, err := p.read(startProc); err != nil {
		return err
	}
	if err := p.readWithName(name, "pop"); err != nil {
		return err
	}
	value = nil
	other, err := p.readValue()
	if err != nil {
		return err
	}
	value = append(value, other...)
	if _, err := p.read(endProc); err != nil {
		return err
	}

	if err := p.readWithName(name, "if"); err != nil {
		return err
	}
	return nil
}

// Reads a procedure.
func (p *parser) readProc() ([]token, error) {
	var value []token
	openProc := 1
	for {
		if p.lexer.peekToken().kind == startProc {
			openProc++
		}

		token, err := p.lexer.nextToken()
		if err != nil {
			return nil, err
		}
		value = append(value, token)

		if token.kind == endProc {
			openProc--
			if openProc == 0 {
				break
			}
		}
	}
	executeonly, err := p.readMaybe(name, "executeonly")
	if err != nil {
		return nil, err
	}
	if executeonly != none {
		value = append(value, executeonly)
	}

	return value, nil
}

// // Parses the binary portion of a Type 1 font.
// func (p *Parser) parseBinary(bytes []byte) error {
// 	var decrypted []byte
// 	// Sometimes, fonts use the hex format, so this needs to be converted before decryption
// 	if isBinary(bytes) {
// 		decrypted = decrypt(bytes, EEXEC_KEY, 4)
// 	} else {
// 		decrypted = decrypt(hexToBinary(bytes), EEXEC_KEY, 4)
// 	}
// 	lexer := lexer{data: decrypted}

// 	// find /Private dict
// 	peekToken := lexer.peekToken()
// 	for peekToken != none && !peekToken.value == "Private" {
// 		// for a more thorough validation, the presence of "begin" before Private
// 		// determines how code before and following charstrings should look
// 		// it is not currently checked anyway
// 		lexer.nextToken()
// 		peekToken = lexer.peekToken()
// 	}
// 	if peekToken == none {
// 		return errors.New("/Private token not found")
// 	}

// 	// Private dict
// 	read(literal, "Private")
// 	length := read(integer).intValue()
// 	read(name, "dict")
// 	// actually could also be "/Private 10 dict def Private begin"
// 	// instead of the "dup"
// 	p.readMaybe(name, "dup")
// 	read(name, "begin")

// 	lenIV := 4 // number of random bytes at start of charstring

// 	for i := 0; i < length; i++ {
// 		// premature end
// 		if lexer.peekToken() == none || lexer.peekToken().kind != literal {
// 			break
// 		}

// 		// key/value
// 		key := read(literal).value

// 		switch key {
// 		case "Subrs":
// 			readSubrs(lenIV)

// 		case "OtherSubrs":
// 			readOtherSubrs()

// 		case "lenIV":
// 			lenIV = readDictValue()[0].intValue()

// 		case "ND":
// 			read(startProc)
// 			// the access restrictions are not mandatory
// 			p.readMaybe(name, "noaccess")
// 			read(name, "def")
// 			read(token.END_PROC)
// 			p.readMaybe(name, "executeonly")
// 			read(name, "def")

// 		case "NP":
// 			read(startProc)
// 			p.readMaybe(name, "noaccess")
// 			read(name)
// 			read(token.END_PROC)
// 			p.readMaybe(name, "executeonly")
// 			read(name, "def")

// 		case "RD":
// 			// /RD {string currentfile exch readstring pop} bind executeonly def
// 			read(startProc)
// 			readProc()
// 			p.readMaybe(name, "bind")
// 			p.readMaybe(name, "executeonly")
// 			read(name, "def")

// 		default:
// 			readPrivate(key, readDictValue())

// 		}
// 	}

// 	// some fonts have "2 index" here, others have "end noaccess put"
// 	// sometimes followed by "put". Either way, we just skip until
// 	// the /CharStrings dict is found
// 	for !(lexer.peekToken().kind == literal &&
// 		lexer.peekToken().value == "CharStrings") {
// 		lexer.nextToken()
// 	}

// 	// CharStrings dict
// 	read(literal, "CharStrings")
// 	readCharStrings(lenIV)
// }

// 	 /**
// 	  * Extracts values from the /Private dictionary.
// 	  */
//func (p *Parser) void readPrivate(String key, List<token> value) error
// 	 {
// 		 switch (key)
// 		 {
// 			 case "BlueValues":
// 				 font.blueValues = arrayToNumbers(value);
// 				 break;
// 			 case "OtherBlues":
// 				 font.otherBlues = arrayToNumbers(value);
// 				 break;
// 			 case "FamilyBlues":
// 				 font.familyBlues = arrayToNumbers(value);
// 				 break;
// 			 case "FamilyOtherBlues":
// 				 font.familyOtherBlues = arrayToNumbers(value);
// 				 break;
// 			 case "BlueScale":
// 				 font.blueScale = value[0].floatValue();
// 				 break;
// 			 case "BlueShift":
// 				 font.blueShift = value[0].intValue();
// 				 break;
// 			 case "BlueFuzz":
// 				 font.blueFuzz = value[0].intValue();
// 				 break;
// 			 case "StdHW":
// 				 font.stdHW = arrayToNumbers(value);
// 				 break;
// 			 case "StdVW":
// 				 font.stdVW = arrayToNumbers(value);
// 				 break;
// 			 case "StemSnapH":
// 				 font.stemSnapH = arrayToNumbers(value);
// 				 break;
// 			 case "StemSnapV":
// 				 font.stemSnapV = arrayToNumbers(value);
// 				 break;
// 			 case "ForceBold":
// 				 font.forceBold = value[0].booleanValue();
// 				 break;
// 			 case "LanguageGroup":
// 				 font.languageGroup = value[0].intValue();
// 				 break;
// 			 default:
// 				 break;
// 		 }
// 	 }

// 	 /**
// 	  * Reads the /Subrs array.
// 	  * @param lenIV The number of random bytes used in charstring encryption.
// 	  */
//func (p *Parser) void readSubrs(int lenIV) error
// 	 {
// 		 // allocate size (array indexes may not be in-order)
// 		  length := read(integer).intValue();
// 		 for (int i = 0; i < length; i++)
// 		 {
// 			 font.subrs.add(none);
// 		 }
// 		 read(name, "array");

// 		 for (int i = 0; i < length; i++)
// 		 {
// 			 // premature end
// 			 if (lexer.peekToken() == none)
// 			 {
// 				 break;
// 			 }
// 			 if (!(lexer.peekToken().kind == name &&
// 				   lexer.peekToken().value == "dup")))
// 			 {
// 				 break;
// 			 }

// 			 read(name, "dup");
// 			 token index = read(integer);
// 			 read(integer);

// 			 // RD
// 			 token charstring = read(token.CHARSTRING);
// 			 font.subrs.set(index.intValue(), decrypt(charstring.getData(), CHARSTRING_KEY, lenIV));
// 			 readPut();
// 		 }
// 		 readDef();
// 	 }

// 	 // OtherSubrs are embedded PostScript procedures which we can safely ignore
//func (p *Parser) void readOtherSubrs() error
// 	 {
// 		 if (lexer.peekToken().kind == token.START_ARRAY)
// 		 {
// 			 readValue();
// 			 readDef();
// 		 }
// 		 else
// 		 {
// 			  length := read(integer).intValue();
// 			 read(name, "array");

// 			 for (int i = 0; i < length; i++)
// 			 {
// 				 read(name, "dup");
// 				 read(integer); // index
// 				 readValue(); // PostScript
// 				 readPut();
// 			 }
// 			 readDef();
// 		 }
// 	 }

// 	 /**
// 	  * Reads the /CharStrings dictionary.
// 	  * @param lenIV The number of random bytes used in charstring encryption.
// 	  */
//func (p *Parser) void readCharStrings(int lenIV) error
// 	 {
// 		  length := read(integer).intValue();
// 		 read(name, "dict");
// 		 // could actually be a sequence ending in "CharStrings begin", too
// 		 // instead of the "dup begin"
// 		 read(name, "dup");
// 		 read(name, "begin");

// 		 for (int i = 0; i < length; i++)
// 		 {
// 			 // premature end
// 			 if (lexer.peekToken() == none)
// 			 {
// 				 break;
// 			 }
// 			 if (lexer.peekToken().kind == name &&
// 				 lexer.peekToken().value == "end"))
// 			 {
// 				 break;
// 			 }
// 			 // key/value
// 			 name := read(literal).value;

// 			 // RD
// 			 read(integer);
// 			 token charstring = read(token.CHARSTRING);
// 			 font.charstrings.put(name, decrypt(charstring.getData(), CHARSTRING_KEY, lenIV));
// 			 readDef();
// 		 }

// 		 // some fonts have one "end", others two
// 		 read(name, "end");
// 		 // since checking ends here, this does not matter ....
// 		 // more thorough checking would see whether there is "begin" before /Private
// 		 // and expect a "def" somewhere, otherwise a "put"
// 	 }

// Reads the sequence "noaccess def" or equivalent.
func (p *parser) readDef() error {
	if _, err := p.readMaybe(name, "readonly"); err != nil {
		return err
	}
	// allows "noaccess ND" (not in the Type 1 spec)
	if _, err := p.readMaybe(name, "noaccess"); err != nil {
		return err
	}

	token, err := p.read(name)
	if err != nil {
		return err
	}
	switch token.value {
	case "ND", "|-":
		return nil
	case "noaccess":
		token, err = p.read(name)
		if err != nil {
			return err
		}
	}
	if token.value == "def" {
		return nil
	}
	return fmt.Errorf("Found %s but expected ND", token.value)
}

// 	 /**
// 	  * Reads the sequence "noaccess put" or equivalent.
// 	  */
//func (p *Parser) void readPut() error
// 	 {
// 		 p.readMaybe(name, "readonly");

// 		 token := read(name);
// 		 switch (token.value)
// 		 {
// 			 case "NP":
// 			 case "|":
// 				 return;
// 			 case "noaccess":
// 				 token = read(name);
// 				 break;
// 			 default:
// 				 break;
// 		 }

// 		 if (token.value == "put"))
// 		 {
// 			 return;
// 		 }
// 		 return errors.New("Found " + token + " but expected NP");
// 	 }

/// Reads the next token and throws an error if it is not of the given kind.
func (p *parser) read(kind kind) (token, error) {
	token, err := p.lexer.nextToken()
	if err != nil {
		return none, err
	}
	if token.kind != kind {
		return none, fmt.Errorf("found token %s (%s) but expected token %s", token.kind, token.value, kind)
	}
	return token, nil
}

// Reads the next token and throws an error if it is not of the given kind
// and does not have the given value.
func (p *parser) readWithName(kind kind, name string) error {
	token, err := p.read(kind)
	if err != nil {
		return err
	}
	if token.value != name {
		return fmt.Errorf("found %s but expected %s", token.value, name)
	}
	return nil
}

// Reads the next token if and only if it is of the given kind and
// has the given value.
func (p *parser) readMaybe(kind kind, name string) (token, error) {
	token := p.lexer.peekToken()
	if token.kind == kind && token.value == name {
		return p.lexer.nextToken()
	}
	return none, nil
}

// 	 /**
// 	  * Type 1 Decryption (eexec, charstring).
// 	  *
// 	  * @param cipherBytes cipher text
// 	  * @param r key
// 	  * @param n number of random bytes (lenIV)
// 	  * @return plain text
// 	  */
//func (p *Parser) byte[] decrypt(byte[] cipherBytes, int r, int n)
// 	 {
// 		 // lenIV of -1 means no encryption (not documented)
// 		 if (n == -1)
// 		 {
// 			 return cipherBytes;
// 		 }
// 		 // empty charstrings and charstrings of insufficient length
// 		 if (len(cipherBytes) == 0 || n > cipherByteslen())
// 		 {
// 			 return new byte[] {};
// 		 }
// 		 // decrypt
// 		 int c1 = 52845;
// 		 int c2 = 22719;
// 		 byte[] plainBytes = new byte[len(cipherBytes) - n];
// 		 for (int i = 0; i < cipherByteslen(); i++)
// 		 {
// 			 int cipher = cipherBytes[i] & 0xFF;
// 			 int plain = cipher ^ r >> 8;
// 			 if (i >= n)
// 			 {
// 				 plainBytes[i - n] = (byte) plain;
// 			 }
// 			 r = (cipher + r) * c1 + c2 & 0xffff;
// 		 }
// 		 return plainBytes;
// 	 }

// 	 // Check whether binary or hex encoded. See Adobe Type 1 Font Format specification
// 	 // 7.2 eexec encryption
//func (p *Parser) boolean isBinary(byte[] bytes)
// 	 {
// 		 if (len(bytes) < 4)
// 		 {
// 			 return true;
// 		 }
// 		 // "At least one of the first 4 ciphertext bytes must not be one of
// 		 // the ASCII hexadecimal character codes (a code for 0-9, A-F, or a-f)."
// 		 for (int i = 0; i < 4; ++i)
// 		 {
// 			 byte by = bytes[i];
// 			 if (by != 0x0a && by != 0x0d && by != 0x20 && by != '\t' &&
// 					 Character.digit((char) by, 16) == -1)
// 			 {
// 				 return true;
// 			 }
// 		 }
// 		 return false;
// 	 }

//func (p *Parser) byte[] hexToBinary(byte[] bytes)
// 	 {
// 		 // calculate needed length
// 		 int len = 0;
// 		 for (byte by : bytes)
// 		 {
// 			 if (Character.digit((char) by, 16) != -1)
// 			 {
// 				 ++len;
// 			 }
// 		 }
// 		 byte[] res = new byte[len / 2];
// 		 int r = 0;
// 		 int prev = -1;
// 		 for (byte by : bytes)
// 		 {
// 			 int digit = Character.digit((char) by, 16);
// 			 if (digit != -1)
// 			 {
// 				 if (prev == -1)
// 				 {
// 					 prev = digit;
// 				 }
// 				 else
// 				 {
// 					 res[r++] = (byte) (prev * 16 + digit);
// 					 prev = -1;
// 				 }
// 			 }
// 		 }
// 		 return res;
// 	 }
//  }
