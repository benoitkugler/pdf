package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
)

func (r resolver) resolveShading(sh model.Object) (map[model.ObjName]*model.ShadingDict, error) {
	sh = r.resolve(sh)
	if sh == nil {
		return nil, nil
	}
	shDict, isDict := sh.(model.ObjDict)
	if !isDict {
		return nil, errType("Shading", sh)
	}
	out := make(map[model.ObjName]*model.ShadingDict, len(shDict))
	for name, sha := range shDict {
		shModel, err := r.resolveOneShading(sha)
		if err != nil {
			return nil, err
		}
		out[model.ObjName(name)] = shModel
	}
	return out, nil
}

func (r resolver) resolveOneShading(shadings model.Object) (*model.ShadingDict, error) {
	shRef, isRef := shadings.(model.ObjIndirectRef)
	if sh := r.shadings[shRef]; isRef && sh != nil {
		return sh, nil
	}
	shadings = r.resolve(shadings)
	var (
		shDict model.ObjDict
		stream model.ObjStream
	)
	switch sh := shadings.(type) {
	case model.ObjDict:
		shDict = sh
	case model.ObjStream:
		shDict = sh.Args
		stream = sh
	default:
		return nil, errType("Shading", shadings)
	}

	var (
		out model.ShadingDict
		err error
	)
	// common fields
	bg, _ := r.resolveArray(shDict["Background"])
	out.Background = r.processFloatArray(bg)
	out.BBox = r.rectangleFromArray(shDict["BBox"])
	out.AntiAlias, _ = r.resolveBool(shDict["AntiAlias"])

	// color space
	out.ColorSpace, err = r.resolveOneColorSpace(shDict["ColorSpace"])
	if err != nil {
		return nil, err
	}

	// specialization
	st, _ := r.resolveInt(shDict["ShadingType"])
	switch st {
	case 1:
		out.ShadingType, err = r.resolveFunctionSh(shDict)
	case 2:
		out.ShadingType, err = r.resolveAxialSh(shDict)
	case 3:
		out.ShadingType, err = r.resolveRadialSh(shDict)
	case 4:
		out.ShadingType, err = r.resolveFreeFormSh(stream)
	case 5:
		out.ShadingType, err = r.resolveLatticeSh(stream)
	case 6:
		var ff model.ShadingFreeForm
		ff, err = r.resolveFreeFormSh(stream)
		out.ShadingType = model.ShadingCoons(ff)
	case 7:
		var ff model.ShadingFreeForm
		ff, err = r.resolveFreeFormSh(stream)
		out.ShadingType = model.ShadingTensorProduct(ff)
	default:
		return nil, fmt.Errorf("invalid shading type %d", st)
	}
	if err != nil {
		return nil, err
	}
	if isRef {
		r.shadings[shRef] = &out
	}
	return &out, nil
}

// may return nil
func (r resolver) resolveOneColorSpace(cs model.Object) (model.ColorSpace, error) {
	cs = r.resolve(cs)
	switch cs := cs.(type) {
	case model.ObjName:
		return model.NewNameColorSpace(string(cs))
	case model.ObjArray:
		return r.resolveArrayCS(cs)
	case nil:
		return nil, nil
	default:
		return nil, errType("Color Space", cs)
	}
}

func (r resolver) resolveArrayCS(ar model.ObjArray) (model.ColorSpace, error) {
	if len(ar) == 0 {
		return nil, fmt.Errorf("array for Color Space is empty")
	}
	csName, _ := r.resolveName(ar[0])
	switch csName {
	case "CalGray":
		return r.resolveCalGray(ar)
	case "CalRGB":
		return r.resolveCalRGB(ar)
	case "Lab":
		return r.resolveLab(ar)
	case "ICCBased":
		return r.resolveICCBased(ar)
	case "Indexed":
		return r.resolveIndexed(ar)
	case "Pattern": // uncoloured tiling pattern
		// we accept a one array element with the name pattern
		// even if it should rather be given as a plain name ...
		if len(ar) == 1 {
			return model.ColorSpacePattern, nil
		}
		if len(ar) != 2 {
			return nil, fmt.Errorf("expected 2-elements array for Pattern color space, got %v", ar)
		}
		cs, err := r.resolveOneColorSpace(ar[1])
		if err != nil {
			return nil, err
		}
		return model.ColorSpaceUncoloredPattern{UnderlyingColorSpace: cs}, nil
	case "Separation":
		return r.resolveSeparation(ar)
	case "DeviceN":
		return r.resolveDeviceN(ar)
	default:
		return nil, fmt.Errorf("invalid color space name %s", csName)
	}
}

