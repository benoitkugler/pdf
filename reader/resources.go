package reader

import (
	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// TODO:
func (r resolver) resolveResources(o pdfcpu.Object) (*model.ResourcesDict, error) {
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		if res := r.resources[ref]; isRef && res != nil {
			return res, nil
		}
		o = r.resolve(ref)
	}
	if o == nil {
		return nil, nil
	}
	resDict, isDict := o.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Resources Dict", o)
	}
	var (
		out model.ResourcesDict
		err error
	)
	// Fonts
	out.Font, err = r.resolveFonts(resDict["Font"])
	if err != nil {
		return nil, err
	}
	if isRef { // write back to the cache
		r.resources[ref] = &out
	}
	return &out, err
}

func (r resolver) resolveFonts(ft pdfcpu.Object) (map[model.Name]*model.Font, error) {
	if ftRef, isRef := ft.(pdfcpu.IndirectRef); isRef {
		ft = r.resolve(ftRef)
	}
	if ft == nil {
		return nil, nil
	}
	ftDict, isDict := ft.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Resources Dict", ft)
	}
	ftMap := make(map[model.Name]*model.Font)
	for name, font := range ftDict {
		fontRef, isFontRef := font.(pdfcpu.IndirectRef)
		if isFontRef {
			if fontModel := r.fonts[fontRef]; isFontRef && fontModel != nil {
				ftMap[model.Name(name)] = fontModel
				continue
			}
			font = r.resolve(fontRef)
		}
		if font == nil { // ignore the name
			continue
		}
		fontDict, isDict := font.(pdfcpu.Dict)
		if !isDict {
			return nil, errType("Font", font)
		}
		fontType, err := r.parseFontDict(fontDict)
		if err != nil {
			return nil, err
		}
		fontModel := &model.Font{Subtype: fontType}
		if isFontRef { //write back to the cache
			r.fonts[fontRef] = fontModel
		}
		ftMap[model.Name(name)] = fontModel
	}
	return ftMap, nil
}

func parseDiffArray(ar pdfcpu.Array) model.Differences {
	var (
		currentCode   byte
		posInNameList int
		out           = make(model.Differences)
	)
	for _, o := range ar {
		switch o := o.(type) {
		case pdfcpu.Integer: // new code start
			currentCode = byte(o)
			posInNameList = 0
		case pdfcpu.Name:
			out[currentCode+byte(posInNameList)] = model.Name(o)
			posInNameList++
		}
	}
	return out
}

func (r resolver) resolveEncoding(encoding pdfcpu.Object) (model.Encoding, error) {
	if encName, isName := encoding.(pdfcpu.Name); isName {
		return model.NewPrededinedEncoding(string(encName)), nil
	}
	// ref or dict, maybe nil
	encRef, isRef := encoding.(pdfcpu.IndirectRef)
	if isRef {
		encoding = r.resolve(encRef)
	}
	if encoding == nil {
		return nil, nil
	}
	encDict, isDict := encoding.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Encoding", encoding)
	}
	var encModel model.EncodingDict
	if name := encDict.NameEntry("BaseEncoding"); name != nil {
		encModel.BaseEncoding = model.Name(*name)
	}
	if diff, ok := r.resolve(encDict["Differences"]).(pdfcpu.Array); ok {
		encModel.Differences = parseDiffArray(diff)
	}
	if isRef { // write back encoding to the cache
		r.encodings[encRef] = &encModel
	}
	return &encModel, nil
}

func (r resolver) parseFontDict(font pdfcpu.Dict) (model.FontType, error) {
	var err error
	switch font["Subtype"] {
	case pdfcpu.Name("Type0"):
		return model.Type0{}, nil
	case pdfcpu.Name("Type1"):
		out := model.Type1{}
		baseFont, _ := font["BaseFont"].(pdfcpu.Name)
		out.BaseFont = model.Name(baseFont)
		out.Encoding, err = r.resolveEncoding(font["Encoding"])
		return out, err
	case pdfcpu.Name("TrueType"):
		return model.TrueType{}, nil
	case pdfcpu.Name("Type3"):
		return model.Type3{}, nil
	default:
		return nil, nil
	}
}
