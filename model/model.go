// Implements the in-memory structure of the PDFs object
// Whenever possible, use static types.
// The structure is not directly the one found or written
// in a PDF, but it serves as an intermediate representation
// to facilitate PDF modifications.
// The entry point of the package is the type `Document`.
package model

import (
	"time"
)

// Document is the top-level object,
// representing a whole PDF file.
type Document struct {
	Trailer Trailer
	Catalog Catalog
}

// Write walks the entire document and writes its content
// using `pdf` as an output.
// It returns two references, needed by a PDF writer to finalize
// the file (that is, to write the trailer)
func (doc Document) Write(pdf PDFOutput) (root, info Reference) {
	wr := PDFWriter{PDFOutput: pdf, cache: make(map[cachable]Reference), pages: make(map[*PageObject]Reference)}

	root = wr.addObject(doc.Catalog.pdfString(wr), nil)
	info = wr.addObject(doc.Trailer.Info.PDFString(pdf), nil)

	return root, info
}

type Catalog struct {
	Extensions        Extensions
	Pages             PageTree
	Names             NameDictionary     // optional
	ViewerPreferences *ViewerPreferences // optional
	PageLayout        Name               // optional
	PageMode          Name               // optional
	AcroForm          *AcroForm          // optional
	Dests             *DestTree          // optional
	PageLabels        *PageLabelsTree    // optional
	StructTreeRoot    *StructureTree     // optional
}

// returns the Dictionary of `cat`
func (cat Catalog) pdfString(pdf PDFWriter) string {
	b := newBuffer()
	b.fmt("<<\n/Type /Catalog\n")

	// Some pages may need to know in advance the
	// object number of an arbitrary page, such as annotation link
	// with GoTo actions
	// So, we first walk the tree to associate an object number
	// to each pages, and we generate the content in a second pass
	// (at this point, the cache `pages` is filled)
	cat.Pages.allocateReferences(pdf)

	ref := pdf.CreateObject()
	content := cat.Pages.pdfString(pdf, ref, -1)
	pdf.WriteObject(content, nil, ref)
	b.fmt("/Pages %s\n", ref)

	if pLabel := cat.PageLabels; pLabel != nil {
		ref := pdf.addObject(pLabel.pdfString(pdf), nil)
		b.fmt("/PageLabels %s\n", ref)
	}

	b.fmt("/Names <<")
	if dests := cat.Names.Dests; dests != nil {
		ref := pdf.addObject(dests.pdfString(pdf), nil)
		b.fmt("/Dests %s", ref)
	}
	if emb := cat.Names.EmbeddedFiles; emb != nil {
		ref := pdf.addObject(emb.pdfString(pdf), nil)
		b.fmt("/EmbeddedFiles %s", ref)
	}
	b.fmt(">>\n")

	if dests := cat.Dests; dests != nil {
		ref := pdf.addObject(dests.pdfString(pdf), nil)
		b.fmt("/Dests %s\n", ref)
	}
	if viewerPref := cat.ViewerPreferences; viewerPref != nil {
		ref := pdf.addObject(viewerPref.pdfString(pdf), nil)
		b.fmt("/ViewerPreferences %s\n", ref)
	}
	if p := cat.PageLayout; p != "" {
		b.fmt("/PageLayout %s\n", p)
	}
	if p := cat.PageMode; p != "" {
		b.fmt("/PageMode %s\n", p)
	}
	if ac := cat.AcroForm; ac != nil {
		ref := pdf.addObject(ac.pdfString(pdf), nil)
		b.fmt("/AcroForm %s\n", ref)
	}
	b.fmt(">>")

	return b.String()
}

type NameDictionary struct {
	EmbeddedFiles EmbeddedFileTree
	Dests         *DestTree // optional
	// AP
}

type ViewerPreferences struct {
	FitWindow    bool
	CenterWindow bool
}

// TODO: ViewerPreferences
func (p ViewerPreferences) pdfString(pdf PDFWriter) string {
	return "<<>>"
}

type Trailer struct {
	Encrypt Encrypt
	Info    Info
}

type Info struct {
	Producer     string
	Title        string
	Subject      string
	Author       string
	Keywords     string
	Creator      string
	CreationDate time.Time
	ModDate      time.Time
}

// PDFString return the Dictionary for `info`
func (info Info) PDFString(pdf PDFOutput) string {
	b := newBuffer()
	b.fmt("<<\n")
	if t := info.Producer; t != "" {
		b.fmt("/Producer %s\n", pdf.EncodeTextString(t))
	}
	if t := info.Title; t != "" {
		b.fmt("/Title %s\n", pdf.EncodeTextString(t))
	}
	if t := info.Subject; t != "" {
		b.fmt("/Subject %s\n", pdf.EncodeTextString(t))
	}
	if t := info.Author; t != "" {
		b.fmt("/Author %s\n", pdf.EncodeTextString(t))
	}
	if t := info.Keywords; t != "" {
		b.fmt("/Keywords %s\n", pdf.EncodeTextString(t))
	}
	if t := info.Creator; t != "" {
		b.fmt("/Creator %s\n", pdf.EncodeTextString(t))
	}
	if t := info.CreationDate; !t.IsZero() {
		b.fmt("/CreationDate %s\n", pdf.EncodeTextString(dateString(t)))
	}
	if t := info.ModDate; !t.IsZero() {
		b.fmt("/ModDate %s\n", pdf.EncodeTextString(dateString(t)))
	}
	b.fmt(">>")
	return b.String()
}

type EncryptionAlgorithm uint8

const (
	Undocumented EncryptionAlgorithm = iota
	AES
	AESExt // encryption key with length greater than 40
	Unpublished
	InDocument
)

type Encrypt struct {
	Filter    Name
	SubFilter Name
	V         EncryptionAlgorithm
	Length    int
}
