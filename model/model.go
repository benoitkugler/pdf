// Implements the in-memory structure of the PDFs object, using static types.
// The structure is not directly the one found or written
// in a PDF, but it serves as an intermediate representation
// to facilitate PDF modifications. Still, this library supports
// the majority of the PDF specification.
//
// This package aims at being used without having to think (to much)
// of the PDF implementations details. In particular, unless stated otherwise,
// all the strings should be UTF-8 encoded. The library
// will take care to encode them when needed. They are a few exceptions, where
// ASCII strings are required: it is then up to the user to make sure
// the given string is ASCII.
// TODO: remove encoding hints from comments of string fields
//
// The entry point of the package is the type `Document`.
package model

import (
	"fmt"
	"io"
	"strings"
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

	// // UserPassword, OwnerPassword are not directly part
	// // of the PDF document, but are used to protect (encrypt)
	// // the contents.
	// UserPassword, OwnerPassword string
}

// Clone returns a deep copy of the document.
// It may be useful when we want to load
// a 'template' document once at server startup, and then
// modyfing it for each request.
// For every type implementing `Referenceable`, the equalities
// between pointers are preserved.
func (doc Document) Clone() Document {
	out := doc
	out.Trailer = doc.Trailer.Clone()
	out.Catalog = doc.Catalog.Clone()
	return doc
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

	wr.writeFooter(doc.Trailer, root, info)

	return wr.err
}

// Catalog contains the main contents of the document.
// See especially the `Pages` tree, the `AcroForm` form
// and the `Outlines` tree.
type Catalog struct {
	Pages             PageTree
	Extensions        Extensions
	Names             NameDictionary     // optional
	ViewerPreferences *ViewerPreferences // optional
	AcroForm          *AcroForm          // optional
	Dests             *DestTree          // optional
	PageLabels        *PageLabelsTree    // optional
	Outlines          *Outline           // optional
	StructTreeRoot    *StructureTree     // optional
	MarkInfo          *MarkDict          // optional
	PageLayout        Name               // optional
	PageMode          Name               // optional
}

// returns the Dictionary of `cat`
// `catalog` is needed by the potential signature fields
func (cat Catalog) pdfString(pdf pdfWriter, catalog Reference) string {
	// some pages may need to know in advance the
	// object number of an arbitrary page, such as annotation link
	// with GoTo actions
	// so, we first walk the tree to associate an object number
	// to each pages, so that the second pass can use the map in `pdf`
	pdf.allocateReferences(&cat.Pages)

	b := newBuffer()
	b.line("<<\n/Type/Catalog")
	pageRef := pdf.pages[&cat.Pages]
	pdf.writeObject(cat.Pages.pdfString(pdf), nil, pageRef)
	b.line("/Pages %s", pageRef)

	if pLabel := cat.PageLabels; pLabel != nil {
		labelsRef := pdf.createObject()
		pdf.writeObject(pLabel.pdfString(pdf, labelsRef), nil, labelsRef)
		b.line("/PageLabels %s", labelsRef)
	}

	b.line("/Names %s", cat.Names.pdfString(pdf))

	if dests := cat.Dests; dests != nil {
		ref := pdf.createObject()
		pdf.writeObject(dests.pdfString(pdf, ref), nil, ref)
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
	if cat.StructTreeRoot != nil {
		stRef := pdf.createObject()
		pdf.writeObject(cat.StructTreeRoot.pdfString(pdf, stRef), nil, stRef)
		b.line("/StructTreeRoot %s", stRef)
	}
	if m := cat.MarkInfo; m != nil {
		b.line("/MarkInfo %s", m)
	}
	b.fmt(">>")

	return b.String()
}

type cloneCache struct {
	refs      map[Referenceable]Referenceable
	pages     map[PageNode]PageNode // concrete type are preserved
	fields    map[*FormFieldDict]*FormFieldDict
	structure map[*StructureElement]*StructureElement
	// outlines map[*OutlineItem]*OutlineItem
}

func newCloneCache() cloneCache {
	return cloneCache{
		refs:      make(map[Referenceable]Referenceable),
		pages:     make(map[PageNode]PageNode),
		fields:    make(map[*FormFieldDict]*FormFieldDict),
		structure: make(map[*StructureElement]*StructureElement),
		// outlines: make(map[*OutlineItem]*OutlineItem),
	}
}

// convenience function to check if the object
// is already cloned and return the clone object, or do the cloning.
// the concrete type of `origin` is preserved, so that the return
// value can be type-asserted back to its original concrete type
func (cache cloneCache) checkOrClone(origin Referenceable) Referenceable {
	if cloned := cache.refs[origin]; cloned != nil {
		return cloned
	}
	out := origin.clone(cache)
	cache.refs[origin] = out // update the cache
	return out
}

// Clone returns a deep copy of the catalog.
func (cat Catalog) Clone() Catalog {
	cache := newCloneCache()
	// Some pages may need to know in advance the
	// pointer to an arbitrary cloned page, such as annotation link
	// with GoTo actions
	// So, we first walk the tree to allocate new memory
	// to each pages, and we do the actual cloning in a second pass
	// (at this point, the cache `pages` is filled)
	cache.allocateClones(&cat.Pages)

	out := cat
	outPage := cat.Pages.clone(cache).(*PageTree)
	out.Pages = *outPage
	out.Names = cat.Names.clone(cache)
	if cat.ViewerPreferences != nil {
		v := *cat.ViewerPreferences
		out.ViewerPreferences = &v
	}
	out.AcroForm = cat.AcroForm.clone(cache)
	if de := cat.Dests; de != nil {
		des := de.clone(cache)
		out.Dests = &des
	}
	if cat.PageLabels != nil {
		pl := out.PageLabels.Clone()
		cat.PageLabels = &pl
	}
	out.Outlines = cat.Outlines.clone(cache)
	out.StructTreeRoot = cat.StructTreeRoot.clone(cache)
	if cat.MarkInfo != nil {
		m := *cat.MarkInfo
		out.MarkInfo = &m
	}
	return cat
}

// NameDictionary establish the correspondence between names and objects
// TODO: add more names
type NameDictionary struct {
	EmbeddedFiles EmbeddedFileTree
	Dests         *DestTree // optional
	// AP
}

func (n NameDictionary) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.WriteString("<<")
	if dests := n.Dests; dests != nil {
		ref := pdf.createObject()
		pdf.writeObject(dests.pdfString(pdf, ref), nil, ref)
		b.fmt("/Dests %s", ref)
	}
	if emb := n.EmbeddedFiles; emb != nil {
		ref := pdf.createObject()
		pdf.writeObject(emb.pdfString(pdf, ref), nil, ref)
		b.fmt("/EmbeddedFiles %s", ref)
	}
	b.WriteString(">>")
	return b.String()
}

