package model

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/text/encoding/unicode"
)

func TestUnicode(t *testing.T) {
	enc := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder()

	encoded, err := enc.String("dlmkskmdskld")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(encoded))
}

func BenchmarkString(b *testing.B) {
	chunks := make([]string, 200)
	s := strings.Join(chunks, ",")

	b.Run("Sprintf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("[%s]", s)
		}
	})

	b.Run("+", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = "[" + s + "]"
		}
	})
}
