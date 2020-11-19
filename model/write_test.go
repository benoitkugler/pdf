package model

import (
	"fmt"
	"strings"
	"testing"
)

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
