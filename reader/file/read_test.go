package file

import (
	"bytes"
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
	if L := len(ctx.xrefTable.objects); L != 126989 {
		t.Errorf("expected 126989 objects, got %d", L)
	}
}

func BenchmarkReadSpec(b *testing.B) {
	f, err := os.Open("../test/PDF_SPEC.pdf")
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err = Read(f, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestLines(t *testing.T) {
	expected := [...]string{
		"abc",
		"d",
		" ",
		"efgh ",
	}
	expectedOffsets := [...]int64{2, 7, 9, 11}
	input := []byte("\r\nabc\r\nd\r \nefgh \r\n\n\n")
	tk := newLineReader(bytes.NewReader(input))
	var (
		sl      []byte
		lines   [4]string
		offsets [4]int64
	)
	sl, offsets[0] = tk.readLine()
	lines[0] = string(sl)
	sl, offsets[1] = tk.readLine()
	lines[1] = string(sl)
	sl, offsets[2] = tk.readLine()
	lines[2] = string(sl)
	sl, offsets[3] = tk.readLine()
	lines[3] = string(sl)

	if lines != expected {
		t.Errorf("expected lines %v, got %v", expected, lines)
	}
	if expectedOffsets != offsets {
		t.Errorf("expected lines %v, got %v", expectedOffsets, offsets)
	}
	if l, _ := tk.readLine(); len(l) != 0 {
		t.Error("unexpected input")
	}
}

func TestBypass(t *testing.T) {
	f, err := os.Open("../test/PDF_SPEC.pdf")
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newContext(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.bypassXrefSection()
	if err != nil {
		t.Fatal(err)
	}
	if L := len(ctx.xrefTable.objects); L != 126988 {
		t.Errorf("expected 126988 objects, got %d", L)
	}
}

func TestCorpus(t *testing.T) {
	files := [...]string{
		"../test/Empty.pdf",
		"../test/descriptif.pdf",
		"../test/f1118s1.pdf",
		"../test/transparents.pdf",
		"../test/ModeleRecuFiscalEditable.pdf",
		"../test/CMYK_OP.pdf",
		"../test/CMYKSpot_OP.pdf",
		"../test/Shading.pdf",
		"../test/Shading4.pdf",
		"../test/Font_Substitution.pdf",
	}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}

		_, err = Read(f, nil)
		if err != nil {
			t.Fatal(err)
		}

		ctx, err := newContext(f, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = ctx.bypassXrefSection()
		if err != nil {
			t.Fatal(err)
		}

		for on := range ctx.xrefTable.objects {
			_, err := ctx.resolveObjectNumber(on)
			if err != nil {
				t.Fatal(file, err)
			}
		}

		f.Close()
	}
}
