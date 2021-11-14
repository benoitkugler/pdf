package file

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/parser"
	tok "github.com/benoitkugler/pstokenizer"
)

// freeHeadGeneration is the predefined generation number for the head of the free list.
const freeHeadGeneration = 65535

var errCorruptHeader = errors.New("headerVersion: corrupt pdf stream - no header version available")

// context represents an environment for processing PDF files.
type context struct {
	rs       io.ReadSeeker
	fileSize int64

	Configuration

	// PDF Version
	HeaderVersion string // The PDF version the source is claiming to us as per its header.
	xrefTable     xRefTableContext
	trailer       trailer

	// AdditionalStreams (array of IndirectRef) is not described in the spec,
	// but may be found in the trailer :e.g., Oasis "Open Doc"
	additionalStreams parser.Array

	tok tok.Tokenizer // buffer

	enc *encrypt // nil for plain document
}

func newContext(rs io.ReadSeeker, conf *Configuration) (*context, error) {
	if conf == nil {
		conf = NewDefaultConfiguration()
	}

	rdCtx := &context{
		rs:            rs,
		Configuration: *conf,
		xrefTable:     newXRefTable(),
	}

	fileSize, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	rdCtx.fileSize = fileSize

	return rdCtx, nil
}

type trailer struct {
	encrypt parser.Object // indirect ref or dict

	root *parser.IndirectRef
	info *parser.IndirectRef // optional
	id   parser.Array        // required in encrypted docs
	size int                 // Object count from PDF trailer dict.
}

// allocate a slice with length `size` and read at `offset`
// into it
func (ctx *context) readAt(size int, offset int64) ([]byte, error) {
	_, err := ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("invalid offset %d: %s", offset, err)
	}
	p := make([]byte, size)
	_, err = io.ReadFull(ctx.rs, p)
	if err != nil {
		return nil, fmt.Errorf("can't read %d bytes at offset %d: %s", size, offset, err)
	}
	return p, nil
}

// position the tokenizer at `offset`
func (ctx *context) tokenizerAt(offset int64) (*tok.Tokenizer, error) {
	_, err := ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	ctx.tok.ResetFromReader(ctx.rs)

	return &ctx.tok, nil
}

// reset the tokenizer with the given `data`
func (ctx *context) tokenizerBytes(data []byte) *tok.Tokenizer {
	ctx.tok.Reset(data)

	return &ctx.tok
}

// look for `pattern`, starting from `skip` bytes from the end of the file (skip >= 0),
// and buffering until `pattern` is reached.
// returns the accumulated buffer, starting right after `pattern`
func (ctx *context) findStringFromFileEnd(skip int64, pattern string) ([]byte, error) {
	rs := ctx.rs

	var (
		workBuf []byte
		bufSize int64 = 512
	)

	// guard for very small files
	if ctx.fileSize < bufSize {
		bufSize = ctx.fileSize
	}

	curBuf := make([]byte, bufSize)
	for i := 1; ; i++ {
		offsetToReach := -int64(i)*bufSize - skip // negative
		if ctx.fileSize+offsetToReach < 0 {
			return nil, fmt.Errorf("didn't found %s (try %d)", pattern, i)
		}

		_, err := rs.Seek(offsetToReach, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("can't find %s: %s", pattern, err)
		}

		_, err = rs.Read(curBuf)
		if err != nil {
			return nil, fmt.Errorf("can't find %s: %s", pattern, err)
		}

		workBuf = append(curBuf, workBuf...)

		j := bytes.LastIndex(workBuf, []byte(pattern))
		if j != -1 { // found it !
			return workBuf[j+len(pattern):], nil
		}
	}
}

// Get the file offset of the last XRefSection.
// Go to end of file and search backwards for the first occurrence of startxref {offset} %%EOF
// xref at 114172
func (ctx *context) offsetLastXRefSection(skip int64) (int64, error) {
	p, err := ctx.findStringFromFileEnd(skip, "startxref")
	if err != nil {
		return 0, err
	}

	posEOF := bytes.Index(p, []byte("%%EOF"))
	if posEOF == -1 {
		return 0, errors.New("no matching %%EOF for startxref")
	}

	p = p[:posEOF]
	targetOffset, err := strconv.ParseInt(string(bytes.TrimSpace(p)), 10, 64)
	if err != nil || targetOffset >= ctx.fileSize {
		return 0, errors.New("corrupted last xref section")
	}

	return targetOffset, nil
}

