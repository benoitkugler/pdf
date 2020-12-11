package filters

import (
	"bytes"
	"image"
	"image/jpeg"
	"math/rand"
)

func randJPEG() ([]byte, error) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for i := range img.Pix {
		img.Pix[i] = uint8(rand.Int())
	}
	err := jpeg.Encode(&buf, img, nil)
	return buf.Bytes(), err
}

// func TestDCTFail(t *testing.T) {
// 	for range [30]int{} {
// 		input := make([]byte, 25000)
// 		rand.Read(input)

// 		lim := LimitedDCTDecoder(bytes.NewReader(input))
// 		_, err := ioutil.ReadAll(lim)
// 		if err == nil {
// 			t.Error("expected error on random input data")
// 		}
// 	}
// }
