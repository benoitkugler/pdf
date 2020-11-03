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

func (r Rotation) Degrees() int {
	return 90 * int(r)
}

// Name is a symbol to be referenced,
// and it is included in PDF without encoding, by prepending /
type Name string

type FunctionType uint8

const (
	Sampled FunctionType = iota
	_
	ExpInterpolation
	Stitching
	PostScript
)

// Range represents an interval [a,b] where a < b
type Range [2]float64

// Function takes m arguments and return n values
type Function struct {
	FunctionType FunctionType
	Domain       []Range // length m
	Range        []Range // length n
}

// Matrix maps an input (x,y) to an output (x',y') defined by
// x′ = a × x + c × y + e
// y′ = b × x + d × y + f
type Matrix [6]float64 // a,b,c,d,e,f
