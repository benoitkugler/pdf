package ccitt

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"
)

func testOneConfig(p CCITTParams) error {
	for range [30]int{} {
		input := make([]byte, 400)
		rand.Read(input)
		de, err := NewReader(bytes.NewReader(input), p)
		if err != nil {
			return err
		}
		_, _ = ioutil.ReadAll(de) // test for crashes only
	}
	return nil
}

func TestRandomExpectedFail(t *testing.T) {
	encs := []int32{-1, 0, 1}
	cols := []int32{4, 9, 100}
	rows := []int32{3, 9, 100}
	for _, enc := range encs {
		for _, col := range cols {
			for _, row := range rows {
				for _, eb := range [2]bool{true, false} {
					for _, el := range [2]bool{true, false} {
						for _, ba := range [2]bool{true, false} {
							for _, bl := range [2]bool{true, false} {
								p := CCITTParams{
									Encoding:   enc,
									Columns:    col,
									Rows:       row,
									EndOfBlock: eb,
									EndOfLine:  el,
									ByteAlign:  ba,
									Black:      bl,
								}
								err := testOneConfig(p)
								if err != nil {
									t.Fatal(err)
								}
							}
						}
					}
				}
			}
		}
	}
}

func invertBytes(b []byte) {
	for i, c := range b {
		b[i] = ^c
	}
}

func compareImages(t *testing.T, img0 image.Image, img1 image.Image) {
	t.Helper()

	b0 := img0.Bounds()
	b1 := img1.Bounds()
	if b0 != b1 {
		t.Fatalf("bounds differ: %v vs %v", b0, b1)
	}

	for y := b0.Min.Y; y < b0.Max.Y; y++ {
		for x := b0.Min.X; x < b0.Max.X; x++ {
			c0 := img0.At(x, y)
			c1 := img1.At(x, y)
			if c0 != c1 {
				t.Fatalf("pixel at (%d, %d) differs: %v vs %v", x, y, c0, c1)
			}
		}
	}
}

func decodePNG(fileName string) (image.Image, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

const width, height = 153, 55

func TestRead(t *testing.T) {
	for _, fileName := range []string{
		// "testdata/bw-gopher-aligned.ccitt_group3", // these file seems broken since fax2tiff error
		// "testdata/bw-gopher-inverted-aligned.ccitt_group3", // these file seems broken since fax2tiff error
		"testdata/bw-gopher.ccitt_group3",
		"testdata/bw-gopher-inverted.ccitt_group3",
		"testdata/bw-gopher.ccitt_group4",
		"testdata/bw-gopher-aligned.ccitt_group4",
		"testdata/bw-gopher-inverted.ccitt_group4",
		"testdata/bw-gopher-inverted-aligned.ccitt_group4",
		"testdata/bw-gopher-truncated0.ccitt_group3",
		"testdata/bw-gopher-truncated0.ccitt_group4",
		"testdata/bw-gopher-truncated1.ccitt_group3",
		"testdata/bw-gopher-truncated1.ccitt_group4",
	} {

		var params CCITTParams
		params.Columns = width
		params.Rows = height
		params.Encoding = 0
		if strings.HasSuffix(fileName, "group4") {
			params.Encoding = -1
		}
		params.ByteAlign = strings.Contains(fileName, "aligned")
		params.Black = strings.Contains(fileName, "inverted")

		truncated := strings.Contains(fileName, "truncated")

		testRead(t, fileName, params, truncated)
	}
}

func testRead(t *testing.T, fileName string, params CCITTParams, truncated bool) {
	// t.Helper()

	got := ""
	{
		f, err := os.Open(fileName)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer f.Close()
		r, err := NewReader(bufio.NewReader(f), params)
		if err != nil {
			t.Fatal(err)
		}
		gotBytes, err := ioutil.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll %s: %v", fileName, err)
		}
		got = string(gotBytes)
	}

	want := ""
	{
		img, err := decodePNG("testdata/bw-gopher.png")
		if err != nil {
			t.Fatalf("decodePNG: %v", err)
		}
		gray, ok := img.(*image.Gray)
		if !ok {
			t.Fatalf("decodePNG: got %T, want *image.Gray", img)
		}
		bounds := gray.Bounds()
		if w := bounds.Dx(); w != width {
			t.Fatalf("width: got %d, want %d", w, width)
		}
		if h := bounds.Dy(); h != height {
			t.Fatalf("height: got %d, want %d", h, height)
		}

		// Prepare to extend each row's width to a multiple of 8, to simplify
		// packing from 1 byte per pixel to 1 bit per pixel.
		extended := make([]byte, (width+7)&^7)

		wantBytes := []byte(nil)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			rowPix := gray.Pix[(y-bounds.Min.Y)*gray.Stride:]
			rowPix = rowPix[:width]
			copy(extended, rowPix)

			// Pack from 1 byte per pixel to 1 bit per pixel, MSB first.
			byteValue := uint8(0)
			for x, pixel := range extended {
				byteValue |= (pixel & 0x80) >> uint(x&7)
				if (x & 7) == 7 {
					wantBytes = append(wantBytes, byteValue)
					byteValue = 0
				}
			}
			// the 7 bits of padding of the last byte depends on revert
			if params.Black {
				b := wantBytes[len(wantBytes)-1]
				wantBytes[len(wantBytes)-1] = b | 0x7F
			}
		}
		want = string(wantBytes)
	}

	// We expect a width of 153 pixels, which is 20 bytes per row (at 1 bit per
	// pixel, plus 7 final bits of padding). Check that want is 20 * height
	// bytes long, and if got != want, format them to split at every 20 bytes.

	if n := len(want); n != 20*height {
		t.Fatalf("len(want): got %d, want %d", n, 20*height)
	}

	format := func(s string) string {
		b := []byte(nil)
		for row := 0; len(s) >= 20; row++ {
			b = append(b, fmt.Sprintf("row%02d: %02X\n", row, s[:20])...)
			s = s[20:]
		}
		if len(s) > 0 {
			b = append(b, fmt.Sprintf("%02X\n", s)...)
		}
		return string(b)
	}

	if got != want {
		t.Fatalf("%s: got:\n%s\nwant:\n%s", fileName, format(got), format(want))
	}

	// Passing AutoDetectHeight should produce the same output, provided that
	// the input hasn't truncated the trailing sequence of consecutive EOL's
	// that marks the end of the image.
	if !truncated {
		f, err := os.Open(fileName)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer f.Close()

		// auto-detect
		params.Rows = 0
		params.EndOfBlock = true

		re, err := NewReader(bufio.NewReader(f), params)
		if err != nil {
			t.Fatal(err)
		}
		adhBytes, err := ioutil.ReadAll(re)
		if err != nil {
			t.Fatalf("ReadAll %s: %v", fileName, err)
		}
		if s := string(adhBytes); s != want {
			t.Fatalf("AutoDetectHeight produced different output.\n"+
				"%s\n\n%s", format(s), format(want))
		}
	}
}
