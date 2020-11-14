package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveShading(sh pdfcpu.Object) (map[model.Name]*model.ShadingDict, error) {
	sh = r.resolve(sh)
	if sh == nil {
		return nil, nil
	}
	shDict, isDict := sh.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Shading", sh)
	}
	out := make(map[model.Name]*model.ShadingDict, len(shDict))
	for name, sha := range shDict {
		shModel, err := r.resolveOneShading(sha)
		if err != nil {
			return nil, err
		}
		out[model.Name(name)] = shModel
	}
	return out, nil
}

func (r resolver) resolveOneShading(shadings pdfcpu.Object) (*model.ShadingDict, error) {
	shRef, isRef := shadings.(pdfcpu.IndirectRef)
	if sh := r.shadings[shRef]; isRef && sh != nil {
		return sh, nil
	}
	shadings = r.resolve(shadings)
	var (
		shDict pdfcpu.Dict
		stream pdfcpu.StreamDict
	)
	switch sh := shadings.(type) {
	case pdfcpu.Dict:
		shDict = sh
	case pdfcpu.StreamDict:
		shDict = sh.Dict
		stream = sh
	default:
		return nil, errType("Shading", shadings)
	}

	var (
		out model.ShadingDict
		err error
	)
	// common fields
	bg, _ := r.resolve(shDict["Background"]).(pdfcpu.Array)
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
	case 4, 5, 7:
		fmt.Println(shDict)
		return nil, errors.New("not supported")
	case 6:
		out.ShadingType, err = r.resolveCoonsSh(stream)
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

func (r resolver) resolveOneColorSpace(cs pdfcpu.Object) (model.ColorSpace, error) {
	cs = r.resolve(cs)
	switch cs := cs.(type) {
	case pdfcpu.Name:
		return model.NewNameColorSpace(cs.Value())
	case pdfcpu.Array:
		return r.resolveArrayCS(cs)
	case nil:
		return nil, nil
	default:
		fmt.Println("not handled", cs)
		return nil, nil
	}
}

func (r resolver) resolveArrayCS(ar pdfcpu.Array) (model.ColorSpace, error) {
	if len(ar) == 0 {
		return nil, fmt.Errorf("array for Color Space is empty")
	}
	csName, _ := r.resolveName(ar[0])
	switch csName {
	case "Separation":
		return r.resolveSeparation(ar)
	case "ICCBased":
		return r.resolveICCBased(ar)
	case "Indexed":
		return r.resolveIndexed(ar)
	case "Pattern": // uncoloured tiling pattern
		if len(ar) != 2 {
			return nil, fmt.Errorf("expected 2-elements array for Pattern color space, got %v", ar)
		}
		cs, err := r.resolveOneColorSpace(ar[1])
		if err != nil {
			return nil, err
		}
		return model.UncoloredTilingPattern{UnderlyingColorSpace: cs}, nil
	default: // TODO
		fmt.Println("TODO array CS", ar)
		return nil, nil
	}
}

func (r resolver) resolveSeparation(ar pdfcpu.Array) (model.SeparationColorSpace, error) {
	var (
		out model.SeparationColorSpace
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

func (r resolver) resolveICCBased(ar pdfcpu.Array) (*model.ICCBasedColorSpace, error) {
	if len(ar) != 2 {
		return nil, fmt.Errorf("expected 2-elements array in ICCBase Color, got %v", ar)
	}
	ref, isRef := ar[1].(pdfcpu.IndirectRef)
	if icc := r.iccs[ref]; isRef && icc != nil {
		return icc, nil
	}
	obj := r.resolve(ar[1]) // ar[1] should be indirect, but we accept direct object
	common, err := r.processContentStream(ar[1])
	if err != nil {
		return nil, err
	}
	if common == nil {
		return nil, errors.New("missing ICCBased stream")
	}
	out := model.ICCBasedColorSpace{ContentStream: *common}
	stream, _ := obj.(pdfcpu.StreamDict) // no error, ar[1] has type Stream

	out.N, _ = r.resolveInt(stream.Dict["N"])

	out.Alternate, err = r.resolveOneColorSpace(stream.Dict["Alternate"])
	if err != nil {
		return nil, err
	}
	ra, _ := r.resolve(stream.Dict["Range"]).(pdfcpu.Array)
	out.Range, err = r.processRange(ra)
	if err != nil {
		return nil, err
	}
	if isRef {
		r.iccs[ref] = &out
	}
	return &out, nil
}

func (r resolver) resolveIndexed(ar pdfcpu.Array) (model.IndexedColorSpace, error) {
	var (
		out model.IndexedColorSpace
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

	if lookupString, is := isString(r.resolve(ar[3])); is {
		out.Lookup = model.ColorTableBytes(lookupString)
	} else { // stream
		lookupRef, isRef := ar[3].(pdfcpu.IndirectRef)
		cs, err := r.processContentStream(ar[3])
		if err != nil {
			return out, err
		}
		if cs == nil {
			return out, errors.New("missing stream for lookup of Indexed color space")
		}
		out.Lookup = (*model.ColorTableStream)(cs)
		if isRef {
			r.colorTableStreams[lookupRef] = (*model.ColorTableStream)(cs)
		}
	}
	return out, nil
}

// check that the alternate is not a special color space
// avoiding potential circle
func (r resolver) resolveAlternateColorSpace(alternate pdfcpu.Object) (model.ColorSpace, error) {
	if ar, ok := r.resolve(alternate).(pdfcpu.Array); ok && len(ar) >= 1 {
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
func (r resolver) resolveFuncOrArray(sh pdfcpu.Object, expectedN int) ([]model.Function, error) {
	if ar, isAr := r.resolve(sh).(pdfcpu.Array); isAr {
		out := make([]model.Function, len(ar))
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
	return []model.Function{*fn}, nil
}

func (r resolver) resolveFunctionSh(sh pdfcpu.Dict) (model.FunctionBased, error) {
	var (
		out model.FunctionBased
		err error
	)

	if domain, _ := r.resolve(sh["Domain"]).(pdfcpu.Array); len(domain) == 4 {
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

func (r resolver) resolveBaseGradient(sh pdfcpu.Dict) (g model.BaseGradient, err error) {
	domain, _ := r.resolve(sh["Domain"]).(pdfcpu.Array)
	if len(domain) == 2 {
		g.Domain[0], _ = r.resolveNumber(domain[0])
		g.Domain[1], _ = r.resolveNumber(domain[1])
	}
	extend, _ := r.resolve(sh["Extend"]).(pdfcpu.Array)
	if len(extend) == 2 {
		g.Extend[0], _ = r.resolveBool(extend[0])
		g.Extend[1], _ = r.resolveBool(extend[1])
	}
	g.Function, err = r.resolveFuncOrArray(sh["Function"], 1)
	return g, err
}

func (r resolver) resolveAxialSh(sh pdfcpu.Dict) (model.Axial, error) {
	g, err := r.resolveBaseGradient(sh)
	if err != nil {
		return model.Axial{}, err
	}
	out := model.Axial{BaseGradient: g}
	coords, _ := r.resolve(sh["Coords"]).(pdfcpu.Array)
	if len(coords) != 4 {
		return out, fmt.Errorf("unexpected Coords for Axial shading %v", coords)
	}
	for i, v := range coords {
		out.Coords[i], _ = r.resolveNumber(v)
	}
	return out, nil
}

func (r resolver) resolveRadialSh(sh pdfcpu.Dict) (model.Radial, error) {
	g, err := r.resolveBaseGradient(sh)
	if err != nil {
		return model.Radial{}, err
	}
	out := model.Radial{BaseGradient: g}
	coords, _ := r.resolve(sh["Coords"]).(pdfcpu.Array)
	if len(coords) != 6 {
		return out, fmt.Errorf("unexpected Coords for Axial shading %v", coords)
	}
	for i, v := range coords {
		out.Coords[i], _ = r.resolveNumber(v)
	}
	return out, nil
}

func (r resolver) resolveCoonsSh(sh pdfcpu.StreamDict) (model.Coons, error) {
	cs, err := r.processContentStream(sh)
	if err != nil {
		return model.Coons{}, err
	}
	if cs == nil {
		return model.Coons{}, errors.New("missing Coons stream")
	}
	out := model.Coons{ContentStream: *cs}
	if bi, ok := r.resolveInt(sh.Dict["BitsPerCoordinate"]); ok {
		out.BitsPerCoordinate = uint8(bi)
	}
	if bi, ok := r.resolveInt(sh.Dict["BitsPerComponent"]); ok {
		out.BitsPerComponent = uint8(bi)
	}
	if bi, ok := r.resolveInt(sh.Dict["BitsPerFlag"]); ok {
		out.BitsPerFlag = uint8(bi)
	}
	decode, _ := r.resolve(sh.Dict["Decode"]).(pdfcpu.Array)
	out.Decode, err = r.processRange(decode)
	if err != nil {
		return out, err
	}
	if fn := sh.Dict["Function"]; fn != nil {
		out.Function, err = r.resolveFuncOrArray(fn, 0)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// ----------------------------- Patterns -----------------------------

func (r resolver) resolvePattern(pattern pdfcpu.Object) (map[model.Name]model.Pattern, error) {
	pattern = r.resolve(pattern)
	if pattern == nil {
		return nil, nil
	}
	patternDict, isDict := pattern.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("Pattern", pattern)
	}
	out := make(map[model.Name]model.Pattern, len(patternDict))
	for name, pat := range patternDict {
		pattern, err := r.resolveOnePattern(pat)
		if err != nil {
			return nil, err
		}
		out[model.Name(name)] = pattern
	}
	return out, nil
}

func (r resolver) resolveOnePattern(pat pdfcpu.Object) (model.Pattern, error) {
	patRef, isRef := pat.(pdfcpu.IndirectRef)
	if pattern := r.patterns[patRef]; isRef && pattern != nil {
		return pattern, nil
	}
	pat = r.resolve(pat)
	var (
		patDict pdfcpu.Dict
		stream  pdfcpu.StreamDict // for a tiling pattern
	)
	switch pa := pat.(type) {
	case pdfcpu.Dict:
		patDict = pa
	case pdfcpu.StreamDict:
		patDict = pa.Dict
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

func (r resolver) resolveTilingPattern(pat pdfcpu.StreamDict) (*model.TilingPatern, error) {
	cs, err := r.processContentStream(pat)
	if err != nil {
		return nil, err
	}
	// since pat is not a ref, cs can't be nil
	out := model.TilingPatern{ContentStream: *cs}

	if pt, ok := r.resolveInt(pat.Dict["PaintType"]); ok {
		out.PaintType = uint8(pt)
	}
	if pt, ok := r.resolveInt(pat.Dict["TilingType"]); ok {
		out.TilingType = uint8(pt)
	}
	if rect := r.rectangleFromArray(pat.Dict["BBox"]); rect != nil {
		out.BBox = *rect
	}
	out.XStep, _ = r.resolveNumber(pat.Dict["XStep"])
	out.YStep, _ = r.resolveNumber(pat.Dict["YStep"])
	rs, err := r.resolveOneResourceDict(pat.Dict["Resources"])
	if err != nil {
		return nil, err
	}
	if rs != nil {
		out.Resources = *rs
	}
	if mat := r.matrixFromArray(pat.Dict["Matrix"]); mat != nil {
		out.Matrix = *mat
	}
	return &out, nil
}

func (r resolver) resolveShadingPattern(pat pdfcpu.Dict) (*model.ShadingPatern, error) {
	sh, err := r.resolveOneShading(pat["Shading"])
	if err != nil {
		return nil, err
	}
	var out model.ShadingPatern
	out.Shading = sh
	if m := r.matrixFromArray(pat["Matrix"]); m != nil {
		out.Matrix = *m
	}
	out.ExtGState, err = r.resolveOneExtGState(pat["ExtGState"])
	return &out, err
}
