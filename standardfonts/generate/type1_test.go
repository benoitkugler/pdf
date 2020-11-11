package type1

import (
	"fmt"
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	f, err := os.Open("afms/Symbol.afm")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(font.charMetrics)

	fmt.Println(font.Widths())
}
