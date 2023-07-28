package filters

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/benoitkugler/pdf/reader/parser/filters/ccitt"
)

type cr struct {
	*bytes.Reader
}

func (c cr) ReadByte() (byte, error) {
	out, err := c.Reader.ReadByte()
	return out, err
}

func TestCCITT(t *testing.T) {
	b, err := os.ReadFile("ccitt/testdata/bw-gopher.ccitt_group3")
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, "dlsm"...)
	re := cr{bytes.NewReader(b)}
	r, err := ccitt.NewReader(re, ccitt.CCITTParams{Columns: 153, Rows: 55, EndOfBlock: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(b[len(b)-20:])
}
