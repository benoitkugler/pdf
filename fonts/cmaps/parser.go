package cmaps

// the main parsing logic is taken
// from https://git.maze.io/go/unipdf/src/branch/master/internal/cmap

import (
	"errors"
	"fmt"
	"io"

	"github.com/benoitkugler/pdf/model"
	tokenizer "github.com/benoitkugler/pstokenizer"
)

// parser parses CMap files, which represents either a character code to unicode mapping or
// a character code to CID mapping, both used in PDF files
// References:
//  https://www.adobe.com/content/dam/acom/en/devnet/acrobat/pdfs/5411.ToUnicode.pdf
//  https://github.com/adobe-type-tools/cmap-resources/releases
type parser struct {
	version string

	// a cmap may contain either CIDs or Unicodes
	unicode UnicodeCMap
	cids    CMap

	tokenizer tokenizer.Tokenizer
}

// parser creates a new instance of the PDF CMap parser from input data.
func newparser(content []byte) *parser {
	ps := parser{}
	ps.tokenizer = *tokenizer.NewTokenizer(content)
	return &ps
}

// parse parses the CMap file and loads into the CMap structure.
func (cmap *parser) parse() error {
	var prev cmapObject
	for {
		o, err := cmap.parseObject()
		if err != nil {
			return err
		}
		switch t := o.(type) {
		case nil: // means EOF
			return nil
		case cmapOperand:
			switch t {
			case "begincodespacerange":
				err := cmap.parseCodespaceRange()
				if err != nil {
					return err
				}
			case "begincidrange":
				err := cmap.parseCIDRange()
				if err != nil {
					return err
				}
			case "beginbfchar":
				err := cmap.parseBfchar()
				if err != nil {
					return err
				}
			case "beginbfrange":
				err := cmap.parseBfrange()
				if err != nil {
					return err
				}
			case "usecmap":
				if prev == nil {
					return ErrBadCMap
				}
				name, ok := prev.(model.ObjName)
				if !ok {
					return ErrBadCMap
				}
				cmap.cids.UseCMap = name
				cmap.unicode.UseCMap = name
			case "CIDSystemInfo":
				// Some PDF generators leave the "/"" off CIDSystemInfo
				// e.g. ~/testdata/459474_809.pdf
				err := cmap.parseSystemInfo()
				if err != nil {
					return err
				}
			}
		case model.ObjName:
			switch t {
			case "CIDSystemInfo":
				err := cmap.parseSystemInfo()
				if err != nil {
					return err
				}
			case "CMapName":
				err := cmap.addName()
				if err != nil {
					return err
				}
			case "CMapType":
				err := cmap.parseType()
				if err != nil {
					return err
				}
			case "CMapVersion":
				err := cmap.parseVersion()
				if err != nil {
					return err
				}
			}
		}
		prev = o
	}
}

// parseName parses a cmap name and adds it to `cmap`.
// cmap names are defined like this:/CMapName/83pv-RKSJ-H def
func (cmap *parser) addName() error {
	var name model.ObjName
	done := false
	for i := 0; i < 10 && !done; i++ {
		o, err := cmap.parseObject()
		if err != nil {
			return err
		}
		switch t := o.(type) {
		case cmapOperand:
			switch t {
			case "def":
				done = true
			default:
				// This is not an error because some PDF files don't have valid PostScript names.
				// e.g. ~/testdata/Papercut vs Equitrac.pdf
				///CMapName/Adobe-SI-*Courier New-6164-0 def
				// We just append the non-existant operator "New-6164-0" to the name
				if name != "" {
					name = model.ObjName(fmt.Sprintf("%s %s", name, t))
				}
			}
		case model.ObjName:
			name = t
		}
	}
	if !done {
		return ErrBadCMap
	}
	cmap.cids.Name = name
	return nil
}

// parseType parses a cmap type and adds it to `cmap`.
// cmap names are defined like this:/CMapType 1 def
func (cmap *parser) parseType() error {
	ctype := 0
	done := false
	for i := 0; i < 3 && !done; i++ {
		o, err := cmap.parseObject()
		if err != nil {
			return err
		}
		switch t := o.(type) {
		case cmapOperand:
			switch t {
			case "def":
				done = true
			default:
				return ErrBadCMap
			}
		case int:
			ctype = t
		}
	}
	cmap.cids.Type = ctype
	return nil
}