func (r resolver) resolveCalGray(ar model.ObjArray) (model.ColorSpaceCalGray, error) {
	if len(ar) != 2 {
		return model.ColorSpaceCalGray{}, fmt.Errorf("expected 2-elements array in CalGray Color, got %v", ar)
	}
	dict, ok := r.resolve(ar[1]).(model.ObjDict)
	if !ok {
		return model.ColorSpaceCalGray{}, errType("CalGray", r.resolve(ar[1]))
	}
	var out model.ColorSpaceCalGray

	wp, _ := r.resolveArray(dict["WhitePoint"])
	if len(wp) != 3 {
		return out, fmt.Errorf("expected 3-elements array in CalGray.WhitePoint, got %v", wp)
	}
	copy(out.WhitePoint[:], r.processFloatArray(wp))

	bp, _ := r.resolveArray(dict["BlackPoint"])
	if len(bp) == 3 { // optional
		copy(out.BlackPoint[:], r.processFloatArray(bp))
	}
	if gamma, ok := r.resolveNumber(dict["Gamma"]); ok {
		out.Gamma = gamma
	}
	return out, nil
}

func (r resolver) resolveCalRGB(ar model.ObjArray) (model.ColorSpaceCalRGB, error) {
	if len(ar) != 2 {
		return model.ColorSpaceCalRGB{}, fmt.Errorf("expected 2-elements array in CalRGB Color, got %v", ar)
	}
	dict, ok := r.resolve(ar[1]).(model.ObjDict)
	if !ok {
		return model.ColorSpaceCalRGB{}, errType("CalRGB", r.resolve(ar[1]))
	}
	var out model.ColorSpaceCalRGB

	wp, _ := r.resolveArray(dict["WhitePoint"])
	if len(wp) != 3 {
		return out, fmt.Errorf("expected 3-elements array in CalRGB.WhitePoint, got %v", wp)
	}
	copy(out.WhitePoint[:], r.processFloatArray(wp))

	bp, _ := r.resolveArray(dict["BlackPoint"])
	if len(bp) == 3 { // optional
		copy(out.BlackPoint[:], r.processFloatArray(bp))
	}

	gamma, _ := r.resolveArray(dict["Gamma"])
	if len(gamma) == 3 { // optional
		copy(out.Gamma[:], r.processFloatArray(gamma))
	}

	mat, _ := r.resolveArray(dict["Matrix"])
	if len(mat) == 9 { // optional
		copy(out.Matrix[:], r.processFloatArray(mat))
	}
	return out, nil
}

func (r resolver) resolveLab(ar model.ObjArray) (model.ColorSpaceLab, error) {
	if len(ar) != 2 {
		return model.ColorSpaceLab{}, fmt.Errorf("expected 2-elements array in Lab Color, got %v", ar)
	}
	dict, ok := r.resolve(ar[1]).(model.ObjDict)
	if !ok {
		return model.ColorSpaceLab{}, errType("Lab", r.resolve(ar[1]))
	}
	var out model.ColorSpaceLab

	wp, _ := r.resolveArray(dict["WhitePoint"])
	if len(wp) != 3 {
		return out, fmt.Errorf("expected 3-elements array in Lab.WhitePoint, got %v", wp)
	}
	copy(out.WhitePoint[:], r.processFloatArray(wp))

	bp, _ := r.resolveArray(dict["BlackPoint"])
	if len(bp) == 3 { // optional
		copy(out.BlackPoint[:], r.processFloatArray(bp))
	}

	range_, _ := r.resolveArray(dict["Range"])
	if len(range_) == 4 { // optional
		copy(out.Range[:], r.processFloatArray(range_))
	}
	return out, nil
}

