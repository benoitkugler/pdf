package file

import (
	"fmt"

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
	for on := range ctx.xrefTable.objects {
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

	// Actually resolve the object. There are two cases:
	//	- the object is compressed inside an object stream
	// 	- the object is a regular object
	// Before recursing, start by assigning null as object,
	// so that malicious loops won't lead to infinite recursion
	entry.object = model.ObjNull{}

	if entry.streamObjectNumber != 0 {
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
			return nil, fmt.Errorf("invalid object (%d): %s", objRef.ObjectNumber, err)
		}

		entry.object, err = parser.NewParserFromTokenizer(tk).ParseObject()
		if err != nil {
			return nil, fmt.Errorf("invalid object (%d): %s", objRef.ObjectNumber, err)
		}

		// stream object are dict with an additional content : lookup up for them
		nt, _ := tk.NextToken()
		if streamHeader, ok := entry.object.(model.ObjDict); nt.IsOther("stream") && ok {
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
			content, err := ctx.extractStreamContent(filters, entry.offset+int64(tk.StreamPosition()), int(length))
			if err != nil {
				return nil, err
			}

			entry.object = model.ObjStream{Args: streamHeader, Content: content}
		}
	}

	var err error
	if ctx.enc != nil {
		entry.object, err = ctx.enc.decryptObject(entry.object, objRef)
	}

	return entry.object, err
}

// xrefEntry is an object entry in the xref table
// it is created with reference information,
// and its Object field is populated when resolved.
type xrefEntry struct {
	object parser.Object // initialy nil

	offset int64

	// for object in object streams
	streamObjectNumber int // The object number of the object stream in which this object is stored.
	streamObjectIndex  int // The index of this object within the object stream.
}
