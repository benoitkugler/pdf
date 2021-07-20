package reader

import (
	"errors"
	"fmt"
	"log"

	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveOneResourceDict(o pdfcpu.Object) (model.ResourcesDict, error) {
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		if res, ok := r.resources[ref]; isRef && ok {
			return res, nil
		}
		o = r.resolve(ref)
	}
	if o == nil {
		return model.ResourcesDict{}, nil
	}
	var (
		out model.ResourcesDict
		err error
	)
	resDict, isDict := o.(pdfcpu.Dict)
	if !isDict {
		return out, errType("Resources Dict", o)
	}
	// Graphic state
	out.ExtGState, err = r.resolveExtGState(resDict["ExtGState"])
	if err != nil {
		return out, err
	}
	// Color spaces
	out.ColorSpace, err = r.resolveColorSpace(resDict["ColorSpace"])
	if err != nil {
		return out, err
	}
	// Shadings
	out.Shading, err = r.resolveShading(resDict["Shading"])
	if err != nil {
		return out, err
	}
	// Patterns
	out.Pattern, err = r.resolvePattern(resDict["Pattern"])
	if err != nil {
		return out, err
	}
	// Fonts
	out.Font, err = r.resolveFonts(resDict["Font"])
	if err != nil {
		return out, err
	}
	// XObjects
	out.XObject, err = r.resolveXObjects(resDict["XObject"])
	if err != nil {
		return out, err
	}
	// Properties
	out.Properties, err = r.resolveProperties(resDict["Properties"])
	if err != nil {
		return out, err
	}

	if isRef { // write back to the cache
		r.resources[ref] = out
	}

	return out, nil
}

func (r resolver) resolveOneFont(font pdfcpu.Object) (*model.FontDict, error) {
	fontRef, isFontRef := font.(pdfcpu.IndirectRef)
	if isFontRef {
		if fontModel := r.fonts[fontRef]; isFontRef && fontModel != nil {
			return fontModel, nil
		}
		font = r.resolve(fontRef)
	}
	if font == nil { // ignore the name
		return nil, nil
	}
	fontDict, isDict := font.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Font", font)
	}
	fontType, err := r.parseFontDict(fontDict)
	if err != nil {
		return nil, err
	}
	fontModel := &model.FontDict{Subtype: fontType}
	fontModel.ToUnicode, err = r.resolveToUnicode(fontDict["ToUnicode"])
	if err != nil {
		return nil, err
	}
	if isFontRef { // write back to the cache
		r.fonts[fontRef] = fontModel
	}
	return fontModel, nil
}

func (r resolver) resolveFonts(ft pdfcpu.Object) (map[model.ObjName]*model.FontDict, error) {
	ft = r.resolve(ft)
	if ft == nil {
		return nil, nil
	}
	ftDict, isDict := ft.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Fonts Dict", ft)
	}
	ftMap := make(map[model.ObjName]*model.FontDict)
	for name, font := range ftDict {
		fontModel, err := r.resolveOneFont(font)
		if err != nil {
			return nil, err
		}
		if fontModel == nil { // ignore the name
			continue
		}
		ftMap[model.ObjName(name)] = fontModel
	}
	return ftMap, nil
}

func (r resolver) parseDiffArray(ar pdfcpu.Array) model.Differences {
	var (
		currentCode   byte
		posInNameList int
		out           = make(model.Differences)
	)
	for _, o := range ar {
		switch o := r.resolve(o).(type) {
		case pdfcpu.Integer: // new code start
			currentCode = byte(o)
			posInNameList = 0
		case pdfcpu.Name:
			out[currentCode+byte(posInNameList)] = model.ObjName(o)
			posInNameList++
		}
	}
	return out
}

