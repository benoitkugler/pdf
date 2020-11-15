// Implements the in-memory structure of the PDFs object, using static types.
// The structure is not directly the one found or written
// in a PDF, but it serves as an intermediate representation
// to facilitate PDF modifications.
// The entry point of the package is the type `Document`.
package model

import (
	"fmt"
	"io"
	"time"
)

// Document is the top-level object,
// representing a whole PDF file.
// Where a PDF file use indirect object to
// link data together, `Document` uses Go pointers,
// making easier to analyse and mutate a document.
// See the package `reader` to create a new `Document`
// from an existing PDF file.
type Document struct {
	Trailer Trailer
	Catalog Catalog
}

// Write walks the entire document and writes its content
// into `output`, producing a valid PDF file.
func (doc Document) Write(output io.Writer) error {
	wr := newWriter(output, doc.Trailer.Encrypt)

	wr.writeHeader()

	root := wr.createObject()
	wr.writeObject(doc.Catalog.pdfString(wr, root), nil, root)
	info := wr.createObject()
	wr.writeObject(doc.Trailer.Info.pdfString(wr, info), nil, info)

	wr.writeFooter(doc.Trailer.Encrypt, root, info)

	return wr.err
}

// Catalog contains the main contents of the document.
// See especially the `Pages` tree, the `AcroForm` form
// and the `Outlines` tree.
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
func (cat Catalog) pdfString(pdf pdfWriter, catalog reference) string {
	b := newBuffer()
	b.line("<<\n/Type/Catalog")

	// Some pages may need to know in advance the
	// object number of an arbitrary page, such as annotation link
	// with GoTo actions
	// So, we first walk the tree to associate an object number
	// to each pages, and we generate the content in a second pass
	// (at this point, the cache `pages` is filled)
	cat.Pages.allocateReferences(pdf)

	pageRef := pdf.createObject()
	content := cat.Pages.pdfString(pdf, pageRef, -1)
	pdf.writeObject(content, nil, pageRef)
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
		ref := pdf.createObject()
		pdf.writeObject(ac.pdfString(pdf, catalog, ref), nil, ref)
		b.line("/AcroForm %s", ref)
	}
	if outline := cat.Outlines; outline != nil {
		outlineRef := pdf.createObject()
		pdf.writeObject(outline.pdfString(pdf, outlineRef), nil, outlineRef)
		b.line("/Outlines %s", outlineRef)
	}
	b.fmt(">>")

	return b.String()
}

// NameDictionary establish the correspondence between names and objects
type NameDictionary struct {
	EmbeddedFiles EmbeddedFileTree
	Dests         *DestTree // optional
	// AP
}

// ViewerPreferences specifies the way the document shall be
// displayed on the screen.
// TODO: ViewerPreferences extend the fields
type ViewerPreferences struct {
	FitWindow    bool
	CenterWindow bool
}

func (p ViewerPreferences) pdfString(pdf pdfWriter) string {
	return fmt.Sprintf("<</FitWindow %v /CenterWindow%v>>", p.FitWindow, p.CenterWindow)
}

type Trailer struct {
	//TODO: check Prev field
	Encrypt Encrypt
	Info    Info
	ID      [2]string // optional (must be not crypted, direct objects)
}

// Info contains metadata about the document
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
func (info Info) pdfString(pdf pdfWriter, ref reference) string {
	b := newBuffer()
	b.fmt("<<\n")
	if t := info.Producer; t != "" {
		b.fmt("/Producer %s\n", pdf.EncodeString(t, TextString, ref))
	}
	if t := info.Title; t != "" {
		b.fmt("/Title %s\n", pdf.EncodeString(t, TextString, ref))
	}
	if t := info.Subject; t != "" {
		b.fmt("/Subject %s\n", pdf.EncodeString(t, TextString, ref))
	}
	if t := info.Author; t != "" {
		b.fmt("/Author %s\n", pdf.EncodeString(t, TextString, ref))
	}
	if t := info.Keywords; t != "" {
		b.fmt("/Keywords %s\n", pdf.EncodeString(t, TextString, ref))
	}
	if t := info.Creator; t != "" {
		b.fmt("/Creator %s\n", pdf.EncodeString(t, TextString, ref))
	}
	if t := info.CreationDate; !t.IsZero() {
		b.fmt("/CreationDate %s\n", pdf.dateString(t, ref))
	}
	if t := info.ModDate; !t.IsZero() {
		b.fmt("/ModDate %s\n", pdf.dateString(t, ref))
	}
	b.fmt(">>")
	return b.String()
}

// EncryptionAlgorithm is a code specifying the algorithm to be used in encrypting and
// decrypting the document
type EncryptionAlgorithm uint8

const (
	Undocumented EncryptionAlgorithm = iota
	AES
	AESExt // encryption key with length greater than 40
	Unpublished
	InDocument
)

// Encrypt stores the encryption-related information
type Encrypt struct {
	Filter    Name
	SubFilter Name
	V         EncryptionAlgorithm
	Length    int
	CF        map[Name]CrypFilter // optional
	StmF      Name                // optional
	StrF      Name                // optional
	EFF       Name                // optional
}

type CrypFilter struct {
	CFM       Name // optional
	AuthEvent Name // optional
	Length    int  // optional

	// byte strings, required for public-key security handlers
	// for Crypt filter decode parameter dictionary,
	// it's a one element array, written in PDF directly as a string
	Recipients      []string
	EncryptMetadata bool // optional, default to false
}
