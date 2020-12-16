package contentstream

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image/color"
	"image/jpeg"
	"io"
	"io/ioutil"
	"strings"

	"github.com/benoitkugler/pdf/model"
)

// supports importing common image type
// we follow the logic from gofpdf

// parseJPG extracts info from io.Reader with JPEG data
func parseJPG(r io.Reader) (out model.XObjectImage, err error) {
	out.Content, err = ioutil.ReadAll(r)
	if err != nil {
		return out, err
	}
	out.Filter = model.Filters{{Name: model.DCT}}

	config, err := jpeg.DecodeConfig(bytes.NewReader(out.Content))
	if err != nil {
		return out, err
	}
	out.Width = config.Width
	out.Height = config.Height
	out.BitsPerComponent = 8
	switch config.ColorModel {
	case color.GrayModel:
		out.ColorSpace = model.ColorSpaceGray
	case color.YCbCrModel:
		out.ColorSpace = model.ColorSpaceRGB
	case color.CMYKModel:
		out.ColorSpace = model.ColorSpaceCMYK
	default:
		return out, fmt.Errorf("image JPEG buffer has unsupported color space (%v)", config.ColorModel)
	}
	return
}

func pngColorSpace(ct byte) (cs model.ColorSpace, colorVal int, err error) {
	colorVal = 1
	switch ct {
	case 0, 4:
		cs = model.ColorSpaceGray
	case 2, 6:
		cs = model.ColorSpaceRGB
		colorVal = 3
	case 3: // the palette will be filled later
		cs = model.ColorSpaceIndexed{Base: model.ColorSpaceRGB}
	default:
		return nil, 0, fmt.Errorf("unknown color type in PNG buffer: %d", ct)
	}
	return
}

func beInt(buf *bytes.Buffer) int {
	var s [4]byte
	_, _ = buf.Read(s[:])
	return int(binary.BigEndian.Uint32(s[:]))
}

// sliceCompress returns a zlib-compressed copy of the specified byte array
func sliceCompress(data []byte) []byte {
	var buf bytes.Buffer
	cmp, _ := zlib.NewWriterLevel(&buf, zlib.BestSpeed)
	cmp.Write(data)
	cmp.Close()
	return buf.Bytes()
}

// sliceUncompress returns an uncompressed copy of the specified zlib-compressed byte array
func sliceUncompress(data []byte) (outData []byte, err error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}

