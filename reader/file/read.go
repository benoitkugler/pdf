package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	tok "github.com/benoitkugler/pdf/reader/parser/tokenizer"
)

var errCorruptHeader = errors.New("headerVersion: corrupt pdf stream - no header version available")

func Read(rs io.ReadSeeker, conf *Configuration) error {
	ctx, err := newContext(rs, conf)
	if err != nil {
		return err
	}

	o, err := ctx.offsetLastXRefSection(0)
	if err != nil {
		return err
	}

	err = ctx.buildXRefTableStartingAt(o)
	if err != nil {
		return err
	}

	return nil
}

// Context represents an environment for processing PDF files.
type Context struct {
	rs       io.ReadSeeker
	fileSize int64

	Configuration
	xrefTable
}

func newContext(rs io.ReadSeeker, conf *Configuration) (*Context, error) {
	if conf == nil {
		conf = NewDefaultConfiguration()
	}

	rdCtx := &Context{
		rs: rs,
		// ObjectStreams: intSet{},
		// XRefStreams:   intSet{},
		Configuration: *conf,
		xrefTable:     xrefTable{Table: make(map[int]xrefEntry)},
	}

	fileSize, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	rdCtx.fileSize = fileSize

	return rdCtx, nil
}

type xrefTable struct {
	// PDF Version
	HeaderVersion string // The PDF version the source is claiming to us as per its header.

	Table map[int]xrefEntry
	Size  int // Object count from PDF trailer dict.
}

type xrefEntry struct {
	free       bool
	offset     int64
	generation int
}

type Configuration struct{}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{}
}

func (ctx *Context) readAt(p []byte, offset int64) error {
	_, err := ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = ctx.rs.Read(p)
	return err
}

// Get the file offset of the last XRefSection.
// Go to end of file and search backwards for the first occurrence of startxref {offset} %%EOF
// xref at 114172
func (ctx *Context) offsetLastXRefSection(skip int64) (int64, error) {

	rs := ctx.rs

	var (
		prevBuf, workBuf []byte
		bufSize          int64 = 512
		offset           int64
	)

	// guard for very small files
	if ctx.fileSize < bufSize {
		bufSize = ctx.fileSize
	}

	for i := 1; offset == 0; i++ {

		_, err := rs.Seek(-int64(i)*bufSize-skip, io.SeekEnd)
		if err != nil {
			return 0, fmt.Errorf("can't find last xref section: %s", err)
		}

		curBuf := make([]byte, bufSize)

		_, err = rs.Read(curBuf)
		if err != nil {
			return 0, fmt.Errorf("can't read last xref section: %s", err)
		}

		workBuf = append(curBuf, prevBuf...)

		j := bytes.LastIndex(workBuf, []byte("startxref"))
		if j == -1 {
			prevBuf = curBuf
			continue
		}

		p := workBuf[j+len("startxref"):]
		posEOF := bytes.Index(p, []byte("%%EOF"))
		if posEOF == -1 {
			return 0, errors.New("no matching %%EOF for startxref")
		}

		p = p[:posEOF]
		offset, err = strconv.ParseInt(string(bytes.TrimSpace(p)), 10, 64)
		if err != nil || offset >= ctx.fileSize {
			return 0, errors.New("corrupted last xref section")
		}
	}
	return offset, nil
}

// Get version from first line of file.
// Beginning with PDF 1.4, the Version entry in the document’s catalog dictionary
// (located via the Root entry in the file’s trailer, as described in 7.5.5, "File Trailer"),
// if present, shall be used instead of the version specified in the Header.
// Save PDF Version from header to xRefTable.
// The header version comes as the first line of the file.
func headerVersion(rs io.ReadSeeker) (v string, err error) {

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
	prefix := "%PDF-"

	if len(s) < 8 || !strings.HasPrefix(s, prefix) {
		return "", errCorruptHeader
	}

	pdfVersion := s[len(prefix) : len(prefix)+3]
	return pdfVersion, nil
}

