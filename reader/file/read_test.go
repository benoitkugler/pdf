package file

import (
	"fmt"
	"os"
	"testing"

	"github.com/benoitkugler/pdf/reader/parser/tokenizer"
)

func TestOffset(t *testing.T) {
	f, err := os.Open("../test/PDF_SPEC.pdf")
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newContext(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, err := ctx.offsetLastXRefSection(0)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, ctx.fileSize-o)
	err = ctx.readAt(buf, o)
	if err != nil {
		t.Fatal(err)
	}

	trs, err := tokenizer.Tokenize(buf)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(trs))
}
