// Package truetype provides support for OpenType and TrueType font formats, used in PDF.
//
// It is vastly copied from github.com/ConradIrwin/font and golang.org/x/image/font/sfnt.
package truetype

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type fixed struct {
	Major int16
	Minor uint16
}

type longdatetime struct {
	SecondsSince1904 uint64
}

var (
	// errMissingHead is returned by ParseOTF when the font has no head section.
	errMissingHead = errors.New("missing head table in font")

	// errInvalidChecksum is returned by ParseOTF if the font's checksum is wrong
	errInvalidChecksum = errors.New("invalid checksum")

	// errUnsupportedFormat is returned from Parse if parsing failed
	errUnsupportedFormat = errors.New("unsupported font format")

	// errMissingTable is returned from *Table if the table does not exist in the font.
	errMissingTable = errors.New("missing table")
)

// Font represents a SFNT font, which is the underlying representation found
// in .otf and .ttf files.
// SFNT is a container format, which contains a number of tables identified by
// Tags. Depending on the type of glyphs embedded in the file which tables will
// exist. In particular, there's a big different between TrueType glyphs (usually .ttf)
// and CFF/PostScript Type 2 glyphs (usually .otf)
type Font struct {
	// Type represents the kind of glyphs in this font.
	// It is one of TypeTrueType, TypeTrueTypeApple, TypePostScript1, TypeOpenType
	Type TableTag

	file File

	tables map[TableTag]*tableSection
}

// tableSection represents a table within the font file.
type tableSection struct {
	offset  uint32 // Offset into the file this table starts.
	length  uint32 // Length of this table within the file.
	zLength uint32 // Uncompressed length of this table.
}

// HeadTable returns the table corresponding to the 'head' tag.
func (font *Font) HeadTable() (*TableHead, error) {
	s, found := font.tables[tagHead]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return nil, err
	}

	return parseTableHead(buf)
}

// NameTable returns the table corresponding to the 'name' tag.
func (font *Font) NameTable() (*TableName, error) {
	s, found := font.tables[tagName]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return nil, err
	}
	return parseTableName(buf)
}

func (font *Font) HheaTable() (*TableHhea, error) {
	s, found := font.tables[tagHhea]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return nil, err
	}

	return parseTableHhea(buf)
}

func (font *Font) OS2Table() (*TableOS2, error) {
	s, found := font.tables[tagOS2]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return nil, err
	}

	return parseTableOS2(buf)
}

// GposTable returns the Glyph Positioning table identified with the 'GPOS' tag.
func (font *Font) GposTable() (*TableLayout, error) {
	s, found := font.tables[tagGpos]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return nil, err
	}

	return parseTableLayout(buf)
}

// GsubTable returns the Glyph Substitution table identified with the 'GSUB' tag.
func (font *Font) GsubTable() (*TableLayout, error) {
	s, found := font.tables[tagGsub]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return nil, err
	}

	return parseTableLayout(buf)
}

// CmapTable returns the Character to Glyph Index Mapping table.
func (font *Font) CmapTable() (Cmap, error) {
	s, found := font.tables[tagCmap]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return nil, err
	}

	return parseTableCmap(buf)
}

// PostTable returns the Post table names
func (font *Font) PostTable() (PostTable, error) {
	s, found := font.tables[tagPost]
	if !found {
		return PostTable{}, errMissingTable
	}

	buf, err := font.findTableBuffer(s)
	if err != nil {
		return PostTable{}, err
	}

	numGlyph, err := font.numGlyphs()
	if err != nil {
		return PostTable{}, err
	}

	return parseTablePost(buf, numGlyph)
}

func (font *Font) numGlyphs() (uint16, error) {
	maxpSection, found := font.tables[tagMaxp]
	if !found {
		return 0, errMissingTable
	}

	buf, err := font.findTableBuffer(maxpSection)
	if err != nil {
		return 0, err
	}

	return parseMaxpTable(buf)
}

// HtmxTable returns the glyphs widths (array of size numGlyphs)
func (font *Font) HtmxTable() ([]int, error) {
	numGlyph, err := font.numGlyphs()
	if err != nil {
		return nil, err
	}

	hhea, err := font.HheaTable()
	if err != nil {
		return nil, err
	}

	htmxSection, found := font.tables[tagHmtx]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(htmxSection)
	if err != nil {
		return nil, err
	}

	return parseHtmxTable(buf, uint16(hhea.NumOfLongHorMetrics), numGlyph)
}