// Build XRefTable by reading XRef streams or XRef sections.
func (ctx *Context) buildXRefTableStartingAt(offset int64) (err error) {
	rs := ctx.rs

	ctx.HeaderVersion, err = headerVersion(rs)
	if err != nil {
		return err
	}

	offs := map[int64]bool{}
	ssCount := 0

	for offset != 0 {
		if offs[offset] {
			offset, err = ctx.offsetLastXRefSection(ctx.fileSize - offset)
			if err != nil {
				return err
			}
			if offs[offset] {
				return nil
			}
		}

		offs[offset] = true

		buf := make([]byte, ctx.fileSize-offset)
		err = ctx.readAt(buf, offset)
		if err != nil {
			return err
		}

		tk := tok.NewTokenizer(buf)

		start, err := tk.PeekToken()
		if err != nil {
			return fmt.Errorf("invalid xref table: %s", err)
		}

		if start == (tok.Token{Kind: tok.Other, Value: "xref"}) { // xref section
			_, _ = tk.NextToken() // consume keyword
			offset, ssCount, err = ctx.parseXRefSection(&tk, ssCount)
			if err != nil {
				return err
			}
		} else { // xref stream
			offset, err = ctx.parseXRefStream(buf, offset)
			if err != nil {
				// Try fix for corrupt single xref section.
				return ctx.bypassXrefSection()
			}
		}
	}

	// A friendly 🤢 to the devs of the HP Scanner & Printer software utility.
	// Hack for #250: If exactly one xref subsection ensure it starts with object #0 instead #1.
	if _, hasZero := ctx.Table[0]; ssCount == 1 && !hasZero {
		for i := 1; i <= ctx.Size; i++ {
			ctx.Table[i-1] = ctx.Table[i]
		}
		delete(ctx.Table, ctx.Size)
	}

	return nil
}

// Parse xRef section into corresponding number of xRef table entries.
func (ctx *Context) parseXRefSection(tk *tok.Tokenizer, ssCount int) (int64, int, error) {
	// Process all sub sections of this xRef section.
	for {
		err := ctx.xrefTable.parseXRefTableSubSection(tk)
		if err != nil {
			return 0, 0, err
		}
		ssCount++

		if next, _ := tk.PeekToken(); next == (tok.Token{Kind: tok.Other, Value: "trailer"}) {
			break
		}
	}

	return ctx.processTrailer(tk)
}

func parseInt(tk *tok.Tokenizer) (int, error) {
	token, err := tk.NextToken()
	if err != nil {
		return 0, err
	}
	return token.Int()
}

// Process xRef table subsection and create corrresponding xRef table entries.
func (xRefTable *xrefTable) parseXRefTableSubSection(tk *tok.Tokenizer) error {
	startObjNumber, err := parseInt(tk)
	if err != nil {
		return err
	}

	objCount, err := parseInt(tk)
	if err != nil {
		return err
	}

	// Process all entries of this subsection into xRefTable entries.
	for i := 0; i < objCount; i++ {
		if err := xRefTable.parseXRefTableEntry(tk, startObjNumber+i); err != nil {
			return err
		}
	}

	return nil
}

// Read next subsection entry and generate corresponding xref table entry.
func (xRefTable *xrefTable) parseXRefTableEntry(tk *tok.Tokenizer, objectNumber int) error {
	// since we read the last xref table first, we skip potential
	// older object definition
	if _, exists := xRefTable.Table[objectNumber]; exists {
		return nil
	}

	offsetTk, err := tk.NextToken()
	if err != nil {
		return err
	}
	offset, err := strconv.ParseInt(offsetTk.Value, 10, 64)
	if err != nil {
		return err
	}

	generation, err := parseInt(tk)
	if err != nil {
		return err
	}

	entryType, err := tk.NextToken()
	if err != nil {
		return err
	}
	if entryType.Kind != tok.Other || (entryType.Value != "f" && entryType.Value != "n") {
		return errors.New("parseXRefTableEntry: corrupt xref subsection entry")
	}

	entry := xrefEntry{
		free:       entryType.Value == "f",
		offset:     offset,
		generation: generation,
	}

	if !entry.free && offset == 0 { // Skip entry for in use object with offset 0
		return nil
	}

	xRefTable.Table[objectNumber] = entry
	return nil
}

// TODO:
func (ctx *Context) processTrailer(tk *tok.Tokenizer) (int64, int, error) {
	return 0, 0, nil
}
func (ctx *Context) bypassXrefSection() error { return nil }
func (ctx *Context) parseXRefStream(buf []byte, offset int64) (int64, error) {
	return 0, nil
}
