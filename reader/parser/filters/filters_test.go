package filters

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"testing"

	"github.com/benoitkugler/pdf/reader/parser/filters/ccitt"
)

var skippers = map[string]Skipper{
	ASCII85:   SkipperAscii85{},
	ASCIIHex:  SkipperAsciiHex{},
	RunLength: SkipperRunLength{},
	LZW:       SkipperLZW{EarlyChange: true},
	Flate:     SkipperFlate{},
	DCT:       SkipperDCT{},
	CCITTFax: SkipperCCITT{
		Params: ccitt.CCITTParams{
			Columns:    153,
			Rows:       55,
			EndOfBlock: true,
		},
	},
}

func forgeEncoded(t *testing.T, fi string) []byte {
	b, err := ioutil.ReadFile("samples/" + fi + ".bin")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDontPassEOD(t *testing.T) {
	for _, fi := range []string{
		ASCII85,
		ASCIIHex,
		RunLength,
		LZW,
		Flate,
		DCT,
		CCITTFax,
	} {
		filtered := forgeEncoded(t, fi)

		fil := skippers[fi]

		// add data passed EOD
		additionalBytes := bytes.Repeat([]byte("')(à'(ààç454658"), 1000)
		filteredPadded := append(filtered, additionalBytes...)

		read1, err := fil.Skip(bytes.NewReader(filteredPadded))
		if err != nil {
			t.Fatal(err)
		}

		// we want to use the number of byte read from the
		// filtered stream to detect EOD
		if read1 != len(filtered) {
			t.Errorf("invalid number of bytes read with filter %s: %d, expected %d", fi, read1, len(filtered))
		}
	}
}

func TestInvalid(t *testing.T) {
	for _, fi := range []string{
		ASCII85,
		ASCIIHex,
		RunLength,
		// LZW,
		Flate,
		DCT,
		CCITTFax,
	} {
		for range [200]int{} {
			// random input
			input := make([]byte, 80)
			_, _ = rand.Read(input)

			// random data may actually be valid since the eod ASCIIHex is easy to get
			if fi == ASCIIHex {
				input = bytes.ReplaceAll(input, []byte{eodHexDecode}, []byte{eodHexDecode + 1})
			} else if fi == RunLength {
				input = bytes.ReplaceAll(input, []byte{eodRunLength}, []byte{eodRunLength + 1})
			}

			fil := skippers[fi]
			_, err := fil.Skip(bytes.NewReader(input))
			if err == nil {
				t.Fatalf("filter %s: expected error on random data %v", fi, input)
			}
		}
	}
}

// forge 30x30 gray images with various filters
// but this rely on pdfcpu filters
// func TestCreateImageStream(t *testing.T) {
// 	in := make([]byte, 30*30)
// 	rand.Read(in)

// 	filtersName := []string{
// 		ASCII85,
// 		ASCIIHex,
// 		Flate,
// 		LZW,
// 		RunLength,
// 	}
// 	for _, fi := range filtersName {
// 		out, err := filter.NewFilter(string(fi), nil)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		r, err := out.Encode(bytes.NewReader(in))
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		encoded, err := ioutil.ReadAll(r)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		err = ioutil.WriteFile("samples/"+string(fi)+"_30x30.bin", encoded, os.ModePerm)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 	}
// }
