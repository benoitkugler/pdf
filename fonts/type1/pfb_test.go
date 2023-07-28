package type1

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	tokenizer "github.com/benoitkugler/pstokenizer"
)

func TestOpen(t *testing.T) {
	for _, filename := range []string{
		"../test/c0419bt_.pfb",
		"../test/CalligrapherRegular.pfb",
		"../test/Z003-MediumItalic.t1",
	} {
		b, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}

		enc, err := ParseEncoding(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		if enc == nil {
			t.Fatal("expected encoding")
		}
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, filename := range []string{
			"../test/CalligrapherRegular.pfb",
			"../test/Z003-MediumItalic.t1",
		} {
			by, err := os.ReadFile(filename)
			if err != nil {
				b.Fatal(err)
			}

			_, err = ParseEncoding(bytes.NewReader(by))
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func TestTokenize(t *testing.T) {
	filename := "../test/CalligrapherRegular.pfb"
	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	s1, err := openPfb(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(s1))

	tks, err := tokenizer.Tokenize(s1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(tks))
}
