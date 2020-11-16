package model

import (
	"reflect"
	"testing"
)

func TestCloneCS(t *testing.T) {
	css := [...]ColorSpace{
		ColorSpaceCalGray{},
		ColorSpaceCalRGB{},
		CSDeviceCMYK,
		CSSeparation,
		&ColorSpaceICCBased{
			Alternate: CSDeviceCMYK,
		},
		ColorSpaceUncoloredPattern{
			UnderlyingColorSpace: ColorSpaceIndexed{},
		},
	}

	cache := newCloneCache()

	for _, cs := range css {
		cs2 := cloneColorSpace(cs, cache)
		if !reflect.DeepEqual(cs, cs2) {
			t.Errorf("expected %v, got %v", cs, cs2)
		}
	}
}
