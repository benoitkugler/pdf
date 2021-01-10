// Pacakge bitmap provides support for bitmap fonts
// such as .pcf and .bdf formats.
package bitmap

import "golang.org/x/image/math/fixed"

// Property is either an `Atom`, an `Int` or a `Cardinal`
type Property interface {
	isProperty()
}

func (Atom) isProperty()     {}
func (Int) isProperty()      {}
func (Cardinal) isProperty() {}

type Atom string

type Int int32

type Cardinal uint32

type Size struct {
	height, width int16

	// size fixed.Point26_6

	XPpem, YPpem fixed.Int26_6
}
