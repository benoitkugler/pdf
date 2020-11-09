package writer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/text/encoding/unicode"
)

type writer struct {
	dst     io.Writer
	err     error // internal error, to defer error checking
	written int   // total number of bytes written to dst

	// encode the object numbers as index (starting from 1)
	// and the byte offsets of objects (starts at 1, [0] is unused)
	objOffsets []int

	doc model.Document

	// caches to avoid duplication of indirect object: ptr -> object number
	formFields        map[*model.FormField]ref
	appearanceDicts   map[*model.AppearanceDict]ref
	appearanceEntries map[*model.AppearanceEntry]ref
	xObjects          map[*model.XObject]ref
	resources         map[*model.ResourcesDict]ref
	fonts             map[*model.Font]ref
	graphicsStates    map[*model.GraphicState]ref
	encodings         map[*model.EncodingDict]ref
	annotations       map[*model.Annotation]ref
	fileSpecs         map[*model.FileSpec]ref
	fileContents      map[*model.EmbeddedFileStream]ref
	pages             map[*model.PageObject]ref
	shadings          map[*model.ShadingDict]ref
	functions         map[*model.Function]ref
	iccs              map[*model.ICCBasedColorSpace]ref
	patterns          map[model.Pattern]ref
}

func newWriter(dest io.Writer) *writer {
	return &writer{dst: dest, objOffsets: []int{0},
		formFields:        make(map[*model.FormField]ref),
		appearanceDicts:   make(map[*model.AppearanceDict]ref),
		appearanceEntries: make(map[*model.AppearanceEntry]ref),
		xObjects:          make(map[*model.XObject]ref),
		resources:         make(map[*model.ResourcesDict]ref),
		fonts:             make(map[*model.Font]ref),
		graphicsStates:    make(map[*model.GraphicState]ref),
		encodings:         make(map[*model.EncodingDict]ref),
		annotations:       make(map[*model.Annotation]ref),
		fileSpecs:         make(map[*model.FileSpec]ref),
		fileContents:      make(map[*model.EmbeddedFileStream]ref),
		pages:             make(map[*model.PageObject]ref),
		shadings:          make(map[*model.ShadingDict]ref),
		functions:         make(map[*model.Function]ref),
		iccs:              make(map[*model.ICCBasedColorSpace]ref),
		patterns:          make(map[model.Pattern]ref),
	}
}

