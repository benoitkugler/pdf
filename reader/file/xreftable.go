package file

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/parser"
)

// xRefTable is the main access to PDF objects
type xRefTable struct {
	// object number -> entry
	objects map[parser.IndirectRef]*xrefEntry

	// object stream are special cases since we
	// don't wan't to process them for each object they contain
	objectStreams map[int]objectStream
}

func newXRefTable() xRefTable {
	return xRefTable{objects: make(map[parser.IndirectRef]*xrefEntry), objectStreams: make(map[int]objectStream)}
}

// populate object field of the xrefTable
func (ctx *context) processAllObjects() error {
	for on, entry := range ctx.xrefTable.objects {
		if entry.free {
			continue
		}

		_, err := ctx.resolveObjectNumber(on)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx *context) resolve(o parser.Object) (parser.Object, error) {
	ref, ok := o.(parser.IndirectRef)
	if !ok {
		return o, nil // return the direct object as it is
	}

	return ctx.resolveObjectNumber(ref)
}

func (ctx *context) resolveObjectNumber(objRef model.ObjIndirectRef) (parser.Object, error) {
	// 7.3.10
	// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
	// it shall be treated as a reference to the null object.
	entry, ok := ctx.xrefTable.objects[objRef]
	if !ok {
		return model.ObjNull{}, nil
	}

	if entry.object != nil { // already resolved
		return entry.object, nil
	}

	isCompressedObject := entry.streamObjectNumber != 0
	// Actually resolve the object. There are two cases:
	//	- the object is compressed inside an object stream
	// 	- the object is a regular object
	// Before recursing, start by assigning null as object,
	// so that malicious loops won't lead to infinite recursion
	entry.object = model.ObjNull{}

	if isCompressedObject {
		ob, err := ctx.processObjectStream(entry.streamObjectNumber)
		if err != nil {
			return nil, err
		}

		if entry.streamObjectIndex >= len(ob) {
			return nil, fmt.Errorf("invalid object index (%d >= %d)", entry.streamObjectIndex, len(ob))
		}

		entry.object = ob[entry.streamObjectIndex]
	} else {
		tk, err := ctx.tokenizerAt(entry.offset)
		if err != nil {
			return nil, fmt.Errorf("invalid offset in xref table (%d): %s", entry.offset, err)
		}

		_, _, err = parseObjectDeclaration(tk)
		if err != nil {
			return nil, fmt.Errorf("invalid object declaration (%v): %s", objRef, err)
		}

		entry.object, err = parser.NewParserFromTokenizer(tk).ParseObject()
		if err != nil {
			return nil, fmt.Errorf("invalid object content (%v): %s", objRef, err)
		}

		// stream object are dict with an additional content : lookup up for them
		nt, _ := tk.NextToken()
		if streamHeader, ok := entry.object.(model.ObjDict); nt.IsOther("stream") && ok {
			// before resolving, we need to save the current tokeniser position,
			// since it may be used during resolution
			streamPosition := entry.offset + int64(tk.StreamPosition())

			filters, err := parser.ParseFilters(streamHeader["Filter"], streamHeader["DecodeParms"], ctx.resolve)
			if err != nil {
				return nil, fmt.Errorf("invalid stream: %s", err)
			}

			lengthO, err := ctx.resolve(streamHeader["Length"])
			if err != nil {
				return nil, fmt.Errorf("invalid stream Length: %s", err)
			}
			length, ok := lengthO.(parser.Integer)
			if !ok {
				return nil, fmt.Errorf("invalid stream Length: expected integer, got %T", lengthO)
			}

			// we want the cryted not decoded content
			content, err := ctx.extractStreamContent(filters, streamPosition, int(length))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %v: %s", objRef, err)
			}

			entry.object = model.ObjStream{Args: streamHeader, Content: content}
		}
	}

	var err error
	if ctx.enc != nil && !isCompressedObject { // object inside streams object shall not be encrypted
		entry.object, err = ctx.enc.decryptObject(entry.object, objRef)
	}

	return entry.object, err
}

