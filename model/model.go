// Implements the in-memory structure of a PDF document, using static types.
// The structure is not exactly the one found or written
// in a PDF, but it serves as an intermediate representation
// to facilitate PDF construction and modification. Still, this library supports
// the majority of the PDF specification.
//
// This package aims at being used without having to think (to much)
// of the PDF implementations details. In particular, unless stated otherwise,
// all the strings should be UTF-8 encoded. The library
// will take care to encode them when needed. They are a few exceptions, where
// ASCII strings are required: it is then up to the user to make sure
// the given string is ASCII.
//
// The entry point of the package is the type `Document`.
package model

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Document is the top-level object,
// representing a whole PDF file.
// Where a PDF file use indirect object to
// link data together, `Document` uses Go pointers,
// making easier to analyse and mutate a document.
// See the package `reader` to create a new `Document`
// from an existing PDF file.
// The zero value represents an empty PDF file.
type Document struct {
	Trailer Trailer
	Catalog Catalog

	// // UserPassword, OwnerPassword are not directly part
	// // of the PDF document, but are used to protect (encrypt)
	// // the contentstream.
	// UserPassword, OwnerPassword string
}

// Clone returns a deep copy of the document.
// It may be useful for example when we want to load
// a 'template' document once at server startup, and then
// modifyng it for each request.
// For every type implementing `Referenceable`, the equalities
// between pointers are preserved, meaning that if a pointer is
// used twice in the original document, it will also be used twice in
// the clone (and not duplicated).
func (doc *Document) Clone() Document {
	out := *doc
	out.Trailer = doc.Trailer.Clone()
	out.Catalog = doc.Catalog.Clone()
	return out
}

// Write walks the entire document and writes its content
// into `output`, producing a valid PDF file.
// `encryption` is an optional encryption dictionary,
// returned by `UseStandardEncryptionHandler`.
func (doc *Document) Write(output io.Writer, encryption *Encrypt) error {
	wr := newWriter(output, encryption)

	wr.writeHeader()

	root := wr.CreateObject()
	wr.WriteObject(doc.Catalog.pdfString(wr, root), root)
	info := wr.CreateObject()
	wr.WriteObject(doc.Trailer.Info.pdfString(wr, info), info)

	var encRef Reference
	if encryption != nil {
		encRef = wr.addObject(encryption.pdfString())
	}

	wr.writeFooter(doc.Trailer, root, info, encRef)

	return wr.err
}

// WriteFile writes the document in the given file.
// See method `Write` for more information.
func (doc *Document) WriteFile(filename string, encryption *Encrypt) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("can't create PDF file output: %s", err)
	}
	err = doc.Write(f, encryption)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("can't close PDF file output: %s", err)
	}
	return nil
}

// Catalog contains the main contents of the document.
// See especially the `Pages` tree, the `AcroForm` form
// and the `Outlines` tree.
type Catalog struct {
	Pages             PageTree
	Extensions        Extensions
	Names             NameDictionary               // optional
	ViewerPreferences *ViewerPreferences           // optional
	AcroForm          AcroForm                     // optional
	Dests             map[Name]DestinationExplicit // optional
	PageLabels        *PageLabelsTree              // optional
	Outlines          *Outline                     // optional
	StructTreeRoot    *StructureTree               // optional
	MarkInfo          *MarkDict                    // optional
	PageLayout        Name                         // optional
	PageMode          Name                         // optional
	// optional. A simple GoTo action to a direct destination
	// may be found as an array in a PDF file.
	OpenAction Action
	URI        string // optional, ASCII string, written in PDF as a dictionary
	Lang       string
}