// parseVersion parses a cmap version and adds it to `cmap`.
// cmap names are defined like this:/CMapType 1 def
// We don't need the version. We do this to eat up the version code in the cmap definition
// to reduce unhandled parse object warnings.
func (cmap *parser) parseVersion() error {
	version := ""
	done := false
	for i := 0; i < 3 && !done; i++ {
		o, err := cmap.parseObject()
		if err != nil {
			return err
		}
		switch t := o.(type) {
		case cmapOperand:
			switch t {
			case "def":
				done = true
			default:
				return ErrBadCMap
			}
		case int:
			version = fmt.Sprintf("%d", t)
		case float64:
			version = fmt.Sprintf("%f", t)
		case string:
			version = t
		}
	}
	cmap.version = version
	return nil
}

// parseSystemInfo parses a cmap CIDSystemInfo and adds it to `cmap`.
// cmap CIDSystemInfo is define like this:
///CIDSystemInfo 3 dict dup begin
//  /Registry (Adobe) def
//  /Ordering (Japan1) def
//  /Supplement 1 def
// end def
func (cmap *parser) parseSystemInfo() error {
	inDict := false
	inDef := false
	var name model.ObjName
	done := false
	systemInfo := model.CIDSystemInfo{}

	// 50 is a generous but arbitrary limit to prevent an endless loop on badly formed cmap files.
	for i := 0; i < 50 && !done; i++ {
		o, err := cmap.parseObject()
		if err != nil {
			return err
		}
		switch t := o.(type) {
		case cmapDict:
			r, ok := t["Registry"].(string)
			if !ok {
				return fmt.Errorf("unexpected type for Registry: %T", t["Registry"])
			}
			systemInfo.Registry = r

			r, ok = t["Ordering"].(string)
			if !ok {
				return fmt.Errorf("unexpected type for Ordering: %T", t["Ordering"])
			}
			systemInfo.Ordering = r

			s, ok := t["Supplement"].(int)
			if !ok {
				return fmt.Errorf("unexpected type for Supplement: %T", t["Supplement"])
			}
			systemInfo.Supplement = s

			done = true
		case cmapOperand:
			switch t {
			case "begin":
				inDict = true
			case "end":
				done = true
			case "def":
				inDef = false
			}
		case model.ObjName:
			if inDict {
				name = t
				inDef = true
			}
		case string:
			if inDef {
				switch name {
				case "Registry":
					systemInfo.Registry = t
				case "Ordering":
					systemInfo.Ordering = t
				}
			}
		case int:
			if inDef {
				switch name {
				case "Supplement":
					systemInfo.Supplement = t
				}
			}
		}
	}
	if !done {
		return ErrBadCMap
	}

	cmap.cids.CIDSystemInfo = systemInfo
	return nil
}

// parseCodespaceRange parses the codespace range section of a CMap.
func (cmap *parser) parseCodespaceRange() error {
	for {
		o, err := cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		hexLow, ok := o.(cmapHexString)
		if !ok {
			if op, isOperand := o.(cmapOperand); isOperand {
				if op == "endcodespacerange" {
					return nil
				}
				return errors.New("unexpected operand")
			}
		}

		o, err = cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		hexHigh, ok := o.(cmapHexString)
		if !ok {
			return errors.New("non-hex high")
		}

		cspace, err := newCodespaceFromBytes(hexLow, hexHigh)
		if err != nil {
			return err
		}
		cmap.cids.Codespaces = append(cmap.cids.Codespaces, cspace)
	}

	if len(cmap.cids.Codespaces) == 0 {
		return ErrBadCMap
	}

	return nil
}

