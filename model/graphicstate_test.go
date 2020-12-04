package model

import (
	"bytes"
	"reflect"
	"testing"
)

func TestCloneCS(t *testing.T) {
	css := [...]ColorSpace{
		ColorSpaceCalGray{},
		ColorSpaceCalRGB{},
		ColorSpaceGray,
		&ColorSpaceICCBased{
			Alternate: ColorSpaceCMYK,
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

func TestWriteColorSpace(t *testing.T) {
	icc := &ColorSpaceICCBased{N: 4, Alternate: ColorSpaceRGB}
	cs := ColorSpaceSeparation{
		Name:           "test",
		AlternateSpace: icc,
	}
	cs2 := ColorSpaceIndexed{cs, 5, &ColorTableStream{}}

	pdf := newWriter(new(bytes.Buffer), nil)
	pdf.addObject(cs2.colorSpaceWrite(pdf), nil)
	if pdf.err != nil {
		t.Error(pdf.err)
	}
}
