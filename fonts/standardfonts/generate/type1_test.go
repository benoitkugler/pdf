package type1

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	f, err := os.Open("afms/Helvetica.afm")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = ParseFont(f)
	if err != nil {
		t.Fatal(err)
	}

	// _, wd := font.Widths()
	// if !reflect.DeepEqual(wd, standardfonts.Helvetica.Widths) {
	// 	t.Error()
	// }

}
