package model

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/unicode"
)

// Reference is the object number of a PDF object.
// It is only needed to write a document.
type Reference uint32

// String return a string to be used when writing a PDF
func (r Reference) String() string {
	return fmt.Sprintf("%d 0 R", r)
}

// output implements the logic needed to write object
// and keep track of the correct byte offsets
type output struct {
	dst     io.Writer
	err     error // internal error, to defer error checking
	written int   // total number of bytes written to dst

	// encode the object numbers as index (starting from 1)
	// and the byte offsets of objects (starts at 1, [0] is unused)
	objOffsets []int
}

func (w *output) bytes(b []byte) {
	if w.err != nil { // write is now a no-op
		return
	}
	n, err := w.dst.Write(b)
	if err != nil {
		w.err = err
		return
	}
	w.written += n
}

// createObject return a new reference
// and grow the `objOffsets` accordingly.
// This is needed to write objects that must reference their "parent".
func (w *output) createObject() Reference {
	ref := Reference(len(w.objOffsets)) // last object is at len(objOffsets) - 1
	w.objOffsets = append(w.objOffsets, 0)
	return ref
}

func (w *output) writeHeader() {
	w.bytes([]byte("%PDF-1.7\n"))
	// If a PDF file contains binary data, as most do (see 7.2, "Lexical Conventions"), the header line shall be
	// immediately followed by a comment line containing at least four binary characters—that is, characters whose
	// codes are 128 or greater.
	w.bytes([]byte("%"))
	w.bytes([]byte{200, 200, 200, 200})
	w.bytes([]byte("\n"))
}

func (w *output) writeFooter(trailer Trailer, root, info, encrypt Reference) {
	var b bytes.Buffer
	// Cross-ref
	o, n := w.written, len(w.objOffsets)-1
	b.WriteString("xref\n")
	b.WriteString(fmt.Sprintf("0 %d\n", n+1))
	b.WriteString("0000000000 65535 f \n")
	for j := 1; j <= n; j++ {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", w.objOffsets[j]))
	}
	// Trailer
	b.WriteString("trailer\n")
	b.WriteString("<<\n")
	b.WriteString(fmt.Sprintf("/Size %d\n", n+1))
	b.WriteString(fmt.Sprintf("/Root %d 0 R\n", root))
	b.WriteString(fmt.Sprintf("/Info %d 0 R\n", info))
	if encrypt > 0 {
		b.WriteString(fmt.Sprintf("/Encrypt %s\n", encrypt))
		b.WriteString(fmt.Sprintf("/ID [%s %s]\n",
			escapeFormatByteString(trailer.ID[0]), escapeFormatByteString(trailer.ID[1])))
	}
	b.WriteString(">>\n")
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n", o))
	b.WriteString("%%EOF")
	w.bytes(b.Bytes())
}

// pdfWriter uses an output and an internal cache
// to write a Document.
// The internal cache avoids duplication of indirect object,
// by associating an object number to a pointer
type pdfWriter struct {
	*output

	cache     map[Referenceable]Reference
	pages     map[PageNode]Reference
	outlines  map[*OutlineItem]Reference
	fields    map[*FormFieldDict]Reference
	structure map[*StructureElement]Reference

	encrypt *Encrypt
}

func newWriter(dest io.Writer, encrypt *Encrypt) pdfWriter {
	return pdfWriter{
		output:   &output{dst: dest, objOffsets: []int{0}},
		cache:    make(map[Referenceable]Reference),
		pages:    make(map[PageNode]Reference),
		outlines: make(map[*OutlineItem]Reference),
		fields:   make(map[*FormFieldDict]Reference),
		encrypt:  encrypt,
	}
}

type PDFStringEncoding uint8

const (
	ByteString PDFStringEncoding = iota // no special treatment, except escaping
	// ASCIIString                          // ASCII encoding and escaping
	HexString  // hex form
	TextString // one of the PDF encoding: PDFDocEncoding or UTF16-BE
)

var (
	replacer = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\r", "\\r")
	utf16Enc = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
)

// EncodeString transforms an UTF-8 string `s` to satisfy the PDF
// format required by `mode`.
// It will also encrypt `s`, if needed, using
// `context`, which is the object number of the containing object.
type PDFStringEncoder interface {
	// EncodeString may panic if `mode` is not one of `ByteString`, `HexString`, `TextString`
	EncodeString(s string, mode PDFStringEncoding, context Reference) string
}