// KernTable returns the kern table, with kerning value expressed in
// glyph units.
// Unless `kernFirst` is true, the priority is given to the GPOS table, then to the kern table.
func (font *Font) KernTable(kernFirst bool) (kerns Kerns, err error) {
	if kernFirst {
		kerns, err = font.kernKerning()
		if err != nil {
			kerns, err = font.gposKerning()
		}
	} else {
		kerns, err = font.gposKerning()
		if err != nil {
			kerns, err = font.kernKerning()
		}
	}
	return
}

func (font *Font) gposKerning() (Kerns, error) {
	gpos, err := font.GposTable()
	if err != nil {
		return nil, err
	}

	return gpos.parseKern()
}

func (font *Font) kernKerning() (Kerns, error) {
	section, found := font.tables[tagKern]
	if !found {
		return nil, errMissingTable
	}

	buf, err := font.findTableBuffer(section)
	if err != nil {
		return nil, err
	}

	return parseKernTable(buf)
}

// File is a combination of io.Reader, io.Seeker and io.ReaderAt.
// This interface is satisfied by most things that you'd want
// to parse, for example *os.File, io.SectionReader or *bytes.Buffer.
type File interface {
	Read([]byte) (int, error)
	ReadAt([]byte, int64) (int, error)
	Seek(int64, int) (int64, error)
}

// Parse parses an OpenType or TrueType file and returns a Font.
// The underlying file is still needed to parse the tables, and must not be closed.
func Parse(file File) (*Font, error) {
	magic, err := readTag(file)
	if err != nil {
		return nil, err
	}

	file.Seek(0, 0)

	switch magic {
	case TypeTrueType, TypeOpenType, TypePostScript1, TypeAppleTrueType:
		return parseOTF(file)
	default:
		return nil, errUnsupportedFormat
	}
}

type otfHeader struct {
	ScalerType    TableTag
	NumTables     uint16
	SearchRange   uint16
	EntrySelector uint16
	RangeShift    uint16
}

const otfHeaderLength = 12
const directoryEntryLength = 16

func (header *otfHeader) checkSum() uint32 {
	return uint32(header.ScalerType) +
		(uint32(header.NumTables)<<16 | uint32(header.SearchRange)) +
		(uint32(header.EntrySelector)<<16 + uint32(header.RangeShift))
}

// An Entry in an OpenType table.
type directoryEntry struct {
	Tag      TableTag
	CheckSum uint32
	Offset   uint32
	Length   uint32
}

func (entry *directoryEntry) checkSum() uint32 {
	return uint32(entry.Tag) + entry.CheckSum + entry.Offset + entry.Length
}

func readOTFHeader(r io.Reader, header *otfHeader) error {
	var buf [12]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}

	header.ScalerType = newTag(buf[0:4])
	header.NumTables = binary.BigEndian.Uint16(buf[4:6])
	header.SearchRange = binary.BigEndian.Uint16(buf[6:8])
	header.EntrySelector = binary.BigEndian.Uint16(buf[8:10])
	header.RangeShift = binary.BigEndian.Uint16(buf[10:12])

	return nil
}

func readDirectoryEntry(r io.Reader, entry *directoryEntry) error {
	var buf [16]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}

	entry.Tag = newTag(buf[0:4])
	entry.CheckSum = binary.BigEndian.Uint32(buf[4:8])
	entry.Offset = binary.BigEndian.Uint32(buf[8:12])
	entry.Length = binary.BigEndian.Uint32(buf[12:16])

	return nil
}

// parseOTF reads an OpenTyp (.otf) or TrueType (.ttf) file and returns a Font.
// If parsing fails, then an error is returned and Font will be nil.
func parseOTF(file File) (*Font, error) {
	var header otfHeader
	if err := readOTFHeader(file, &header); err != nil {
		return nil, err
	}

	font := &Font{
		file: file,

		Type:   header.ScalerType,
		tables: make(map[TableTag]*tableSection, header.NumTables),
	}

	for i := 0; i < int(header.NumTables); i++ {
		var entry directoryEntry
		if err := readDirectoryEntry(file, &entry); err != nil {
			return nil, err
		}

		// TODO Check the checksum.

		if _, found := font.tables[entry.Tag]; found {
			return nil, fmt.Errorf("found multiple %q tables", entry.Tag)
		}

		font.tables[entry.Tag] = &tableSection{
			offset: entry.Offset,
			length: entry.Length,
		}
	}

	if _, ok := font.tables[tagHead]; !ok {
		return nil, errMissingHead
	}

	return font, nil
}
