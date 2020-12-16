package file

import (
	"fmt"
	"os"
	"testing"
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

	err = ctx.buildXRefTableStartingAt(o)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(ctx.xrefTable))
	// buf := make([]byte, ctx.fileSize-o)
	// err = ctx.readAt(buf, o)
	// if err != nil {
	// 	t.Fatal(err)
	// }
}

func BenchmarkReadXRef(b *testing.B) {
	f, err := os.Open("../test/PDF_SPEC.pdf")
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err = Read(f, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
