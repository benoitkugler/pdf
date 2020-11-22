package type1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseFont read a .afm file and return the associated font
// Some default values are also set
func ParseFont(source io.Reader) (Font, error) {
	f := defautFontValues
	// deep copy to avoid state sharing
	f.charMetrics = map[string]charMetric{}
	f.charCodeToCharName = map[byte]string{}
	f.kernPairs = map[string][]kernPair{}

	err := f.parse(source)

	f.encodingScheme = strings.TrimSpace(f.encodingScheme)

	// f.createEncoding()

	return f, err
}

// safely try to read one token; returns an error
// if it's not found
func readToken(tokens []string, index int) (string, error) {
	if index >= len(tokens) {
		return "", fmt.Errorf("invalid line %s : expected %d tokens", strings.Join(tokens, " "), index+1)
	}
	return tokens[index], nil
}

func readIntToken(tokens []string, index int) (int, error) {
	s, err := readToken(tokens, index)
	if err != nil {
		return 0, err
	}
	out, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid int in line %s (%s)", strings.Join(tokens, " "), err)
	}
	return out, nil
}

func readFloatToken(tokens []string, index int) (Fl, error) {
	s, err := readToken(tokens, index)
	if err != nil {
		return 0, err
	}
	out, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float in line %s (%s)", strings.Join(tokens, " "), err)
	}
	return Fl(out), nil
}

func (f *Font) parse(source io.Reader) error {
	scanner := bufio.NewScanner(source)
	isMetrics := false
	for scanner.Scan() {
		line := scanner.Text()

		tok := strings.Fields(line)
		if len(tok) == 0 {
			continue
		}
		ident := tok[0]
		var err error
		switch ident {
		case "FontName":
			f.fontName, err = readToken(tok, 1)
		case "FullName":
			f.fullName, err = readToken(tok, 1)
		case "FamilyName":
			f.familyName, err = readToken(tok, 1)
		case "Weight":
			f.weight, err = readToken(tok, 1)
		case "ItalicAngle":
			f.ItalicAngle, err = readFloatToken(tok, 1)
		case "IsFixedPitch":
			var s string
			s, err = readToken(tok, 1)
			f.isFixedPitch = s == "true"
		case "CharacterSet":
			f.characterSet, err = readToken(tok, 1)
		case "FontBBox":
			f.Llx, err = readFloatToken(tok, 1)
			if err != nil {
				break
			}
			f.Lly, err = readFloatToken(tok, 2)
			if err != nil {
				break
			}
			f.Urx, err = readFloatToken(tok, 3)
			if err != nil {
				break
			}
			f.Ury, err = readFloatToken(tok, 4)
		case "UnderlinePosition":
			f.underlinePosition, err = readIntToken(tok, 1)
		case "UnderlineThickness":
			f.underlineThickness, err = readIntToken(tok, 1)
		case "EncodingScheme":
			f.encodingScheme, err = readToken(tok, 1)
		case "CapHeight":
			f.CapHeight, err = readFloatToken(tok, 1)
		case "XHeight":
			f.xHeight, err = readIntToken(tok, 1)
		case "Ascender":
			f.Ascender, err = readFloatToken(tok, 1)
		case "Descender":
			f.Descender, err = readFloatToken(tok, 1)
		case "StdHW":
			f.stdHw, err = readIntToken(tok, 1)
		case "StdVW":
			f.stdVw, err = readIntToken(tok, 1)
		}
		if err != nil {
			return err
		}
		if ident == "StartCharMetrics" {
			isMetrics = true
			break
		}
	}

	if !isMetrics {
		return errors.New("missing StartCharMetrics in font file")
	}

	for scanner.Scan() {
		line := scanner.Text()
		tok := strings.Fields(line)
		if len(tok) == 0 {
			continue
		}
		ident := tok[0]
		if ident == "EndCharMetrics" {
			isMetrics = false
			break
		}

		met := charMetric{width: 250}
		tok = strings.Split(line, ";")
		for len(tok) > 0 {
			tokc := strings.Fields(tok[0])
			tok = tok[1:] // go to next token
			if len(tokc) == 0 {
				continue
			}
			ident := tokc[0]
			var (
				err error
				c   int
			)
			switch ident {
			case "C":
				c, err = readIntToken(tokc, 1)
				if c == -1 { // not encoded
					goto endloop
				}
				if c < 0 || c > 255 {
					panic("byte overflow")
				}
				met.code = byte(c)
			case "WX":
				met.width, err = readIntToken(tokc, 1)
			case "N":
				met.name, err = readToken(tokc, 1)
			case "B":
				met.charBBox[0], err = readIntToken(tokc, 1)
				if err != nil {
					break
				}
				met.charBBox[1], err = readIntToken(tokc, 2)
				if err != nil {
					break
				}
				met.charBBox[2], err = readIntToken(tokc, 3)
				if err != nil {
					break
				}
				met.charBBox[3], err = readIntToken(tokc, 4)
			}
			if err != nil {
				return err
			}
		}
		f.charMetrics[met.name] = met
		f.charCodeToCharName[met.code] = met.name

	endloop:
	}

	if isMetrics {
		return errors.New("missing EndCharMetrics in font file")
	}
	for scanner.Scan() {
		line := scanner.Text()
		tok := strings.Fields(line)
		if len(tok) == 0 {
			continue
		}
		ident := tok[0]
		if ident == "EndFontMetrics" {
			goto end
		}
		if ident == "StartKernPairs" {
			isMetrics = true
			break
		}
	}
	if !isMetrics {
		return errors.New("missing EndFontMetrics in font file")
	}

	for scanner.Scan() {
		line := scanner.Text()
		tok := strings.Fields(line)
		if len(tok) == 0 {
			continue
		}
		ident := tok[0]
		if ident == "KPX" {
			first, err := readToken(tok, 1)
			if err != nil {
				return err
			}
			second, err := readToken(tok, 2)
			if err != nil {
				return err
			}
			width, err := readIntToken(tok, 3)
			if err != nil {
				return err
			}
			f.kernPairs[first] = append(f.kernPairs[first], kernPair{sndChar: second, kerningDistance: width})
		} else if ident == "EndKernPairs" {
			isMetrics = false
			break
		}
	}
	if isMetrics {
		return errors.New("missing EndKernPairs in font file")
	}
end:
	err := scanner.Err()
	return err
}
