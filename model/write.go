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

// PDFOutput abstracts away the complexity of
// writing a PDF file.
// Package `writer` provides a defaut PDFOutput implementation.
type PDFOutput interface {
	// EncodeTextString should encode `s` to one of the PDF encoding:
	// PDFDocEncoding or UTF16-BE
	// If should also encrypt `s`, if needed
	EncodeTextString(s string) string

	// ASCIIString should return the PDF form of `s`,
	// which should already be ASCII,
	// escaped and encrypted if need
	ASCIIString(s string) string

	// CreateObject return a new reference which shoud be update later on.
	// This is needed to write objects that must reference their "parent".
	CreateObject() Reference

	// WriteObject write the content of the object `ref`
	// This method will be called at most once for each reference.
	WriteObject(content []byte, ref Reference)
}

type cachable interface {
	isCachable()
	PDFBytes(pdf PDFWriter) []byte
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

// PDFWriter uses `PDFOutput` and an internal cache
// to write a Document.
// The internal cache avoids duplication of indirect object,
// by associating an object number to a pointer
type PDFWriter struct {
	PDFOutput
	cache map[cachable]Reference
	pages map[*PageObject]Reference
}

func NewPDFWritter(w PDFOutput) PDFWriter {
	return PDFWriter{PDFOutput: w, cache: make(map[cachable]Reference), pages: make(map[*PageObject]Reference)}
}

// addObject is a convenience shortcut to write `content` into a new object
// and return the created reference
func (p PDFWriter) addObject(content []byte) Reference {
	ref := p.CreateObject()
	p.WriteObject(content, ref)
	return ref
}

// Write walks the entire document and writes its content
// using `pdf` as an output.
// It returns two references, needed by a PDF writer to finalize
// the file (that is, to write the trailer)
func (pdf PDFWriter) Write(doc Document) (root, info Reference) {
	root = pdf.addObject(doc.Catalog.PDFBytes(pdf))
	info = pdf.addObject(doc.Trailer.Info.PDFBytes(pdf))
	return root, info
}

// writerCache

// check the cache and write a new item if not found
func (pdf PDFWriter) addItem(item cachable) Reference {
	if ref, has := pdf.cache[item]; has {
		return ref
	}
	ref := pdf.addObject(item.PDFBytes(pdf))
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
	return fmt.Sprintf("[ %s]", strings.Join(b, " "))
}

func writeRangeArray(rs []Range) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%.3f %.3f ", a[0], a[1])
	}
	return fmt.Sprintf("[ %s]", strings.Join(b, " "))
}

func dateString(t time.Time) string {
	_, tz := t.Zone()
	return fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		tz/60/60, tz/60%60)
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
