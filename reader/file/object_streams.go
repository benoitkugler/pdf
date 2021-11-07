package file

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/benoitkugler/pdf/reader/parser"
)

// parsed version of an object stream
type objectStream []parser.Object // with length N

// check the cache and process the given object stream number
func (ctx *context) processObjectStream(on int) (objectStream, error) {
	if os, ok := ctx.xrefTable.objectStreams[on]; ok {
		return os, nil
	}
	// process the object stream

	entry, ok := ctx.xrefTable.objects[on]
	if !ok {
		return nil, fmt.Errorf("missing object stream for reference %d", on)
	}

	streamHeader, err := ctx.parseStreamDictAt(entry.offset)
	if err != nil {
		return nil, fmt.Errorf("invalid stream at %d; %s", entry.offset, err)
	}

	filters, err := parser.ParseFilters(streamHeader.dict["Filter"], streamHeader.dict["DecodeParms"], func(o parser.Object) (parser.Object, error) {
		// TODO: actually resolve using xref
		return o, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid object stream: %s", err)
	}

	lengthO, err := ctx.resolve(streamHeader.dict["Length"])
	if err != nil {
		return nil, fmt.Errorf("invalid object stream Length: %s", err)
	}
	length, ok := lengthO.(parser.Integer)
	if !ok {
		return nil, fmt.Errorf("invalid object stream Length: expected integer, got %T", lengthO)
	}

	// TODO: handle encryption
	decoded, err := ctx.decodeStreamContent(filters, streamHeader.contentOffset, int(length))
	if err != nil {
		return nil, fmt.Errorf("invalid object stream: %s", err)
	}

	firstObjectOffset, ok := streamHeader.dict["First"].(parser.Integer)
	if !ok {
		return nil, fmt.Errorf("invalid object stream First: expected integer, got %T", streamHeader.dict["First"])
	}
	if int(firstObjectOffset) > len(decoded) {
		return nil, fmt.Errorf("out of bounds object stream First: %d > %d", firstObjectOffset, len(decoded))
	}
	prolog := decoded[:firstObjectOffset]

	// N pairs of integers separated by white space, where the first integer in each pair shall represent the object
	// number of a compressed object and the second integer shall represent the byte offset in the decoded
	// stream of that object, relative to the first object stored in the object stream, the value of the stream's first
	// entry. The offsets shall be in increasing order.
	// Note: The separator used in the prolog shall be white space but some PDF writers use 0x00.
	prolog = bytes.ReplaceAll(prolog, []byte{0x00}, []byte{0x20})
	fields := bytes.Fields(prolog)
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("odd number of fields (%d) in object stream prolog", len(fields))
	}

	offsets := make([]int, len(fields)/2)
	for i := range offsets {
		offsets[i], err = strconv.Atoi(string(fields[2*i+1]))
		if err != nil {
			return nil, fmt.Errorf("invalid object offset in object stream: %v", fields[2*i+1])
		}
		offsets[i] += int(firstObjectOffset)
		if offsets[i] > len(decoded) {
			return nil, fmt.Errorf("invalid object offset in object stream: %d", offsets[i])
		}
	}

	objects := make(objectStream, len(offsets))
	for i := range objects {
		start, end := offsets[i], len(decoded)
		if i+1 < len(offsets) {
			end = offsets[i+1]
		}

		objects[i], err = parser.ParseObject(decoded[start:end])
		if err != nil {
			return nil, fmt.Errorf("invalid object in object stream: %s", err)
		}
	}

	if _, has := streamHeader.dict["Extents"]; has {
		return nil, fmt.Errorf("unsupported Extents in object stream")
	}

	// cache it
	ctx.xrefTable.objectStreams[on] = objects
	return objects, nil
}
