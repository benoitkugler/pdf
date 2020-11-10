package reader

import (
	"fmt"
	"log"

	"github.com/benoitkugler/pdf/model"
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

func (r resolver) resolveOneFont(font pdfcpu.Object) (*model.Font, error) {
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
	fontModel := &model.Font{Subtype: fontType}
	if isFontRef { //write back to the cache
		r.fonts[fontRef] = fontModel
	}
	return fontModel, nil
}

func (r resolver) resolveFonts(ft pdfcpu.Object) (map[model.Name]*model.Font, error) {
	ft = r.resolve(ft)
	if ft == nil {
		return nil, nil
	}
	ftDict, isDict := ft.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Fonts Dict", ft)
	}
	ftMap := make(map[model.Name]*model.Font)
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

func (r resolver) resolveEncoding(encoding pdfcpu.Object) (model.SimpleEncoding, error) {
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

func (r resolver) resolveFontTT1orTT(font pdfcpu.Dict) (model.Type1, error) {
	var err error
	out := model.Type1{}

	baseFont, _ := font["BaseFont"].(pdfcpu.Name)
	out.BaseFont = model.Name(baseFont)

	out.Encoding, err = r.resolveEncoding(font["Encoding"])
	if err != nil {
		return model.Type1{}, err
	}
	if fc := font.IntEntry("FirstChar"); fc != nil {
		if *fc > 255 {
			return out, fmt.Errorf("overflow for FirstChar %d", *fc)
		}
		out.FirstChar = byte(*fc)
	}
	if lc := font.IntEntry("LastChar"); lc != nil {
		if *lc > 255 {
			return out, fmt.Errorf("overflow for FirstChar %d", *lc)
		}
		out.LastChar = byte(*lc)
	}

	widths, _ := r.resolve(font["Widths"]).(pdfcpu.Array)
	out.Widths = processFloatArray(widths)
	// be careful to byte overflow when LastChar = 255 and FirstChar = 0
	if exp := int(out.LastChar) - int(out.FirstChar) + 1; exp != len(out.Widths) {
		log.Printf("invalid length for font Widths array: expected %d, got %d", exp, len(out.Widths))
	}

	out.FontDescriptor, err = r.resolveFontDescriptor(font["FontDescriptor"])
	return out, err
}

func (r resolver) resolveFontDescriptor(entry pdfcpu.Object) (model.FontDescriptor, error) {
	fd := r.resolve(entry)
	fontDescriptor, isDict := fd.(pdfcpu.Dict)
	if !isDict {
		return model.FontDescriptor{}, errType("FontDescriptor", fd)
	}
	var out model.FontDescriptor
	if f, ok := isNumber(fontDescriptor["Ascent"]); ok {
		out.Ascent = f
	}
	if f, ok := isNumber(fontDescriptor["Descent"]); ok {
		out.Descent = f
	}
	if f, ok := isNumber(fontDescriptor["Leading"]); ok {
		out.Leading = f
	}
	if f, ok := isNumber(fontDescriptor["CapHeight"]); ok {
		out.CapHeight = f
	}
	if f, ok := isNumber(fontDescriptor["XHeight"]); ok {
		out.XHeight = f
	}
	if f, ok := isNumber(fontDescriptor["StemV"]); ok {
		out.StemV = f
	}
	if f, ok := isNumber(fontDescriptor["StemH"]); ok {
		out.StemH = f
	}
	if f, ok := isNumber(fontDescriptor["AvgWidth"]); ok {
		out.AvgWidth = f
	}
	if f, ok := isNumber(fontDescriptor["MaxWidth"]); ok {
		out.MaxWidth = f
	}
	if f, ok := isNumber(fontDescriptor["MissingWidth"]); ok {
		out.MissingWidth = f
	}
	if it, ok := isNumber(fontDescriptor["ItalicAngle"]); ok {
		out.ItalicAngle = it
	}
	if fl := fontDescriptor.IntEntry("Flags"); fl != nil && *fl >= 0 {
		out.Flags = uint32(*fl)
	}
	if name := fontDescriptor.NameEntry("FontName"); name != nil {
		out.FontName = model.Name(*name)
	}
	if bbox := rectangleFromArray(fontDescriptor.ArrayEntry("FontBBox")); bbox != nil {
		out.FontBBox = *bbox
	}
	return out, nil
}

func (r resolver) resolveFontT0(font pdfcpu.Dict) (model.Type0, error) {
	var err error
	out := model.Type0{}

	baseFont, _ := font["BaseFont"].(pdfcpu.Name)
	out.BaseFont = model.Name(baseFont)

	if enc, ok := font["Encoding"].(pdfcpu.Name); ok {
		out.Encoding = model.PredefinedCMapEncoding(enc)
	} else {
		// should'nt be common, dont bother tracking ref
		enc, err := r.processContentStream(font["Encoding"])
		if err != nil {
			return model.Type0{}, err
		}
		if enc != nil {
			out.Encoding = model.EmbeddedCMapEncoding(*enc)
		}
	}

	desc := font.ArrayEntry("DescendantFonts")
	if len(desc) != 1 {
		return model.Type0{}, fmt.Errorf("expected array of one indirect object, got %s", desc)
	}
	// we track the ref from the main font object
	// no need to track the descendants
	descFont := r.resolve(desc[0])
	descFontDict, isDict := descFont.(pdfcpu.Dict)
	if !isDict {
		return model.Type0{}, errType("DescendantFonts", descFont)
	}
	out.DescendantFonts, err = r.resolveCIDFontDict(descFontDict)
	if err != nil {
		return out, err
	}
	out.ToUnicode, err = r.processContentStream(font["ToUnicode"])
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
	if subtype := cid.NameEntry("Subtype"); subtype != nil {
		out.Subtype = model.Name(*subtype)
	}
	if baseFont := cid.NameEntry("BaseFont"); baseFont != nil {
		out.BaseFont = model.Name(*baseFont)
	}

	cidSystem := r.resolve(cid["CIDSystemInfo"])
	cidSystemDict, isDict := cidSystem.(pdfcpu.Dict)
	if !isDict {
		return model.CIDFontDictionary{}, errType("CIDSystemInfo", cidSystem)
	}
	if reg, ok := isString(cidSystemDict["Registry"]); ok {
		out.CIDSystemInfo.Registry = reg
	}
	if ord, ok := isString(cidSystemDict["Ordering"]); ok {
		out.CIDSystemInfo.Ordering = ord
	}
	if sup := cidSystemDict.IntEntry("Supplement"); sup != nil {
		out.CIDSystemInfo.Supplement = *sup
	}

	out.FontDescriptor, err = r.resolveFontDescriptor(cid["FontDescriptor"])
	if err != nil {
		return out, err
	}

	if w := cid.IntEntry("DW"); w != nil {
		out.DW = *w
	}
	dw2 := cid.ArrayEntry("DW2")
	if len(dw2) == 2 {
		dw21, _ := dw2[0].(pdfcpu.Integer)
		dw22, _ := dw2[1].(pdfcpu.Integer)
		out.DW2[0], out.DW2[1] = dw21.Value(), dw22.Value()
	}
	out.W = processCIDWidths(r.resolve(cid["W"]))
	out.W2 = processCIDWidths(r.resolve(cid["W2"]))
	return out, nil
}

func processCIDWidths(wds pdfcpu.Object) []model.CIDWidth {
	ar, _ := wds.(pdfcpu.Array)
	var out []model.CIDWidth
	for i := 0; i < len(ar); {
		first, _ := ar[i].(pdfcpu.Integer)
		if i+1 >= len(ar) {
			// invalid, ignore last element
			return out
		}
		switch next := ar[i+1].(type) {
		case pdfcpu.Integer:
			last := next
			if i+2 >= len(ar) {
				// invalid, ignore last element
				return out
			}
			w, _ := ar[i+2].(pdfcpu.Integer)
			out = append(out, model.CIDWidthRange{
				First: rune(first), Last: rune(last),
				Width: w.Value(),
			})
			i += 3
		case pdfcpu.Array:
			cid := model.CIDWidthArray{
				Start: rune(first),
				W:     make([]int, len(next)),
			}
			for j, w := range next {
				wi, _ := w.(pdfcpu.Integer)
				cid.W[j] = wi.Value()
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

func (r resolver) parseFontDict(font pdfcpu.Dict) (model.FontType, error) {
	switch font["Subtype"] {
	case pdfcpu.Name("Type0"):
		return r.resolveFontT0(font)
	case pdfcpu.Name("Type1"):
		return r.resolveFontTT1orTT(font)
	case pdfcpu.Name("TrueType"):
		t1, err := r.resolveFontTT1orTT(font)
		return model.TrueType(t1), err
	case pdfcpu.Name("Type3"):
		// TODO:
		fmt.Println(font)
		return model.Type3{}, nil
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
	if lw, ok := isNumber(state["LW"]); ok {
		out.LW = lw
	}
	if lc := state.IntEntry("LC"); lc != nil {
		out.LC = *lc
	}
	if lj := state.IntEntry("LJ"); lj != nil {
		out.LJ = *lj
	}
	if ml, ok := isNumber(state["ML"]); ok {
		out.ML = ml
	}
	if ri := state.NameEntry("RI"); ri != nil {
		out.RI = model.Name(*ri)
	}
	if ca, ok := isNumber(state["CA"]); ok {
		out.CA = ca
	}
	if ca, ok := isNumber(state["Ca"]); ok {
		out.Ca = ca
	}
	if ais := state.BooleanEntry("AIS"); ais != nil {
		out.AIS = *ais
	}
	if sa := state.BooleanEntry("SA"); sa != nil {
		out.SA = *sa
	}
	if sm, ok := isNumber(state["SM"]); ok {
		out.SM = sm
	}
	d := state.ArrayEntry("D")
	if len(d) == 2 {
		dash, _ := d[0].(pdfcpu.Array)
		phase, _ := isNumber(d[1])
		out.D.Array = processFloatArray(dash)
		out.D.Phase = phase
	}
	font := state.ArrayEntry("Font")
	if len(font) == 2 {
		out.Font.Size, _ = isNumber(font[1])
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
