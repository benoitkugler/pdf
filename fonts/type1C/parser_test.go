package type1c

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
)

func TestParseCFF(t *testing.T) {
	files := []string{
		"../test/AAAPKB+SourceSansPro-Bold.cff",
		"../test/AdobeMingStd-Light-Identity-H.cff",
		"../test/YPTQCA+CMR17.cff",
	}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ParseEncoding(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err, "in", file)
		}
	}
}

func TestBulk(t *testing.T) {
	for _, file := range []string{
		"../test/AAAPKB+SourceSansPro-Bold.cff",
		"../test/AdobeMingStd-Light-Identity-H.cff",
		"../test/YPTQCA+CMR17.cff",
	} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for range [100]int{} {
			for range [500]int{} { // random mutation
				i := rand.Intn(len(b))
				b[i] = byte(rand.Intn(256))
			}
			_, _ = ParseEncoding(bytes.NewReader(b)) // we just check for crashes
		}
	}
}
