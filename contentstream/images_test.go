package contentstream

import (
	"fmt"
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
