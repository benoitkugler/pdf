package bitmap

import (
	"io/ioutil"
	"testing"
)

func TestParse(t *testing.T) {
	for _, file := range []string{
		"test/4x6.pcf",
		"test/8x16.pcf",
	} {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		_, err = parse(b)
		if err != nil {
			t.Fatal(err)
		}
	}
}