func (r resolver) resolveICCBased(ar model.ObjArray) (*model.ColorSpaceICCBased, error) {
	if len(ar) != 2 {
		return nil, fmt.Errorf("expected 2-elements array in ICCBase Color, got %v", ar)
	}
	ref, isRef := ar[1].(model.ObjIndirectRef)
	if icc := r.iccs[ref]; isRef && icc != nil {
		return icc, nil
	}
	obj := r.resolve(ar[1]) // ar[1] should be indirect, but we accept direct object
	common, ok, err := r.resolveStream(ar[1])
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("missing ICCBased stream")
	}
	out := model.ColorSpaceICCBased{Stream: common}
	stream, _ := obj.(model.ObjStream) // no error, ar[1] has type Stream

	out.N, _ = r.resolveInt(stream.Args["N"])

	out.Alternate, err = r.resolveOneColorSpace(stream.Args["Alternate"])
	if err != nil {
		return nil, err
	}
	ra, _ := r.resolveArray(stream.Args["Range"])
	out.Range, err = r.processPoints(ra)
	if err != nil {
		return nil, err
	}
	if isRef {
		r.iccs[ref] = &out
	}
	return &out, nil
}

func (r resolver) resolveIndexed(ar model.ObjArray) (model.ColorSpaceIndexed, error) {
	var (
		out model.ColorSpaceIndexed
		err error
	)
	if len(ar) != 4 {
		return out, fmt.Errorf("expected 4-elements array in Indexed Color, got %v", ar)
	}
	out.Base, err = r.resolveOneColorSpace(ar[1])
	if err != nil {
		return out, err
	}

	hival, _ := r.resolveInt(ar[2])
	out.Hival = uint8(hival)

	if lookupString, is := file.IsString(r.resolve(ar[3])); is {
		out.Lookup = model.ColorTableBytes(lookupString)
	} else { // stream
		lookupRef, isRef := ar[3].(model.ObjIndirectRef)
		cs, ok, err := r.resolveStream(ar[3])
		if err != nil {
			return out, err
		}
		if !ok {
			return out, errors.New("missing stream for lookup of Indexed color space")
		}
		out.Lookup = (*model.ColorTableStream)(&cs)
		if isRef {
			r.colorTableStreams[lookupRef] = (*model.ColorTableStream)(&cs)
		}
	}
	return out, nil
}

func (r resolver) resolveSeparation(ar model.ObjArray) (model.ColorSpaceSeparation, error) {
	var (
		out model.ColorSpaceSeparation
		err error
	)
	if len(ar) != 4 {
		return out, fmt.Errorf("expected 4-elements array in Separation Color, got %v", ar)
	}
	out.Name, _ = r.resolveName(ar[1])
	out.AlternateSpace, err = r.resolveAlternateColorSpace(ar[2])
	if err != nil {
		return out, err
	}
	fn, err := r.resolveFunction(ar[3])
	if err != nil {
		return out, err
	}
	out.TintTransform = *fn
	return out, nil
}

