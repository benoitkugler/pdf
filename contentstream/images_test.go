package contentstream

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestPng(t *testing.T) {
	for _, file := range []string{
		"test/test_gray.png",
		"test/test_alpha.png",
		"test/test_indexed.png",
	} {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		img, dpi, err := parsePNG(f, true, true)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(img.PDFCommonFields(true), dpi)
		fmt.Println(img.ColorSpace)
		if img.SMask != nil {
			fmt.Println(img.SMask.Length())
		}
		f.Seek(0, io.SeekStart)
		_, _, err = parsePNG(f, true, false)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

func TestJPG(t *testing.T) {
	f, err := os.Open("test/test.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := parseJPG(f)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(img.PDFCommonFields(true))
}
