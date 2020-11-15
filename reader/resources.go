package reader

import (
	"fmt"
	"log"

	"errors"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/standardfonts"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveOneResourceDict(o pdfcpu.Object) (*model.ResourcesDict, error) {
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
	// Graphic state
	out.ExtGState, err = r.resolveExtGState(resDict["ExtGState"])
	if err != nil {
		return nil, err
	}
	// Color spaces
	out.ColorSpace, err = r.resolveColorSpace(resDict["ColorSpace"])
	if err != nil {
		return nil, err
	}
	// Shadings
	out.Shading, err = r.resolveShading(resDict["Shading"])
	if err != nil {
		return nil, err
	}
	// Patterns
	out.Pattern, err = r.resolvePattern(resDict["Pattern"])
	if err != nil {
		return nil, err
	}
	// Fonts
	out.Font, err = r.resolveFonts(resDict["Font"])
	if err != nil {
		return nil, err
	}
	// XObjects
	out.XObject, err = r.resolveXObjects(resDict["XObject"])
	if err != nil {
		return nil, err
	}

	if isRef { // write back to the cache
		r.resources[ref] = &out
	}

	return &out, nil
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
	if isFontRef { //write back to the cache
		r.fonts[fontRef] = fontModel
	}
	return fontModel, nil
}

func (r resolver) resolveFonts(ft pdfcpu.Object) (map[model.Name]*model.FontDict, error) {
	ft = r.resolve(ft)
	if ft == nil {
		return nil, nil
	}
	ftDict, isDict := ft.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Fonts Dict", ft)
	}
	ftMap := make(map[model.Name]*model.FontDict)
	for name, font := range ftDict {
		fontModel, err := r.resolveOneFont(font)
		if err != nil {
			return nil, err
		}
		if fontModel == nil { // ignore the name
			continue
		}
		ftMap[model.Name(name)] = fontModel
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
			out[currentCode+byte(posInNameList)] = model.Name(o)
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
		encModel.BaseEncoding = name
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
	if standard, ok := standardfonts.Fonts[string(out.BaseFont)]; ok {
		out.FirstChar = standard.FirstChar
		out.Widths = standard.Widths
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
	out.CharProcs = make(map[model.Name]model.ContentStream, len(charProcsDict))
	for name, proc := range charProcsDict {
		// char proc propably wont be shared accros fonts,
		// so we dont track the refs
		cs, err := r.resolveStream(proc)
		if err != nil {
			return out, err
		}
		if cs == nil {
			log.Printf("missing content stream for CharProc %s\n", name)
			continue
		}
		out.CharProcs[model.Name(name)] = model.ContentStream{Stream: *cs}
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

	out.ToUnicode, err = r.resolveStream(font["ToUnicode"])
	if err != nil {
		return out, err
	}

	return out, nil
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
		out.MissingWidth = f
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
	cs, err := r.resolveStream(object)
	if err != nil || cs == nil {
		return nil, err
	}

	stream, _ := r.resolve(object).(pdfcpu.StreamDict) // here, we know object is a StreamDict
	out := model.FontFile{Stream: *cs}

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

func (r resolver) resolveFontT0(font pdfcpu.Dict) (model.FontType0, error) {
	var err error
	out := model.FontType0{}

	out.BaseFont, _ = r.resolveName(font["BaseFont"])

	if enc, ok := r.resolveName(font["Encoding"]); ok {
		out.Encoding = model.CMapEncodingPredefined(enc)
	} else {
		// should'nt be common, dont bother tracking ref
		enc, err := r.resolveStream(font["Encoding"])
		if err != nil {
			return model.FontType0{}, err
		}
		if enc != nil {
			out.Encoding = model.CMapEncodingEmbedded(*enc)
		}
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
	out.ToUnicode, err = r.resolveStream(font["ToUnicode"])
	if err != nil {
		return out, err
	}
	return out, nil
}

func (r resolver) resolveCIDFontDict(cid pdfcpu.Dict) (model.CIDFontDictionary, error) {
	var (
		out model.CIDFontDictionary
		err error
	)
	out.Subtype, _ = r.resolveName(cid["Subtype"])
	out.BaseFont, _ = r.resolveName(cid["BaseFont"])

	cidSystem := r.resolve(cid["CIDSystemInfo"])
	cidSystemDict, isDict := cidSystem.(pdfcpu.Dict)
	if !isDict {
		return model.CIDFontDictionary{}, errType("CIDSystemInfo", cidSystem)
	}
	out.CIDSystemInfo.Registry, _ = isString(r.resolve(cidSystemDict["Registry"]))
	out.CIDSystemInfo.Ordering, _ = isString(r.resolve(cidSystemDict["Ordering"]))
	out.CIDSystemInfo.Supplement, _ = r.resolveInt(cidSystemDict["Supplement"])

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
	out.W2 = r.processCIDWidths(cid["W2"])
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
				First: rune(first), Last: rune(last),
				Width: w,
			})
			i += 3
		case pdfcpu.Array:
			cid := model.CIDWidthArray{
				Start: rune(first),
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

func (r resolver) resolveExtGState(states pdfcpu.Object) (map[model.Name]*model.GraphicState, error) {
	states = r.resolve(states)
	if states == nil {
		return nil, nil
	}
	statesDict, isDict := states.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Graphics state Dict", states)
	}
	out := make(map[model.Name]*model.GraphicState)
	for name, state := range statesDict {
		gs, err := r.resolveOneExtGState(state)
		if err != nil {
			return nil, err
		}
		if gs == nil { // ignore the name
			continue
		}
		out[model.Name(name)] = gs
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
	var out model.GraphicState
	// undefined values
	out.LC = model.Undef
	out.LJ = model.Undef
	out.CA = model.Undef
	out.Ca = model.Undef
	out.SM = model.Undef

	out.LW, _ = r.resolveNumber(state["LW"])
	out.ML, _ = r.resolveNumber(state["ML"])
	out.RI, _ = r.resolveName(state["RI"])

	if lc, ok := r.resolveInt(state["LC"]); ok { // 0 is not a default value
		out.LC = lc
	}
	if lj, ok := r.resolveInt(state["LJ"]); ok { // 0 is not a default value
		out.LJ = lj
	}
	if ca, ok := r.resolveNumber(state["CA"]); ok { // 0 is not a default value
		out.CA = ca
	}
	if ca, ok := r.resolveNumber(state["Ca"]); ok { // 0 is not a default value
		out.Ca = ca
	}
	if sm, ok := r.resolveNumber(state["SM"]); ok { // 0 is not a default value
		out.SM = sm
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
	return &out, nil
}

func (r resolver) resolveColorSpace(colorSpace pdfcpu.Object) (map[model.Name]model.ColorSpace, error) {
	colorSpace = r.resolve(colorSpace)
	if colorSpace == nil {
		return nil, nil
	}
	colorSpaceDict, isDict := colorSpace.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Color space Dict", colorSpace)
	}
	out := make(map[model.Name]model.ColorSpace)
	for name, cs := range colorSpaceDict {
		gs, err := r.resolveOneColorSpace(cs)
		if err != nil {
			return nil, err
		}
		if gs == nil { // ignore the name
			continue
		}
		out[model.Name(name)] = gs
	}
	return out, nil
}
