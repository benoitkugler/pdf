package file

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func TestFindPattern(t *testing.T) {
	last := strings.Repeat(" dkm sqdlqs à)çç)à(-à(-ç))", 200)
	data := strings.Repeat("ùsdd", 100) + "slmkd87x4csm xrefstream" + last
	ctx, _ := newContext(strings.NewReader(data), nil)
	buf, err := ctx.findStringFromFileEnd(0, "xrefstream")
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != len(last) {
		t.Fatalf("unexpected length %d", len(buf))
	}
}

func TestSpecOffset(t *testing.T) {
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

	_, err = ReadFile("../test/PDF_SPEC.pdf", nil)
	if err != nil {
		t.Fatal(err)
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

func filesFromDir(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	var out []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		out = append(out, filepath.Join(dir, f.Name()))
	}
	return out
}

func TestCorpus(t *testing.T) {
	for _, file := range filesFromDir("../test/corpus") {
		_, err := ReadFile(file, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCorpus2(t *testing.T) {
	// this corpus is copied from pdfcpu
	for _, file := range filesFromDir("../test/corpus/testdata") {
		_, err := ReadFile(file, nil)
		if err != nil {
			t.Fatal(file, err)
		}
	}
}

func TestXrefTableOffsets(t *testing.T) {
	f, err := os.Open("../test/ProtectedRC4.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ctx, err := newContext(f, NewDefaultConfiguration())
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

	expectedOffsets := []int64{
		0,
		213,
		300,
		9,
		87,
		412,
		544,
		659,
	}
	if len(ctx.xrefTable.objects) != len(expectedOffsets) {
		t.Fatal(ctx.xrefTable.objects)
	}
	for i := range expectedOffsets {
		generation := 0
		if i == 0 {
			generation = 65535
		}
		if off := ctx.xrefTable.objects[model.ObjIndirectRef{ObjectNumber: i, GenerationNumber: generation}].offset; off != expectedOffsets[i] {
			t.Fatalf("expected offset %d for %d, got %d", expectedOffsets[i], i, off)
		}
	}
}
