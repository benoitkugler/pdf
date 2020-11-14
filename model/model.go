// Implements the in-memory structure of the PDFs object
// Whenever possible, use static types.
// The structure is not directly the one found or written
// in a PDF, but it serves as an intermediate representation
// to facilitate PDF modifications.
// The entry point of the package is the type `Document`.
package model

import (
	"io"
	"time"
)

// Document is the top-level object,
// representing a whole PDF file.
type Document struct {
	Trailer Trailer
	Catalog Catalog
}

// Write walks the entire document and writes its content
// into `output`, producing a valid PDF file.
func (doc Document) Write(output io.Writer) error {
	wr := newWriter(output)

	wr.writeHeader()

	root := wr.CreateObject()
	wr.WriteObject(doc.Catalog.pdfString(wr, root), nil, root)
	info := wr.addObject(doc.Trailer.Info.pdfString(wr), nil)

	wr.writeFooter(doc.Trailer.Encrypt, root, info)

	return wr.err
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
// `catalog` is needed by the potential signature fields
func (cat Catalog) pdfString(pdf pdfWriter, catalog Reference) string {
	b := newBuffer()
	b.line("<<\n/Type/Catalog")

	// Some pages may need to know in advance the
	// object number of an arbitrary page, such as annotation link
	// with GoTo actions
	// So, we first walk the tree to associate an object number
	// to each pages, and we generate the content in a second pass
	// (at this point, the cache `pages` is filled)
	cat.Pages.allocateReferences(pdf)

	pageRef := pdf.CreateObject()
	content := cat.Pages.pdfString(pdf, pageRef, -1)
	pdf.WriteObject(content, nil, pageRef)
	b.line("/Pages %s", pageRef)

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
		ref := pdf.addObject(ac.pdfString(pdf, catalog), nil)
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
		b.fmt("/Producer %s\n", pdf.encodeString(t, textString))
	}
	if t := info.Title; t != "" {
		b.fmt("/Title %s\n", pdf.encodeString(t, textString))
	}
	if t := info.Subject; t != "" {
		b.fmt("/Subject %s\n", pdf.encodeString(t, textString))
	}
	if t := info.Author; t != "" {
		b.fmt("/Author %s\n", pdf.encodeString(t, textString))
	}
	if t := info.Keywords; t != "" {
		b.fmt("/Keywords %s\n", pdf.encodeString(t, textString))
	}
	if t := info.Creator; t != "" {
		b.fmt("/Creator %s\n", pdf.encodeString(t, textString))
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
