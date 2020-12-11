package filters

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

func TestReacher(t *testing.T) {
	input := []byte("789456zesd45679998989")
	rd := bytes.NewReader(input)
	r := newReacher(rd, []byte("456"))
	_, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		t.Fatal(err)
	}
	nbRead := len(input) - rd.Len()
	if nbRead != 6 {
		t.Error()
	}

	rd = bytes.NewReader(input)
	r = newReacher(rd, []byte("998"))
	_, err = io.Copy(ioutil.Discard, r)
	if err != nil {
		t.Fatal(err)
	}
	nbRead = len(input) - rd.Len()
	if nbRead != len("789456zesd45679998") {
		t.Error()
	}
}