func (w *writer) bytes(b []byte) {
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

func (w *writer) write(b buffer) {
	w.bytes((*bytes.Buffer)(&b).Bytes())
}

// writeObject write the content, and update the offsets
// ref must have been allocated previously
func (w *writer) writeObject(content []byte, ref ref) {
	w.objOffsets[ref] = w.written
	w.bytes([]byte(fmt.Sprintf("%d 0 obj\n", ref)))
	w.bytes(content)
	w.bytes([]byte("\nendobj\n"))
}

// allocate a new object and imediatly write its content
func (w *writer) defineObject(content []byte) ref {
	ref := w.allocateObject()
	w.writeObject(content, ref)
	return ref
}

// allocateObjects return a new reference
// and grow the `objOffsets` accordingly
// which shoud be update later on
// this is needed to write objects that must reference their "parent"
func (w *writer) allocateObject() ref {
	ref := ref(len(w.objOffsets)) // last object is at len(objOffsets) - 1
	w.objOffsets = append(w.objOffsets, 0)
	return ref
}

type ref int

func (r ref) String() string {
	return fmt.Sprintf("%d 0 R", r)
}

type buffer bytes.Buffer

func (b *buffer) fmt(format string, arg ...interface{}) {
	fmt.Fprintf((*bytes.Buffer)(b), format, arg...)
}

func (b *buffer) bytes() []byte {
	return (*bytes.Buffer)(b).Bytes()
}

func (w *writer) writeHeader() {
	w.bytes([]byte("%PDF-1.7\n"))
}

func (w *writer) writeFooter(root, info ref) {
	var b buffer
	// Cross-ref
	o, n := w.written, len(w.objOffsets)-1
	b.fmt("xref\n")
	b.fmt("0 %d\n", n+1)
	b.fmt("0000000000 65535 f \n")
	for j := 1; j <= n; j++ {
		b.fmt("%010d 00000 n \n", w.objOffsets[j])
	}
	// Trailer
	b.fmt("trailer\n")
	b.fmt("<<\n")
	b.fmt("/Size %d\n", n+1)
	b.fmt("/Root %d 0 R\n", root)
	b.fmt("/Info %d 0 R\n", info)
	if w.doc.Trailer.Encrypt.V != 0 {
		// TODO:
		// b.fmt("/Encrypt %d 0 R", f.protect.objNum)
		// f.out("/ID [()()]")
	}
	b.fmt(">>\n")
	b.fmt("startxref\n")
	b.fmt("%d\n", o)
	b.fmt("%%%%EOF")
	w.write(b)
}

func (w *writer) writeCatalog() (root ref) {
	var b buffer
	cat := w.doc.Catalog
	b.fmt("<<\n/Type /Catalog\n")

	ref := w.writePageTree(cat.Pages, -1)
	b.fmt("/Pages %s\n", ref)

	if pLabel := cat.PageLabels; pLabel != nil {
		ref := w.writePageLabels(*pLabel)
		b.fmt("/PageLabels %s\n", ref)
	}
	b.fmt("/Names <<")
	if dests := cat.Names.Dests; dests != nil {
		ref := w.writeNameDests(*dests)
		b.fmt("/Dests %s", ref)
	}
	if emb := cat.Names.EmbeddedFiles; emb != nil {
		ref := w.writeNameEmbeddedFiles(emb)
		b.fmt("/EmbeddedFiles %s", ref)
	}
	b.fmt(">>\n")
	if dests := cat.Dests; dests != nil {
		ref := w.writeDests(*dests)
		b.fmt("/Dests %s\n", ref)
	}
	if viewerPref := cat.ViewerPreferences; viewerPref != nil {
		ref := w.writeViewerPref(*viewerPref)
		b.fmt("/ViewerPreferences %s\n", ref)
	}
	if p := cat.PageLayout; p != "" {
		b.fmt("/PageLayout %s\n", p.PDFString())
	}
	if p := cat.PageMode; p != "" {
		b.fmt("/PageMode %s\n", p.PDFString())
	}
	if ac := cat.AcroForm; ac != nil {
		ref := w.writeAcroForm(*ac)
		b.fmt("/AcroForm %s\n", ref)
	}
	b.fmt(">>")

	return w.defineObject(b.bytes())
}

func (w *writer) writeInfo() (info ref) {
	var b buffer
	b.fmt("<<\n")
	inf := w.doc.Trailer.Info
	if t := inf.Producer; t != "" {
		b.fmt("/Producer %s\n", w.encodeTextString(t))
	}
	if t := inf.Title; t != "" {
		b.fmt("/Title %s\n", w.encodeTextString(t))
	}
	if t := inf.Subject; t != "" {
		b.fmt("/Subject %s\n", w.encodeTextString(t))
	}
	if t := inf.Author; t != "" {
		b.fmt("/Author %s\n", w.encodeTextString(t))
	}
	if t := inf.Keywords; t != "" {
		b.fmt("/Keywords %s\n", w.encodeTextString(t))
	}
	if t := inf.Creator; t != "" {
		b.fmt("/Creator %s\n", w.encodeTextString(t))
	}
	if t := inf.CreationDate; !t.IsZero() {
		b.fmt("/CreationDate %s\n", w.encodeTextString(dateString(t)))
	}
	if t := inf.ModDate; !t.IsZero() {
		b.fmt("/ModDate %s\n", w.encodeTextString(dateString(t)))
	}
	b.fmt(">>")
	return w.defineObject(b.bytes())
}

var (
	replacer = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\r", "\\r")
	utf16Enc = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
)

// encodeTextString use UTF16-BE to encode `s`
// it also espace charac and add ()
func (w *writer) encodeTextString(s string) string {
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

func Write(document model.Document, dest io.Writer) error {
	w := newWriter(dest)
	w.doc = document

	w.writeHeader()

	root := w.writeCatalog()

	info := w.writeInfo()

	w.writeFooter(root, info)

	return w.err
}
