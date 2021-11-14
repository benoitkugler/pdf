package file

import (
	"fmt"
	"testing"
)

func TestReadFDF(t *testing.T) {
	f, err := ReadFDFFile("../../formfill/test/sample.fdf")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(f.Root)
	fmt.Println(f.ResolveObject(f.Root))
}
