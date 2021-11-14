package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

// if not error return a non nil pointer
func (r resolver) resolveFunction(fn model.Object) (*model.FunctionDict, error) {
	fnRef, isRef := fn.(model.ObjIndirectRef)
	if fnM := r.functions[fnRef]; isRef && fnM != nil {
		return fnM, nil
	}
	fn = r.resolve(fn)
	var (
		out    model.FunctionDict
		err    error
		dict   model.ObjDict
		stream model.ObjStream
	)
	// fn is either a dict (type 2 and 3) or a content stream (type 0 and 4)
	switch fn := fn.(type) {
	case model.ObjDict:
		dict = fn
	case model.ObjStream:
		dict = fn.Args
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
		stream, ok, err := r.resolveStream(stream)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("missing content stream for PostScriptCalculator")
		}
		out.FunctionType = model.FunctionPostScriptCalculator(stream)
	}
	if err != nil {
		return nil, err
	}

	// common fields
	domain, _ := r.resolveArray(dict["Domain"])
	out.Domain, err = r.processRange(domain)
	if err != nil {
		return nil, err
	}
	range_, _ := r.resolveArray(dict["Range"])
	out.Range, err = r.processRange(range_)
	if err != nil {
		return nil, err
	}

	if isRef {
		r.functions[fnRef] = &out
	}
	return &out, nil
}

func (r resolver) processRange(range_ model.ObjArray) ([]model.Range, error) {
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

// do not impose a < b
func (r resolver) processPoints(range_ model.ObjArray) ([][2]Fl, error) {
	if len(range_)%2 != 0 {
		return nil, fmt.Errorf("expected even length for array, got %v", range_)
	}
	out := make([][2]Fl, len(range_)/2)
	for i := range out {
		a, _ := r.resolveNumber(range_[2*i])
		b, _ := r.resolveNumber(range_[2*i+1])
		out[i] = [2]Fl{a, b}
	}
	return out, nil
}

func (r resolver) processExpInterpolationFn(fn model.ObjDict) (model.FunctionExpInterpolation, error) {
	C0, _ := r.resolveArray(fn["C0"])
	C1, _ := r.resolveArray(fn["C1"])
	if len(C0) != len(C1) {
		return model.FunctionExpInterpolation{}, errors.New("array length must be equal for C0 and C1")
	}
	var out model.FunctionExpInterpolation
	out.C0 = r.processFloatArray(C0)
	out.C1 = r.processFloatArray(C1)
	if N, ok := r.resolveInt(fn["N"]); ok {
		out.N = N
	}
	return out, nil
}

func (r resolver) resolveStitchingFn(fn model.ObjDict) (model.FunctionStitching, error) {
	fns, _ := r.resolveArray(fn["Functions"])
	K := len(fns)
	var out model.FunctionStitching
	out.Functions = make([]model.FunctionDict, K)
	for i, f := range fns {
		fn, err := r.resolveFunction(f)
		if err != nil {
			return out, err
		}
		out.Functions[i] = *fn
	}
	bounds, _ := r.resolveArray(fn["Bounds"])
	if len(bounds) != K-1 {
		return out, fmt.Errorf("expected k-1 elements array for Bounds, got %v", bounds)
	}
	out.Bounds = r.processFloatArray(bounds)

	encode, _ := r.resolveArray(fn["Encode"])
	if len(encode) != 2*K {
		return out, fmt.Errorf("expected 2 x k elements array for Bounds, got %v", encode)
	}
	out.Encode = make([][2]Fl, K)
	for i := range out.Encode {
		out.Encode[i][0], _ = r.resolveNumber(encode[2*i])
		out.Encode[i][1], _ = r.resolveNumber(encode[2*i+1])
	}
	return out, nil
}

func (r resolver) processSampledFn(stream model.ObjStream) (model.FunctionSampled, error) {
	cs, ok, err := r.resolveStream(stream)
	if err != nil {
		return model.FunctionSampled{}, err
	}
	if !ok {
		return model.FunctionSampled{}, errors.New("missing stream for Sampled function")
	}
	out := model.FunctionSampled{Stream: cs}
	size, _ := r.resolveArray(stream.Args["Size"])
	m := len(size)
	out.Size = make([]int, m)
	for i, s := range size {
		out.Size[i], _ = r.resolveInt(s)
	}
	if bs, ok := r.resolveInt(stream.Args["BitsPerSample"]); ok {
		out.BitsPerSample = uint8(bs)
	}
	if o, ok := r.resolveInt(stream.Args["Order"]); ok {
		out.Order = uint8(o)
	}
	encode, _ := r.resolveArray(stream.Args["Encode"])
	if len(encode) != 2*m && len(encode) != 0 {
		return out, fmt.Errorf("expected 2 x m elements array for Bounds, got %v", encode)
	}
	out.Encode, err = r.processPoints(encode)
	if err != nil {
		return out, err
	}

	decode, _ := r.resolveArray(stream.Args["Decode"])
	out.Decode, err = r.processPoints(decode)

	return out, err
}