// Get version from first line of file.
// Beginning with PDF 1.4, the Version entry in the document’s catalog dictionary
// (located via the Root entry in the file’s trailer, as described in 7.5.5, "File Trailer"),
// if present, shall be used instead of the version specified in the Header.
// Save PDF Version from header to xrefTable.
// The header version comes as the first line of the file.
func headerVersion(rs io.ReadSeeker, prefix string) (v string, err error) {
	// Get first line of file which holds the version of this PDFFile.
	// We call this the header version.
	if _, err = rs.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	buf := make([]byte, 100)
	if _, err = rs.Read(buf); err != nil {
		return "", err
	}

	s := string(buf)

	if len(s) < 8 || !strings.HasPrefix(s, prefix) {
		return "", errCorruptHeader
	}

	pdfVersion := s[len(prefix) : len(prefix)+3]
	return pdfVersion, nil
}

// Build XRefTable by reading XRef streams or XRef sections.
func (ctx *context) buildXRefTableStartingAt(offset int64) (err error) {
	seenOffsets := map[int64]bool{}
	ssCount := 0

	for offset != 0 {
		if seenOffsets[offset] {
			offset, err = ctx.offsetLastXRefSection(ctx.fileSize - offset)
			if err != nil {
				return err
			}
			if seenOffsets[offset] {
				return nil
			}
		}

		seenOffsets[offset] = true

		buf, err := ctx.readAt(int(ctx.fileSize-offset), offset)
		if err != nil {
			return err
		}

		tk := ctx.tokenizerBytes(buf)

		start, err := tk.PeekToken()
		if err != nil {
			return fmt.Errorf("invalid xref table: %s", err)
		}

		if start.IsOther("xref") { // xref section
			_, _ = tk.NextToken() // consume keyword
			offset, ssCount, err = ctx.parseXRefSectionAndTrailer(tk, ssCount)
			if err != nil {
				return err
			}
		} else { // xref stream
			offset, err = ctx.parseXRefStream(offset)
			if err != nil {
				log.Printf("reading PDF file: invalid xref stream (%s), trying fix\n", err)
				// Try fix for corrupt single xref section.
				return ctx.bypassXrefSection()
			}
		}
	}

	return nil
}

// Parse xRef section into corresponding number of xRef table entries,
// and the following trailer
func (ctx *context) parseXRefSectionAndTrailer(tk *tok.Tokenizer, ssCount int) (int64, int, error) {
	// Process all sub sections of this xRef section.
	for {
		err := ctx.xrefTable.parseXRefTableSubSection(tk)
		if err != nil {
			return 0, 0, err
		}
		ssCount++

		if next, _ := tk.PeekToken(); next.IsOther("trailer") {
			break
		}
	}
	// consume trailer
	_, _ = tk.NextToken()

	offset, err := ctx.processTrailer(tk)
	return offset, ssCount, err
}

func parseInt(tk *tok.Tokenizer) (int, error) {
	token, err := tk.NextToken()
	if err != nil {
		return 0, err
	}
	return token.Int()
}

// Process xRef table subsection and create corrresponding xRef table entries.
func (xrefTable *xRefTableContext) parseXRefTableSubSection(tk *tok.Tokenizer) error {
	startObjNumber, err := parseInt(tk)
	if err != nil {
		return fmt.Errorf("parseXRefTableSubSection: invalid start object number %s", err)
	}

	objCount, err := parseInt(tk)
	if err != nil {
		return fmt.Errorf("parseXRefTableSubSection: invalid object count %s", err)
	}

	// Process all entries of this subsection into xrefTable entries.
	for i := 0; i < objCount; i++ {
		entry, generationNumber, err := xrefTable.parseXRefTableEntry(tk)
		if err != nil {
			return err
		}

		objectNumber := startObjNumber + i

		if entry.offset == 0 && !entry.free { // skip entry for in use object with offset 0
			continue
		}

		ref := model.ObjIndirectRef{ObjectNumber: objectNumber, GenerationNumber: generationNumber}

		// since we read the last xref table first, we skip potential
		// older object definition
		if _, exists := xrefTable.objects[ref]; !exists {
			xrefTable.objects[ref] = entry
		}

	}

	return nil
}

