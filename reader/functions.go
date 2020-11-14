package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// if not error return a non nil pointer
func (r resolver) resolveFunction(fn pdfcpu.Object) (*model.Function, error) {
	fnRef, isRef := fn.(pdfcpu.IndirectRef)
	if fnM := r.functions[fnRef]; isRef && fnM != nil {
		return fnM, nil
	}
	fn = r.resolve(fn)
	var (
		out    model.Function
		err    error
		dict   pdfcpu.Dict
		stream pdfcpu.StreamDict
	)
	// fn is either a dict (type 2 and 3) or a content stream (type 0 and 4)
	switch fn := fn.(type) {
	case pdfcpu.Dict:
		dict = fn
	case pdfcpu.StreamDict:
		dict = fn.Dict
		stream = fn
	default:
		return nil, errType("Function", fn)
	}

	// specialization
	fType, _ := r.resolveInt(dict["FunctionType"])
	switch fType {
	case 0:
		out.FunctionType, err = r.processSampledFn(stream)
	case 2:
		out.FunctionType, err = r.processExpInterpolationFn(dict)
	case 3:
		out.FunctionType, err = r.resolveStitchingFn(dict)
	case 4:
		stream, err := r.processContentStream(stream)
		if err != nil {
			return nil, err
		}
		out.FunctionType = model.PostScriptCalculatorFunction(*stream)
	}
	if err != nil {
		return nil, err
	}

	// common fields
	domain, _ := r.resolve(dict["Domain"]).(pdfcpu.Array)
	out.Domain, err = r.processRange(domain)
	if err != nil {
		return nil, err
	}
	range_, _ := r.resolve(dict["Range"]).(pdfcpu.Array)
	out.Range, err = r.processRange(range_)
	if err != nil {
		return nil, err
	}

	if isRef {
		r.functions[fnRef] = &out
	}
	return &out, nil
}

func (r resolver) processRange(range_ pdfcpu.Array) ([]model.Range, error) {
	if len(range_)%2 != 0 {
		return nil, fmt.Errorf("expected even length for array, got %v", range_)
	}
	out := make([]model.Range, len(range_)/2)
	for i := range out {
		a, _ := r.resolveNumber(range_[2*i])
		b, _ := r.resolveNumber(range_[2*i+1])
		if a > b {
			return nil, fmt.Errorf("invalid ranges range %v > %v", a, b)
		}
		out[i] = model.Range{a, b}
	}
	return out, nil
}

func (r resolver) processExpInterpolationFn(fn pdfcpu.Dict) (model.ExpInterpolationFunction, error) {
	C0, _ := r.resolve(fn["C0"]).(pdfcpu.Array)
	C1, _ := r.resolve(fn["C1"]).(pdfcpu.Array)
	if len(C0) != len(C1) {
		return model.ExpInterpolationFunction{}, errors.New("array length must be equal for C0 and C1")
	}
	var out model.ExpInterpolationFunction
	out.C0 = r.processFloatArray(C0)
	out.C1 = r.processFloatArray(C1)
	if N, ok := r.resolveInt(fn["N"]); ok {
		out.N = N
	}
	return out, nil
}

func (r resolver) resolveStitchingFn(fn pdfcpu.Dict) (model.StitchingFunction, error) {
	fns, _ := r.resolve(fn["Functions"]).(pdfcpu.Array)
	K := len(fns)
	var out model.StitchingFunction
	out.Functions = make([]model.Function, K)
	for i, f := range fns {
		fn, err := r.resolveFunction(f)
		if err != nil {
			return out, err
		}
		out.Functions[i] = *fn
	}
	bounds, _ := r.resolve(fn["Bounds"]).(pdfcpu.Array)
	if len(bounds) != K-1 {
		return out, fmt.Errorf("expected k-1 elements array for Bounds, got %v", bounds)
	}
	out.Bounds = r.processFloatArray(bounds)

	encode, _ := r.resolve(fn["Encode"]).(pdfcpu.Array)
	if len(encode) != 2*K {
		return out, fmt.Errorf("expected 2 x k elements array for Bounds, got %v", encode)
	}
	out.Encode = make([][2]float64, K)
	for i := range out.Encode {
		out.Encode[i][0], _ = r.resolveNumber(encode[2*i])
		out.Encode[i][1], _ = r.resolveNumber(encode[2*i+1])
	}
	return out, nil
}

func (r resolver) processSampledFn(stream pdfcpu.StreamDict) (model.SampledFunction, error) {
	cs, err := r.processContentStream(stream)
	if err != nil {
		return model.SampledFunction{}, err
	}
	if cs == nil {
		return model.SampledFunction{}, errors.New("missing stream for Sampled function")
	}
	out := model.SampledFunction{ContentStream: *cs}
	size, _ := r.resolve(stream.Dict["Size"]).(pdfcpu.Array)
	m := len(size)
	out.Size = make([]int, m)
	for i, s := range size {
		out.Size[i], _ = r.resolveInt(s)
	}
	if bs, ok := r.resolveInt(stream.Dict["BitsPerSample"]); ok {
		out.BitsPerSample = uint8(bs)
	}
	if o, ok := r.resolveInt(stream.Dict["Order"]); ok {
		out.Order = uint8(o)
	}
	encode, _ := r.resolve(stream.Dict["Encode"]).(pdfcpu.Array)
	if len(encode) != 2*m {
		return out, fmt.Errorf("expected 2 x m elements array for Bounds, got %v", encode)
	}
	out.Encode = make([][2]float64, m)
	for i := range out.Encode {
		out.Encode[i][0], _ = r.resolveNumber(encode[2*i])
		out.Encode[i][1], _ = r.resolveNumber(encode[2*i+1])
	}

	decode, _ := r.resolve(stream.Dict["Decode"]).(pdfcpu.Array)
	out.Decode, err = r.processRange(decode)

	return out, err
}
