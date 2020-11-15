package model

import (
	"bytes"
	"fmt"
)

// implements basic types found in PDF files

type Rectangle struct {
	Llx, Lly, Urx, Ury float64 // lower-left x, lower-left y, upper-right x, and upper-right y coordinates of the rectangle
}

func (r Rectangle) String() string {
	return writeFloatArray([]float64{r.Llx, r.Lly, r.Urx, r.Ury})
}

// Rotation encodes a clock-wise rotation
type Rotation uint8

const (
	Zero Rotation = iota
	Quarter
	Half
	ThreeQuarter
)

// NewRotation validate the input and returns
// a rotation, which may be nil
func NewRotation(degrees int) *Rotation {
	if degrees%90 != 0 {
		return nil
	}
	r := Rotation((degrees / 90) % 4)
	return &r
}

func (r Rotation) Degrees() int {
	return 90 * int(r)
}

// Name is a symbol to be referenced,
// and it is included in PDF without encoding, by prepending/
type Name string

// String returns the PDF representation of a name
func (n Name) String() string {
	return "/" + string(n)
}

type FunctionType interface {
	isFunction()
	Clone() FunctionType
}

func (SampledFunction) isFunction()              {}
func (ExpInterpolationFunction) isFunction()     {}
func (StitchingFunction) isFunction()            {}
func (PostScriptCalculatorFunction) isFunction() {}

// Range represents an interval [a,b] where a < b
type Range [2]float64

// Function takes m arguments and return n values
type Function struct {
	FunctionType FunctionType
	Domain       []Range // length m
	Range        []Range // length n, optionnal for ExpInterpolationFunction and StitchingFunction
}

// pdfContent return the object content of `f`
// `pdf` is used to write and reference the sub-functions of a `StitchingFunction`
func (f Function) pdfContent(pdf pdfWriter) (string, []byte) {
	baseArgs := fmt.Sprintf("/Domain %s", writeRangeArray(f.Domain))
	if len(f.Range) != 0 {
		baseArgs += fmt.Sprintf("/Range %s", writeRangeArray(f.Range))
	}
	var (
		content string
		stream  []byte
	)
	switch ft := f.FunctionType.(type) {
	case SampledFunction:
		content, stream = ft.pdfContent(baseArgs)
	case ExpInterpolationFunction:
		content = ft.pdfString(baseArgs)
	case StitchingFunction:
		// start by writing the "child" functions
		content = ft.pdfString(baseArgs, pdf)
	case PostScriptCalculatorFunction:
		content, stream = ft.pdfContent(baseArgs)
	}
	return content, stream
}

// Clone returns a deep copy of the function
func (f Function) Clone() Function {
	var out Function
	out.FunctionType = f.FunctionType.Clone()
	out.Domain = append([]Range(nil), f.Domain...)
	out.Range = append([]Range(nil), f.Range...)
	return out
}

// convenience: write the functions and returns the corresponding reference
func (pdf pdfWriter) writeFunctions(fns []Function) []Reference {
	refs := make([]Reference, len(fns))
	for i, f := range fns {
		refs[i] = pdf.addObject(f.pdfContent(pdf))
	}
	return refs
}

type SampledFunction struct {
	Stream

	Size          []int        // length m
	BitsPerSample uint8        // 1, 2, 4, 8, 12, 16, 24 or 32
	Order         uint8        // 1 (linear) or 3 (cubic), optional, default to 1
	Encode        [][2]float64 // length m, optional, default to [ 0 (Size_0 − 1) 0 (Size_1 − 1) ... ]
	Decode        []Range      // length n, optionnal, default to Range
}

// adds to the common arguments the specificities of a `SampledFunction`
func (f SampledFunction) pdfContent(baseArgs string) (string, []byte) {
	var b bytes.Buffer
	b.WriteString("<</FunctionType 0 ")
	b.WriteString(baseArgs)
	b.WriteString(f.Stream.PDFCommonFields())
	b.WriteString(fmt.Sprintf("/Size %s/BitsPerSample %d", writeIntArray(f.Size), f.BitsPerSample))
	if f.Order != 0 {
		b.WriteString(fmt.Sprintf("/Order %d", f.Order))
	}
	if len(f.Encode) != 0 {
		b.WriteString("/Encode [ ")
		for _, v := range f.Encode {
			b.WriteString(fmt.Sprintf("%.3f %.3f ", v[0], v[1]))
		}
		b.WriteByte(']')
	}
	if len(f.Decode) != 0 {
		b.WriteString("/Decode ")
		b.WriteString(writeRangeArray(f.Decode))
	}
	b.WriteString(" >>")
	return b.String(), f.Content
}

