package parser

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/parser/filters"
)

var errBIExpressionCorrupt = errors.New("corrupt BI (inline image) expression")

func (pr *Parser) parseInlineImage(res model.ResourcesColorSpace) (contentstream.OpBeginImage, error) {
	var (
		out                   contentstream.OpBeginImage
		filters, decodeParams Object // parsing delayed
	)
	if err := assertLength(pr.opsStack, 0); err != nil {
		return out, err
	}
	// process the image characteristics
	for {
		obj, err := pr.ParseObject()
		if err != nil {
			return out, err
		}
		if obj == Command("ID") {
			// done with the characteristics;
			err = pr.parseImageData(&out, filters, decodeParams, res)
			// EI is consumed in parseImageData
			return out, err
		} else {
			// we expect a name and a value
			name, ok := obj.(Name)
			if !ok {
				return out, errBIExpressionCorrupt
			}
			value, err := pr.ParseObject()
			if err != nil {
				return out, errBIExpressionCorrupt
			}
			o1, o2, err := parseOneImgField(name, value, &out)
			if err != nil {
				return out, err
			}
			if o1 != nil { // only true for the Filter key
				filters = o1
			}
			if o2 != nil { // only true for the DecodeParms key
				decodeParams = o2
			}
		}
	}
}

// since DecodeParms and Filter are a same object in the model
// we have to return them separately
func parseOneImgField(name Name, value Object, img *contentstream.OpBeginImage) (filters, decodeParams Object, err error) {
	switch name {
	case "BitsPerComponent", "BPC":
		i, ok := value.(Integer)
		if !ok {
			return nil, nil, errBIExpressionCorrupt
		}
		img.Image.BitsPerComponent = uint8(i)
	case "Width", "W":
		i, ok := value.(Integer)
		if !ok {
			return nil, nil, errBIExpressionCorrupt
		}
		img.Image.Width = int(i)
	case "Height", "H":
		i, ok := value.(Integer)
		if !ok {
			return nil, nil, errBIExpressionCorrupt
		}
		img.Image.Height = int(i)
	case "Decode", "D":
		arr, ok := value.(Array)
		if !ok {
			return nil, nil, errBIExpressionCorrupt
		}
		img.Image.Decode, err = processPoints(arr)
	case "ImageMask", "IM":
		b, ok := value.(Bool)
		if !ok {
			return nil, nil, errBIExpressionCorrupt
		}
		img.Image.ImageMask = bool(b)
	case "Intent":
		in, ok := value.(Name)
		if !ok {
			return nil, nil, errBIExpressionCorrupt
		}
		img.Image.Intent = model.ObjName(in)
	case "Interpolate", "I":
		b, ok := value.(Bool)
		if !ok {
			return nil, nil, errBIExpressionCorrupt
		}
		img.Image.Interpolate = bool(b)
	case "ColorSpace", "CS":
		switch value := value.(type) {
		case Name:
			img.ColorSpace = contentstream.ImageColorSpaceName{ColorSpaceName: model.ColorSpaceName(value)}
		case Array:
			img.ColorSpace, err = processIndexedCS(value)
		}
	case "Filter", "F": // parsing is delayed
		return value, nil, nil
	case "DecodeParms", "DP": // parsing is delayed
		return nil, value, nil
	}

	return nil, nil, err
}

func processPoints(arr Array) ([][2]Fl, error) {
	if len(arr)%2 != 0 {
		return nil, fmt.Errorf("expected even length for array, got %v", arr)
	}
	out := make([][2]Fl, len(arr)/2)
	for i := range out {
		a, err := assertNumber(arr[2*i])
		if err != nil {
			return nil, err
		}
		b, err := assertNumber(arr[2*i+1])
		if err != nil {
			return nil, err
		}
		out[i] = [2]Fl{a, b}
	}
	return out, nil
}

func processIndexedCS(arr Array) (contentstream.ImageColorSpaceIndexed, error) {
	var out contentstream.ImageColorSpaceIndexed
	if len(arr) != 4 {
		return out, errBIExpressionCorrupt
	}
	b, ok := arr[1].(Name)
	if !ok {
		return out, errBIExpressionCorrupt
	}
	out.Base = model.ColorSpaceName(b)
	h, ok := arr[2].(Integer)
	if !ok {
		return out, errBIExpressionCorrupt
	}
	out.Hival = uint8(h)
	switch table := arr[3].(type) {
	case StringLiteral:
		out.Lookup = model.ColorTableBytes(table)
	case HexLiteral:
		out.Lookup = model.ColorTableBytes(table)
	default:
		return out, errBIExpressionCorrupt
	}
	return out, nil
}

