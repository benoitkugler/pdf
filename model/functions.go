package model

import (
	"strconv"
)

// Function is one of FunctionSample, FunctionExpInterpolation,
// FunctionStitching, FunctionPostScriptCalculator
type Function interface {
	isFunction()
	Clone() Function
}

func (FunctionSampled) isFunction()              {}
func (FunctionExpInterpolation) isFunction()     {}
func (FunctionStitching) isFunction()            {}
func (FunctionPostScriptCalculator) isFunction() {}

// Range represents an interval [a,b] where a < b
type Range [2]Fl

// FunctionDict takes m arguments and return n values
type FunctionDict struct {
	FunctionType Function
	Domain       []Range // length m
	Range        []Range // length n, optionnal for ExpInterpolationFunction and StitchingFunction
}

// pdfContent return the object content of `f`
// `pdf` is used to write and reference the sub-functions of a `StitchingFunction`
func (f *FunctionDict) pdfContent(pdf pdfWriter, ref Reference) (StreamHeader, string, []byte) {
	baseArgs := map[ObjName]string{"Domain": writeRangeArray(f.Domain)}
	if len(f.Range) != 0 {
		baseArgs["Range"] = writeRangeArray(f.Range)
	}
	var (
		header StreamHeader
		stream []byte
	)
	switch ft := f.FunctionType.(type) {
	case FunctionSampled:
		header, stream = ft.pdfContent(baseArgs)
	case FunctionExpInterpolation:
		header = ft.pdfString(baseArgs)
	case FunctionStitching:
		// start by writing the "child" functions
		header = ft.pdfString(baseArgs, pdf)
	case FunctionPostScriptCalculator:
		header, stream = ft.pdfContent(baseArgs)
	}
	return header, "", stream
}

// Clone returns a deep copy of the function
func (f FunctionDict) Clone() FunctionDict {
	var out FunctionDict
	out.FunctionType = f.FunctionType.Clone()
	out.Domain = append([]Range(nil), f.Domain...)
	out.Range = append([]Range(nil), f.Range...)
	return out
}

func (f *FunctionDict) clone(cloneCache) Referenceable {
	if f == nil {
		return f
	}
	out := f.Clone()
	return &out
}

// convenience: write the functions and returns the corresponding reference
func (pdf pdfWriter) writeFunctions(fns []FunctionDict) []Reference {
	refs := make([]Reference, len(fns))
	for i, f := range fns {
		sh, _, sb := f.pdfContent(pdf, 0)
		refs[i] = pdf.addStream(sh, sb)
	}
	return refs
}

// FunctionSampled (type 0)
type FunctionSampled struct {
	Stream

	Size          []int   // length m
	BitsPerSample uint8   // 1, 2, 4, 8, 12, 16, 24 or 32
	Order         uint8   // 1 (linear) or 3 (cubic), optional, default to 1
	Encode        [][2]Fl // length m, optional, default to [ 0 (Size_0 − 1) 0 (Size_1 − 1) ... ]
	Decode        [][2]Fl // length n, optionnal, default to Range
}

// adds to the common arguments the specificities of a `SampledFunction`
func (f FunctionSampled) pdfContent(b map[Name]string) (StreamHeader, []byte) {
	out := f.Stream.PDFCommonFields(true)
	out.updateWith(b)
	out.Fields["FunctionType"] = "0"
	out.Fields["Size"] = writeIntArray(f.Size)
	out.Fields["BitsPerSample"] = strconv.Itoa(int(f.BitsPerSample))
	if f.Order != 0 {
		out.Fields["Order"] = strconv.Itoa(int(f.Order))
	}
	if len(f.Encode) != 0 {
		out.Fields["Encode"] = writePointsArray(f.Encode)
	}
	if len(f.Decode) != 0 {
		out.Fields["Decode"] = writePointsArray(f.Decode)
	}
	return out, f.Content
}

// Clone returns a deep copy of the function
// (with concrete type `SampledFunction`)
func (f FunctionSampled) Clone() Function {
	out := f
	out.Stream = f.Stream.Clone()
	out.Size = append([]int(nil), f.Size...)
	out.Encode = append([][2]Fl(nil), f.Encode...)
	out.Decode = append([][2]Fl(nil), f.Decode...)
	return out
}