func (n NameDictionary) clone(cache cloneCache) NameDictionary {
	out := n
	out.EmbeddedFiles = n.EmbeddedFiles.clone(cache)
	if n.Dests != nil {
		ds := n.Dests.clone(cache)
		out.Dests = &ds
	}
	return out
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

func (t Trailer) Clone() Trailer {
	out := t
	out.Encrypt = t.Encrypt.Clone()
	return out
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
func (info Info) pdfString(pdf pdfWriter, ref Reference) string {
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
	_ EncryptionAlgorithm = iota
	Key40
	KeyExt // encryption key with length greater than 40
	_
	KeySecurityHandler
)

// Encrypt stores the encryption-related information
// It will be filled when reading an existing PDF document.
// Note that to encrypt a document when writting it,
// a call to Trailer.SetStandardEncryptionHandler is needed
// (partly because password are needed, which are not contained in the PDF).
type Encrypt struct {
	EncryptionHandler EncryptionHandler
	Filter            Name
	SubFilter         Name
	V                 EncryptionAlgorithm
	// in bytes, from 5 to 16, optional, default to 5
	// written in pdf as bit length
	Length uint8
	CF     map[Name]CrypFilter // optional
	StmF   Name                // optional
	StrF   Name                // optional
	EFF    Name                // optional
	P      UserPermissions
}

func (e Encrypt) Clone() Encrypt {
	out := e
	if e.EncryptionHandler != nil {
		out.EncryptionHandler = e.EncryptionHandler.Clone()
	}
	if e.CF != nil { // preserve reflet.DeepEqual
		out.CF = make(map[Name]CrypFilter, len(e.CF))
		for k, v := range e.CF {
			out.CF[k] = v.Clone()
		}
	}
	return out
}

func (e Encrypt) pdfString() string {
	b := newBuffer()
	b.fmt("<</Filter %s /SubFilter %s /V %d /Length %d", e.Filter, e.SubFilter, e.V, e.Length)
	if e.EncryptionHandler != nil {
		b.line(e.EncryptionHandler.encryptionAddFields())
	}
	if e.CF != nil {
		b.fmt("/CF <<")
		for n, v := range e.CF {
			b.fmt("%s %s ", n, v.pdfString(true))
		}
		b.line(">>")
	}
	if e.StmF != "" {
		b.fmt("/StmF %s", e.StmF)
	}
	if e.StrF != "" {
		b.fmt("/StrF %s", e.StrF)
	}
	if e.EFF != "" {
		b.fmt("/EFF %s", e.EFF)
	}
	b.fmt("/P %d>>", e.P)
	return b.String()
}

type CrypFilter struct {
	CFM       Name // optional
	AuthEvent Name // optional
	Length    int  // optional

	// byte strings, required for public-key security handlers
	// for Crypt filter decode parameter dictionary,
	// a one element array is written in PDF directly as a string
	Recipients []string
	// optional, default to false
	// written in PDF under the key /EncryptMetadata
	DontEncryptMetadata bool
}

func (c CrypFilter) pdfString(fromCrypt bool) string {
	out := "<<"
	if c.CFM != "" {
		out += "/CFM " + c.CFM.String()
	}
	if c.AuthEvent != "" {
		out += "/AuthEvent " + c.AuthEvent.String()
	}
	if c.Length != 0 {
		out += fmt.Sprintf("/Length %d", c.Length)
	}
	if fromCrypt && len(c.Recipients) == 1 {
		out += "/Recipients " + escapeFormatByteString(c.Recipients[0])
	}
	out += fmt.Sprintf("/EncryptMetadata %v>>", !c.DontEncryptMetadata)
	return out
}

// Clone returns a deep copy
func (c CrypFilter) Clone() CrypFilter {
	out := c
	out.Recipients = append([]string(nil), c.Recipients...)
	return out
}

//EncryptionHandler is either EncryptionStandard or EncryptionPublicKey
type EncryptionHandler interface {
	encryptionAddFields() string
	// Clone returns a deep copy, preserving the concrete type.
	Clone() EncryptionHandler
	// Crypt transform the incoming `data` in place, using `n`
	// as the object number of its context.
	Crypt(n Reference, data []byte)
}

// EncryptionPublicKey is written in PDF under the /Recipients key.
type EncryptionPublicKey []string

func (e EncryptionPublicKey) encryptionAddFields() string {
	chunks := make([]string, len(e))
	for i, s := range e {
		chunks[i] = escapeFormatByteString(s)
	}
	return fmt.Sprintf("/Recipients [%s]", strings.Join(chunks, " "))
}

func (e EncryptionPublicKey) Clone() EncryptionHandler {
	return append(EncryptionPublicKey(nil), e...)
}
