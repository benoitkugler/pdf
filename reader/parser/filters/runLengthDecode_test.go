package filters

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestInvalidRunLength(t *testing.T) {
	fil := SkipperRunLength{}
	for range [200]int{} {
		input := make([]byte, 20)
		_, _ = rand.Read(input)
		input = bytes.ReplaceAll(input, []byte{eodRunLength}, []byte{eodRunLength - 1})

		_, err := fil.Skip(bytes.NewReader(input))
		if err == nil {
			t.Fatalf("expected error on random data %v", input)
		}
	}
}