// parseCIDRange parses the CID range section of a CMap.
func (cmap *parser) parseCIDRange() error {
	for {
		// Parse character code interval start.
		o, err := cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		hexStart, ok := o.(cmapHexString)
		if !ok {
			if op, isOperand := o.(cmapOperand); isOperand {
				if op == "endcidrange" {
					return nil
				}
				return errors.New("cid interval start must be a hex string")
			}
		}

		// Parse character code interval end.
		o, err = cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		hexEnd, ok := o.(cmapHexString)
		if !ok {
			return errors.New("cid interval end must be a hex string")
		}

		// Parse interval start CID.
		o, err = cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		cidStart, ok := o.(int)
		if !ok {
			return errors.New("cid start value must be an decimal number")
		}
		if cidStart < 0 {
			return errors.New("invalid cid start value")
		}

		codespace, err := newCodespaceFromBytes(hexStart, hexEnd)
		if err != nil {
			return err
		}
		if cidStart >= (1 << 16) {
			return fmt.Errorf("%d overflow CID range", cidStart)
		}
		cidRange := CIDRange{Codespace: codespace, CIDStart: model.CID(cidStart)}
		cmap.cids.CIDs = append(cmap.cids.CIDs, cidRange)
	}

	return nil
}

// parseBfchar parses a bfchar section of a CMap file.
func (cmap *parser) parseBfchar() error {
	for {
		// Src code.
		o, err := cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		var code model.CID

		switch v := o.(type) {
		case cmapOperand:
			if v == "endbfchar" {
				return nil
			}
			return errors.New("unexpected operand")
		case cmapHexString:
			code, err = hexToCID(v)
			if err != nil {
				return err
			}
		default:
			return errors.New("unexpected type")
		}

		// Target code.
		o, err = cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		var target rune
		switch v := o.(type) {
		case cmapOperand:
			if v == "endbfchar" {
				return nil
			}
			return ErrBadCMap
		case cmapHexString:
			target = hexToRune(v)
		case model.ObjName:
			target = MissingCodeRune
		default:
			return ErrBadCMap
		}

		cmap.unicode.Mappings = append(cmap.unicode.Mappings, ToUnicodePair{
			From: code, Dest: target,
		})
	}

	return nil
}

// parseBfrange parses a bfrange section of a CMap file.
func (cmap *parser) parseBfrange() error {
	for {
		// The specifications are in triplets.
		// <srcCodeFrom> <srcCodeTo> <target>
		// where target can be either <destFrom> as a hex code, or a list.

		// Src code from.
		var srcCodeFrom model.CID
		o, err := cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch v := o.(type) {
		case cmapOperand:
			if v == "endbfrange" {
				return nil
			}
			return errors.New("unexpected operand")
		case cmapHexString:
			srcCodeFrom, err = hexToCID(v)
			if err != nil {
				return err
			}
		default:
			return errors.New("unexpected type")
		}

		// Src code to.
		var srcCodeTo model.CID
		o, err = cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch v := o.(type) {
		case cmapOperand:
			return ErrBadCMap
		case cmapHexString:
			srcCodeTo, err = hexToCID(v)
			if err != nil {
				return err
			}
		default:
			return ErrBadCMap
		}

		// target(s).
		o, err = cmap.parseObject()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch v := o.(type) {
		case cmapArray:
			if len(v) != int(srcCodeTo-srcCodeFrom)+1 {
				return ErrBadCMap
			}
			arr := ToUnicodeArray{From: srcCodeFrom, To: srcCodeTo, Runes: make([]rune, len(v))}
			for code := srcCodeFrom; code <= srcCodeTo; code++ {
				o := v[code-srcCodeFrom]
				hexs, ok := o.(cmapHexString)
				if !ok {
					return errors.New("non-hex string in array")
				}
				// we only support one-rune string
				r := hexToRune(hexs)
				arr.Runes[code-srcCodeFrom] = r
			}
			cmap.unicode.Mappings = append(cmap.unicode.Mappings, arr)
		case cmapHexString:
			// <codeFrom> <codeTo> <dst>, maps [from,to] to [dst,dst+to-from].
			// we only support one-rune string
			r := hexToRune(v)
			tr := ToUnicodeTranslation{From: srcCodeFrom, To: srcCodeTo, Dest: r}
			cmap.unicode.Mappings = append(cmap.unicode.Mappings, tr)
		default:
			return ErrBadCMap
		}
	}

	return nil
}
