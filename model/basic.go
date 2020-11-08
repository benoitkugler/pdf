package model

// implements basic types found in PDF files

type Rectangle struct {
	Llx, Lly, Urx, Ury float64 // lower-left x, lower-left y, upper-right x, and upper-right y coordinates of the rectangle
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
// and it is included in PDF without encoding, by prepending /
type Name string

type FunctionType interface {
	isFunction()
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
	Range        []Range // length n
}

// TODO:
type SampledFunction struct{}

// ExpInterpolationFunction defines an exponential interpolation of one input
// value and n output values
type ExpInterpolationFunction struct {
	C0 []float64 // length n, optional, default to 0
	C1 []float64 // length n, optional, default to 1
	N  int       // interpolation exponent (N=1 for linear interpolation)
}

// StitchingFunction defines a stitching of the subdomains of several 1-input functions
// to produce a single new 1-input function
type StitchingFunction struct {
	Functions []*Function  // array of k 1-input functions
	Bounds    []float64    // array of k − 1 numbers
	Encode    [][2]float64 // length k
}

// TODO:
type PostScriptCalculatorFunction struct{}

// Matrix maps an input (x,y) to an output (x',y') defined by
// x′ = a × x + c × y + e
// y′ = b × x + d × y + f
type Matrix [6]float64 // a,b,c,d,e,f
