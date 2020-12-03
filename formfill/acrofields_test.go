package formfill

import (
	"image/color"
	"reflect"
	"testing"
)

func TestDAParse(t *testing.T) {
	das := []string{
		"/Helv 9 Tf 0 g",
		"0.5 0.25 0.25 rg",
		"0.5 0.25 0.25 0.5 k",
		"0.5 g",
	}
	expecteds := []daConfig{
		{font: "Helv", size: 9},
		{color: color.NRGBA{R: 127, G: 63, B: 63, A: 255}},
		{color: color.CMYK{C: 127, M: 63, Y: 63, K: 127}},
		{color: color.Gray{127}},
	}
	for i, da := range das {
		conf, err := splitDAelements(da)
		if err != nil {
			t.Fatal(err)
		}
		if exp := expecteds[i]; !reflect.DeepEqual(conf, exp) {
			t.Errorf("expected %v, got %v", exp, conf)
		}
	}
}