// FunctionExpInterpolation (type 2) defines an exponential interpolation of one input
// value and n output values
type FunctionExpInterpolation struct {
	C0 []Fl // length n, optional, default to 0
	C1 []Fl // length n, optional, default to 1
	N  int  // interpolation exponent (N=1 for linear interpolation)
}

// adds to the common arguments the specificities of a `ExpInterpolationFunction`
func (f FunctionExpInterpolation) pdfString(baseArgs map[Name]string) StreamHeader {
	if len(f.C0) != 0 {
		baseArgs["C0"] = writeFloatArray(f.C0)
	}
	if len(f.C1) != 0 {
		baseArgs["C1"] = writeFloatArray(f.C1)
	}
	baseArgs["FunctionType"] = "2"
	baseArgs["N"] = strconv.Itoa(f.N)
	return StreamHeader{Fields: baseArgs}
}

// Clone returns a deep copy of the function
// (with concrete type `ExpInterpolationFunction`)
func (f FunctionExpInterpolation) Clone() Function {
	out := f
	out.C0 = append([]Fl(nil), f.C0...)
	out.C1 = append([]Fl(nil), f.C1...)
	return out
}

// FunctionStitching (type 3) defines a stitching of the subdomains of several 1-input functions
// to produce a single new 1-input function
type FunctionStitching struct {
	Functions []FunctionDict // array of k 1-input functions
	Bounds    []Fl           // array of k − 1 numbers
	Encode    [][2]Fl        // length k
}

// FunctionEncodeRepeat return a slice of k [0,1] encode domains.
func FunctionEncodeRepeat(k int) [][2]Fl {
	out := make([][2]Fl, k)
	for i := range out {
		out[i][1] = 1
	}
	return out
}

// adds to the common arguments the specificities of a `StitchingFunction`.
func (f FunctionStitching) pdfString(baseArgs map[Name]string, pdf pdfWriter) StreamHeader {
	// start by writing the "child" functions
	refs := pdf.writeFunctions(f.Functions)
	baseArgs["FunctionType"] = "3"
	baseArgs["Functions"] = writeRefArray(refs)
	baseArgs["Bounds"] = writeFloatArray(f.Bounds)
	baseArgs["Encode"] = writePointArray(f.Encode)
	return StreamHeader{Fields: baseArgs}
}

// Clone returns a deep copy of the function
// (with concrete type `StitchingFunction`)
func (f FunctionStitching) Clone() Function {
	var out FunctionStitching
	out.Functions = make([]FunctionDict, len(f.Functions))
	for i, fu := range f.Functions {
		out.Functions[i] = fu.Clone()
	}
	out.Bounds = append([]Fl(nil), f.Bounds...)
	out.Encode = append([][2]Fl(nil), f.Encode...)
	return out
}

// FunctionPostScriptCalculator (type 4) is stream
// containing code written in a small subset of the PostScript language
type FunctionPostScriptCalculator Stream

// adds to the common arguments the specificities of a `PostScriptCalculatorFunction`.
func (f FunctionPostScriptCalculator) pdfContent(baseArgs map[Name]string) (StreamHeader, []byte) {
	out := Stream(f).PDFCommonFields(true)
	out.updateWith(baseArgs)
	out.Fields["FunctionType"] = "4"
	return out, f.Content
}

// Clone returns a deep copy of the function
// (with concrete type `PostScriptCalculatorFunction`)
func (f FunctionPostScriptCalculator) Clone() Function {
	return FunctionPostScriptCalculator(Stream(f).Clone())
}

// Matrix maps an input (x,y) to an output (x',y') defined by
// x′ = a × x + c × y + e
// y′ = b × x + d × y + f
type Matrix [6]Fl // [a,b,c,d,e,f]

// String return the PDF representation of the matrix
func (m Matrix) String() string {
	return writeFloatArray(m[:])
}

// Multiply returns the product m * m2
func (m Matrix) Multiply(m2 Matrix) Matrix {
	a, b, c, d, e, f := m[0], m[1], m[2], m[3], m[4], m[5]
	a2, b2, c2, d2, e2, f2 := m2[0], m2[1], m2[2], m2[3], m2[4], m2[5]
	var out Matrix
	out[0] = a*a2 + c*b2
	out[1] = b*a2 + d*b2
	out[2] = a*c2 + c*d2
	out[3] = b*c2 + d*d2
	out[4] = e*a2 + f*b2 + e2
	out[5] = e*c2 + f*d2 + f2
	return out
}
