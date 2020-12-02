package tokenizer

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestCharString(t *testing.T) {
	b, err := ioutil.ReadFile("test/charstrings.ps")
	if err != nil {
		t.Fatal(err)
	}
	tks, err := Tokenize(b)
	if err != nil {
		t.Fatal(err)
	}
	nbCs := 0
	for _, tk := range tks {
		if tk.Kind == CharString {
			nbCs++
		}
	}
	if nbCs != 269 {
		t.Fatalf("expected 269 CharStrings, got %d", nbCs)
	}
}

func TestFloats(t *testing.T) {
	fl := []float64{12e1, -124e7, 12e-7, 98.78, -45.4, 45}
	for i, st := range []string{
		"+12e1", "-124e7", "12e-7", "98.78", "-45.4", "45.",
	} {
		tk, err := Tokenize([]byte(st))
		if err != nil {
			t.Fatal(err)
		}
		if len(tk) != 1 {
			t.Errorf("expected 1 token, got %v", tk)
		}
		if tk[0].Kind != Float {
			t.Errorf("expected Float, got %s", tk[0].Kind)
		}
		if f, err := tk[0].Float(); err != nil || f != fl[i] {
			t.Errorf("expected %v got %v", fl[i], f)
		}
	}
}

func TestConvert(t *testing.T) {
	tk := Token{Value: "78.45"}
	_, err := tk.Float()
	if err != nil {
		t.Error(err)
	}
	tk = Token{Value: "78."}
	_, err = tk.Float()
	if err != nil {
		t.Error(err)
	}
	_, err = tk.Int()
	if err != nil {
		t.Error(err)
	}

	if tk.IsNumber() {
		t.Error()
	}
}

func TestStrings(t *testing.T) {
	for i := range [CharString + 1]int{} {
		if Kind(i).String() == "<invalid token>" {
			t.Error()
		}
	}
	if Kind(CharString+1).String() != "<invalid token>" {
		t.Error()
	}
}

func TestSkipBinary(t *testing.T) {
	out, err := Tokenize([]byte("7 8 stream dmslsùdm"))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Errorf("expected 3 tokens, got %v", out)
	}
	out, err = Tokenize([]byte("7 BI 8 ID dmslsùdm"))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 {
		t.Errorf("expected 4 tokens, got %v", out)
	}
}

func TestResume(t *testing.T) {
	input := []byte("7 8 9 4 5 6 4")
	tk := NewTokenizer(input)
	nplus2, err := tk.PeekPeekToken()
	if err != nil {
		t.Fatal(err)
	}
	if exp := (Token{Kind: Integer, Value: "8"}); nplus2 != exp {
		t.Errorf("expected %v got %v", exp, nplus2)
	}
	_, err = tk.NextToken()
	if err != nil {
		t.Fatal(err)
	}
	chunk := tk.SkipBytes(2)
	if !bytes.Equal(chunk, []byte(" 8")) {
		t.Errorf("expected %v got %v", []byte(" 8"), chunk)
	}
	next, err := tk.NextToken()
	if err != nil {
		t.Fatal(err)
	}
	if next != (Token{Kind: Integer, Value: "9"}) {
		t.Errorf("expected %v, got %v", Token{Kind: Integer, Value: "9"}, next)
	}
	if p := tk.CurrentPosition(); p != 5 {
		t.Errorf("expected %d, got %d", 5, p)
	}

	chunk = tk.SkipBytes(5)
	if !bytes.Equal(chunk, []byte(" 4 5 ")) {
		t.Errorf("expected %v got %v", []byte(" 4 5 "), chunk)
	}
	next, err = tk.NextToken()
	if err != nil {
		t.Fatal(err)
	}
	if next != (Token{Kind: Integer, Value: "6"}) {
		t.Errorf("expected %v, got %v", Token{Kind: Integer, Value: "6"}, next)
	}
	chunk = tk.SkipBytes(456)
	if !bytes.Equal(chunk, []byte(" 4")) {
		t.Errorf("expected %s got %s", []byte(" 4"), chunk)
	}
}
func TestBytes(t *testing.T) {
	input := []byte("7 8 9")
	tk := NewTokenizer(input)
	if len(tk.Bytes()) != len(input) {
		t.Error()
	}
	tk.NextToken()
	if len(tk.Bytes()) != len(input)-1 {
		t.Error()
	}
	tk.NextToken()
	if len(tk.Bytes()) != len(input)-3 {
		t.Error()
	}
	tk.NextToken()
	if tk.Bytes() != nil {
		t.Error()
	}
}

func TestEOL(t *testing.T) {
	input := []byte("a /Key \n 5 \r6 4")
	tk := NewTokenizer(input)
	_, err := tk.NextToken()
	if tk.HasEOLBeforeToken() {
		t.Error()
	}
	if err != nil {
		t.Error(err)
	}
	_, err = tk.NextToken()
	if err != nil {
		t.Error(err)
	}
	if !tk.HasEOLBeforeToken() {
		t.Error()
	}
}
