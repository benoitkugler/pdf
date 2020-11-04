package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// TODO:
func (r resolver) resolveResources(o pdfcpu.Object) (*model.ResourcesDict, error) {
	if ref, isRef := o.(pdfcpu.IndirectRef); isRef {
		if res := r.resources[ref]; isRef && res != nil {
			return res, nil
		}
		o = r.resolveRef(ref)
	}
	if o == nil {
		return nil, nil
	}
	// Fonts
	resDict, isDict := o.(pdfcpu.Dict)
	if !isDict {
		return nil, fmt.Errorf("unexpected type for Resources Dict: %T", o)
	}
	ft := resDict["Font"]
	if ftRef, isRef := ft.(pdfcpu.IndirectRef); isRef {
		ft = r.resolveRef(ftRef)
	}
	if ft != nil {
		ftDict, isDict := ft.(pdfcpu.Dict)
		if !isDict {
			return nil, fmt.Errorf("unexpected type for Resources.Font Dict: %T", ft)
		}
		ftMap := make(map[model.Name]*model.Font)
		for name, font := range ftDict {
			fontRef, isFontRef := font.(pdfcpu.IndirectRef)
			if isFontRef {
				if fontModel := r.fonts[fontRef]; isFontRef && fontModel != nil {
					ftMap[model.Name(name)] = fontModel
					continue
				}
				font = r.resolveRef(fontRef)
			}
			if font == nil { // ignore the name
				continue
			}
			fontDict, isDict := font.(pdfcpu.Dict)
			if !isDict {
				return nil, fmt.Errorf("unexpected type for Font: %T", font)
			}
			fontModel := parseFontDict(fontDict)
			if fontModel == nil {
				return nil, fmt.Errorf("invalid Font dictionnary: %s", fontDict)
			}
			fmt.Printf("%v", fontModel)
		}
	}

	return &model.ResourcesDict{ExtGState: map[model.Name]*model.GraphicState{"TODO": nil}}, nil
}

func parseFontDict(font pdfcpu.Dict) model.FontType {
	switch font["Subtype"] {
	case pdfcpu.Name("Type0"):
		return model.Type0{}
	case pdfcpu.Name("Type1"):
		out := model.Type1{}
		name, _ := font["BaseFont"].(pdfcpu.Name)
		out.BaseFont = model.Name(name)
		return out
	case pdfcpu.Name("TrueType"):
		return model.TrueType{}
	case pdfcpu.Name("Type3"):
		return model.Type3{}
	default:
		return nil
	}
}
