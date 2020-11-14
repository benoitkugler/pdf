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
type Reference int

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

// WriteObject write the content of the object `ref`, and update the offsets.
// This method will be called at most once for each reference.
// For stream object, `content` will contain the dictionary,
// and `stream` the inner stream bytes. For other objects, `stream` will be nil.
// Stream content will be encrypted if needed.
func (w *output) WriteObject(content string, stream []byte, ref Reference) {
	w.objOffsets[ref] = w.written
	w.bytes([]byte(fmt.Sprintf("%d 0 obj\n", ref)))
	w.bytes([]byte(content))
	if stream != nil { // TODO: encryption
		w.bytes([]byte("\nstream\n"))
		w.bytes(stream)
		if len(stream) > 0 && stream[len(stream)-1] != '\n' {
			// There should be an end-of-line marker after the data and before endstream
			w.bytes([]byte{'\n'})
		}
		w.bytes([]byte("endstream"))
	}
	w.bytes([]byte("\nendobj\n"))
}

// CreateObject return a new reference
// and grow the `objOffsets` accordingly.
// This is needed to write objects that must reference their "parent".
func (w *output) CreateObject() Reference {
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

func (w *output) writeFooter(encrypt Encrypt, root, info Reference) {
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
	if encrypt.V != 0 {
		// TODO:
		// b.WriteString("/Encrypt %d 0 R", f.protect.objNum)
		// f.out("/ID [()()]")
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

	cache    map[cachable]Reference
	pages    map[*PageObject]Reference
	outlines map[*OutlineItem]Reference
	fields   map[*FormField]Reference
}

func newWriter(dest io.Writer) pdfWriter {
	return pdfWriter{
		output:   &output{dst: dest, objOffsets: []int{0}},
		cache:    make(map[cachable]Reference),
		pages:    make(map[*PageObject]Reference),
		outlines: make(map[*OutlineItem]Reference),
		fields:   make(map[*FormField]Reference),
	}
}

type stringEncoding uint8

const (
	aSCIIString stringEncoding = iota // ASCII encoding and escaping
	byteString                        // no special treatment, except escaping
	hexString                         // hex form
	textString                        // one of the PDF encoding: PDFDocEncoding or UTF16-BE
)

var (
	replacer = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\r", "\\r")
	utf16Enc = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
)

// encodeString should transform an UTF-8 string `s` to satisfy the PDF
// format required by `mode`
// It should also encrypt `s`, if needed
func (p pdfWriter) encodeString(s string, mode stringEncoding) string {
	if p.err != nil {
		return ""
	}

	// TODO: encryption
	// if f.protect.encrypted {
	// 	b := []byte(s)
	// 	f.protect.rc4(uint32(f.n), &b)
	// 	s = string(b)
	// }

	switch mode {
	case aSCIIString, byteString: // TODO: check is we must ensure ASCII
		s = replacer.Replace(s)
		return "(" + s + ")"
	case hexString:
		return "<" + hex.EncodeToString([]byte(s)) + ">"
	case textString:
		s, err := utf16Enc.NewEncoder().String(s)
		if err != nil {
			p.err = fmt.Errorf("invalid text string %s: %w", s, err)
			return ""
		}
		s = replacer.Replace(s)
		return "(" + s + ")"
	default:
		panic("should be an exhaustive switch")
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

// check the cache and write a new item if not found
func (pdf pdfWriter) addItem(item cachable) Reference {
	if ref, has := pdf.cache[item]; has {
		return ref
	}
	ref := pdf.addObject(item.pdfContent(pdf))
	pdf.cache[item] = ref
	return ref
}
