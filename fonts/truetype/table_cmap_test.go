package truetype

import (
	"fmt"
	"os"
	"testing"
)

func TestCmap(t *testing.T) {
	for _, file := range []string{
		"testdata/Roboto-BoldItalic.ttf",
		"testdata/Raleway-v4020-Regular.otf",
		"testdata/Castoro-Regular.ttf",
		"testdata/Castoro-Italic.ttf",
		"testdata/FreeSerif.ttf",
		"testdata/AnjaliOldLipi-Regular.ttf",
	} {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		font, err := Parse(f)
		if err != nil {
			t.Fatal(err)
		}
		cmap, err := font.CmapTable()
		if err != nil {
			t.Fatal(err)
		}
		all := cmap.Compile()
		fmt.Println("	cmap:", len(all))
		for r, c := range all {
			c2 := cmap.Lookup(r)
			if c2 != c {
				t.Errorf("inconsistent lookup for rune %d : got %d and %d", r, c, c2)
			}
		}

		f.Close()

	}
}
