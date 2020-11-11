package writer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/text/encoding/unicode"
)

var _ model.PDFOutput = (*output)(nil)

type output struct {
	dst     io.Writer
	err     error // internal error, to defer error checking
	written int   // total number of bytes written to dst

	// encode the object numbers as index (starting from 1)
	// and the byte offsets of objects (starts at 1, [0] is unused)
	objOffsets []int
}

func newWriter(dest io.Writer) *output {
	return &output{dst: dest, objOffsets: []int{0}}
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

var (
	replacer = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\r", "\\r")
	utf16Enc = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
)

// encodeTextString use UTF16-BE to encode `s`
// it also espace charac and add ()
func (w output) EncodeTextString(s string) string {
	if w.err != nil {
		return ""
	}
	s = replacer.Replace(s)
	s, err := utf16Enc.NewEncoder().String(s)
	if err != nil {
		w.err = fmt.Errorf("invalid text string %s: %w", s, err)
		return ""
	}
	//TODO:
	// if f.protect.encrypted {
	// 	b := []byte(s)
	// 	f.protect.rc4(uint32(f.n), &b)
	// 	s = string(b)
	// }
	return "(" + s + ")"
}

func (w output) ASCIIString(s string) string {
	if w.err != nil {
		return ""
	}
	s = replacer.Replace(s)
	//TODO:
	// if f.protect.encrypted {
	// 	b := []byte(s)
	// 	f.protect.rc4(uint32(f.n), &b)
	// 	s = string(b)
	// }
	return "(" + s + ")"
}

// WriteObject write the content, and update the offsets
// `ref` must have been allocated previously
func (w *output) WriteObject(content string, stream []byte, ref model.Reference) {
	w.objOffsets[ref] = w.written
	w.bytes([]byte(fmt.Sprintf("%d 0 obj\n", ref)))
	w.bytes([]byte(content))
	if stream != nil { // TODO: encryption
		w.bytes([]byte("\nstream\n"))
		w.bytes(stream)
		w.bytes([]byte("\nendstream"))
	}
	w.bytes([]byte("\nendobj\n"))
}

// CreateObject return a new reference
// and grow the `objOffsets` accordingly
func (w *output) CreateObject() model.Reference {
	ref := model.Reference(len(w.objOffsets)) // last object is at len(objOffsets) - 1
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

func (w *output) writeFooter(encrypt model.Encrypt, root, info model.Reference) {
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

// Write write a PDF file, described by `document`, into
// the writer `dest`
func Write(document model.Document, dest io.Writer) error {
	w := newWriter(dest)

	w.writeHeader()

	root, info := document.Write(w)

	w.writeFooter(document.Trailer.Encrypt, root, info)

	return w.err
}
