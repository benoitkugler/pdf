package sfnt

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestChars(t *testing.T) {
	for _, name := range [...]string{
		"gobold",
		"gomono",
		"goregular",
	} {
		f, err := Parse(fontData(name))
		if err != nil {
			t.Errorf("Parse(%q): %v", name, err)
			continue
		}
		assertCharIndex(t, name, f)
	}

	data, err := ioutil.ReadFile(filepath.FromSlash("../../../../../golang.org/x/image/font/testdata/cmapTest.ttf"))
	if err != nil {
		t.Fatal(err)
	}

	for _, format := range []int{-1, 0, 4, 12} {
		testCharMap(t, data, format)
	}
}

func assertCharIndex(t *testing.T, name string, f *Font) {
	for r, index := range f.Chars() {
		ind, err := f.GlyphIndex(nil, r)
		if err != nil {
			t.Errorf("GlyphIndex(%q): %v", name, err)
			continue
		}
		if ind != index {
			t.Errorf("unexpected index from char map: %d (wanted %d)", index, ind)
		}
	}
}

func testCharMap(t *testing.T, data []byte, cmapFormat int) {
	if cmapFormat >= 0 {
		originalSupportedCmapFormat := supportedCmapFormat
		defer func() {
			supportedCmapFormat = originalSupportedCmapFormat
		}()
		supportedCmapFormat = func(format, pid, psid uint16) bool {
			return int(format) == cmapFormat && originalSupportedCmapFormat(format, pid, psid)
		}
	}

	f, err := Parse(data)
	if err != nil {
		t.Errorf("cmapFormat=%d: %v", cmapFormat, err)
		return
	}
	assertCharIndex(t, fmt.Sprintf("cmapTest-format:%d", cmapFormat), f)
}
