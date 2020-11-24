package simpleencodings

import (
	"fmt"
	"testing"
)

func TestNames(t *testing.T) {
	for _, e := range encsNameRunes {
		fmt.Println(len(e))
	}
}