func (r resolver) resolveDeviceN(ar model.ObjArray) (model.ColorSpaceDeviceN, error) {
	var (
		out model.ColorSpaceDeviceN
		err error
	)
	if len(ar) != 4 && len(ar) != 5 {
		return out, fmt.Errorf("expected 4 or 5 elements array in DeviceN Color, got %v", ar)
	}
	names, _ := r.resolveArray(ar[1])
	out.Names = make([]model.ObjName, len(names))
	for i, n := range names {
		out.Names[i], _ = r.resolveName(n)
	}
	out.AlternateSpace, err = r.resolveAlternateColorSpace(ar[2])
	if err != nil {
		return out, err
	}
	fn, err := r.resolveFunction(ar[3])
	if err != nil {
		return out, err
	}
	out.TintTransform = *fn
	if len(ar) == 5 { // optional attributes
		out.Attributes, err = r.resolveDeviceNAttributes(ar[4])
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func (r resolver) resolveDeviceNAttributes(obj model.Object) (*model.ColorSpaceDeviceNAttributes, error) {
	obj = r.resolve(obj)
	dict, ok := obj.(model.ObjDict)
	if !ok {
		return nil, nil // accept null or invalid value silently
	}
	var (
		out model.ColorSpaceDeviceNAttributes
		err error
	)
	out.Subtype, _ = r.resolveName(dict["Subtype"])

	colorants, _ := r.resolve(dict["Colorants"]).(model.ObjDict)
	out.Colorants = make(map[model.ObjName]model.ColorSpaceSeparation, len(colorants))
	for name, col := range colorants {
		col, _ := r.resolveArray(col)
		out.Colorants[model.ObjName(name)], err = r.resolveSeparation(col)
		if err != nil {
			return nil, err
		}
	}

	processDict, _ := r.resolve(dict["Process"]).(model.ObjDict)
	out.Process.ColorSpace, err = r.resolveAlternateColorSpace(processDict["ColorSpace"]) // may return nil
	if err != nil {
		return nil, err
	}
	comps, _ := r.resolveArray(processDict["Components"])
	out.Process.Components = make([]model.ObjName, len(comps))
	for i, n := range comps {
		out.Process.Components[i], _ = r.resolveName(n)
	}

	if mix, ok := r.resolve(processDict["MixingHints"]).(model.ObjDict); ok {
		var m model.ColorSpaceDeviceNMixingHints

		sold, _ := r.resolve(mix["Solidities"]).(model.ObjDict)
		m.Solidities = make(map[model.ObjName]Fl, len(sold))
		for i, s := range sold {
			m.Solidities[model.ObjName(i)], _ = r.resolveNumber(s)
		}

		dot, _ := r.resolve(mix["DotGain"]).(model.ObjDict)
		m.DotGain = make(map[model.ObjName]model.FunctionDict, len(dot))
		for i, s := range dot {
			fn, err := r.resolveFunction(s)
			if err != nil {
				return nil, err
			}
			m.DotGain[model.ObjName(i)] = *fn
		}

		print, _ := r.resolveArray(processDict["PrintingOrder"])
		m.PrintingOrder = make([]model.ObjName, len(print))
		for i, n := range print {
			m.PrintingOrder[i], _ = r.resolveName(n)
		}
		out.MixingHints = &m
	}
	return &out, nil
}

// check that the alternate is not a special color space
// avoiding potential circle
func (r resolver) resolveAlternateColorSpace(alternate model.Object) (model.ColorSpace, error) {
	if ar, ok := r.resolveArray(alternate); ok && len(ar) >= 1 {
		name, _ := r.resolveName(ar[0])
		switch name {
		case "Pattern", "Indexed", "Separation", "DeviceN": // forbiden special colour spaces
			return nil, fmt.Errorf("alternate space must not be a special color space")
		}
	}
	return r.resolveOneColorSpace(alternate)
}

// resolve a func (possibly indirect) or an array of func
// returns the result as an array of funcs
// if `expectedN` is > 0, check that the dimension of the domain is `expectedN`
func (r resolver) resolveFuncOrArray(sh model.Object, expectedN int) ([]model.FunctionDict, error) {
	if ar, isAr := r.resolveArray(sh); isAr {
		out := make([]model.FunctionDict, len(ar))
		for i, f := range ar {
			fn, err := r.resolveFunction(f)
			if err != nil {
				return out, err
			}
			if expectedN > 0 && len(fn.Domain) != expectedN {
				return out, fmt.Errorf("expected %d-arg function, got %v", expectedN, out)
			}
			out[i] = *fn
		}
		return out, nil
	}
	fn, err := r.resolveFunction(sh)
	if err != nil {
		return nil, err
	}
	if expectedN > 0 && len(fn.Domain) != expectedN {
		return nil, fmt.Errorf("expected %d-arg function, got %v", expectedN, fn)
	}
	return []model.FunctionDict{*fn}, nil
}

func (r resolver) resolveFunctionSh(sh model.ObjDict) (model.ShadingFunctionBased, error) {
	var (
		out model.ShadingFunctionBased
		err error
	)

	if domain, _ := r.resolveArray(sh["Domain"]); len(domain) == 4 {
		for i, v := range domain {
			out.Domain[i], _ = r.resolveNumber(v)
		}
	}
	if mat := r.matrixFromArray(sh["Matrix"]); mat != nil {
		out.Matrix = *mat
	}

	out.Function, err = r.resolveFuncOrArray(sh["Function"], 2)
	return out, err
}

func (r resolver) resolveBaseGradient(sh model.ObjDict) (g model.BaseGradient, err error) {
	domain, _ := r.resolveArray(sh["Domain"])
	if len(domain) == 2 {
		g.Domain[0], _ = r.resolveNumber(domain[0])
		g.Domain[1], _ = r.resolveNumber(domain[1])
	}
	extend, _ := r.resolveArray(sh["Extend"])
	if len(extend) == 2 {
		g.Extend[0], _ = r.resolveBool(extend[0])
		g.Extend[1], _ = r.resolveBool(extend[1])
	}
	g.Function, err = r.resolveFuncOrArray(sh["Function"], 1)
	return g, err
}

func (r resolver) resolveAxialSh(sh model.ObjDict) (model.ShadingAxial, error) {
	g, err := r.resolveBaseGradient(sh)
	if err != nil {
		return model.ShadingAxial{}, err
	}
	out := model.ShadingAxial{BaseGradient: g}
	coords, _ := r.resolveArray(sh["Coords"])
	if len(coords) != 4 {
		return out, fmt.Errorf("unexpected Coords for Axial shading %v", coords)
	}
	for i, v := range coords {
		out.Coords[i], _ = r.resolveNumber(v)
	}
	return out, nil
}

func (r resolver) resolveRadialSh(sh model.ObjDict) (model.ShadingRadial, error) {
	g, err := r.resolveBaseGradient(sh)
	if err != nil {
		return model.ShadingRadial{}, err
	}
	out := model.ShadingRadial{BaseGradient: g}
	coords, _ := r.resolveArray(sh["Coords"])
	if len(coords) != 6 {
		return out, fmt.Errorf("unexpected Coords for Axial shading %v", coords)
	}
	for i, v := range coords {
		out.Coords[i], _ = r.resolveNumber(v)
	}
	return out, nil
}

func (r resolver) resolveStreamSh(sh model.ObjStream) (model.ShadingStream, error) {
	cs, ok, err := r.resolveStream(sh)
	if err != nil {
		return model.ShadingStream{}, err
	}
	if !ok {
		return model.ShadingStream{}, errors.New("missing Shading stream")
	}
	out := model.ShadingStream{Stream: cs}
	if bi, ok := r.resolveInt(sh.Args["BitsPerCoordinate"]); ok {
		out.BitsPerCoordinate = uint8(bi)
	}
	if bi, ok := r.resolveInt(sh.Args["BitsPerComponent"]); ok {
		out.BitsPerComponent = uint8(bi)
	}

	decode, _ := r.resolveArray(sh.Args["Decode"])
	out.Decode, err = r.processPoints(decode)
	if err != nil {
		return out, err
	}
	if fn := sh.Args["Function"]; fn != nil {
		out.Function, err = r.resolveFuncOrArray(fn, 0)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func (r resolver) resolveFreeFormSh(sh model.ObjStream) (out model.ShadingFreeForm, err error) {
	out.ShadingStream, err = r.resolveStreamSh(sh)
	if err != nil {
		return out, err
	}
	if bi, ok := r.resolveInt(sh.Args["BitsPerFlag"]); ok {
		out.BitsPerFlag = uint8(bi)
	}
	return out, nil
}

func (r resolver) resolveLatticeSh(sh model.ObjStream) (out model.ShadingLattice, err error) {
	out.ShadingStream, err = r.resolveStreamSh(sh)
	if err != nil {
		return out, err
	}
	out.VerticesPerRow, _ = r.resolveInt(sh.Args["VerticesPerRow"])
	return out, nil
}

// ----------------------------- Patterns -----------------------------

func (r resolver) resolvePattern(pattern model.Object) (map[model.ObjName]model.Pattern, error) {
	pattern = r.resolve(pattern)
	if pattern == nil {
		return nil, nil
	}
	patternDict, isDict := pattern.(model.ObjDict)
	if !isDict {
		return nil, errType("Pattern", pattern)
	}
	out := make(map[model.ObjName]model.Pattern, len(patternDict))
	for name, pat := range patternDict {
		pattern, err := r.resolveOnePattern(pat)
		if err != nil {
			return nil, err
		}
		out[model.ObjName(name)] = pattern
	}
	return out, nil
}

func (r resolver) resolveOnePattern(pat model.Object) (model.Pattern, error) {
	patRef, isRef := pat.(model.ObjIndirectRef)
	if pattern := r.patterns[patRef]; isRef && pattern != nil {
		return pattern, nil
	}
	pat = r.resolve(pat)
	var (
		patDict model.ObjDict
		stream  model.ObjStream // for a tiling pattern
	)
	switch pa := pat.(type) {
	case model.ObjDict:
		patDict = pa
	case model.ObjStream:
		patDict = pa.Args
		stream = pa
	default:
		return nil, errType("Pattern", pat)
	}

	var (
		out model.Pattern
		err error
	)
	patType, _ := r.resolveInt(patDict["PatternType"])
	switch patType {
	case 1:
		out, err = r.resolveTilingPattern(stream)
	case 2:
		out, err = r.resolveShadingPattern(patDict)
	default:
		err = fmt.Errorf("unexpected type for Pattern %d", patType)
	}
	if err != nil {
		return nil, err
	}
	if isRef {
		r.patterns[patRef] = out
	}
	return out, nil
}

func (r resolver) resolveTilingPattern(pat model.ObjStream) (*model.PatternTiling, error) {
	cs, _, err := r.resolveStream(pat)
	if err != nil {
		return nil, err
	}
	// since pat is not a ref, ok must be true
	out := model.PatternTiling{ContentStream: model.ContentStream{Stream: cs}}

	if pt, ok := r.resolveInt(pat.Args["PaintType"]); ok {
		out.PaintType = uint8(pt)
	}
	if pt, ok := r.resolveInt(pat.Args["TilingType"]); ok {
		out.TilingType = uint8(pt)
	}
	if rect := r.rectangleFromArray(pat.Args["BBox"]); rect != nil {
		out.BBox = *rect
	}
	out.XStep, _ = r.resolveNumber(pat.Args["XStep"])
	out.YStep, _ = r.resolveNumber(pat.Args["YStep"])
	out.Resources, err = r.resolveOneResourceDict(pat.Args["Resources"])
	if err != nil {
		return nil, err
	}
	if mat := r.matrixFromArray(pat.Args["Matrix"]); mat != nil {
		out.Matrix = *mat
	}
	return &out, nil
}

func (r resolver) resolveShadingPattern(pat model.ObjDict) (*model.PatternShading, error) {
	sh, err := r.resolveOneShading(pat["Shading"])
	if err != nil {
		return nil, err
	}
	var out model.PatternShading
	out.Shading = sh
	if m := r.matrixFromArray(pat["Matrix"]); m != nil {
		out.Matrix = *m
	}
	out.ExtGState, err = r.resolveOneExtGState(pat["ExtGState"])
	return &out, err
}