func (r resolver) resolveEncoding(encoding pdfcpu.Object) (model.SimpleEncoding, error) {
	if encName, isName := r.resolveName(encoding); isName {
		return model.NewSimpleEncodingPredefined(string(encName)), nil
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
	var encModel model.SimpleEncodingDict
	if name, ok := r.resolveName(encDict["BaseEncoding"]); ok {
		if be := model.NewSimpleEncodingPredefined(string(name)); be != nil {
			encModel.BaseEncoding = be.(model.SimpleEncodingPredefined)
		}
	}
	if diff, ok := r.resolveArray(encDict["Differences"]); ok {
		encModel.Differences = r.parseDiffArray(diff)
	}
	if isRef { // write back encoding to the cache
		r.encodings[encRef] = &encModel
	}
	return &encModel, nil
}

func (r resolver) resolveFontTT1orTT(font pdfcpu.Dict) (out model.FontType1, err error) {
	out.BaseFont, _ = r.resolveName(font["BaseFont"])

	out.Encoding, err = r.resolveEncoding(font["Encoding"])
	if err != nil {
		return model.FontType1{}, err
	}

	// for the standard fonts, the font descriptor, first char and widths might be omited
	// add it, taking into account the encoding
	if standard, ok := standardfonts.Fonts[string(out.BaseFont)]; ok {
		var names [256]string
		switch enc := out.Encoding.(type) {
		case model.SimpleEncodingPredefined: // enc is validated by resolveEncoding
			names = *standardfonts.PredefinedEncodings[enc]
		case *model.SimpleEncodingDict:
			if enc.BaseEncoding != "" { // baseEncoding is validated by resolveEncoding
				names = *standardfonts.PredefinedEncodings[enc.BaseEncoding]
			} else {
				names = standard.Builtin
			}
			names = enc.Differences.Apply(names)
		default:
			names = standard.Builtin
		}
		f, w := standard.WidthsWithEncoding(names)
		out.FirstChar = f
		out.Widths = w
		out.FontDescriptor = standard.Descriptor
		return out, nil
	}

	out.FirstChar, out.Widths, err = r.resolveFontMetrics(font)
	if err != nil {
		return out, err
	}

	out.FontDescriptor, err = r.resolveFontDescriptor(font["FontDescriptor"])
	return out, err
}

func (r resolver) resolveFontT3(font pdfcpu.Dict) (out model.FontType3, err error) {
	bbox := r.rectangleFromArray(font["FontBBox"])
	if bbox == nil {
		return out, errors.New("missing FontBBox entry")
	}
	out.FontBBox = *bbox

	matrix := r.matrixFromArray(font["FontMatrix"])
	if matrix == nil {
		return out, errors.New("missing FontMatrix entry")
	}
	out.FontMatrix = *matrix

	charProcs := r.resolve(font["CharProcs"])
	charProcsDict, ok := charProcs.(pdfcpu.Dict)
	if !ok {
		return out, errType("Font.CharProcs", charProcs)
	}
	out.CharProcs = make(map[model.ObjName]model.ContentStream, len(charProcsDict))
	for name, proc := range charProcsDict {
		// char proc propably wont be shared accros fonts,
		// so we dont track the refs
		cs, ok, err := r.resolveStream(proc)
		if err != nil {
			return out, err
		}
		if !ok {
			log.Printf("missing content stream for CharProc %s\n", name)
			continue
		}
		out.CharProcs[model.ObjName(name)] = model.ContentStream{Stream: cs}
	}

	out.Encoding, err = r.resolveEncoding(font["Encoding"])
	if err != nil {
		return out, err
	}

	out.FirstChar, out.Widths, err = r.resolveFontMetrics(font)
	if err != nil {
		return out, err
	}

	if fd := r.resolve(font["FontDescriptor"]); fd != nil {
		fontD, err := r.resolveFontDescriptor(fd)
		if err != nil {
			return out, err
		}
		out.FontDescriptor = &fontD
	}

	out.Resources, err = r.resolveOneResourceDict(font["Resources"])
	if err != nil {
		return out, err
	}

	return out, nil
}

func (r resolver) resolveToUnicode(obj pdfcpu.Object) (*model.UnicodeCMap, error) {
	// keep track of the ref to avoid loops
	ref, _ := obj.(pdfcpu.IndirectRef)
	stream, ok, err := r.resolveStream(obj)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	dict, _ := r.resolve(obj).(pdfcpu.StreamDict)
	var out model.UnicodeCMap
	out.Stream = stream
	if kidRef, isRef := dict.Dict["UseCMap"].(pdfcpu.IndirectRef); isRef && kidRef == ref {
		// invalid loop, return early
		return &out, nil
	} else if name, ok := r.resolveName(dict.Dict["UseCMap"]); ok {
		out.UseCMap = model.UnicodeCMapBasePredefined(name)
	} else { // try another stream (maybe nil)
		u, err := r.resolveToUnicode(dict.Dict["UseCMap"])
		if err != nil || u == nil {
			return nil, err
		}
		out.UseCMap = *u
	}
	return &out, nil
}

// properies common to TrueType, Type1 and Type3
func (r resolver) resolveFontMetrics(font pdfcpu.Dict) (firstChar byte, widths []int, err error) {
	if fc, ok := r.resolveInt(font["FirstChar"]); ok {
		if fc > 255 {
			err = fmt.Errorf("overflow for FirstChar %d", fc)
			return
		}
		firstChar = byte(fc)
	}
	var lastChar byte // store it to check the length of Widths
	if lc, ok := r.resolveInt(font["LastChar"]); ok {
		if lc > 255 {
			err = fmt.Errorf("overflow for FirstChar %d", lc)
			return
		}
		lastChar = byte(lc)
	}

	wds, _ := r.resolveArray(font["Widths"])
	widths = make([]int, len(wds))
	for i, w := range wds {
		wf, _ := r.resolveNumber(w) // also accept float
		widths[i] = int(wf)
	}
	// be careful to byte overflow when LastChar = 255 and FirstChar = 0
	if exp := int(lastChar) - int(firstChar) + 1; widths != nil && exp != len(widths) {
		log.Printf("invalid length for font Widths array: expected %d, got %d", exp, len(widths))
	}

	return
}

func (r resolver) resolveFontDescriptor(entry pdfcpu.Object) (model.FontDescriptor, error) {
	fd := r.resolve(entry)
	fontDescriptor, isDict := fd.(pdfcpu.Dict)
	if !isDict {
		return model.FontDescriptor{}, errType("FontDescriptor", fd)
	}
	var out model.FontDescriptor
	if f, ok := r.resolveNumber(fontDescriptor["Ascent"]); ok {
		out.Ascent = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["Descent"]); ok {
		out.Descent = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["Leading"]); ok {
		out.Leading = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["CapHeight"]); ok {
		out.CapHeight = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["XHeight"]); ok {
		out.XHeight = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["StemV"]); ok {
		out.StemV = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["StemH"]); ok {
		out.StemH = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["AvgWidth"]); ok {
		out.AvgWidth = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["MaxWidth"]); ok {
		out.MaxWidth = f
	}
	if f, ok := r.resolveNumber(fontDescriptor["MissingWidth"]); ok {
		out.MissingWidth = int(f)
	}
	if it, ok := r.resolveNumber(fontDescriptor["ItalicAngle"]); ok {
		out.ItalicAngle = it
	}
	if fl, ok := r.resolveInt(fontDescriptor["Flags"]); ok && fl >= 0 {
		out.Flags = model.FontFlag(fl)
	}
	if name, ok := r.resolveName(fontDescriptor["FontName"]); ok {
		out.FontName = name
	}
	if bbox := r.rectangleFromArray(fontDescriptor["FontBBox"]); bbox != nil {
		out.FontBBox = *bbox
	}

	var err error
	if fontFile := fontDescriptor["FontFile"]; fontFile != nil {
		out.FontFile, err = r.processFontFile(fontFile)
	} else if fontFile := fontDescriptor["FontFile2"]; fontFile != nil {
		out.FontFile, err = r.processFontFile(fontFile)
	} else if fontFile := fontDescriptor["FontFile3"]; fontFile != nil {
		out.FontFile, err = r.processFontFile(fontFile)
	}
	if err != nil {
		return out, err
	}

	if charSet, ok := isString(r.resolve(fontDescriptor["CharSet"])); ok {
		out.CharSet = charSet
	}

	return out, nil
}

func (r resolver) processFontFile(object pdfcpu.Object) (*model.FontFile, error) {
	cs, ok, err := r.resolveStream(object)
	if err != nil || !ok { // return nil, nil on missing stream
		return nil, err
	}

	stream, _ := r.resolve(object).(pdfcpu.StreamDict) // here, we know object is a StreamDict
	out := model.FontFile{Stream: cs}

	out.Subtype, _ = r.resolveName(stream.Dict["Subtype"])

	if l, ok := r.resolveInt(stream.Dict["Length1"]); ok {
		out.Length1 = l
	}
	if l, ok := r.resolveInt(stream.Dict["Length2"]); ok {
		out.Length2 = l
	}
	if l, ok := r.resolveInt(stream.Dict["Length3"]); ok {
		out.Length3 = l
	}
	return &out, nil
}

func (r resolver) resolveCMapEncoding(enc pdfcpu.Object) (model.CMapEncoding, error) {
	if enc, ok := r.resolveName(enc); ok {
		return model.CMapEncodingPredefined(enc), nil
	}
	// keep the indirect to check for invalid loop
	ref, isRef := enc.(pdfcpu.IndirectRef)

	stream, ok, err := r.resolveStream(enc)
	if err != nil || !ok { // return nil, nil on missing stream
		return nil, err
	}
	encDict, _ := r.resolve(enc).(pdfcpu.StreamDict)
	var cmap model.CMapEncodingEmbedded
	cmap.Stream = stream
	cmap.CMapName, _ = r.resolveName(encDict.Dict["CMapName"])
	cmap.CIDSystemInfo, err = r.resolveCIDSystemInfo(encDict.Dict["CIDSystemInfo"])
	if err != nil {
		return nil, err
	}
	if wmode, _ := r.resolveInt(encDict.Dict["WMode"]); wmode == 1 {
		cmap.WMode = true
	}
	if otherRef, ok := encDict.Dict["UseCMap"]; isRef && ok && otherRef == ref {
		// avoid circle
	} else {
		cmap.UseCMap, err = r.resolveCMapEncoding(encDict.Dict["UserCMap"])
		if err != nil {
			return nil, err
		}
	}
	return cmap, nil
}

func (r resolver) resolveFontT0(font pdfcpu.Dict) (model.FontType0, error) {
	var err error
	out := model.FontType0{}

	out.BaseFont, _ = r.resolveName(font["BaseFont"])

	out.Encoding, err = r.resolveCMapEncoding(font["Encoding"])
	if err != nil {
		return out, err
	}
	if out.Encoding == nil {
		return out, errors.New("encoding is required in Type0 font dictionary")
	}

	desc, _ := r.resolveArray(font["DescendantFonts"])
	if len(desc) != 1 {
		return model.FontType0{}, fmt.Errorf("expected array of one indirect object, got %s", desc)
	}
	// we track the ref from the main font object
	// no need to track the descendants
	descFont := r.resolve(desc[0])
	descFontDict, isDict := descFont.(pdfcpu.Dict)
	if !isDict {
		return model.FontType0{}, errType("DescendantFonts", descFont)
	}
	out.DescendantFonts, err = r.resolveCIDFontDict(descFontDict)
	if err != nil {
		return out, err
	}
	return out, nil
}

func (r resolver) resolveCIDSystemInfo(object pdfcpu.Object) (out model.CIDSystemInfo, err error) {
	cidSystem := r.resolve(object)
	cidSystemDict, isDict := cidSystem.(pdfcpu.Dict)
	if !isDict {
		return model.CIDSystemInfo{}, errType("CIDSystemInfo", cidSystem)
	}
	out.Registry, _ = isString(r.resolve(cidSystemDict["Registry"]))
	out.Ordering, _ = isString(r.resolve(cidSystemDict["Ordering"]))
	out.Supplement, _ = r.resolveInt(cidSystemDict["Supplement"])
	return out, nil
}

func (r resolver) resolveCIDFontDict(cid pdfcpu.Dict) (model.CIDFontDictionary, error) {
	var (
		out model.CIDFontDictionary
		err error
	)
	out.Subtype, _ = r.resolveName(cid["Subtype"])
	out.BaseFont, _ = r.resolveName(cid["BaseFont"])

	out.CIDSystemInfo, err = r.resolveCIDSystemInfo(cid["CIDSystemInfo"])
	if err != nil {
		return out, err
	}

	out.FontDescriptor, err = r.resolveFontDescriptor(cid["FontDescriptor"])
	if err != nil {
		return out, err
	}

	out.DW, _ = r.resolveInt(cid["DW"])

	if dw2, _ := r.resolveArray(cid["DW2"]); len(dw2) == 2 {
		out.DW2[0], _ = r.resolveInt(dw2[0])
		out.DW2[1], _ = r.resolveInt(dw2[1])
	}
	out.W = r.processCIDWidths(cid["W"])

	out.W2, err = r.processCIDVerticalMetrics(cid["W2"])
	if err != nil {
		return out, err
	}

	if id, _ := r.resolveName(cid["CIDToGIDMap"]); id == "Identity" {
		out.CIDToGIDMap = model.CIDToGIDMapIdentity{}
	} else {
		stream, ok, err := r.resolveStream(cid["CIDToGIDMap"])
		if err != nil {
			return out, err
		}
		if ok {
			out.CIDToGIDMap = model.CIDToGIDMapStream{Stream: stream}
		}
	}
	return out, nil
}

func (r resolver) processCIDWidths(wds pdfcpu.Object) []model.CIDWidth {
	ar, _ := r.resolveArray(wds)
	var out []model.CIDWidth
	for i := 0; i < len(ar); {
		first, _ := r.resolveInt(ar[i])
		if i+1 >= len(ar) {
			// invalid, ignore last element
			return out
		}
		switch next := r.resolve(ar[i+1]).(type) {
		case pdfcpu.Integer:
			last := next
			if i+2 >= len(ar) {
				// invalid, ignore last element
				return out
			}
			w, _ := r.resolveInt(ar[i+2])
			out = append(out, model.CIDWidthRange{
				First: model.CID(first), Last: model.CID(last),
				Width: w,
			})
			i += 3
		case pdfcpu.Array:
			cid := model.CIDWidthArray{
				Start: model.CID(first),
				W:     make([]int, len(next)),
			}
			for j, w := range next {
				cid.W[j], _ = r.resolveInt(w)
			}
			out = append(out, cid)
			i += 2
		default:
			// invalid, return
			return out
		}
	}
	return out
}

func (r resolver) processCIDVerticalMetrics(wds pdfcpu.Object) ([]model.CIDVerticalMetric, error) {
	ar, _ := r.resolveArray(wds)
	var out []model.CIDVerticalMetric
	for i := 0; i < len(ar); {
		first, _ := r.resolveInt(ar[i])
		if i+1 >= len(ar) {
			return out, errors.New("invalid W2 entry")
		}
		switch next := r.resolve(ar[i+1]).(type) {
		case pdfcpu.Integer:
			last := next
			if i+4 >= len(ar) {
				return out, errors.New("invalid W2 entry")
			}
			w, _ := r.resolveInt(ar[i+2])
			vx, _ := r.resolveInt(ar[i+3])
			vy, _ := r.resolveInt(ar[i+4])
			out = append(out, model.CIDVerticalMetricRange{
				First: model.CID(first), Last: model.CID(last),
				VerticalMetric: model.VerticalMetric{Vertical: w, Position: [2]int{vx, vy}},
			})
			i += 5
		case pdfcpu.Array:
			if len(next)%3 != 0 {
				return out, errors.New("invalid W2 entry")
			}
			cid := model.CIDVerticalMetricArray{
				Start:     model.CID(first),
				Verticals: make([]model.VerticalMetric, len(next)/3),
			}
			for j := range cid.Verticals {
				cid.Verticals[j].Vertical, _ = r.resolveInt(next[3*j])
				cid.Verticals[j].Position[0], _ = r.resolveInt(next[3*j+1])
				cid.Verticals[j].Position[1], _ = r.resolveInt(next[3*j+2])
			}
			out = append(out, cid)
			i += 2
		default:
			// invalid, return
			return out, errType("vertical metric", next)
		}
	}
	return out, nil
}

func (r resolver) parseFontDict(font pdfcpu.Dict) (model.Font, error) {
	subtype, _ := r.resolveName(font["Subtype"])
	switch subtype {
	case "Type0":
		return r.resolveFontT0(font)
	case "Type1":
		return r.resolveFontTT1orTT(font)
	case "TrueType":
		t1, err := r.resolveFontTT1orTT(font)
		return model.FontTrueType(t1), err
	case "Type3":
		return r.resolveFontT3(font)
	default:
		return nil, nil
	}
}

func (r resolver) resolveExtGState(states pdfcpu.Object) (map[model.ObjName]*model.GraphicState, error) {
	states = r.resolve(states)
	if states == nil {
		return nil, nil
	}
	statesDict, isDict := states.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Graphics state Dict", states)
	}
	out := make(map[model.ObjName]*model.GraphicState)
	for name, state := range statesDict {
		gs, err := r.resolveOneExtGState(state)
		if err != nil {
			return nil, err
		}
		if gs == nil { // ignore the name
			continue
		}
		out[model.ObjName(name)] = gs
	}
	return out, nil
}

