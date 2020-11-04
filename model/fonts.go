package model

// Font is a PDF font dictionnary
type Font struct {
	Subtype FontType
}

type FontType interface {
	isFontType()
}

type Type0 struct{}

type Type1 struct {
	BaseFont            Name
	FirstChar, LastChar byte
	Widths              []float64 // length (LastChar − FirstChar + 1) index i is char FirstChar + i
	FontDescriptor      FontDescriptor
	Encoding            Encoding // optional
}
type TrueType Type1

type Type3 struct {
	FontBBox            Rectangle
	FontMatrix          Matrix
	CharProcs           map[Name]ContentStream
	Encoding            Encoding
	FirstChar, LastChar byte
	Widths              []float64 // length (LastChar − FirstChar + 1) index i is char FirstChar + i
	FontDescriptor      FontDescriptor
	Resources           ResourcesDict
}

func (Type0) isFontType()    {}
func (Type1) isFontType()    {}
func (Type3) isFontType()    {}
func (TrueType) isFontType() {}

type FontFlag uint32

const (
	FixedPitch  FontFlag = 1
	Serif       FontFlag = 1 << 2
	Symbolic    FontFlag = 1 << 3
	Script      FontFlag = 1 << 4
	Nonsymbolic FontFlag = 1 << 6
	Italic      FontFlag = 1 << 7
	AllCap      FontFlag = 1 << 17
	SmallCap    FontFlag = 1 << 18
	ForceBold   FontFlag = 1 << 19
)

type FontDescriptor struct {
	FontName        Name
	Flags           uint32
	FontBBox        Rectangle
	ItalicAngle     int
	Ascent, Descent float64
	Leading         float64
	CapHeight       float64
	XHeight         float64
	StemV, StemH    float64
	AvgWidth        float64
	MaxWidth        float64
	MissingWidth    float64
}

type Encoding interface {
	isEncoding()
}

func (PredefinedEncoding) isEncoding() {}

type PredefinedEncoding Name

const (
	MacRomanEncoding  PredefinedEncoding = "MacRomanEncoding"
	MacExpertEncoding PredefinedEncoding = "MacExpertEncoding"
	WinAnsiEncoding   PredefinedEncoding = "WinAnsiEncoding"
)

// Differences describes the differences from the encoding specified by BaseEncoding
// It is written in a PDF file as a more condensed form: it's an array:
// 	[ code1, name1_1, name1_2, code2, name2_1, name2_2, name2_3 ... ]
type Differences map[byte]Name

type EncodingDict struct {
	BaseEncoding Name        // optionnal
	Differences  Differences // optionnal
}