// xrefEntry is an object entry in the xref table
// it is created with reference information,
// and its Object field is populated when resolved.
type xrefEntry struct {
	object parser.Object // initialy nil

	free   bool // if true, won't be resolved
	offset int64

	// for object in object streams
	streamObjectNumber int // The object number of the object stream in which this object is stored.
	streamObjectIndex  int // The index of this object within the object stream.
}

// return the previous offset (0 if it does not exists)
func (ctx *context) parseXRefStream(offset int64) (int64, error) {
	// parse this object
	streamHeader, err := ctx.parseStreamDictAt(offset)
	if err != nil {
		return 0, err
	}

	streamOffset := streamHeader.contentOffset
	sd, decoded, err := ctx.xRefStreamDict(streamHeader.dict, streamOffset)
	if err != nil {
		return 0, err
	}

	err = ctx.trailer.parseTrailerInfo(streamHeader.dict)
	if err != nil {
		return 0, err
	}

	// Parse xRefStream and create xRefTable entries for embedded objects.
	err = ctx.extractXRefTableEntriesFromXRefStream(decoded, sd)
	if err != nil {
		return 0, err
	}

	// since xRef streams are not regular objects, we do not save them in the xref table
	// in particular, it avoids issue with decryption

	return sd.prev, nil
}

func (ctx *context) xRefStreamDict(d parser.Dict, streamOffset int64) (xrefStreamDict, []byte, error) {
	// The values of all entries shown in Table 17 shall be direct objects; indirect references shall not be
	// permitted. For arrays (the Index and W entries), all of their elements shall be direct objects as well. If the
	// stream is encoded, the Filter and DecodeParms entries in Table 5 shall also be direct objects.

	details, err := parseXRefStreamDict(d)
	if err != nil {
		return details, nil, err
	}

	filters, err := parser.ParseDirectFilters(d["Filter"], d["DecodeParms"])
	if err != nil {
		return details, nil, err
	}

	// we do not use decodeStreamContent since :
	// 1) The cross-reference stream shall not be encrypted and strings appearing in the cross-reference stream
	// dictionary shall not be encrypted. It shall not have a Filter entry that specifies a Crypt filter (see 7.4.10,
	// "Crypt Filter").
	// 2) there is no object number for xref stream
	content, err := ctx.extractStreamContent(filters, streamOffset, details.count()*details.entrySize())
	if err != nil {
		return details, nil, err
	}

	// Decode stream content:
	r, err := filters.DecodeReader(bytes.NewReader(content))
	if err != nil {
		return details, nil, err
	}
	decoded, err := ioutil.ReadAll(r)
	if err != nil {
		return details, nil, err
	}

	return details, decoded, nil
}

// bufToInt64 interprets the content of buf as an int64.
func bufToInt64(buf []byte) (i int64) {
	for _, b := range buf {
		i <<= 8
		i |= int64(b)
	}
	return i
}

// For each object embedded in this xRefStream create the corresponding xRef table entry.
func (ctx *context) extractXRefTableEntriesFromXRefStream(buf []byte, xrefDict xrefStreamDict) error {
	// Note:
	// A value of zero for an element in the W array indicates that the corresponding field shall not be present in the stream,
	// and the default value shall be used, if there is one.
	// If the first element is zero, the type field shall not be present, and shall default to type 1.

	xrefEntryLen, count := xrefDict.entrySize(), xrefDict.count()
	L := count * xrefEntryLen
	if len(buf) < L {
		return fmt.Errorf("extractXRefTableEntriesFromXRefStream: corrupted xrefstream (%d < %d)", len(buf), L)
	}

	// Sometimes there is an additional xref entry not accounted for by "Index".
	// We ignore such a entries and do not treat this as an error.
	buf = buf[:L]

	i1 := xrefDict.w[0]
	i2 := xrefDict.w[1]
	i3 := xrefDict.w[2]

	j := 0 // current index of object (0 <= j < count)
	for _, subsection := range xrefDict.index {
		firstObj, nb := subsection[0], subsection[1]
		for i := 0; i < nb; i++ {
			objectNumber := firstObj + i

			offsetEntry := j * xrefEntryLen
			c2 := bufToInt64(buf[offsetEntry+i1 : offsetEntry+i1+i2])
			c3 := bufToInt64(buf[offsetEntry+i1+i2 : offsetEntry+i1+i2+i3])

			var (
				xRefTableEntry xrefEntry
				generation     int
			)
			switch buf[offsetEntry] {
			case 0x00: // free object, ignore
				xRefTableEntry = xrefEntry{
					offset: c2,
					free:   true,
				}
				generation = int(c3)
			case 0x01: // in use object
				xRefTableEntry = xrefEntry{
					offset: c2,
				}
				generation = int(c3)
			case 0x02: // compressed object; generation always 0.
				xRefTableEntry = xrefEntry{
					streamObjectNumber: int(c2),
					streamObjectIndex:  int(c3),
				}
			}

			ref := model.ObjIndirectRef{ObjectNumber: objectNumber, GenerationNumber: generation}
			// skip already assigned
			if _, has := ctx.xrefTable.objects[ref]; !has {
				ctx.xrefTable.objects[ref] = &xRefTableEntry
			}
			j++
		}
	}

	return nil
}