// return a pdf compatible string
// this function should't generaly be used (see EncodeString)
// but is useful when strings must not be encrypted
func escapeFormatByteString(s string) string {
	s = replacer.Replace(s)
	return "(" + s + ")"
}

func (p pdfWriter) EncodeString(s string, mode PDFStringEncoding, context Reference) string {
	if p.err != nil {
		return ""
	}

	if mode == TextString {
		// TODO: choose between utf16 and pdfencoding
		var err error
		s, err = utf16Enc.NewEncoder().String(s)
		if err != nil {
			p.err = fmt.Errorf("invalid text string %s: %w", s, err)
			return ""
		}
	}

	if p.encrypt != nil && p.encrypt.EncryptionHandler != nil {
		sb := []byte(s)
		p.encrypt.EncryptionHandler.crypt(context, sb)
		s = string(sb)
	}

	switch mode {
	case ByteString, TextString:
		return escapeFormatByteString(s) // string litteral
	case HexString:
		return "<" + hex.EncodeToString([]byte(s)) + ">" // hex string
	default:
		panic("invalid encoding mode")
	}
}

// writeObject write the content of the object `ref`, and update the offsets.
// This method will be called at most once for each reference.
// For stream object, `content` will contain the dictionary,
// and `stream` the inner stream bytes. For other objects, `stream` will be nil.
// Stream content will be encrypted if needed.
func (w pdfWriter) writeObject(content string, stream []byte, ref Reference) {
	w.objOffsets[ref] = w.written
	w.bytes([]byte(fmt.Sprintf("%d 0 obj\n", ref)))
	w.bytes([]byte(content))
	if stream != nil {
		w.bytes([]byte("\nstream\n"))
		if w.encrypt != nil && w.encrypt.EncryptionHandler != nil {
			// we must ensure we dont modify the original stream
			// which may be a Stream.Content slice
			stream = append([]byte(nil), stream...)
			w.encrypt.EncryptionHandler.crypt(ref, stream)
		}
		w.bytes(stream)
		// There should be an end-of-line marker after the data and before endstream
		w.bytes([]byte("\nendstream"))
	}
	w.bytes([]byte("\nendobj\n"))
}

// addObject is a convenience shortcut to write `content` into a new object
// and return the created reference
func (p pdfWriter) addObject(content string, stream []byte) Reference {
	ref := p.createObject()
	p.writeObject(content, stream, ref)
	return ref
}

// writerCache

// Referenceable is a private interface implemented
// by the structures idenfied by pointers.
// For such a structure, two usage of the same pointer
// in a `Document` will be written in the PDF file using the same
// object number, avoiding unnecessary duplications.
type Referenceable interface {
	IsReferenceable()
	// clone returns a deep copy, preserving the concrete type
	// it will use the `cache` for child items which are themselves
	// `Referenceable`
	// see cloneCache.checkOrClone to avoid unwanted allocations
	clone(cache cloneCache) Referenceable
	pdfContent(pdf pdfWriter, objectRef Reference) (content string, stream []byte)
}

func (*FontDict) IsReferenceable()           {}
func (*GraphicState) IsReferenceable()       {}
func (*SimpleEncodingDict) IsReferenceable() {}
func (*AnnotationDict) IsReferenceable()     {}
func (*FileSpec) IsReferenceable()           {}
func (*EmbeddedFileStream) IsReferenceable() {}
func (*ShadingDict) IsReferenceable()        {}
func (*FunctionDict) IsReferenceable()       {}
func (*PatternTiling) IsReferenceable()      {}
func (*PatternShading) IsReferenceable()     {}
func (*ColorSpaceICCBased) IsReferenceable() {}
func (*ColorTableStream) IsReferenceable()   {}
func (*XObjectForm) IsReferenceable()        {}
func (*XObjectImage) IsReferenceable()       {}

// check the cache and write a new item if not found
func (pdf pdfWriter) addItem(item Referenceable) Reference {
	if ref, has := pdf.cache[item]; has {
		return ref
	}
	ref := pdf.createObject()
	pdf.cache[item] = ref
	s, b := item.pdfContent(pdf, ref)
	pdf.writeObject(s, b, ref)
	return ref
}
