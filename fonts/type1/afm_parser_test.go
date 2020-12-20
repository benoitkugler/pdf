package type1

import (
	"os"
	"testing"

	"github.com/benoitkugler/pdf/fonts/simpleencodings"
	"golang.org/x/exp/errors/fmt"
)

func TestParse(t *testing.T) {
	f, err := os.Open("test/Times-Bold.afm")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = ParseAFMFile(f)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKernings(t *testing.T) {
	f, err := os.Open("test/Times-Bold.afm")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	font, err := ParseAFMFile(f)
	if err != nil {
		t.Fatal(err)
	}

	out := font.Metrics().KernsWithEncoding(simpleencodings.MacExpert)
	fmt.Println(len(out))
}
