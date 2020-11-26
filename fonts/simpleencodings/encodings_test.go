package simpleencodings

import (
	"fmt"
	"testing"
)

func TestNames(t *testing.T) {
	for _, e := range encs {
		fmt.Println(len(e.NamesToRune))
	}
}
