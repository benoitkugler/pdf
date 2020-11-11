package model

import (
	"fmt"
)

// PageNode is either a `PageTree` or a `PageObject`
type PageNode interface {
	isPageNode()
	count() int
}

func (*PageTree) isPageNode()   {}
func (*PageObject) isPageNode() {}

// PageTree describe the page hierarchy
// of a PDF file.
type PageTree struct {
	Parent    *PageTree
	Kids      []PageNode
	Resources *ResourcesDict // if nil, will be inherited from the parent

	countValue *int // cached for performance
}

// count returns the number of Page objects (leaf node)
// in all the descendants of `p` (not only in its direct children)
// Its result is cached, meaning that the page tree must not
// be mutated after calling it
func (p *PageTree) count() int {
	if p.countValue != nil {
		return *p.countValue
	}
	out := 0
	for _, kid := range p.Kids {
		out += kid.count()
	}
	p.countValue = &out
	return out
}

// Flatten returns all the leaf of the tree,
// respecting the indexing convention for pages (0-based):
// the page with index i is Flatten()[i].
// Be aware that inherited resource are not resolved
func (p PageTree) Flatten() []*PageObject {
	var out []*PageObject
	for _, kid := range p.Kids {
		switch kid := kid.(type) {
		case *PageTree:
			out = append(out, kid.Flatten()...)
		case *PageObject:
			out = append(out, kid)
		}
	}
	return out
}

// walk to associate an object number to each pages object (leaf)
// in the `pages` attribute of `pdf`
func (p PageTree) allocateReferences(pdf PDFWriter) {
	for _, kid := range p.Kids {
		switch kid := kid.(type) {
		case *PageTree:
			kid.allocateReferences(pdf)
		case *PageObject:
			pdf.pages[kid] = pdf.CreateObject()
		}
	}
}

// returns the Dictionary for `pages`.
// It requires a reference passed to its children, and its parent reference.
// `parentReference` will be negative (or zero) only for the root node.
func (pages *PageTree) pdfString(pdf PDFWriter, ownReference, parentRef Reference) string {
	kidRefs := make([]Reference, len(pages.Kids))
	for i, page := range pages.Kids {
		kidRefs[i] = writePageNode(pdf, page, ownReference)
	}
	parent := ""
	if parentRef > 0 {
		parent = fmt.Sprintf(" /Parent %s", parentRef)
	}
	res := ""
	if pages.Resources != nil {
		res = fmt.Sprintf(" /Resources %s", pages.Resources.PDFString(pdf))
	}
	content := fmt.Sprintf("<</Type /Pages /Count %d /Kids %s%s%s>>",
		pages.count(), writeRefArray(kidRefs), parent, res)
	return content
}

func writePageNode(pdf PDFWriter, page PageNode, parentRef Reference) Reference {
	switch page := page.(type) {
	case *PageTree:
		ownRef := pdf.CreateObject()
		content := page.pdfString(pdf, ownRef, parentRef)
		pdf.WriteObject(content, nil, ownRef)
		return ownRef
	case *PageObject:
		ref := pdf.pages[page] // previously allocated
		content := page.pdfString(pdf, parentRef)
		pdf.WriteObject(content, nil, ref)
		return ref
	default:
		panic("exhaustive switch")
	}

}

type PageObject struct {
	Parent                    *PageTree
	Resources                 *ResourcesDict  // if nil, will be inherited from the parent
	MediaBox                  *Rectangle      // if nil, will be inherited from the parent
	CropBox                   *Rectangle      // if nil, will be inherited. if still nil, default to MediaBox
	BleedBox, TrimBox, ArtBox *Rectangle      // if nil, default to CropBox
	Rotate                    *Rotation       // if nil, will be inherited from the parent. Only multiple of 90 are allowed
	Annots                    []*Annotation   // optional
	Contents                  []ContentStream // array of stream (often of length 1)
}

