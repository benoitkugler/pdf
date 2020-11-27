package type1

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestEmbed(t *testing.T) {
	files, err := ioutil.ReadDir("afms")
	if err != nil {
		t.Fatal(err)
	}
	var fonts []Font
	for _, info := range files {
		if !strings.HasSuffix(info.Name(), ".afm") {
			continue // licence file
		}

		f, err := os.Open("afms/" + info.Name())
		if err != nil {
			t.Fatal(err)
		}

		font, err := ParseFont(f)
		if err != nil {
			t.Fatalf("can't parse file %s : %s", info.Name(), err)
		}
		f.Close()

		fonts = append(fonts, font)
	}
	if err = dumpFontDescriptor(fonts); err != nil {
		t.Fatal(err)
	}
}