// Clone returns a deep copy of the function
// (with concrete type `SampledFunction`)
func (f SampledFunction) Clone() FunctionType {
	out := f
	out.Stream = f.Stream.Clone()
	out.Size = append([]int(nil), f.Size...)
	out.Encode = append([][2]float64(nil), f.Encode...)
	out.Decode = append([]Range(nil), f.Decode...)
	return out
}

// ExpInterpolationFunction defines an exponential interpolation of one input
// value and n output values
type ExpInterpolationFunction struct {
	C0 []float64 // length n, optional, default to 0
	C1 []float64 // length n, optional, default to 1
	N  int       // interpolation exponent (N=1 for linear interpolation)
}

// adds to the common arguments the specificities of a `ExpInterpolationFunction`
func (f ExpInterpolationFunction) pdfString(baseArgs string) string {
	c0, c1 := "", ""
	if len(f.C0) != 0 {
		c0 = "/C0 " + writeFloatArray(f.C0)
	}
	if len(f.C1) != 0 {
		c1 = "/C1 " + writeFloatArray(f.C1)
	}
	return fmt.Sprintf("<</FunctionType 2 %s%s%s/N %d>>", baseArgs, c0, c1, f.N)
}

// Clone returns a deep copy of the function
// (with concrete type `ExpInterpolationFunction`)
func (f ExpInterpolationFunction) Clone() FunctionType {
	out := f
	out.C0 = append([]float64(nil), f.C0...)
	out.C1 = append([]float64(nil), f.C1...)
	return out
}

// StitchingFunction defines a stitching of the subdomains of several 1-input functions
// to produce a single new 1-input function
type StitchingFunction struct {
	Functions []Function   // array of k 1-input functions
	Bounds    []float64    // array of k − 1 numbers
	Encode    [][2]float64 // length k
}

// adds to the common arguments the specificities of a `StitchingFunction`.
func (f StitchingFunction) pdfString(baseArgs string, pdf pdfWriter) string {
	// start by writing the "child" functions
	refs := pdf.writeFunctions(f.Functions)
	return fmt.Sprintf("<</FunctionType 3 %s/Functions %s/Bounds %s/Encode %s>>",
		baseArgs, writeRefArray(refs), writeFloatArray(f.Bounds), writePointArray(f.Encode))
}

// Clone returns a deep copy of the function
// (with concrete type `StitchingFunction`)
func (f StitchingFunction) Clone() FunctionType {
	var out StitchingFunction
	out.Functions = make([]Function, len(f.Functions))
	for i, fu := range f.Functions {
		out.Functions[i] = fu.Clone()
	}
	out.Bounds = append([]float64(nil), f.Bounds...)
	out.Encode = append([][2]float64(nil), f.Encode...)
	return out
}

// PostScriptCalculatorFunction is stream
// containing code written in a small subset of the PostScript language
type PostScriptCalculatorFunction Stream

// adds to the common arguments the specificities of a `PostScriptCalculatorFunction`.
func (f PostScriptCalculatorFunction) pdfContent(baseArgs string) (string, []byte) {
	s := Stream(f).PDFCommonFields()
	return fmt.Sprintf("<</FunctionType 4 %s %s>>\n", baseArgs, s), f.Content
}

// Clone returns a deep copy of the function
// (with concrete type `PostScriptCalculatorFunction`)
func (f PostScriptCalculatorFunction) Clone() FunctionType {
	return PostScriptCalculatorFunction(Stream(f).Clone())
}

// Matrix maps an input (x,y) to an output (x',y') defined by
// x′ = a × x + c × y + e
// y′ = b × x + d × y + f
type Matrix [6]float64 // [a,b,c,d,e,f]

// String return the PDF representation of the matrix
func (m Matrix) String() string {
	return writeFloatArray(m[:])
}
