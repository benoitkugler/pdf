package contentstream

import (
	"fmt"
	"image"
	"os"
	"testing"
)

var imagesFiles = [...]string{
	"test/test_gray.png",
	"test/test_alpha.png",
	"test/test_indexed.png",
	"test/test.jpg",
	"test/test.tiff",
	"test/test.gif",
	"test/from_gif.png",
	"test/test.png",
}

func TestImportImages(t *testing.T) {
	for _, file := range imagesFiles {
		fmt.Println(file)
		img, _, err := ParseImageFile(file)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%T\n", img.ColorSpace)
	}
}

func TestImageFormat(t *testing.T) {
	for _, file := range imagesFiles {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		_, format, err := image.DecodeConfig(f)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(file, format)
	}
}
