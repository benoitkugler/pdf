package model

// Font is a PDF font dictionnary
type Font struct {
	Type FontType
}

type FontType interface {
	isFontType()
}

type Type0 struct{}
type Type1 struct{}
type MMType1 struct{}
type Type3 struct{}

func (Type0) isFontType()   {}
func (Type1) isFontType()   {}
func (MMType1) isFontType() {}
func (Type3) isFontType()   {}
