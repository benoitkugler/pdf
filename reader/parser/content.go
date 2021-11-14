package parser

import (
	"fmt"

	cs "github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/model"
)

// ParseContentElement parse one operation and avances.
// `ContentStreamMode` must have been set to true, and EOF
// should be checked before calling with method.
// See `ParseContent` for a convenient way of parsing a whole content stream.
func (pr *Parser) ParseContentElement(res model.ResourcesColorSpace) (cs.Operation, error) {
	for {
		if pr.tokens.IsEOF() {
			return nil, fmt.Errorf("unexpected end of content stream")
		}

		obj, err := pr.ParseObject()
		if err != nil {
			return nil, err
		}
		switch obj := obj.(type) {
		case Command:
			var cmd cs.Operation
			// special case
			if obj == "BI" {
				cmd, err = pr.parseInlineImage(res)
				if err != nil {
					return nil, err
				}
			} else {
				// use the current stack to try and parse
				// the command arguments
				cmd, err = parseCommand(string(obj), pr.opsStack)
				if err != nil {
					return nil, fmt.Errorf("invalid command %s with args %v: %s", obj, pr.opsStack, err)
				}
			}
			pr.opsStack = pr.opsStack[:0] // keep the capacity
			return cmd, nil
		default:
			// store the object
			pr.opsStack = append(pr.opsStack, obj)
		}
	}
}

// ParseContent parse a decrypted Content Stream.
// A resource dictionary is needed to handle inline image data,
// which can refer to a color space.
func ParseContent(content []byte, res model.ResourcesColorSpace) ([]cs.Operation, error) {
	var out []cs.Operation

	pr := NewParser(content)
	pr.ContentStreamMode = true
	pr.opsStack = make([]Object, 0, 6)

	for !pr.tokens.IsEOF() {
		cmd, err := pr.ParseContentElement(res)
		if err != nil {
			return nil, err
		}
		out = append(out, cmd)
	}
	return out, nil
}

// ParseContentResources return the resources needed by content.
// Note that only the names in the returned dicts are valid, all the values will be nil.
func ParseContentResources(content []byte, res model.ResourcesColorSpace) (model.ResourcesDict, error) {
	pr := NewParser(content)
	pr.ContentStreamMode = true
	pr.opsStack = make([]Object, 0, 6)

	out := model.NewResourcesDict()

	for !pr.tokens.IsEOF() {
		cmd, err := pr.ParseContentElement(res)
		if err != nil {
			return out, err
		}
		switch cmd := cmd.(type) {
		case cs.OpSetFillColorSpace:
			switch cmd.ColorSpace {
			case "DeviceGray", "DeviceRGB", "DeviceCMYK", "Pattern": // ignored
			default:
				out.ColorSpace[cmd.ColorSpace] = nil
			}
		case cs.OpSetStrokeColorSpace:
			switch cmd.ColorSpace {
			case "DeviceGray", "DeviceRGB", "DeviceCMYK", "Pattern": // ignored
			default:
				out.ColorSpace[cmd.ColorSpace] = nil
			}
		case cs.OpSetExtGState:
			out.ExtGState[cmd.Dict] = nil
		case cs.OpXObject:
			out.XObject[cmd.XObject] = nil
		case cs.OpShFill:
			out.Shading[cmd.Shading] = nil
		case cs.OpSetFillColorN:
			if cmd.Pattern != "" {
				out.Pattern[cmd.Pattern] = nil
			}
		case cs.OpSetStrokeColorN:
			if cmd.Pattern != "" {
				out.Pattern[cmd.Pattern] = nil
			}
		case cs.OpSetFont:
			out.Font[cmd.Font] = nil
		case cs.OpBeginMarkedContent:
			if pn, ok := cmd.Properties.(cs.PropertyListName); ok {
				out.Properties[model.ObjName(pn)] = model.PropertyList{}
			}
		case cs.OpMarkPoint:
			if pn, ok := cmd.Properties.(cs.PropertyListName); ok {
				out.Properties[model.ObjName(pn)] = model.PropertyList{}
			}
		case cs.OpBeginImage:
			var csName model.ColorSpaceName
			switch c := cmd.ColorSpace.(type) {
			case cs.ImageColorSpaceIndexed:
				csName = c.Base
			case cs.ImageColorSpaceName:
				csName = c.ColorSpaceName
			}
			switch csName {
			case "", model.ColorSpaceRGB, model.ColorSpaceCMYK, model.ColorSpaceGray:
				// ignored
			default:
				out.ColorSpace[model.ObjName(csName)] = nil
			}
		}
	}
	return out, nil
}