// Read next subsection entry and generate corresponding xref table entry.
func (xrefTable xRefTableContext) parseXRefTableEntry(tk *tok.Tokenizer) (*xrefEntry, int, error) {
	offsetTk, err := tk.NextToken()
	if err != nil {
		return nil, 0, err
	}
	offset, err := strconv.ParseInt(string(offsetTk.Value), 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("parseXRefTableEntry: invalid offset: %s", err)
	}

	generation, err := parseInt(tk)
	if err != nil {
		return nil, 0, fmt.Errorf("parseXRefTableEntry: invalid generation number: %s", err)
	}

	entryType, err := tk.NextToken()
	if err != nil {
		return nil, 0, err
	}
	v := string(entryType.Value)
	if entryType.Kind != tok.Other || (v != "f" && v != "n") {
		return nil, 0, errors.New("parseXRefTableEntry: corrupt xref subsection entry")
	}

	entry := xrefEntry{offset: offset, free: v == "f"}
	return &entry, generation, nil
}

func (ctx *context) processTrailer(tk *tok.Tokenizer) (int64, error) {
	p := parser.NewParserFromTokenizer(tk)
	o, err := p.ParseObject()
	if err != nil {
		return 0, err
	}

	trailerDict, ok := o.(parser.Dict)
	if !ok {
		return 0, fmt.Errorf("processTrailer: expected dict, got %T", o)
	}

	// Parse trailer dict and return any offset of a previous xref section.
	// An offset of 0 means no prev entry

	err = ctx.trailer.parseTrailerInfo(trailerDict)
	if err != nil {
		return 0, err
	}

	if streams, ok := trailerDict["AdditionalStreams"].(parser.Array); ok {
		var arr parser.Array
		for _, v := range streams {
			if _, ok := v.(parser.IndirectRef); ok {
				arr = append(arr, v)
			}
		}
		ctx.additionalStreams = arr
	}

	// Prev entry
	// The spec is not very clear, since it says:
	// "Present only if the file has more than one cross-reference section; shall be
	// an indirect reference"
	// but in pratice it is always found as a direct object.
	// However certain buggy PDF generators generate "/Prev NNN 0 R" instead
	// of "/Prev NNN", maybe to try and follow the spec ?
	// we then accept both integer and reference

	offset, _ := offsetFromObject(trailerDict["Prev"])

	offsetXRefStream, ok := trailerDict["XRefStm"].(parser.Integer)
	if !ok {
		// No cross reference stream
		// continue to parse previous xref section, if there is any.
		return offset, nil
	}

	// 1.5 conformant readers process hidden objects contained
	// in XRefStm before continuing to process any previous XRefSection.
	// Previous XRefSection is expected to have free entries for hidden entries.
	// May appear in XRefSections only.
	if err := ctx.parseHybridXRefStream(int64(offsetXRefStream)); err != nil {
		return 0, err
	}

	return offset, nil
}

// accept Int or XXX 0 R
func offsetFromObject(o parser.Object) (int64, bool) {
	switch pref := o.(type) {
	case parser.Integer:
		return int64(pref), true
	case parser.IndirectRef:
		return int64(pref.ObjectNumber), true
	default:
		return 0, false
	}
}

