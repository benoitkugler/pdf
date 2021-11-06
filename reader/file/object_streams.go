package file

import (
	"fmt"
	"io"

	"github.com/benoitkugler/pdf/reader/parser"
	tok "github.com/benoitkugler/pstokenizer"
)

func (ctx *context) processObjectStreams() error {
	// select the object streams to process
	objectStreamNumbers := map[int]bool{}
	for _, entry := range ctx.xrefTable {
		if entry.streamObjectNumber != 0 {
			objectStreamNumbers[entry.streamObjectNumber] = true
		}
	}

	fmt.Println("processing object streams", objectStreamNumbers)

	for on := range objectStreamNumbers {
		entry, ok := ctx.xrefTable[on]
		if !ok {
			return fmt.Errorf("missing object stream for reference %d", on)
		}

		_, err := ctx.rs.Seek(entry.offset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("invalid offset in xref table (%d); %s", entry.offset, err)
		}

		tk := tok.NewTokenizerFromReader(ctx.rs)
		// parse this object
		streamHeader, err := parseStreamDict(tk)
		if err != nil {
			return fmt.Errorf("invalid object stream: %s", err)
		}

		fmt.Println(streamHeader)
		filters, err := parser.ParseFilters(streamHeader.dict["Filter"], streamHeader.dict["DecodeParms"], func(o parser.Object) (parser.Object, error) {
			// TODO: actually resolve using xref
			return o, nil
		})
		if err != nil {
			return fmt.Errorf("invalid object stream: %s", err)
		}
		fmt.Println(filters)
	}

	return nil
}