func (p PageObject) pdfString(pdf PDFWriter, parentReference Reference) string {
	b := newBuffer()
	b.line("<<")
	b.line("/Type /Page")
	b.line("/Parent %s", parentReference)
	if p.Resources != nil {
		b.line("/Resources %s", p.Resources.PDFString(pdf))
	}
	if p.MediaBox != nil {
		b.line("/MediaBox %s", p.MediaBox.PDFstring())
	}
	if p.CropBox != nil {
		b.line("/CropBox %s", p.CropBox.PDFstring())
	}
	if p.BleedBox != nil {
		b.line("/BleedBox %s", p.BleedBox.PDFstring())
	}
	if p.TrimBox != nil {
		b.line("/TrimBox %s", p.TrimBox.PDFstring())
	}
	if p.ArtBox != nil {
		b.line("/ArtBox %s", p.ArtBox.PDFstring())
	}
	if p.Rotate != nil {
		b.line("/Rotate %d", p.Rotate.Degrees())
	}
	annots := make([]Reference, len(p.Annots))
	for i, a := range p.Annots {
		annots[i] = pdf.addItem(a)
	}
	if len(p.Annots) != 0 {
		b.line("/Annots %s", writeRefArray(annots))
	}
	contents := make([]Reference, len(p.Contents))
	for i, c := range p.Contents {
		contents[i] = pdf.addObject(c.PDFContent())
	}
	if len(contents) != 0 {
		b.line("/Contents %s", writeRefArray(contents))
	}
	b.WriteString(">>")
	return b.String()
}

func (PageObject) count() int { return 1 }

type ResourcesDict struct {
	ExtGState  map[Name]*GraphicState // optionnal
	ColorSpace map[Name]ColorSpace    // optionnal
	Shading    map[Name]*ShadingDict  // optionnal
	Pattern    map[Name]Pattern       // optionnal
	Font       map[Name]*Font         // optionnal
	XObject    map[Name]XObject       // optionnal
}

func (r ResourcesDict) PDFString(pdf PDFWriter) string {
	b := newBuffer()
	b.line("<<")
	if r.ExtGState != nil {
		b.line("/ExtGState <<")
		for n, item := range r.ExtGState {
			ref := pdf.addItem(item)
			b.line("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.ColorSpace != nil {
		b.line("/ColorSpace <<")
		for n, item := range r.ColorSpace {
			cs := writeColorSpace(item, pdf)
			b.line("%s %s", n, cs)
		}
		b.line(">>")
	}
	if r.Shading != nil {
		b.line("/Shading <<")
		for n, item := range r.Shading {
			ref := pdf.addItem(item)
			b.line("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.Pattern != nil {
		b.line("/Pattern <<")
		for n, item := range r.Pattern {
			ref := pdf.addItem(item)
			b.line("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.Font != nil {
		b.line("/Font <<")
		for n, item := range r.Font {
			ref := pdf.addItem(item)
			b.line("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.XObject != nil {
		b.line("/XObject <<")
		for n, item := range r.XObject {
			ref := pdf.addItem(item)
			b.line("%s %s", n, ref)
		}
		b.line(">>")
	}
	b.fmt(">>")
	return b.String()
}

// ----------------------- structure -----------------------

type StructureTree struct {
	K          []*StructureElement // 1-array may be written in PDF directly as a dict
	IDTree     IDTree
	ParentTree ParentTree
	RoleMap    map[Name]Name
	ClassMap   map[Name][]AttributeObject // for each key, 1-array may be written in PDF directly
}

// An integer greater than any key in the parent tree, which shall be
// used as a key for the next entry added to the tree
func (s StructureTree) ParentTreeNextKey() int {
	high := s.ParentTree.Limits()[1]
	return high + 1
}

type StructureElement struct {
	S          Name
	P          *StructureElement // parent
	ID         string            // byte string
	Pg         *PageObject       // optional
	K          []ContentItem     // 1-array may be written in PDF directly
	A          []AttributeObject // 1-array may be written in PDF directly
	C          []Name            // 1-array may be written in PDF directly
	R          int               // optional
	T          string            // optional, text string
	Lang       string            // optional, text string
	Alt        string            // optional, text string
	E          string            // optional, text string
	ActualText string            // optional, text string
}

// ContentItem may be *StructureElement,
type ContentItem interface {
	//TODO:
	isContentItem()
}

//TODO:
type AttributeObject struct{}
