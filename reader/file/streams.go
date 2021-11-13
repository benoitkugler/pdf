package file

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/parser"
)

type streamDictHeader struct {
	dict          parser.Dict
	ref           model.ObjIndirectRef
	contentOffset int64 // start of the actual content (from the start of the file)
}

func (ctx *context) parseStreamDictAt(offset int64) (out streamDictHeader, err error) {
	tk, err := ctx.tokenizerAt(offset)
	if err != nil {
		return out, err
	}

	out.ref.ObjectNumber, out.ref.GenerationNumber, err = parseObjectDeclaration(tk)
	if err != nil {
		return out, err
	}

	// parse this object
	pr := parser.NewParserFromTokenizer(tk)
	o, err := pr.ParseObject()
	if err != nil {
		return out, fmt.Errorf("parseStreamDict: no object: %s", err)
	}

	d, ok := o.(parser.Dict)
	if !ok {
		return out, fmt.Errorf("parseStreamDict: expected dict, got %T", o)
	}

	streamStart, err := tk.NextToken()
	if err != nil {
		return out, err
	}
	if !streamStart.IsOther("stream") {
		return out, fmt.Errorf("parseStreamDict: unexpected token %s", streamStart)
	}

	out.dict = d
	out.contentOffset = offset + int64(tk.StreamPosition())
	return out, nil
}

// read a stream starting at `offset`
// Although, according to the SPEC, `expectedLength` should be sufficient, in practice it is not.
// We apply the following heuristics :
//	- if the content is not filtered, or is encrypted, we can't use the format to find the end
// 	of the stream. Thus, we either use `expectedLength` or, if it is 0, look for "endstream"
// 	- else we use the EOD of the filter (which if the most reliable method)
func (ctx *context) extractStreamContent(filters model.Filters, offset int64, expectedLength int) ([]byte, error) {
	if ctx.enc != nil || len(filters) == 0 {
		// we can't use information provided by a potential filter
		return ctx.readStreamFromLength(offset, expectedLength)
	}

	// rely on EOD
	out, err := ctx.readStreamWithEOD(filters[0], offset)
	if err != nil {
		// if the filtered content is badly formatted, try again
		// with the heuristic approaches
		log.Printf("reading PDF filtered stream : %s. trying to fix\n", err)

		return ctx.readStreamFromLength(offset, expectedLength)
	}

	return out, nil
}

// extract, decrypt, and decode a stream at `offset`
// ref if used for decryption
func (ctx *context) decodeStreamContent(ref model.ObjIndirectRef, filters model.Filters, offset int64, expectedLengthPlain int) (content []byte, err error) {
	content, err = ctx.extractStreamContent(filters, offset, expectedLengthPlain)
	if err != nil {
		return nil, fmt.Errorf("invalid stream content: %s", err)
	}

	if ctx.enc != nil {
		// Special case if the "Identity" crypt filter is used: we do not need to decrypt.
		if len(filters) == 1 && filters[0].Name == "Crypt" {
		} else {
			content, err = ctx.enc.decryptStream(content, ref)
			if err != nil {
				return nil, fmt.Errorf("invalid stream content: %s", err)
			}
		}
	}

	// Decode stream content:
	r, err := filters.DecodeReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("invalid stream content: %s", err)
	}
	return ioutil.ReadAll(r)
}

// readStreamFromLength try to locate the end of the stream using `expectedLength`,
// which is not always realiable
func (ctx *context) readStreamFromLength(offset int64, expectedLength int) ([]byte, error) {
	if expectedLength == 0 || expectedLength > int(ctx.fileSize) {
		// corrupted length
		return ctx.readStreamBlindly(offset)
	}

	return ctx.readStreamMaxLength(offset, expectedLength)
}

func (ctx *context) readStreamBlindly(offset int64) ([]byte, error) {
	// corrupted length => apply a weak heuristic :
	// buffer content until we found "endstream"
	_, err := ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	var (
		eod   = []byte("endstream")
		buf   [1024]byte
		total []byte
	)
	for {
		n, err := ctx.rs.Read(buf[:])
		if err == nil || err == io.EOF {
			total = append(total, buf[:n]...)
			// look for endstream
			searchStart := len(total) - n - len(eod)
			if searchStart < 0 {
				searchStart = 0
			}
			if index := bytes.Index(total[searchStart:], eod); index != -1 {
				total = total[:searchStart+index]
				break
			} else {
				if err == io.EOF {
					return nil, fmt.Errorf("invalid stream: EOF")
				} // else read another chunk
			}
		}
	}

	total = bytes.TrimRight(total, "\n\r")
	return total, nil
}

// try to read `maxLength`. If it fails, look backward to find "endstream"
// which may help with corrupted lengths
func (ctx *context) readStreamMaxLength(offset int64, maxLength int) ([]byte, error) {
	_, err := ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, maxLength) // maxLength has been check by the caller
	_, err = io.ReadFull(ctx.rs, buf)
	if err == io.ErrUnexpectedEOF {
		// Weak heuristic to detect the actual end of this stream
		// once we have reached EOF due to incorrect streamLength.
		eob := bytes.Index(buf, []byte("endstream"))
		if eob < 0 { // not found, actually error
			return nil, err
		}
		return buf[:eob], nil
	} else if err != nil {
		return nil, err
	}

	return buf, nil
}

// read starting from `offset` until the EOD marker expected by `filter` is reached
func (ctx *context) readStreamWithEOD(filter model.Filter, offset int64) ([]byte, error) {
	skipper, err := parser.SkipperFromFilter(filter)
	if err != nil {
		return nil, err
	}
	_, err = ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("invalid stream offset %d: %s", offset, err)
	}
	trueLength, err := skipper.Skip(ctx.rs)
	if err != nil {
		return nil, fmt.Errorf("failed to locate stream end: %s (filter: %s)", err, filter.Name)
	}
	return ctx.readAt(trueLength, offset)
}