// '7.5.6 - Incremental Updates' says :
// The added trailer shall contain all the entries except the Prev
// entry (if present) from the previous trailer, whether modified or not.
// We are a bit more liberal, allowing individual field update
func (current *trailer) parseTrailerInfo(d parser.Dict) error {
	if enc := d["Encrypt"]; enc != nil {
		current.encrypt = enc
	}

	if current.size == 0 {
		size, ok := d["Size"].(parser.Integer)
		if !ok {
			return errors.New("parseTrailerInfo: missing entry \"Size\"")
		}
		// Not reliable!
		// Patched after all read in.
		current.size = int(size)
	}

	if current.root == nil {
		root, ok := d["Root"].(parser.IndirectRef)
		if !ok {
			return errors.New("parseTrailerInfo: missing entry \"Root\"")
		}
		current.root = &root
	}

	if current.info == nil {
		if info, ok := d["Info"].(parser.IndirectRef); ok {
			current.info = &info
		}
	}

	if current.id == nil {
		id, ok := d["ID"].(parser.Array)
		// If there is an Encrypt entry this array and the two
		// byte-strings shall be direct objects and shall be unencrypted
		if !ok && current.encrypt != nil {
			return errors.New("parseTrailerInfo: missing entry \"ID\" in encrypted document")
		}
		current.id = id
	}

	return nil
}

// Parse an xRefStream for a hybrid PDF file.
func (ctx *context) parseHybridXRefStream(offset int64) error {
	_, err := ctx.parseXRefStream(offset)
	return err
}

type lineReader struct {
	src    *bufio.Reader
	buf    []byte // avoid allocations
	offset int64
}

func newLineReader(f io.Reader) lineReader {
	return lineReader{src: bufio.NewReader(f)}
}

func (l *lineReader) read() (byte, bool) {
	c, err := l.src.ReadByte()
	if err != nil {
		return 0, false
	}
	l.offset += 1
	return c, true
}

// return the line and the offset of the first byte in the
// underlying file
// the returned slice will be mutated in the next call
func (l *lineReader) readLine() ([]byte, int64) {
	// consume initial empty lines
	c, ok := l.read()
	for ; c == '\n' || c == '\r'; c, ok = l.read() {
	}
	if !ok {
		return nil, 0
	}
	offset := l.offset - 1
	l.buf = l.buf[:0] // do not re-allocate
	for {
		l.buf = append(l.buf, c)
		c, ok = l.read()
		if !ok || c == '\n' || c == '\r' {
			return l.buf, offset
		}
	}
}

// bypassXrefSection is a hack for digesting corrupt xref sections.
// It populates the xRefTable by reading in all indirect objects line by line
// and works on the assumption of a single xref section - meaning no incremental updates have been made.
func (ctx *context) bypassXrefSection() error {
	ctx.xrefTable = newXRefTable()

	_, err := ctx.rs.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	lr := newLineReader(ctx.rs)

	var (
		withinObj  bool
		withinXref bool
	)
	for {
		line, lineOffset := lr.readLine()
		if len(line) == 0 {
			return nil
		}
		tk := ctx.tokenizerBytes(line)
		firstToken, _ := tk.PeekToken()

		if withinObj { // lookfor "endobj"
			if firstToken.IsOther("endobj") {
				withinObj = false
			}
		} else if withinXref {
			if firstToken.IsOther("trailer") {
				// consume the token and read the end of the file
				_, _ = tk.NextToken()
				pos := lineOffset + int64(tk.CurrentPosition())
				buf, err := ctx.readAt(int(ctx.fileSize-pos), pos)
				if err != nil {
					return err
				}
				tk = ctx.tokenizerBytes(buf)
				_, err = ctx.processTrailer(tk)
				return err
			}
			// Ignore all until "trailer".
		} else if firstToken.IsOther("xref") {
			withinXref = true
		} else { // look for a declaration object XXX XX obj
			objNr, generation, err := parseObjectDeclaration(tk)
			if err == nil {
				ctx.xrefTable.objects[model.ObjIndirectRef{ObjectNumber: objNr, GenerationNumber: generation}] = &xrefEntry{
					// we do not account for potential whitespace
					// is this an issue ?
					offset: lineOffset,
					free:   false,
				}
				withinObj = true
			}
		}
	}
}

func parseObjectDeclaration(tk *tok.Tokenizer) (objectNumber, generationNumber int, err error) {
	objectNumber, err = parseInt(tk)
	if err != nil {
		return
	}
	generationNumber, err = parseInt(tk)
	if err != nil {
		return
	}
	objTk, err := tk.NextToken()
	if err != nil {
		return
	}
	if !objTk.IsOther("obj") {
		err = fmt.Errorf("parseObjectDeclaration: unexpected token %v", objTk)
	}
	return
}
