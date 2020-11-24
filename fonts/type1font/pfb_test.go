package type1font

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestOpen(t *testing.T) {
	file := "test/CalligrapherRegular.pfb"
	b, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	s1, s2, err := OpenPfb(b)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(s1), len(s2))

	font, err := Parse(s1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(font.Encoding)
}
