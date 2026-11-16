package cmaps

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestEncoding(t *testing.T) {
	b, _ := utf16Enc.Bytes([]byte(string(rune(0x2003e))))

	fmt.Println(hex.EncodeToString(b))
}
