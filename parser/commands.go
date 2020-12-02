package parser

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/model"
)

type Fl = contentstream.Fl

func assertLength(stack []Object, L int) error {
	if L != len(stack) {
		return fmt.Errorf("expected %d operands, got %d", L, len(stack))
	}
	return nil
}

func assertOneName(stack []Object) (model.Name, error) {
	if err := assertLength(stack, 1); err != nil {
		return "", err
	}
	name, ok := stack[0].(Name)
	if !ok {
		return "", fmt.Errorf("expected Name, got %v", stack[0])
	}
	return model.Name(name), nil
}

func assertOneString(stack []Object) (string, error) {
	if err := assertLength(stack, 1); err != nil {
		return "", err
	}
	s := stack[0]
	switch s := s.(type) {
	case StringLiteral:
		return string(s), nil
	case HexLiteral:
		return string(s), nil
	default:
		return "", fmt.Errorf("expected string, got %v", s)
	}
}

func assertNumber(t Object) (Fl, error) {
	switch t := t.(type) {
	case Float:
		return Fl(t), nil
	case Integer:
		return Fl(t), nil
	default:
		return 0, fmt.Errorf("expected number, got %v", t)
	}
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

// checkt the validity of the current tokens, with respect to
// the command
// stack does not contain the command
func parseCommand(command string, stack []Object) (contentstream.Operation, error) {
	switch command {
	// case "\"":  OpMoveSetShowText{},
	case "'":
		str, err := assertOneString(stack)
		return contentstream.OpMoveShowText{Text: str}, err
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
		// property is either a name or a dict
		switch p := stack[1].(type) {
		case Dict, Name:
			return contentstream.OpBeginMarkedContent{Tag: name, Properties: p}, nil
		default:
			return nil, fmt.Errorf("expected name or dictionary, got %v", p)
		}
	// case "BI":  OpBeginImage{},
	case "BMC":
		name, err := assertOneName(stack[0:1])
		return contentstream.OpBeginMarkedContent{Tag: name}, err
	case "BT":
		err := assertLength(stack, 0)
		return contentstream.OpBeginText{}, err
	// case "BX":  OpBeginIgnoreUndef{},
	case "CS":
		name, err := assertOneName(stack)
		return contentstream.OpSetStrokeColorSpace{ColorSpace: name}, err
	case "DP":
		if err := assertLength(stack, 2); err != nil {
			return nil, err
		}
		name, err := assertOneName(stack[0:1])
		if err != nil {
			return nil, err
		}
		// property is either a name or a dict
		switch p := stack[1].(type) {
		case Dict, Name:
			return contentstream.OpMarkPoint{Tag: name, Properties: p}, nil
		default:
			return nil, fmt.Errorf("expected name or dictionary, got %v", p)
		}
	case "Do":
		name, err := assertOneName(stack)
		return contentstream.OpXObject{XObject: name}, err
	// case "EI":  OpEndImage{},
	case "EMC":
		err := assertLength(stack, 0)
		return contentstream.OpEndMarkedContent{}, err
	case "ET":
		err := assertLength(stack, 0)
		return contentstream.OpEndText{}, err
		// case "EX":  OpEndIgnoreUndef{},
		// case "F":   OpFill{},
		// case "G":   OpSetStrokeGray{},
		// case "ID":  OpImageData{},
		// case "J":   OpSetLineCap{},
		// case "K":   OpSetStrokeCMYKColor{},
		// case "M":   OpSetMiterLimit{},
	case "MP":
		name, err := assertOneName(stack)
		return contentstream.OpMarkPoint{Tag: name}, err
	case "Q":
		err := assertLength(stack, 0)
		return contentstream.OpRestore{}, err
	case "RG":
		nbs, err := assertNumbers(stack, 3)
		if err != nil {
			return nil, err
		}
		return contentstream.OpSetStrokeRGBColor{R: nbs[0], G: nbs[1], B: nbs[2]}, nil
	case "S":
		err := assertLength(stack, 0)
		return contentstream.OpStroke{}, err
	// case "SC":  OpSetStrokeColor{},
	// case "SCN": OpSetStrokeColorN{},
	// case "T*":  OpTextNextLine{},
	// case "TD":  OpTextMoveSet{},
	// case "TJ":  OpShowSpaceText{},
	case "TL":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return contentstream.OpSetTextLeading{L: nbs[0]}, nil
	// case "Tc":  OpSetCharSpacing{},
	case "Td":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return contentstream.OpTextMove{X: nbs[0], Y: nbs[1]}, nil
	case "Tf":
		if err := assertLength(stack, 2); err != nil {
			return nil, err
		}
		name, err := assertOneName(stack[0:1])
		if err != nil {
			return nil, err
		}
		f, err := assertNumber(stack[1])
		return contentstream.OpSetFont{Font: name, Size: f}, err
	case "Tj":
		str, err := assertOneString(stack)
		return contentstream.OpShowText{Text: str}, err
	case "Tm":
		nbs, err := assertNumbers(stack, 6)
		if err != nil {
			return nil, err
		}
		var mat model.Matrix
		copy(mat[:], nbs)
		return contentstream.OpSetTextMatrix{Matrix: mat}, nil
	// case "Tr":  OpSetTextRender{},
	// case "Ts":  OpSetTextRise{},
	// case "Tw":  OpSetWordSpacing{},
	// case "Tz":  OpSetHorizScaling{},
	case "W":
		err := assertLength(stack, 0)
		return contentstream.OpClip{}, err
	// case "W*":  OpEOClip{},
	// case "b":   OpCloseFillStroke{},
	// case "b*":  OpCloseEOFillStroke{},
	// case "c":   OpCurveTo{},
	// case "cm":  OpConcat{},
	case "cs":
		name, err := assertOneName(stack)
		return contentstream.OpSetFillColorSpace{ColorSpace: name}, err
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
		return contentstream.OpSetDash{Dash: model.DashPattern{Array: dash, Phase: phase}}, err
	// case "d0":  OpSetCharWidth{},
	// case "d1":  OpSetCacheDevice{},
	case "f":
		err := assertLength(stack, 0)
		return contentstream.OpFill{}, err
	// case "f*":  OpEOFill{},
	case "g":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return contentstream.OpSetFillGray{G: nbs[0]}, err
	case "gs":
		name, err := assertOneName(stack)
		return contentstream.OpSetExtGState{Dict: name}, err
	// case "h":   OpClosePath{},
	// case "i":   OpSetFlat{},
	// case "j":   OpSetLineJoin{},
	// case "k":   OpSetFillCMYKColor{},
	case "l":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return contentstream.OpLineTo{X: nbs[0], Y: nbs[1]}, err
	case "m":
		nbs, err := assertNumbers(stack, 2)
		if err != nil {
			return nil, err
		}
		return contentstream.OpMoveTo{X: nbs[0], Y: nbs[1]}, err
	case "n":
		err := assertLength(stack, 0)
		return contentstream.OpEndPath{}, err
	case "q":
		err := assertLength(stack, 0)
		return contentstream.OpSave{}, err
	case "re":
		nbs, err := assertNumbers(stack, 4)
		if err != nil {
			return nil, err
		}
		return contentstream.OpRectangle{X: nbs[0], Y: nbs[1], W: nbs[2], H: nbs[3]}, err
	case "rg":
		nbs, err := assertNumbers(stack, 3)
		if err != nil {
			return nil, err
		}
		return contentstream.OpSetFillRGBColor{R: nbs[0], G: nbs[1], B: nbs[2]}, nil
	case "ri":
		name, err := assertOneName(stack)
		return contentstream.OpSetRenderingIntent{Intent: name}, err
	// case "s":   OpCloseStroke{},
	case "sc":
		nbs, err := assertNumbers(stack, -1)
		return contentstream.OpSetFillColor{Color: nbs}, err
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
		return contentstream.OpSetFillColorN{Color: nbs, Pattern: model.Name(name)}, nil
	case "sh":
		name, err := assertOneName(stack)
		return contentstream.OpShFill{Shading: name}, err
	// case "v":   OpCurveTo1{},
	case "w":
		nbs, err := assertNumbers(stack, 1)
		if err != nil {
			return nil, err
		}
		return contentstream.OpSetLineWidth{W: nbs[0]}, nil
		// case "y":   OpCurveTo{},
	default:
		return nil, fmt.Errorf("invalid command name %s", command)
	}
}
