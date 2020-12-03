package parser

import (
	"errors"
	"fmt"

	cs "github.com/benoitkugler/pdf/contentstream"
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
	// we "normalize" the entry by adding consecutive spaces"
	// not that it would not be correct to do it for strings as well,
	// since (A) (B) is not the same as (AB): in the first case A and B overlap
	for _, arg := range args {
		if s, ok := model.IsString(arg); ok {
			// start a new TextSpaced
			out.Texts = append(out.Texts, cs.TextSpaced{Text: s})
		} else if f, ok := model.IsNumber(arg); ok { // we accept float
			if L := len(out.Texts); L > 0 {
				// if we already have a TextSpaced add the number to it
				out.Texts[L-1].SpaceSubtractedAfter += int(f)
			} else {
				// the array starts by a number
				out.Texts = append(out.Texts, cs.TextSpaced{SpaceSubtractedAfter: int(f)})
			}
		} else {
			return out, fmt.Errorf("invalid type in TJ array: %v %T", arg, arg)
		}
	}
	return out, nil
}

// checkt the validity of the current tokens, with respect to
// the command
// stack does not contain the command
func parseCommand(command string, stack []Object) (cs.Operation, error) {
	switch command {
	// the special case of inline image in handled separatly
	// case "ID":  OpImageData{},
	// case "BI":  OpBeginImage{},
	// case "EI":  OpEndImage{},

	// case "\"":  OpMoveSetShowText{},
	case "'":
		str, err := assertOneString(stack)
		return cs.OpMoveShowText{Text: str}, err
	// case "B":   OpFillStroke{},
	// case "B*":  OpEOFillStroke{},
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
	// case "BX":  OpBeginIgnoreUndef{},
	case "CS":
		name, err := assertOneName(stack)
		return cs.OpSetStrokeColorSpace{ColorSpace: name}, err
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
		// case "EX":  OpEndIgnoreUndef{},
		// case "F":   OpFill{},
		// case "G":   OpSetStrokeGray{},

		// case "J":   OpSetLineCap{},
		// case "K":   OpSetStrokeCMYKColor{},
		// case "M":   OpSetMiterLimit{},
	case "MP":
		name, err := assertOneName(stack)
		return cs.OpMarkPoint{Tag: name}, err
	case "Q":
		err := assertLength(stack, 0)
		return cs.OpRestore{}, err
	case "RG":
		nbs, err := assertNumbers(stack, 3)
		if err != nil {
			return nil, err
		}
		return cs.OpSetStrokeRGBColor{R: nbs[0], G: nbs[1], B: nbs[2]}, nil
	case "S":
		err := assertLength(stack, 0)
		return cs.OpStroke{}, err
	case "SC":
		nbs, err := assertNumbers(stack, -1)
		return cs.OpSetStrokeColor{Color: nbs}, err
	case "SCN":
		out, err := parseSCN(stack)
		return cs.OpSetStrokeColorN(out), err
	// case "T*":  OpTextNextLine{},
	// case "TD":  OpTextMoveSet{},
	case "TJ":
		return parseTextSpaces(stack)
	case "TL":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetTextLeading{L: nbs[0]}, nil
	// case "Tc":  OpSetCharSpacing{},
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
	// case "Tr":  OpSetTextRender{},
	// case "Ts":  OpSetTextRise{},
	// case "Tw":  OpSetWordSpacing{},
	// case "Tz":  OpSetHorizScaling{},
	case "W":
		err := assertLength(stack, 0)
		return cs.OpClip{}, err
	// case "W*":  OpEOClip{},
	// case "b":   OpCloseFillStroke{},
	// case "b*":  OpCloseEOFillStroke{},
	// case "c":   OpCurveTo{},
	// case "cm":  OpConcat{},
	case "cs":
		name, err := assertOneName(stack)
		return cs.OpSetFillColorSpace{ColorSpace: name}, err
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
	// case "d0":  OpSetCharWidth{},
	// case "d1":  OpSetCacheDevice{},
	case "f":
		err := assertLength(stack, 0)
		return cs.OpFill{}, err
	// case "f*":  OpEOFill{},
	case "g":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetFillGray{G: nbs[0]}, err
	case "gs":
		name, err := assertOneName(stack)
		return cs.OpSetExtGState{Dict: name}, err
	// case "h":   OpClosePath{},
	// case "i":   OpSetFlat{},
	// case "j":   OpSetLineJoin{},
	// case "k":   OpSetFillCMYKColor{},
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
	case "ri":
		name, err := assertOneName(stack)
		return cs.OpSetRenderingIntent{Intent: name}, err
	// case "s":   OpCloseStroke{},
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
	// case "v":   OpCurveTo1{},
	case "w":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return cs.OpSetLineWidth{W: nbs[0]}, nil
		// case "y":   OpCurveTo{},
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