var errFiltersCorrupted = errors.New("corrupted filter expression")

// ParseDirectFiltersis the same as ParseFilters, but for direct objects.
// It is the case in image inline parameters and xRefStream dicts.
func ParseDirectFilters(filters, decodeParams Object) (model.Filters, error) {
	return ParseFilters(filters, decodeParams, func(o Object) (Object, error) { return o, nil })
}

// ParseFilters process the given filters and their (optionnal) parameters.
// `resolver` is called to resolve the potential indirect objects
// An empty list may be returned if the filters are nil.
func ParseFilters(filters, decodeParams Object, resolver func(Object) (Object, error)) (model.Filters, error) {
	var err error
	filters, err = resolver(filters)
	if err != nil {
		return nil, err
	}
	if filters == nil {
		return nil, nil
	}

	if filterName, isName := filters.(Name); isName {
		filters = Array{filterName}
	}
	ar, ok := filters.(Array)
	if !ok {
		return nil, errFiltersCorrupted
	}
	var out model.Filters
	for _, name := range ar {
		name, err = resolver(name)
		if err != nil {
			return nil, err
		}

		if filterName, isName := name.(Name); isName {
			out = append(out, model.Filter{Name: model.ObjName(filterName)})
		} else {
			return nil, errFiltersCorrupted
		}
	}

	decodeParams, err = resolver(decodeParams)
	if err != nil {
		return nil, err
	}

	switch decodeParams := decodeParams.(type) {
	case Array: // one dict param per filter
		if len(decodeParams) != len(out) {
			return nil, fmt.Errorf("unexpected length for DecodeParms array: %d", len(decodeParams))
		}
		for i, parms := range decodeParams {
			parms, err = resolver(parms)
			if err != nil {
				return nil, err
			}
			out[i].DecodeParms = processOneDecodeParms(parms)
		}
	case Dict: // one filter and one dict param
		if len(out) != 1 {
			return nil, fmt.Errorf("DecodeParms as dict only supported for one filter, got %d", len(out))
		}
		out[0].DecodeParms = processOneDecodeParms(decodeParams)
	case nil: // OK
	default:
		return nil, errFiltersCorrupted
	}

	return out, nil
}

func processOneDecodeParms(parms Object) map[string]int {
	parmsDict, _ := parms.(Dict)
	parmsModel := make(map[string]int)
	for paramName, paramVal := range parmsDict {
		var intVal int
		switch val := paramVal.(type) {
		case Bool:
			if val {
				intVal = 1
			} else {
				intVal = 0
			}
		case Integer:
			intVal = int(val)
		case Float:
			intVal = int(val)
		default:
			continue
		}
		parmsModel[string(paramName)] = intVal
	}
	return parmsModel
}

// read the inline data, store its content in img, and skip EI command
func (pr *Parser) parseImageData(img *contentstream.OpBeginImage, fils, decodeParams Object, res model.ResourcesColorSpace) error {
	var err error
	// first we check update the filter list
	img.Image.Filter, err = ParseDirectFilters(fils, decodeParams)
	if err != nil {
		return err
	}

	// to read the binary data, there are 2 cases
	// 	- if the data is not filtered, we use the image metadata to deduce the length
	//	- if the data is filtered, we have to rely on the filter format End Of Data marker

	if len(img.Image.Filter) == 0 {
		bits, comps, err := img.Metrics(res)
		if err != nil {
			return err
		}
		n := img.Image.Height * ((img.Image.Width*comps*bits + 7) / 8)

		img.Image.Content = pr.tokens.SkipBytes(n + 1) // with space after ID
	} else {
		pr.tokens.SkipBytes(1) // with space after ID
		input := pr.tokens.Bytes()

		// we only apply the first filter
		fi := img.Image.Stream.Filter[0]
		skipper, err := filters.SkipperFromFilter(string(fi.Name), fi.DecodeParms)
		if err != nil {
			return err
		}
		encodedLength, err := skipper.Skip(bytes.NewReader(input))
		if err != nil {
			return fmt.Errorf("can't read compressed inline image data: %s", err)
		}
		// we return the compressed version ...
		img.Image.Content = input[0:encodedLength]
		// ... and move the tokenizer
		pr.tokens.SkipBytes(encodedLength)
	}
	o, err := pr.ParseObject() // EI
	if err != nil {
		return err
	}
	if o != Command("EI") {
		return fmt.Errorf("expected end of inline image, got %v", o)
	}
	return nil
}
