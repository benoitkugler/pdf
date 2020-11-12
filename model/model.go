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
	wr := newWriter(pdf)

	root = wr.addObject(doc.Catalog.pdfString(wr), nil)
	info = wr.addObject(doc.Trailer.Info.pdfString(wr), nil)

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
	Outlines          *Outline           // optional
	StructTreeRoot    *StructureTree     // optional
}

// returns the Dictionary of `cat`
func (cat Catalog) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.line("<<\n/Type /Catalog")

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
	b.line("/Pages %s", ref)

	if pLabel := cat.PageLabels; pLabel != nil {
		ref := pdf.addObject(pLabel.pdfString(pdf), nil)
		b.line("/PageLabels %s", ref)
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
	b.line(">>")

	if dests := cat.Dests; dests != nil {
		ref := pdf.addObject(dests.pdfString(pdf), nil)
		b.line("/Dests %s", ref)
	}
	if viewerPref := cat.ViewerPreferences; viewerPref != nil {
		ref := pdf.addObject(viewerPref.pdfString(pdf), nil)
		b.line("/ViewerPreferences %s", ref)
	}
	if p := cat.PageLayout; p != "" {
		b.line("/PageLayout %s", p)
	}
	if p := cat.PageMode; p != "" {
		b.line("/PageMode %s", p)
	}
	if ac := cat.AcroForm; ac != nil {
		ref := pdf.addObject(ac.pdfString(pdf), nil)
		b.line("/AcroForm %s", ref)
	}
	if outline := cat.Outlines; outline != nil {
		outlineRef := pdf.CreateObject()
		pdf.WriteObject(outline.pdfString(pdf, outlineRef), nil, outlineRef)
		b.line("/Outlines %s", outlineRef)
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
func (p ViewerPreferences) pdfString(pdf pdfWriter) string {
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

// pdfString return the Dictionary for `info`
func (info Info) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.fmt("<<\n")
	if t := info.Producer; t != "" {
		b.fmt("/Producer %s\n", pdf.EncodeString(t, TextString))
	}
	if t := info.Title; t != "" {
		b.fmt("/Title %s\n", pdf.EncodeString(t, TextString))
	}
	if t := info.Subject; t != "" {
		b.fmt("/Subject %s\n", pdf.EncodeString(t, TextString))
	}
	if t := info.Author; t != "" {
		b.fmt("/Author %s\n", pdf.EncodeString(t, TextString))
	}
	if t := info.Keywords; t != "" {
		b.fmt("/Keywords %s\n", pdf.EncodeString(t, TextString))
	}
	if t := info.Creator; t != "" {
		b.fmt("/Creator %s\n", pdf.EncodeString(t, TextString))
	}
	if t := info.CreationDate; !t.IsZero() {
		b.fmt("/CreationDate %s\n", pdf.dateString(t))
	}
	if t := info.ModDate; !t.IsZero() {
		b.fmt("/ModDate %s\n", pdf.dateString(t))
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
