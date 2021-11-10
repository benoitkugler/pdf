package file

import (
	"os"
	"testing"
)

func TestReadProtected(t *testing.T) {
	file := "../test/Protected.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Read(f, nil)
	if err != nil {
		t.Fatal(err)
	}
}