func parsePNG(r io.Reader, readDPI, compressPalette bool) (img model.XObjectImage, dpi float64, err error) {
	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(r)
	if err != nil {
		return
	}
	// 	Check signature
	if string(buf.Next(8)) != "\x89PNG\x0d\x0a\x1a\x0a" {
		return img, 0, fmt.Errorf("not a PNG buffer")
	}
	// Read header chunk
	_ = buf.Next(4)
	if string(buf.Next(4)) != "IHDR" {
		return img, 0, fmt.Errorf("incorrect PNG buffer")
	}
	img.Width = beInt(buf)
	img.Height = beInt(buf)

	img.BitsPerComponent, err = buf.ReadByte()
	if err != nil {
		return img, 0, err
	}
	if img.BitsPerComponent > 8 {
		return img, 0, fmt.Errorf("16-bit depth not supported in PNG file")
	}

	ct, err := buf.ReadByte()
	if err != nil {
		return img, 0, err
	}

	cs, colorVal, err := pngColorSpace(ct)
	if err != nil {
		return img, 0, err
	}
	img.Filter = model.Filters{{Name: model.Flate, DecodeParms: map[string]int{
		"Predictor": 15,
		"Colors":    colorVal,
		"Columns":   img.Width,
	}}}

	if b, err := buf.ReadByte(); b != 0 || err != nil {
		return img, 0, fmt.Errorf("unknown compression method in PNG buffer")
	}
	if b, err := buf.ReadByte(); b != 0 || err != nil {
		return img, 0, fmt.Errorf("unknown filter method in PNG buffer")
	}
	if b, err := buf.ReadByte(); b != 0 || err != nil {
		return img, 0, fmt.Errorf("interlacing not supported in PNG buffer")
	}
	_ = buf.Next(4)

	// Scan chunks looking for palette, transparency and image data
	pal := make([]byte, 0, 32)
	data := make([]byte, 0, 32)
	var trns []int
	loop := true
	for loop {
		n := beInt(buf)
		switch string(buf.Next(4)) {
		case "PLTE": // Read palette
			pal = buf.Next(n)
			_ = buf.Next(4)
		case "tRNS": // Read transparency info
			t := buf.Next(n)
			switch ct {
			case 0:
				trns = []int{int(t[1])}
			case 2:
				trns = []int{int(t[1]), int(t[3]), int(t[5])}
			default:
				pos := strings.Index(string(t), "\x00")
				if pos >= 0 {
					trns = []int{pos}
				}
			}
			_ = buf.Next(4)
		case "IDAT": // Read image data block
			data = append(data, buf.Next(n)...)
			_ = buf.Next(4)
		case "IEND":
			loop = false
		case "pHYs":
			// png files theoretically support different x/y dpi
			// but we ignore files like this
			// but if they're the same then we can stamp our info
			// object with it
			x := beInt(buf)
			y := beInt(buf)
			units, err := buf.ReadByte()
			if err != nil {
				return img, 0, err
			}
			// only modify the info block if the user wants us to
			if x == y && readDPI {
				switch units {
				// if units is 1 then measurement is px/meter
				case 1:
					dpi = float64(x) / 39.3701 // inches per meter
				default:
					dpi = float64(x)
				}
			}
			_ = buf.Next(4)
		default:
			_ = buf.Next(n + 4)
		}

		loop = loop && n > 0
	}

	// palette
	if indexed, ok := cs.(model.ColorSpaceIndexed); ok {
		if len(pal) == 0 {
			return img, 0, fmt.Errorf("missing palette in PNG buffer")
		}
		indexed.Hival = uint8(len(pal)/3 - 1)
		if compressPalette {
			pal = sliceCompress(pal)
			indexed.Lookup = &model.ColorTableStream{
				Content: pal,
				Filter:  model.Filters{{Name: model.Flate}},
			}
		} else {
			indexed.Lookup = &model.ColorTableStream{
				Content: pal,
			}
		}
		img.ColorSpace = indexed
	} else {
		img.ColorSpace = cs
	}

	if len(trns) > 0 {
		mask := make(model.MaskColor, len(trns))
		for i, v := range trns {
			mask[i] = [2]int{v, v}
		}
		img.Mask = mask
	}

	var smask []byte
	if ct >= 4 {
		// Separate alpha and color channels
		data, err = sliceUncompress(data)
		if err != nil {
			return img, 0, err
		}
		var color, alpha bytes.Buffer
		if ct == 4 {
			// Gray image
			length := 2 * img.Width
			var pos, elPos int
			for i := 0; i < img.Height; i++ {
				pos = (1 + length) * i
				color.WriteByte(data[pos])
				alpha.WriteByte(data[pos])
				elPos = pos + 1
				for k := 0; k < img.Width; k++ {
					color.WriteByte(data[elPos])
					alpha.WriteByte(data[elPos+1])
					elPos += 2
				}
			}
		} else {
			// RGB image
			length := 4 * img.Width
			var pos, elPos int
			for i := 0; i < img.Height; i++ {
				pos = (1 + length) * i
				color.WriteByte(data[pos])
				alpha.WriteByte(data[pos])
				elPos = pos + 1
				for k := 0; k < img.Width; k++ {
					color.Write(data[elPos : elPos+3])
					alpha.WriteByte(data[elPos+3])
					elPos += 4
				}
			}
		}
		data = sliceCompress(color.Bytes())
		smask = sliceCompress(alpha.Bytes())
	}
	img.Content = data

	// 	Soft mask
	if len(smask) > 0 {
		smask := model.ImageSMask{
			Image: model.Image{
				Stream: model.Stream{
					Content: smask,
					Filter: model.Filters{
						{Name: model.Flate, DecodeParms: map[string]int{
							"Predictor": 15,
							"Colors":    1,
							"Columns":   img.Width,
						}}},
				},
				Width:            img.Width,
				Height:           img.Height,
				BitsPerComponent: 8,
			},
		}
		img.SMask = &smask
	}
	return
}
