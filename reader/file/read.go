package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/benoitkugler/pdf/reader/parser"
	tok "github.com/benoitkugler/pdf/reader/parser/tokenizer"
)

// freeHeadGeneration is the predefined generation number for the head of the free list.
const freeHeadGeneration = 65535

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

	// PDF Version
	HeaderVersion string // The PDF version the source is claiming to us as per its header.
	xrefTable     xrefTable
	trailer       trailer

	// AdditionalStreams (array of IndirectRef) is not described in the spec,
	// but may be found in the trailer :e.g., Oasis "Open Doc"
	AdditionalStreams parser.Array
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
		xrefTable:     make(map[int]xrefEntry),
	}

	fileSize, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	rdCtx.fileSize = fileSize

	return rdCtx, nil
}

type xrefTable map[int]xrefEntry

type xrefEntry struct {
	free       bool
	offset     int64
	generation int

	// for object in object streams
	streamObjectNumber int // The object number of the object stream in which this object is stored.
	streamObjectIndex  int // The index of this object within the object stream.
}

type trailer struct {
	encrypt parser.Object // indirect ref or dict

	size int // Object count from PDF trailer dict.
	root *parser.IndirectRef
	info *parser.IndirectRef // optional
	id   parser.Array        // required in encrypted docs
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
// Save PDF Version from header to xrefTable.
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
			offset, ssCount, err = ctx.parseXRefSection(tk, ssCount)
			if err != nil {
				return err
			}
		} else { // xref stream
			offset, err = ctx.parseXRefStream(offset)
			if err != nil {
				// Try fix for corrupt single xref section.
				return ctx.bypassXrefSection()
			}
		}
	}

	// A friendly 🤢 to the devs of the HP Scanner & Printer software utility.
	// Hack for #250: If exactly one xref subsection ensure it starts with object #0 instead #1.
	if _, hasZero := ctx.xrefTable[0]; ssCount == 1 && !hasZero {
		for i := 1; i <= ctx.trailer.size; i++ {
			ctx.xrefTable[i-1] = ctx.xrefTable[i]
		}
		delete(ctx.xrefTable, ctx.trailer.size)
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
func (xrefTable *xrefTable) parseXRefTableSubSection(tk *tok.Tokenizer) error {
	startObjNumber, err := parseInt(tk)
	if err != nil {
		return err
	}

	objCount, err := parseInt(tk)
	if err != nil {
		return err
	}

	// Process all entries of this subsection into xrefTable entries.
	for i := 0; i < objCount; i++ {
		if err := xrefTable.parseXRefTableEntry(tk, startObjNumber+i); err != nil {
			return err
		}
	}

	return nil
}

// Read next subsection entry and generate corresponding xref table entry.
func (xrefTable xrefTable) parseXRefTableEntry(tk *tok.Tokenizer, objectNumber int) error {
	// since we read the last xref table first, we skip potential
	// older object definition
	if _, exists := xrefTable[objectNumber]; exists {
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

	xrefTable[objectNumber] = entry
	return nil
}

func (ctx *Context) processTrailer(tk *tok.Tokenizer) (int64, error) {
	p := parser.NewParserFromTokenizer(tk)
	o, err := p.ParseObject()
	if err != nil {
		return 0, err
	}

	trailerDict, ok := o.(parser.Dict)
	if !ok {
		return 0, fmt.Errorf("processTrailer: expected dict, got %T", o)
	}

	return ctx.parseTrailerDict(trailerDict)
}

// accept Int or XXX 0 R
func offsetFromObject(o parser.Object) (int64, bool) {
	var offset int64
	switch pref := o.(type) {
	case parser.Integer:
		offset = int64(pref)
	case parser.IndirectRef:
		offset = int64(pref.ObjectNumber)
	default:
		return 0, false
	}
	return offset, true
}

// Parse trailer dict and return any offset of a previous xref section.
// An offset of 0 means no prev entry
func (ctx *Context) parseTrailerDict(trailerDict parser.Dict) (int64, error) {

	err := ctx.trailer.parseTrailerInfo(trailerDict)
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
		ctx.AdditionalStreams = arr
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
		current.encrypt = id
	}

	return nil
}

// Parse an xRefStream for a hybrid PDF file.
func (ctx *Context) parseHybridXRefStream(offset int64) error {
	_, err := ctx.parseXRefStream(offset)
	return err
}

// TODO:
// bypassXrefSection is a hack for digesting corrupt xref sections.
// It populates the xRefTable by reading in all indirect objects line by line
// and works on the assumption of a single xref section - meaning no incremental updates have been made.
func (ctx *Context) bypassXrefSection() error {
	ctx.xrefTable[0] = xrefEntry{
		free:       true,
		offset:     0,
		generation: freeHeadGeneration,
	}

	_, err := ctx.rs.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	tk := tok.NewTokenizerFromReader(ctx.rs)

	var (
		withinObj     bool
		withinXref    bool
		withinTrailer bool
	)

	for {
		line := tk.ReadLine()
		if len(line) == 0 {
			break
		}
		if withinXref {
			// offset += int64(len(line) + eolCount)
			if withinTrailer {
				i := bytes.Index(line, []byte("startxref"))
				if i >= 0 {
					// Parse trailer.
					_, err = ctx.processTrailer(tk)
					return err
				}
				continue
			}
			// Ignore all until "trailer".
			i := bytes.Index(line, []byte("trailer"))
			if i >= 0 {
				// bb = append(bb, line...)
				withinTrailer = true
			}
			continue
		}
		i := bytes.Index(line, []byte("xref"))
		if i >= 0 {
			// offset += int64(len(line) + eolCount)
			withinXref = true
			continue
		}
		if !withinObj {
			i := bytes.Index(line, []byte("obj"))
			if i >= 0 {
				withinObj = true
				// off = offset
				// bb = append(bb, line[:i+3]...)
			}
			// offset += int64(len(line) + eolCount)
			continue
		}

		// within obj
		// offset += int64(len(line) + eolCount)
		// bb = append(bb, ' ')
		// bb = append(bb, line...)
		i = bytes.Index(line, []byte("endobj"))
		if i >= 0 {
			// l := string(bb)
			// objNr, generation, err := parseObjectAttributes(&l)
			// if err != nil {
			// 	return err
			// }
			// of := off
			// ctx.Table[*objNr] = &XRefTableEntry{
			// 	Free:       false,
			// 	Offset:     &of,
			// 	Generation: generation}
			// bb = nil
			withinObj = false
		}
	}
	return nil
}

// return the previous offset (0 if it does not exists)
func (ctx *Context) parseXRefStream(offset int64) (int64, error) {
	_, err := ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	tk := tok.NewTokenizerFromReader(ctx.rs)

	objectNumber, err := parseInt(tk)
	if err != nil {
		return 0, err
	}
	generationNumber, err := parseInt(tk)
	if err != nil {
		return 0, err
	}
	objTk, err := tk.NextToken()
	if err != nil {
		return 0, err
	}
	if objTk != (tok.Token{Kind: tok.Other, Value: "obj"}) {
		return 0, fmt.Errorf("parseXRefStream: unexpected token %v", objTk)
	}

	// parse this object
	pr := parser.NewParserFromTokenizer(tk)
	o, err := pr.ParseObject()
	if err != nil {
		return 0, fmt.Errorf("parseXRefStream: no object: %s", err)
	}

	d, ok := o.(parser.Dict)
	if !ok {
		return 0, fmt.Errorf("xRefStreamDict: expected dict, got %T", o)
	}

	streamStart, err := tk.NextToken()
	if err != nil {
		return 0, err
	}
	if streamStart != (tok.Token{Kind: tok.Other, Value: "stream"}) {
		return 0, fmt.Errorf("unexpected token %s", streamStart)
	}
	tk.SkipWhitespaceStream()

	streamOffset := offset + int64(tk.CurrentPosition())

	sd, decoded, err := ctx.xRefStreamDict(d, streamOffset)
	if err != nil {
		return 0, err
	}

	err = ctx.trailer.parseTrailerInfo(d)
	if err != nil {
		return 0, err
	}

	// Parse xRefStream and create xRefTable entries for embedded objects.
	err = ctx.extractXRefTableEntriesFromXRefStream(decoded, sd)
	if err != nil {
		return 0, err
	}

	// Skip entry if already assigned
	if _, has := ctx.xrefTable[objectNumber]; !has {
		// Create xRefTableEntry for XRefStreamDict.
		entry := xrefEntry{
			free:       false,
			offset:     offset,
			generation: generationNumber,
		}
		ctx.xrefTable[objectNumber] = entry
		// ctx.XRefStreams[*objectNumber] = true // TODO: check
	}

	return sd.prev, nil
}

func (ctx *Context) xRefStreamDict(d parser.Dict, streamOffset int64) (xrefStreamDict, []byte, error) {
	details, err := parseXRefStreamDict(d)
	if err != nil {
		return details, nil, err
	}

	filterPipeline, err := parser.ParseDirectFilters(d["Filter"], d["DecodeParms"])
	if err != nil {
		return details, nil, err
	}

	// we do not trust the stream length; instead we either
	//		- use count
	// 		- buffer a maximum of length and read until EOD of a filter is found (image filters are not supported,
	//			but should never be used here)

	var content []byte
	if len(filterPipeline) == 0 {
		length := details.count() * details.entrySize()
		content = make([]byte, length)
		err := ctx.readAt(content, streamOffset)
		if err != nil {
			return details, nil, err
		}
	} else {
		skipper, err := parser.SkipperFromFilter(filterPipeline[0])
		if err != nil {
			return details, nil, err
		}
		content = make([]byte, details.length)
		err = ctx.readAt(content, streamOffset)
		if err != nil && err != io.EOF {
			return details, nil, err
		}
		read, err := skipper.Skip(content)
		if err != nil {
			return details, nil, err
		}
		content = content[:read]
	}

	// Decode xrefstream content
	r, err := filterPipeline.DecodeReader(bytes.NewReader(content))
	if err != nil {
		return details, nil, err
	}
	decoded, err := ioutil.ReadAll(r)
	if err != nil {
		return details, nil, err
	}

	return details, decoded, nil
}

// For each object embedded in this xRefStream create the corresponding xRef table entry.
func (ctx *Context) extractXRefTableEntriesFromXRefStream(buf []byte, xrefDict xrefStreamDict) error {
	// Note:
	// A value of zero for an element in the W array indicates that the corresponding field shall not be present in the stream,
	// and the default value shall be used, if there is one.
	// If the first element is zero, the type field shall not be present, and shall default to type 1.

	xrefEntryLen, count := xrefDict.entrySize(), xrefDict.count()
	L := count * xrefEntryLen
	if len(buf) < L {
		return errors.New("pdfcpu: extractXRefTableEntriesFromXRefStream: corrupt xrefstream")
	}
	// Sometimes there is an additional xref entry not accounted for by "Index".
	// We ignore such a entries and do not treat this as an error.
	buf = buf[:L]

	i1 := xrefDict.w[0]
	i2 := xrefDict.w[1]
	i3 := xrefDict.w[2]

	// bufToInt64 interprets the content of buf as an int64.
	bufToInt64 := func(buf []byte) (i int64) {
		for _, b := range buf {
			i <<= 8
			i |= int64(b)
		}
		return i
	}

	j := 0 // current index of object (0 <= j < count)
	for _, subsection := range xrefDict.index {
		firstObj, nb := subsection[0], subsection[1]
		for i := 0; i < nb; i++ {
			objectNumber := firstObj + i

			offsetEntry := j * xrefEntryLen
			c2 := bufToInt64(buf[offsetEntry+i1 : offsetEntry+i1+i2])
			c3 := bufToInt64(buf[offsetEntry+i1+i2 : offsetEntry+i1+i2+i3])

			var xRefTableEntry xrefEntry
			switch buf[offsetEntry] {
			case 0x00: // free object
				xRefTableEntry = xrefEntry{
					free:       true,
					offset:     c2,
					generation: int(c3),
				}
			case 0x01: // in use object
				xRefTableEntry = xrefEntry{
					free:       false,
					offset:     c2,
					generation: int(c3),
				}
			case 0x02: // compressed object; generation always 0.
				xRefTableEntry = xrefEntry{
					free:               false,
					streamObjectNumber: int(c2),
					streamObjectIndex:  int(c3),
				}
				// TODO: check
				// ctx.ObjectStreams[objNumberRef] = true
			}

			// skip already assigned
			if _, has := ctx.xrefTable[objectNumber]; !has {
				ctx.xrefTable[objectNumber] = xRefTableEntry
			}
			j++
		}
	}
	return nil
}

type xrefStreamDict struct {
	length int
	size   int
	index  [][2]int
	w      [3]int
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