type xrefStreamDict struct {
	index  [][2]int
	w      [3]int
	length int
	size   int
	prev   int64
}

// returns the number of entries, as described by the 'index'
func (x xrefStreamDict) count() int {
	total := 0
	for _, subsection := range x.index {
		total += subsection[1]
	}
	return total
}

func (x xrefStreamDict) entrySize() int {
	return x.w[0] + x.w[1] + x.w[2]
}

var (
	errXrefStreamCorruptIndex = errors.New("parseXRefStreamDict: corrupted Index entry")
	errXrefStreamCorruptW     = errors.New("parseXRefStreamDict: corrupted entry W: expecting array of 3 int")
)

// parseXRefStreamDict creates a XRefStreamDict out of a StreamDict.
func parseXRefStreamDict(dict parser.Dict) (xrefStreamDict, error) {
	var out xrefStreamDict

	out.prev, _ = offsetFromObject(dict["Prev"])

	length, ok := dict["Length"].(parser.Integer)
	if !ok {
		return out, errors.New("parseXRefStreamDict: \"Length\" not available")
	}
	out.length = int(length)

	size, ok := dict["Size"].(parser.Integer)
	if !ok {
		return out, errors.New("parseXRefStreamDict: \"Size\" not available")
	}
	out.size = int(size)

	//	Read optional parameter Index
	indArr, _ := dict["Index"].(parser.Array)
	if len(indArr) != 0 {
		if len(indArr)%2 > 1 {
			return out, errXrefStreamCorruptIndex
		}
		out.index = make([][2]int, len(indArr)/2)
		for i := range out.index {
			startObj, ok := indArr[i*2].(parser.Integer)
			if !ok {
				return out, errXrefStreamCorruptIndex
			}
			count, ok := indArr[i*2+1].(parser.Integer)
			if !ok {
				return out, errXrefStreamCorruptIndex
			}
			out.index = append(out.index, [2]int{int(startObj), int(count)})
		}
	} else {
		out.index = [][2]int{{0, out.size}}
	}

	// Read parameter W in order to decode the xref table.
	// array of integers representing the size of the fields in a single cross-reference entry.

	w, _ := dict["W"].(parser.Array) // validate array with 3 positive integers
	if len(w) < 3 {
		return out, errXrefStreamCorruptW
	}

	f := func(ok bool, i parser.Integer) bool {
		return !ok || i < 0
	}

	i1, ok := w[0].(parser.Integer)
	if f(ok, i1) {
		return out, errXrefStreamCorruptW
	}
	out.w[0] = int(i1)

	i2, ok := w[1].(parser.Integer)
	if f(ok, i2) {
		return out, errXrefStreamCorruptW
	}
	out.w[1] = int(i2)

	i3, ok := w[2].(parser.Integer)
	if f(ok, i3) {
		return out, errXrefStreamCorruptW
	}
	out.w[2] = int(i3)
	return out, nil
}
