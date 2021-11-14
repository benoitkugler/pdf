package filters

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
)

// the post processing functions are copied from pdfcpu/filters

type SkipperFlate struct{}

// Skip implements Skipper for a Flate filter.
func (f SkipperFlate) Skip(encoded io.Reader) (int, error) {
	r := newCountReader(encoded)
	rc, err := zlib.NewReader(r)
	if err != nil {
		return 0, err
	}
	_, err = ioutil.ReadAll(rc)
	if err != nil {
		return 0, err
	}
	err = rc.Close()
	return r.totalRead, err
}

func flateDecoder(params flateDecodeParams, src io.Reader) (io.Reader, error) {
	rc, err := zlib.NewReader(src)
	if err != nil {
		return nil, err
	}

	// Optional decode parameters need postprocessing.
	return params.decodePostProcess(rc)
}

// post process params
type flateDecodeParams struct {
	predictor int

	colors  int
	bpc     int
	columns int
}

func processFlateParams(params map[string]int) (out flateDecodeParams, err error) {
	predictor := params["Predictor"]
	switch predictor {
	case 0, 1, 2, 10, 11, 12, 13, 14, 15:
	default:
		return out, fmt.Errorf("filter FlateDecode: unexpected Predictor: %d", predictor)
	}

	// Colors, int
	// The number of interleaved colour components per sample.
	// Valid values are 1 to 4 (PDF 1.0) and 1 or greater (PDF 1.3). Default value: 1.
	// Used by PredictorTIFF only.
	colors, found := params["Colors"]
	if !found {
		colors = 1
	} else if colors == 0 {
		return out, fmt.Errorf("filter FlateDecode: Colors must be > 0, got %d", colors)
	}

	// BitsPerComponent, int
	// The number of bits used to represent each colour component in a sample.
	// Valid values are 1, 2, 4, 8, and (PDF 1.5) 16. Default value: 8.
	// Used by PredictorTIFF only.
	bpc, found := params["BitsPerComponent"]
	if !found {
		bpc = 8
	} else {
		switch bpc {
		case 1, 2, 4, 8, 16:
		default:
			return out, fmt.Errorf("filter FlateDecode: unexpected BitsPerComponent: %d", bpc)
		}
	}

	// Columns, int
	// The number of samples in each row. Default value: 1.
	columns, found := params["Columns"]
	if !found {
		columns = 1
	}

	return flateDecodeParams{predictor: predictor, colors: colors, bpc: bpc, columns: columns}, nil
}

func (f flateDecodeParams) rowSize() int {
	return f.bpc * f.colors * f.columns / 8
}

// decodePostProcess
func (f flateDecodeParams) decodePostProcess(r io.Reader) (io.Reader, error) {
	if f.predictor == 0 || f.predictor == 1 { // nothing to do
		return r, nil
	}

	bytesPerPixel := (f.bpc*f.colors + 7) / 8

	rowSize := f.rowSize()
	if f.predictor != 2 {
		// PNG prediction uses a row filter byte prefixing the pixelbytes of a row.
		rowSize++
	}

	// cr and pr are the bytes for the current and previous row.
	cr := make([]byte, rowSize)
	pr := make([]byte, rowSize)

	// Output buffer
	var out []byte

	for {

		// Read decompressed bytes for one pixel row.
		_, err := io.ReadFull(r, cr)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			// else : eof
			break
		}

		d, err := processRow(pr, cr, f.predictor, f.colors, bytesPerPixel)
		if err != nil {
			return nil, err
		}

		out = append(out, d...)

		// Swap byte slices.
		pr, cr = cr, pr
	}

	if len(out)%f.rowSize() != 0 {
		return nil, fmt.Errorf("filter FlateDecode: postprocessing failed (%d %d)", len(out), f.rowSize())
	}

	return bytes.NewReader(out), nil
}

func processRow(pr, cr []byte, p, colors, bytesPerPixel int) ([]byte, error) {
	if p == 2 { // TIFF
		return applyHorDiff(cr, colors)
	}

	// Apply the filter.
	cdat := cr[1:]
	pdat := pr[1:]

	// Get row filter from 1st byte
	f := int(cr[0])

	// The value of Predictor supplied by the decoding filter need not match the value
	// used when the data was encoded if they are both greater than or equal to 10.

	switch f {
	case 0:
		// No operation.
	case 1:
		for i := bytesPerPixel; i < len(cdat); i++ {
			cdat[i] += cdat[i-bytesPerPixel]
		}
	case 2:
		for i, p := range pdat {
			cdat[i] += p
		}
	case 3:
		// The average of the two neighboring pixels (left and above).
		// Raw(x) - floor((Raw(x-bpp)+Prior(x))/2)
		for i := 0; i < bytesPerPixel; i++ {
			cdat[i] += pdat[i] / 2
		}
		for i := bytesPerPixel; i < len(cdat); i++ {
			cdat[i] += uint8((int(cdat[i-bytesPerPixel]) + int(pdat[i])) / 2)
		}
	case 4:
		filterPaeth(cdat, pdat, bytesPerPixel)
	}

	return cdat, nil
}

func applyHorDiff(row []byte, colors int) ([]byte, error) {
	// This works for 8 bits per color only.
	for i := 1; i < len(row)/colors; i++ {
		for j := 0; j < colors; j++ {
			row[i*colors+j] += row[(i-1)*colors+j]
		}
	}
	return row, nil
}

func abs(x int32) int32 {
	const intSize = 32

	// m := -1 if x < 0. m := 0 otherwise.
	m := x >> (intSize - 1)

	// In two's complement representation, the negative number
	// of any number (except the smallest one) can be computed
	// by flipping all the bits and add 1. This is faster than code with a branch.
	// See Hacker's Delight, section 2-4.
	return (x ^ m) - m
}

// filterPaeth applies the Paeth filter to the cdat slice.
// cdat is the current row's data, pdat is the previous row's data.
func filterPaeth(cdat, pdat []byte, bytesPerPixel int) {
	var a, b, c, pa, pb, pc int32
	for i := 0; i < bytesPerPixel; i++ {
		a, c = 0, 0
		for j := i; j < len(cdat); j += bytesPerPixel {
			b = int32(pdat[j])
			pa = b - c
			pb = a - c
			pc = abs(pa + pb)
			pa = abs(pa)
			pb = abs(pb)
			if pa <= pb && pa <= pc {
				// No-op.
			} else if pb <= pc {
				a = b
			} else {
				a = c
			}
			a += int32(cdat[j])
			a &= 0xff
			cdat[j] = uint8(a)
			c = b
		}
	}
}
