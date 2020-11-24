package formfill

import (
	"io/ioutil"
	"testing"
)

func TestFDF(t *testing.T) {
	b, err := ioutil.ReadFile("test/test.fdf")
	if err != nil {
		t.Fatal(err)
	}
	err = ReadFDF(b)
	if err != nil {
		t.Fatal(err)
	}
}