// returns the Dictionary of `cat`
// `catalog` is needed by the potential signature fields
func (cat *Catalog) pdfString(pdf pdfWriter, catalog Reference) string {
	// some pages may need to know in advance the
	// object number of an arbitrary page, such as annotation link
	// with GoTo actions
	// so, we first walk the tree to associate an object number
	// to each pages, so that the second pass can use the map in `pdf`
	pdf.allocateReferences(&cat.Pages)

	b := newBuffer()
	b.line("<<\n/Type/Catalog")
	pageRef := pdf.pages[&cat.Pages]
	pdf.WriteObject(cat.Pages.pdfString(pdf), pageRef)
	b.line("/Pages %s", pageRef)

	if pLabel := cat.PageLabels; pLabel != nil {
		labelsRef := pdf.CreateObject()
		pdf.WriteObject(pLabel.pdfString(pdf, labelsRef), labelsRef)
		b.line("/PageLabels %s", labelsRef)
	}

	if names := cat.Names.pdfString(pdf); names != "<<>>" { // avoid writing empty dict
		b.line("/Names %s", names)
	}

	if dests := cat.Dests; len(dests) != 0 {
		b.line("/Dests <<")
		for name, dest := range dests {
			b.line("%s %s", name, dest.pdfDestination(pdf, catalog))
		}
		b.line(">>")
	}
	if viewerPref := cat.ViewerPreferences; viewerPref != nil {
		ref := pdf.addObject(viewerPref.pdfString(pdf))
		b.line("/ViewerPreferences %s", ref)
	}
	if p := cat.PageLayout; p != "" {
		b.line("/PageLayout %s", p)
	}
	if p := cat.PageMode; p != "" {
		b.line("/PageMode %s", p)
	}
	if ac := cat.AcroForm; len(ac.Fields) != 0 {
		ref := pdf.CreateObject()
		pdf.WriteObject(ac.pdfString(pdf, catalog, ref), ref)
		b.line("/AcroForm %s", ref)
	}
	if outline := cat.Outlines; outline != nil && outline.First != nil {
		outlineRef := pdf.CreateObject()
		pdf.WriteObject(outline.pdfString(pdf, outlineRef), outlineRef)
		b.line("/Outlines %s", outlineRef)
	}
	if cat.StructTreeRoot != nil {
		stRef := pdf.CreateObject()
		pdf.WriteObject(cat.StructTreeRoot.pdfString(pdf, stRef), stRef)
		b.line("/StructTreeRoot %s", stRef)
	}
	if m := cat.MarkInfo; m != nil {
		b.line("/MarkInfo %s", m)
	}
	if cat.URI != "" {
		b.line("/URI <</Base %s>>", pdf.EncodeString(cat.URI, ByteString, catalog))
	}
	if cat.OpenAction.ActionType != nil {
		b.line("/OpenAction %s", cat.OpenAction.pdfString(pdf, catalog))
	}
	if cat.Lang != "" {
		b.fmt("/Lang " + pdf.EncodeString(cat.Lang, TextString, catalog))
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
	if cat.Dests != nil {
		out.Dests = make(map[Name]DestinationExplicit, len(cat.Dests))
		for k, v := range cat.Dests {
			out.Dests[k] = v.clone(cache).(DestinationExplicit)
		}
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
	out.OpenAction = cat.OpenAction.clone(cache)
	return out
}

// NameDictionary establish the correspondence between names and objects.
// All fields are optional.
// TODO: add more names
type NameDictionary struct {
	EmbeddedFiles EmbeddedFileTree
	Dests         DestTree
	AP            AppearanceTree
	Pages         TemplateTree
	Templates     TemplateTree
}

func (n NameDictionary) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.WriteString("<<")
	if dests := n.Dests; !dests.IsEmpty() {
		ref := pdf.CreateObject()
		pdf.WriteObject(dests.pdfString(pdf, ref), ref)
		b.fmt("/Dests %s", ref)
	}
	if emb := n.EmbeddedFiles; len(emb) != 0 {
		ref := pdf.CreateObject()
		pdf.WriteObject(emb.pdfString(pdf, ref), ref)
		b.fmt("/EmbeddedFiles %s", ref)
	}
	if aps := n.AP; !aps.IsEmpty() {
		ref := pdf.CreateObject()
		pdf.WriteObject(aps.pdfString(pdf, ref), ref)
		b.fmt("/AP %s", ref)
	}
	if pages := n.Pages; !pages.IsEmpty() {
		ref := pdf.CreateObject()
		pdf.WriteObject(pages.pdfString(pdf, ref), ref)
		b.fmt("/Pages %s", ref)
	}
	if templates := n.Templates; !templates.IsEmpty() {
		ref := pdf.CreateObject()
		pdf.WriteObject(templates.pdfString(pdf, ref), ref)
		b.fmt("/Templates %s", ref)
	}
	b.WriteString(">>")
	return b.String()
}

func (n NameDictionary) clone(cache cloneCache) NameDictionary {
	out := n
	out.EmbeddedFiles = n.EmbeddedFiles.clone(cache)
	out.Dests = n.Dests.clone(cache)
	out.AP = n.AP.clone(cache)
	out.Pages = n.Pages.clone(cache)
	out.Templates = n.Templates.clone(cache)
	return out
}

// ViewerPreferences specifies the way the document shall be
// displayed on the screen.
// TODO: ViewerPreferences extend the fields
type ViewerPreferences struct {
	FitWindow    bool
	CenterWindow bool
	// right to left: determine the relative positioning
	// of pages when displayed side by side or printed n-up
	DirectionRTL bool
}

func (p ViewerPreferences) pdfString(pdf pdfWriter) string {
	direction := Name("L2R")
	if p.DirectionRTL {
		direction = "R2L"
	}
	return fmt.Sprintf("<</FitWindow %v /CenterWindow %v /Direction %s>>", p.FitWindow, p.CenterWindow, direction)
}

type Trailer struct {
	// TODO: check Prev field
	// Encrypt Encrypt
	Info Info
	ID   [2]string // optional (must be not crypted, direct objects)
}

func (t Trailer) Clone() Trailer {
	out := t
	// out.Encrypt = t.Encrypt.Clone()
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