func (r resolver) resolveOneExtGState(state pdfcpu.Object) (*model.GraphicState, error) {
	stateRef, isRef := state.(pdfcpu.IndirectRef)
	if isRef {
		if gState := r.graphicsStates[stateRef]; isRef && gState != nil {
			return gState, nil
		}
		state = r.resolve(stateRef)
	}
	if state == nil {
		return nil, nil
	}
	stateDict, isDict := state.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Font", state)
	}
	gStateModel, err := r.parseStateDict(stateDict)
	if err != nil {
		return nil, err
	}
	if isRef {
		r.graphicsStates[stateRef] = gStateModel
	}
	return gStateModel, nil
}

func (r resolver) parseStateDict(state pdfcpu.Dict) (*model.GraphicState, error) {
	var (
		out model.GraphicState
		err error
	)

	out.LW, _ = r.resolveNumber(state["LW"])
	out.ML, _ = r.resolveNumber(state["ML"])
	out.RI, _ = r.resolveName(state["RI"])

	if lc, ok := r.resolveInt(state["LC"]); ok { // 0 is not a default value
		out.LC = model.ObjInt(lc)
	}
	if lj, ok := r.resolveInt(state["LJ"]); ok { // 0 is not a default value
		out.LJ = model.ObjInt(lj)
	}
	if ca, ok := r.resolveNumber(state["CA"]); ok { // 0 is not a default value
		out.CA = model.ObjFloat(ca)
	}
	if ca, ok := r.resolveNumber(state["ca"]); ok { // 0 is not a default value
		out.Ca = model.ObjFloat(ca)
	}
	if sm, ok := r.resolveNumber(state["SM"]); ok { // 0 is not a default value
		out.SM = model.ObjFloat(sm)
	}
	out.AIS, _ = r.resolveBool(state["AIS"])
	out.SA, _ = r.resolveBool(state["SA"])

	if d, _ := r.resolveArray(state["D"]); len(d) == 2 {
		dash, _ := r.resolveArray(d[0])
		phase, _ := r.resolveNumber(d[1])
		out.D.Array = r.processFloatArray(dash)
		out.D.Phase = phase
	}

	if font, _ := r.resolveArray(state["Font"]); len(font) == 2 {
		out.Font.Size, _ = r.resolveNumber(font[1])
		fontModel, err := r.resolveOneFont(font[0])
		if err != nil {
			return nil, err
		}
		out.Font.Font = fontModel
	}

	if bm, ok := r.resolveName(state["BM"]); ok {
		out.BM = []model.Name{bm}
	} else if bms, ok := r.resolveArray(state["BM"]); ok {
		out.BM = make([]model.ObjName, len(bms))
		for i, bm := range bms {
			out.BM[i], _ = r.resolveName(bm)
		}
	}

	out.SMask, err = r.resolveSoftMaskDict(state["SMask"])
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (r resolver) resolveSoftMaskDict(obj pdfcpu.Object) (model.SoftMaskDict, error) {
	var out model.SoftMaskDict
	obj = r.resolve(obj)
	switch obj := obj.(type) {
	case nil:
		return out, nil
	case pdfcpu.Name:
		if obj == "None" {
			out.S = "None"
			return out, nil
		} else {
			return out, fmt.Errorf("invalid name on SMask entry: %s", obj)
		}
	case pdfcpu.Dict:
		out.S, _ = r.resolveName(obj["S"])
		gObj := obj["G"]
		var g model.XObjectForm
		err := r.resolveXFormObjectFields(gObj, &g)
		if err != nil {
			return out, err
		}
		// here we known resolved gObj is a valid StreamDict
		gDict := r.resolve(gObj).(pdfcpu.StreamDict).Dict
		out.G = &model.XObjectTransparencyGroup{XObjectForm: g}
		group, _ := r.resolve(gDict["Group"]).(pdfcpu.Dict)
		out.G.CS, err = r.resolveOneColorSpace(group["CS"])
		if err != nil {
			return out, err
		}
		out.G.I, _ = r.resolveBool(group["I"])
		out.G.K, _ = r.resolveBool(group["K"])
		return out, nil
	default:
		return out, errType("SoftMaskDict", obj)
	}
}

func (r resolver) resolveColorSpace(colorSpace pdfcpu.Object) (model.ResourcesColorSpace, error) {
	colorSpace = r.resolve(colorSpace)
	if colorSpace == nil {
		return nil, nil
	}
	colorSpaceDict, isDict := colorSpace.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Color space Dict", colorSpace)
	}
	out := make(map[model.ObjName]model.ColorSpace)
	for name, cs := range colorSpaceDict {
		gs, err := r.resolveOneColorSpace(cs)
		if err != nil {
			return nil, err
		}
		if gs == nil { // ignore the name
			continue
		}
		out[model.ObjName(name)] = gs
	}
	return out, nil
}

func (r resolver) resolveProperties(obj pdfcpu.Object) (map[model.ObjName]model.PropertyList, error) {
	dict, _ := r.resolve(obj).(pdfcpu.Dict)
	out := map[model.ObjName]model.PropertyList{}
	var err error
	for k, v := range dict {
		vDict, _ := r.resolve(v).(pdfcpu.Dict)
		propDict := make(model.ObjDict)
		for pName, pValue := range vDict {
			// special case Metadata, which is common
			if pName == "Metadata" && r.customResolve == nil {
				cs, ok, err := r.resolveStream(pValue)
				if err != nil {
					return nil, fmt.Errorf("invalid Metadata entry: %s", err)
				}
				if ok {
					propDict["Metadata"] = model.MetadataStream{Stream: cs}
				}
			} else {
				propDict[model.ObjName(pName)], err = r.resolveCustomObject(pValue)
				if err != nil {
					return nil, fmt.Errorf("invalid property %s: %s", pName, err)
				}
			}
		}
		out[model.ObjName(k)] = propDict
	}
	return out, nil
}
