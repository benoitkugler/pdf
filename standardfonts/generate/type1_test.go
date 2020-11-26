package type1

import (
	"os"
	"testing"

	"github.com/benoitkugler/pdf/fonts/simpleencodings"
)

func TestParse(t *testing.T) {
	f, err := os.Open("afms/Helvetica.afm")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		t.Fatal(err)
	}

	// _, wd := font.Widths()
	// if !reflect.DeepEqual(wd, standardfonts.Helvetica.Widths) {
	// 	t.Error()
	// }

	font.WidthsWithEncoding(simpleencodings.WinAnsi.Names)
}
