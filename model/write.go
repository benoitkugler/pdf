package model

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Reference is the object number of a PDF object.
// It is only needed to write a document.
type Reference int

// String return a string to be used when writing a PDF
func (r Reference) String() string {
	return fmt.Sprintf("%d 0 R", r)
}

type StringEncoding uint8

const (
	ASCIIString StringEncoding = iota // ASCII encoding and escaping
	ByteString                        // no special treatment, except escaping
	HexString                         // hex form
	TextString                        // one of the PDF encoding: PDFDocEncoding or UTF16-BE
)

// PDFOutput abstracts away the complexity of
// writing a PDF file.
// Package `writer` provides a defaut PDFOutput implementation.
type PDFOutput interface {
	// EncodeString should transform an UTF-8 string `s` to satisfy the PDF
	// format required
	// It should also encrypt `s`, if needed
	EncodeString(s string, mode StringEncoding) string

	// CreateObject return a new reference which shoud be update later on.
	// This is needed to write objects that must reference their "parent".
	CreateObject() Reference

	// WriteObject write the content of the object `ref`
	// This method will be called at most once for each reference.
	// For stream object, `content` will contain the dictionary,
	// and `stream` the inner stream bytes. For other objects, `stream` will be nil.
	// Stream content should be encrypted if needed.
	WriteObject(content string, stream []byte, ref Reference)
}

type cachable interface {
	isCachable()
	pdfContent(pdf pdfWriter) (content string, stream []byte)
}

func (*FormField) isCachable()          {}
func (*ResourcesDict) isCachable()      {}
func (*Font) isCachable()               {}
func (*GraphicState) isCachable()       {}
func (*EncodingDict) isCachable()       {}
func (*Annotation) isCachable()         {}
func (*FileSpec) isCachable()           {}
func (*EmbeddedFileStream) isCachable() {}
func (*ShadingDict) isCachable()        {}
func (*Function) isCachable()           {}
func (*TilingPatern) isCachable()       {}
func (*ShadingPatern) isCachable()      {}
func (*ICCBasedColorSpace) isCachable() {}
func (*ColorTableStream) isCachable()   {}
func (*XObjectForm) isCachable()        {}
func (*XObjectImage) isCachable()       {}

// pdfWriter uses `PDFOutput` and an internal cache
// to write a Document.
// The internal cache avoids duplication of indirect object,
// by associating an object number to a pointer
type pdfWriter struct {
	PDFOutput
	cache    map[cachable]Reference
	pages    map[*PageObject]Reference
	outlines map[*OutlineItem]Reference
	fields   map[*FormField]Reference
}

func newWriter(pdf PDFOutput) pdfWriter {
	return pdfWriter{PDFOutput: pdf,
		cache:    make(map[cachable]Reference),
		pages:    make(map[*PageObject]Reference),
		outlines: make(map[*OutlineItem]Reference),
		fields:   make(map[*FormField]Reference),
	}
}

// addObject is a convenience shortcut to write `content` into a new object
// and return the created reference
func (p pdfWriter) addObject(content string, stream []byte) Reference {
	ref := p.CreateObject()
	p.WriteObject(content, stream, ref)
	return ref
}

// writerCache

// check the cache and write a new item if not found
func (pdf pdfWriter) addItem(item cachable) Reference {
	if ref, has := pdf.cache[item]; has {
		return ref
	}
	ref := pdf.addObject(item.pdfContent(pdf))
	pdf.cache[item] = ref
	return ref
}

func writeIntArray(as []int) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = strconv.Itoa(a)
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeFloatArray(as []float64) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = fmt.Sprintf("%.3f", a)
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeRefArray(as []Reference) string {
	b := make([]string, len(as))
	for i, ref := range as {
		b[i] = ref.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writePointArray(rs [][2]float64) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%.3f %.3f ", a[0], a[1])
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeRangeArray(rs []Range) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%.3f %.3f ", a[0], a[1])
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeNameArray(rs []Name) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = a.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func (pdf pdfWriter) dateString(t time.Time) string {
	_, tz := t.Zone()
	str := fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		tz/60/60, tz/60%60)
	return pdf.EncodeString(str, TextString)
}

func (pdf pdfWriter) stringsArray(ar []string, mode StringEncoding) string {
	chunks := make([]string, len(ar))
	for i, val := range ar {
		chunks[i] = pdf.EncodeString(val, mode)
	}
	return fmt.Sprintf("[%s]", strings.Join(chunks, " "))
}

// helper to shorten the writting of formatted strings
type buffer struct {
	*bytes.Buffer
}

func newBuffer() buffer {
	return buffer{Buffer: &bytes.Buffer{}}
}

func (b buffer) fmt(format string, arg ...interface{}) {
	fmt.Fprintf(b.Buffer, format, arg...)
}

// add a formatted line
func (b buffer) line(format string, arg ...interface{}) {
	b.fmt(format, arg...)
	b.WriteByte('\n')
}
