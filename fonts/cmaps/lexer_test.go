package cmaps

import (
	"math/rand"
	"testing"

	"github.com/benoitkugler/pdf/parser/tokenizer"
)

func TestTokenize(t *testing.T) {
	for _, cmap := range [...]string{
		cmap1Data,
		cmap2Data,
		cmap3Data,
		cmap4Data,
	} {
		_, err := tokenizer.Tokenize([]byte(cmap))
		if err != nil {
			t.Error(err)
		}
	}
}

func TestArray(t *testing.T) {
	data := "[1 10 2555]"
	p := newparser([]byte(data))
	o, err := p.parseObject()
	if err != nil {
		t.Fatal(err)
	}
	arr, isArr := o.(cmapArray)
	if !isArr {
		t.Errorf("expected array, got %T", o)
	}
	if len(arr) != 3 {
		t.Errorf("expected array with length 3, got %v", arr)
	}
}

func TestDict(t *testing.T) {
	data := "<< /ks [1 10 2555]/dlmskd/s /dsmldlm 7 def>>"
	p := newparser([]byte(data))
	o, err := p.parseObject()
	if err != nil {
		t.Fatal(err)
	}
	dict, isDict := o.(cmapDict)
	if !isDict {
		t.Errorf("expected dict, got %T", o)
	}
	if len(dict) != 3 {
		t.Errorf("expected dict with length 2, got %v", dict)
	}
}

func TestBad(t *testing.T) {
	for _, data := range [...]string{
		"<< /ks >>",
		"[ << ]",
		"<< ( >>",
	} {
		p := newparser([]byte(data))
		o, err := p.parseObject()
		if err == nil && o != nil {
			t.Fatalf("expected error for input %s, got %v", data, o)
		}
	}
}

func randString(l int) string {
	var chars = []rune("           \n\n\n\n\n\n\nazero\tpqsdfv[]{}+-* bn,;:ù89456123<>ù!?./%µ")
	out := make([]rune, l)
	for i := range out {
		n := rand.Intn(len(chars))
		out[i] = chars[n]
	}
	return string(out)
}

func TestRandom(t *testing.T) {
	for range [200]int{} {
		s := randString(200)
		p := newparser([]byte(s))
		err := p.parse()
		if err == nil {
			t.Errorf("expected error on random data %s, got %v %v", s, p.cids, p.unicode)
		}
	}
}
