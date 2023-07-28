package contentstream

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/image/tiff"
)

// supports importing common image type
// we follow the logic from gofpdf

// RenderingDims defines the size of an image
// in the page. PDF image objects don't contain such
// information, which are added via a tranformation matrix
// associated to the image.
type RenderingDims struct {
	Width, Height RenderingSize // See EffectiveSize()
}

// EffectiveSize performs automatic width and height calculation.
//
// Only the width and height (in terms of columns and rows) of `img` are used.
//
// If both `Width` and `Height` are nil, the image is rendered at 96 dpi.
// If either `Width` or `Height` is nil, it will be
// calculated from the other dimension so that the aspect ratio is maintained.
// Otherwise, `Width` or `Height` may be a dpi or a length.
func (r RenderingDims) EffectiveSize(img *model.XObjectImage) (width Fl, height Fl) {
	intrasecWidth, intrasecHeight := img.Width, img.Height
	if r.Width == nil && r.Height == nil { // Put image at 96 dpi
		r.Width = RenderingDPI(96)
		r.Height = RenderingDPI(96)
	}
	// resolve the non nil values
	if r.Width != nil {
		width = r.Width.effectiveLength(intrasecWidth)
	}
	if r.Height != nil {
		height = r.Height.effectiveLength(intrasecHeight)
	}
	// use ratio for nil values: the first condition
	// ensure that not both width and height are zero
	if r.Width == nil {
		width = height * Fl(intrasecWidth) / Fl(intrasecHeight)
	}
	if r.Height == nil {
		height = width * Fl(intrasecHeight) / Fl(intrasecWidth)
	}
	return width, height
}

// RenderingSize is either RenderingDPI or RenderingLength.
type RenderingSize interface {
	effectiveLength(intrasec int) Fl
}

// RenderingDPI specifies a length in DPI
type RenderingDPI Fl

func (dpi RenderingDPI) effectiveLength(intrasec int) Fl {
	return Fl(intrasec) * 72.0 / Fl(dpi)
}

// RenderingLength specifies a length in user space units.
type RenderingLength Fl

func (l RenderingLength) effectiveLength(int) Fl { return Fl(l) }

// ParseImageFile read the image type from the file extension.
// See `ParseImage` for more details.
func ParseImageFile(filename string) (*model.XObjectImage, Fl, error) {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	f, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	return ParseImage(f, mimeType)
}

// ParseImage supports importing JPEG, PNG, GIFF and TIFF images,
// according to the given MIME type.
// A dpi is returned: it's a default value (72) for JPG/JPEG images,
// and the one found in the image for PNG and GIFF.
func ParseImage(r io.Reader, mimeType string) (*model.XObjectImage, Fl, error) {
	switch mimeType {
	case "image/jpeg":
		out, err := parseJPG(r)
		return out, 72, err
	case "image/png":
		return parsePNG(r)
	case "image/gif":
		return parseGIF(r)
	case "image/tiff":
		return parseTIFF(r)
	default:
		return nil, 0, fmt.Errorf("unsupported image format: %s", mimeType)
	}
}

// parseJPG extracts info from io.Reader with JPEG data
func parseJPG(r io.Reader) (out *model.XObjectImage, err error) {
	out = new(model.XObjectImage)
	out.Content, err = io.ReadAll(r)
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

// parseGIF imports the first image from a GIF file, via PNG conversion
func parseGIF(r io.Reader) (*model.XObjectImage, Fl, error) {
	img, err := gif.Decode(r)
	if err != nil {
		return nil, 0, err
	}
	pngBuf := new(bytes.Buffer)
	err = png.Encode(pngBuf, img)
	if err != nil {
		return nil, 0, err
	}
	return parsePNG(pngBuf)
}

// parseTIFF import a TIFF image, via PNG conversion
// TODO: is it better to use LZW filter directly ? How ?
func parseTIFF(r io.Reader) (*model.XObjectImage, Fl, error) {
	img, err := tiff.Decode(r)
	if err != nil {
		return nil, 0, err
	}
	pngBuf := new(bytes.Buffer)
	err = png.Encode(pngBuf, img)
	if err != nil {
		return nil, 0, err
	}
	return parsePNG(pngBuf)
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

	return io.ReadAll(r)
}

// the potential compression of the palette (for indexed color space)
// is not done here
// we won't use the standard library because of the tRNS information
// which is not exposed (and seems to modify the value of the pixels color ?)
// so we have to write a custom png parser...
func parsePNG(r io.Reader) (img *model.XObjectImage, dpi model.Fl, err error) {
	img = new(model.XObjectImage)
	dpi = 72 // default value

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
			if x == y {
				switch units {
				// if units is 1 then measurement is px/meter
				case 1:
					dpi = model.Fl(x) / 39.3701 // inches per meter
				default:
					dpi = model.Fl(x)
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
		if len(pal) >= 100 {
			indexed.Lookup = &model.ColorTableStream{
				Content: pal,
			}
		} else {
			indexed.Lookup = model.ColorTableBytes(pal)
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
	img.Stream = model.Stream{
		Content: data,
		Filter: model.Filters{{Name: model.Flate, DecodeParms: map[string]int{
			"Predictor":        15,
			"Colors":           colorVal,
			"Columns":          img.Width,
			"BitsPerComponent": int(img.BitsPerComponent),
		}}},
	}

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
						}},
					},
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
