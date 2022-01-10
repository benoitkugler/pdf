package parser

import (
	"errors"
	"fmt"

	cs "github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
)

type Fl = model.Fl

func assertLength(stack []Object, L int) error {
	if L != len(stack) {
		return fmt.Errorf("expected %d operands, got %d", L, len(stack))
	}
	return nil
}

func assertOneName(stack []Object) (model.ObjName, error) {
	if err := assertLength(stack, 1); err != nil {
		return "", err
	}
	name, ok := stack[0].(Name)
	if !ok {
		return "", fmt.Errorf("expected Name, got %v", stack[0])
	}
	return model.ObjName(name), nil
}

func assertOneString(stack []Object) (string, error) {
	if err := assertLength(stack, 1); err != nil {
		return "", err
	}
	st := stack[0]
	s, ok := model.IsString(st)
	if !ok {
		return "", fmt.Errorf("expected string, got %v", st)
	}
	return s, nil
}

func assertNumber(t Object) (Fl, error) {
	f, ok := model.IsNumber(t)
	if !ok {
		return 0, fmt.Errorf("expected number, got %v", t)
	}
	return f, nil
}

// accepts int and numbers
// pass -1 not to check the length
func assertNumbers(stack []Object, L int) ([]Fl, error) {
	if err := assertLength(stack, L); L >= 0 && err != nil {
		return nil, err
	}
	if len(stack) == 0 { // preserve nil-ness, useful in test with reflect.DeepEqual
		return nil, nil
	}
	out := make([]Fl, len(stack))
	var err error
	for i, t := range stack {
		out[i], err = assertNumber(t)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// shared with scn
func parseSCN(stack []Object) (cs.OpSetFillColorN, error) {
	// optional last name argument
	if len(stack) == 0 {
		return cs.OpSetFillColorN{}, errors.New("missing operands for scn/SCN")
	}
	name, ok := stack[len(stack)-1].(Name)
	if ok {
		stack = stack[0 : len(stack)-1] // remove the name
	}
	nbs, err := assertNumbers(stack, -1)
	if err != nil {
		return cs.OpSetFillColorN{}, err
	}
	return cs.OpSetFillColorN{Color: nbs, Pattern: model.ObjName(name)}, nil
}

// property is either a name or a dict
func parsePropertyList(p Object) (cs.PropertyList, error) {
	switch p := p.(type) {
	case Name:
		return cs.PropertyListName(p), nil
	case Dict:
		if err := checkPropertyValue(p); err != nil {
			return nil, err
		}
		return cs.PropertyListDict(p), nil
	default:
		return nil, fmt.Errorf("expected name or dictionary, got %v", p)
	}
}

func parseTextSpaces(stack []Object) (cs.OpShowSpaceText, error) {
	var out cs.OpShowSpaceText
	if err := assertLength(stack, 1); err != nil {
		return out, err
	}
	args, ok := stack[0].(Array)
	if !ok {
		return out, fmt.Errorf("expected Array in TJ command, got %v", args)
	}
	// we "normalize" the entry by adding consecutive spaces
	var (
		current fonts.TextSpaced
		last    uint8 // 0 at the start, 1 for text, 2 for number
	)
	for _, arg := range args {
		if s, ok := model.IsString(arg); ok {
			if last == 2 {
				// add the last and start a new TextSpaced
				out.Texts = append(out.Texts, current)
				current = fonts.TextSpaced{CharCodes: []byte(s)}
			} else {
				// we concatenat the strings
				current.CharCodes = append(current.CharCodes, s...)
			}
			last = 1
		} else if f, ok := model.IsNumber(arg); ok { // we accept float
			current.SpaceSubtractedAfter += int(f)
			last = 2
		} else {
			return out, fmt.Errorf("invalid type in TJ array: %v %T", arg, arg)
		}
	}
	if current.CharCodes != nil || current.SpaceSubtractedAfter != 0 {
		out.Texts = append(out.Texts, current)
	}
	return out, nil
}

// checkt the validity of the current tokens, with respect to
// the command
// stack does not contain the command
func parseCommand(command string, stack []Object) (cs.Operation, error) {
	switch command {
	// the special case of inline image in handled separatly
	// case "ID":  cs.OpImageData{},
	// case "BI":  cs.OpBeginImage{},
	// case "EI":  cs.OpEndImage{},

	case "\"":
		if err := assertLength(stack, 3); err != nil {
			return nil, err
		}
		fls, err := assertNumbers(stack[:2], 2)
		if err != nil {
			return nil, err
		}
		str, err := assertOneString(stack[2:])
		if err != nil {
			return nil, err
		}
		return cs.OpMoveSetShowText{WordSpacing: fls[0], CharacterSpacing: fls[1], Text: str}, nil
	case "'":
		str, err := assertOneString(stack)
		return cs.OpMoveShowText{Text: str}, err
	case "B":
		err := assertLength(stack, 0)
		return cs.OpFillStroke{}, err
	case "B*":
		err := assertLength(stack, 0)
		return cs.OpEOFillStroke{}, err
	case "BDC":
		if err := assertLength(stack, 2); err != nil {
			return nil, err
		}
		name, err := assertOneName(stack[0:1])
		if err != nil {
			return nil, err
		}
		props, err := parsePropertyList(stack[1])
		return cs.OpBeginMarkedContent{Tag: name, Properties: props}, err
	case "BMC":
		name, err := assertOneName(stack[0:1])
		return cs.OpBeginMarkedContent{Tag: name}, err
	case "BT":
		err := assertLength(stack, 0)
		return cs.OpBeginText{}, err
	case "BX":
		err := assertLength(stack, 0)
		return cs.OpBeginIgnoreUndef{}, err
	case "CS":
		name, err := assertOneName(stack)
		return cs.OpSetStrokeColorSpace{ColorSpace: model.ColorSpaceName(name)}, err
	case "DP":
		if err := assertLength(stack, 2); err != nil {
			return nil, err
		}
		name, err := assertOneName(stack[0:1])
		if err != nil {
			return nil, err
		}
		props, err := parsePropertyList(stack[1])
		return cs.OpMarkPoint{Tag: name, Properties: props}, err
	case "Do":
		name, err := assertOneName(stack)
		return cs.OpXObject{XObject: name}, err
	case "EMC":
		err := assertLength(stack, 0)
		return cs.OpEndMarkedContent{}, err
	case "ET":
		err := assertLength(stack, 0)
		return cs.OpEndText{}, err
	case "EX":
		err := assertLength(stack, 0)
		return cs.OpEndIgnoreUndef{}, err
	case "F":
		err := assertLength(stack, 0)
		return cs.OpFill{}, err
	case "G":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetStrokeGray{G: nbs[0]}, err
	case "g":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetFillGray{G: nbs[0]}, err
	case "J":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		s := uint8(nbs[0])
		return cs.OpSetLineCap{Style: s}, nil
	case "M":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetMiterLimit{Limit: nbs[0]}, nil
	case "MP":
		name, err := assertOneName(stack)
		return cs.OpMarkPoint{Tag: name}, err
	case "Q":
		err := assertLength(stack, 0)
		return cs.OpRestore{}, err
	case "S":
		err := assertLength(stack, 0)
		return cs.OpStroke{}, err
	case "SC":
		nbs, err := assertNumbers(stack, -1)
		return cs.OpSetStrokeColor{Color: nbs}, err
	case "SCN":
		out, err := parseSCN(stack)
		return cs.OpSetStrokeColorN(out), err
	case "T*":
		err := assertLength(stack, 0)
		return cs.OpTextNextLine{}, err
	case "TD":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return cs.OpTextMoveSet{X: nbs[0], Y: nbs[1]}, nil
	case "TJ":
		return parseTextSpaces(stack)
	case "TL":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetTextLeading{L: nbs[0]}, nil
	case "Tc":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetCharSpacing{CharSpace: nbs[0]}, nil
	case "Td":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return cs.OpTextMove{X: nbs[0], Y: nbs[1]}, nil
	case "Tf":
		if err := assertLength(stack, 2); err != nil {
			return nil, err
		}
		name, err := assertOneName(stack[0:1])
		if err != nil {
			return nil, err
		}
		f, err := assertNumber(stack[1])
		return cs.OpSetFont{Font: name, Size: f}, err
	case "Tj":
		str, err := assertOneString(stack)
		return cs.OpShowText{Text: str}, err
	case "Tm":
		nbs, err := assertNumbers(stack, 6)
		if err != nil {
			return nil, err
		}
		var mat model.Matrix
		copy(mat[:], nbs)
		return cs.OpSetTextMatrix{Matrix: mat}, nil
	case "Tr":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetTextRender{Render: nbs[0]}, nil
	case "Ts":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetTextRise{Rise: nbs[0]}, nil
	case "Tw":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetWordSpacing{WordSpace: nbs[0]}, nil
	case "Tz":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetHorizScaling{Scale: nbs[0]}, nil
	case "W":
		err := assertLength(stack, 0)
		return cs.OpClip{}, err
	case "W*":
		err := assertLength(stack, 0)
		return cs.OpEOClip{}, err
	case "b":
		err := assertLength(stack, 0)
		return cs.OpCloseFillStroke{}, err
	case "b*":
		err := assertLength(stack, 0)
		return cs.OpCloseEOFillStroke{}, err
	case "c":
		fls, err := assertNumbers(stack, 6)
		if err != nil {
			return nil, err
		}
		return cs.OpCubicTo{X1: fls[0], Y1: fls[1], X2: fls[2], Y2: fls[3], X3: fls[4], Y3: fls[5]}, nil
	case "cm":
		nbs, err := assertNumbers(stack, 6)
		if err != nil {
			return nil, err
		}
		var mat model.Matrix
		copy(mat[:], nbs)
		return cs.OpConcat{Matrix: mat}, nil
	case "cs":
		name, err := assertOneName(stack)
		return cs.OpSetFillColorSpace{ColorSpace: model.ColorSpaceName(name)}, err
	case "d":
		if err := assertLength(stack, 2); err != nil {
			return nil, err
		}
		arr, ok := stack[0].(Array)
		if !ok {
			return nil, fmt.Errorf("expected array, got %v", stack[0])
		}
		dash, err := assertNumbers(arr, -1)
		if err != nil {
			return nil, err
		}
		phase, err := assertNumber(stack[1])
		return cs.OpSetDash{Dash: model.DashPattern{Array: dash, Phase: phase}}, err
	case "d0":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return cs.OpSetCharWidth{WX: int(nbs[0]), WY: int(nbs[1])}, nil
	case "d1":
		nbs, err := assertNumbers(stack, 6)
		if err != nil {
			return nil, err
		}
		return cs.OpSetCacheDevice{
			WX: int(nbs[0]), WY: int(nbs[1]),
			LLX: int(nbs[2]), LLY: int(nbs[3]), URX: int(nbs[4]), URY: int(nbs[5]),
		}, nil
	case "f":
		err := assertLength(stack, 0)
		return cs.OpFill{}, err
	case "f*":
		err := assertLength(stack, 0)
		return cs.OpEOFill{}, err
	case "gs":
		name, err := assertOneName(stack)
		return cs.OpSetExtGState{Dict: name}, err
	case "h":
		err := assertLength(stack, 0)
		return cs.OpClosePath{}, err
	case "i":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetFlat{Flatness: nbs[0]}, nil
	case "j":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		s := uint8(nbs[0])
		return cs.OpSetLineJoin{Style: s}, nil
	case "k":
		nbs, err := assertNumbers(stack, 4)
		if err != nil {
			return nil, err
		}
		return cs.OpSetFillCMYKColor{C: nbs[0], M: nbs[1], Y: nbs[2], K: nbs[3]}, nil
	case "K":
		nbs, err := assertNumbers(stack, 4)
		if err != nil {
			return nil, err
		}
		return cs.OpSetStrokeCMYKColor{C: nbs[0], M: nbs[1], Y: nbs[2], K: nbs[3]}, nil

	case "l":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return cs.OpLineTo{X: nbs[0], Y: nbs[1]}, err
	case "m":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return cs.OpMoveTo{X: nbs[0], Y: nbs[1]}, err
	case "n":
		err := assertLength(stack, 0)
		return cs.OpEndPath{}, err
	case "q":
		err := assertLength(stack, 0)
		return cs.OpSave{}, err
	case "re":
		nbs, err := assertNumbers(stack, 4)
		if err != nil {
			return nil, err
		}
		return cs.OpRectangle{X: nbs[0], Y: nbs[1], W: nbs[2], H: nbs[3]}, err
	case "rg":
		nbs, err := assertNumbers(stack, 3)
		if err != nil {
			return nil, err
		}
		return cs.OpSetFillRGBColor{R: nbs[0], G: nbs[1], B: nbs[2]}, nil
	case "RG":
		nbs, err := assertNumbers(stack, 3)
		if err != nil {
			return nil, err
		}
		return cs.OpSetStrokeRGBColor{R: nbs[0], G: nbs[1], B: nbs[2]}, nil
	case "ri":
		name, err := assertOneName(stack)
		return cs.OpSetRenderingIntent{Intent: name}, err
	case "s":
		err := assertLength(stack, 0)
		return cs.OpCloseStroke{}, err
	case "sc":
		nbs, err := assertNumbers(stack, -1)
		return cs.OpSetFillColor{Color: nbs}, err
	case "scn":
		// optional last name argument
		if len(stack) == 0 {
			return nil, errors.New("missing operands for scn")
		}
		name, ok := stack[len(stack)-1].(Name)
		if ok {
			stack = stack[0 : len(stack)-1] // remove the name
		}
		nbs, err := assertNumbers(stack, -1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetFillColorN{Color: nbs, Pattern: model.ObjName(name)}, nil
	case "sh":
		name, err := assertOneName(stack)
		return cs.OpShFill{Shading: name}, err
	case "v":
		nbs, err := assertNumbers(stack, 4)
		if err != nil {
			return nil, err
		}
		return cs.OpCurveTo1{X2: nbs[0], Y2: nbs[1], X3: nbs[2], Y3: nbs[3]}, nil
	case "w":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetLineWidth{W: nbs[0]}, nil
	case "y":
		nbs, err := assertNumbers(stack, 4)
		if err != nil {
			return nil, err
		}
		return cs.OpCurveTo{X1: nbs[0], Y1: nbs[1], X3: nbs[2], Y3: nbs[3]}, nil
	default:
		return nil, fmt.Errorf("invalid command name %s", command)
	}
}

// recursively check for invalid content like refs and streams
func checkPropertyValue(v Object) error {
	switch v := v.(type) {
	case nil, Command, IndirectRef, model.ObjStream:
		return fmt.Errorf("invalid property value %v (type %T not allowed)", v, v)
	case Array:
		for _, av := range v {
			if err := checkPropertyValue(av); err != nil {
				return err
			}
		}
	case Dict:
		for _, av := range v {
			if err := checkPropertyValue(av); err != nil {
				return err
			}
		}
	}
	return nil
}
