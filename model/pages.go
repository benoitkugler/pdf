package model

import (
	"fmt"
)

// PageNode is either a `PageTree` or a `PageObject`
type PageNode interface {
	isPageNode()
	// Count returns the number of Page objects (leaf node)
	// in all the descendants of `p` (including `p`)
	Count() int
	pdfString(pdf pdfWriter) string
	clone(cache cloneCache) PageNode
}

func (*PageTree) isPageNode()   {}
func (*PageObject) isPageNode() {}

// PageTree describe the page hierarchy
// of a PDF file.
type PageTree struct {
	Kids      []PageNode
	Resources *ResourcesDict // if nil, will be inherited from the parent

	parent *PageTree // cache, set up during pre-allocation
}

// Count returns the number of Page objects (leaf node)
// in all the descendants of `p` (not only in its direct children)
func (p *PageTree) Count() int {
	out := 0
	for _, kid := range p.Kids {
		out += kid.Count()
	}
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

// walk to associate an object number to each page nodes
// in the `pages` attribute of `pdf`
// also build up the parent to simplify the writing
// see catalog.pdfString for more details
func (pdf pdfWriter) allocateReferences(p *PageTree) {
	pdf.pages[p] = pdf.createObject()
	for _, kid := range p.Kids {
		switch kid := kid.(type) {
		case *PageTree:
			kid.parent = p
			pdf.allocateReferences(kid)
		case *PageObject:
			kid.parent = p
			pdf.pages[kid] = pdf.createObject()
		}
	}
}

// walk to associate a clone (new memory, unfiled) to each pages object (leaf)
// in the `pages` attribute of `cache`
// see catalog.clone for more details
func (cache cloneCache) allocateClones(p *PageTree) {
	cache.pages[p] = new(PageTree)
	for _, kid := range p.Kids {
		switch kid := kid.(type) {
		case *PageTree:
			cache.allocateClones(kid)
		case *PageObject:
			cache.pages[kid] = new(PageObject)
		}
	}
}

// returns the Dictionary for `pages`
// the `pdf.pages` map must have been previously filled
func (pages *PageTree) pdfString(pdf pdfWriter) string {
	kidRefs := make([]Reference, len(pages.Kids))
	for i, page := range pages.Kids {
		kidRef := pdf.pages[page]
		pdf.writeObject(page.pdfString(pdf), nil, kidRef)
		kidRefs[i] = kidRef
	}
	parent := ""
	if pages.parent != nil {
		parent = fmt.Sprintf("/Parent %s", pdf.pages[pages.parent])
	}
	res := ""
	if !pages.Resources.IsEmpty() {
		res = fmt.Sprintf("/Resources %s", pdf.addObject(pages.Resources.pdfString(pdf), nil))
	}
	content := fmt.Sprintf("<</Type/Pages/Count %d/Kids %s%s%s>>",
		pages.Count(), writeRefArray(kidRefs), parent, res)
	return content
}

func (p *PageTree) clone(cache cloneCache) PageNode {
	out := cache.pages[p].(*PageTree)
	// ignoring parent since it is not used until writing
	if p.Resources != nil {
		res := p.Resources.clone(cache)
		out.Resources = &res
	}
	if p.Kids != nil { // preserve reflect.DeepEqual
		out.Kids = make([]PageNode, len(p.Kids))
	}
	for i, k := range p.Kids {
		out.Kids[i] = k.clone(cache)
	}
	return out
}

// PageObject
// Since Widget annotation are only used with form fields,
// we choose to define them only in the AcroForm catalog entry.
// Thus, no AnnotationWidget should be added to the Annots entry
// (it will be done automatically when writing the PDF).
type PageObject struct {
	// TODO: complete fields
	Resources                 *ResourcesDict    // if nil, will be inherited from the parent
	MediaBox                  *Rectangle        // if nil, will be inherited from the parent
	CropBox                   *Rectangle        // if nil, will be inherited. if still nil, default to MediaBox
	BleedBox, TrimBox, ArtBox *Rectangle        // if nil, default to CropBox
	Rotate                    *Rotation         // if nil, will be inherited from the parent. Only multiple of 90 are allowed
	Annots                    []*AnnotationDict // optional, should not contain annotation widget
	Contents                  []ContentStream   // array of stream (often of length 1)
	StructParents             MaybeInt          // Required if the page contains structural content items
	Tabs                      Name              // optional, one of R , C or S

	parent *PageTree // cache, set up during pre-allocation
}

// AddFormFieldWidget creates a new form field widget
// and adds it both on the page (via the PageObject.Annots list) and to the
// form field tree (via the FormFieldDict.Widgets list)
func (p *PageObject) AddFormFieldWidget(f *FormFieldDict, base BaseAnnotation, widget AnnotationWidget) {
	annot := FormFieldWidget{AnnotationDict: &AnnotationDict{BaseAnnotation: base, Subtype: widget}}
	p.Annots = append(p.Annots, annot.AnnotationDict)
	f.Widgets = append(f.Widgets, annot)
}

// the pdf page map is used to fetch the object number
func (p *PageObject) pdfString(pdf pdfWriter) string {
	parentReference := pdf.pages[p.parent]
	b := newBuffer()
	b.line("<<")
	b.line("/Type/Page")
	b.line("/Parent %s", parentReference)
	if !p.Resources.IsEmpty() {
		b.line("/Resources %s", pdf.addObject(p.Resources.pdfString(pdf), nil))
	}
	if p.MediaBox != nil {
		b.line("/MediaBox %s", p.MediaBox.String())
	}
	if p.CropBox != nil {
		b.line("/CropBox %s", p.CropBox.String())
	}
	if p.BleedBox != nil {
		b.line("/BleedBox %s", p.BleedBox.String())
	}
	if p.TrimBox != nil {
		b.line("/TrimBox %s", p.TrimBox.String())
	}
	if p.ArtBox != nil {
		b.line("/ArtBox %s", p.ArtBox.String())
	}
	if p.Rotate != nil {
		b.line("/Rotate %d", p.Rotate.Degrees())
	}
	if len(p.Annots) != 0 {
		annots := make([]Reference, len(p.Annots))
		for i, a := range p.Annots {
			annots[i] = pdf.addItem(a)
		}
		b.line("/Annots %s", writeRefArray(annots))
	}
	contents := make([]Reference, len(p.Contents))
	for i, c := range p.Contents {
		contents[i] = pdf.addObject(c.PDFContent())
	}
	if len(contents) != 0 {
		b.line("/Contents %s", writeRefArray(contents))
	}
	if p.StructParents != nil {
		b.fmt("/StructParents %d", p.StructParents.(Int))
	}
	if p.Tabs != "" {
		b.fmt("/Tabs %s", p.Tabs)
	}
	b.WriteString(">>")
	return b.String()
}

// Count return the number of PageObject-that is 1
func (*PageObject) Count() int { return 1 }

// return a deep copy, with concrete type *PageObject
// cache.pages must have been previsouly filled
func (po *PageObject) clone(cache cloneCache) PageNode {
	out := cache.pages[po].(*PageObject)
	// ignoring parent since it is not used until writing
	if po.Resources != nil {
		res := po.Resources.clone(cache)
		out.Resources = &res
	}
	if po.MediaBox != nil {
		r := *po.MediaBox
		out.MediaBox = &r
	}
	if po.CropBox != nil {
		r := *po.CropBox
		out.CropBox = &r
	}
	if po.BleedBox != nil {
		r := *po.BleedBox
		out.BleedBox = &r
	}
	if po.TrimBox != nil {
		r := *po.TrimBox
		out.TrimBox = &r
	}
	if po.ArtBox != nil {
		r := *po.ArtBox
		out.ArtBox = &r
	}
	if po.Rotate != nil {
		r := *po.Rotate
		out.Rotate = &r
	}
	if po.Annots != nil { // preserve reflect.DeepEqual
		out.Annots = make([]*AnnotationDict, len(po.Annots))
	}
	for i, a := range po.Annots {
		out.Annots[i] = cache.checkOrClone(a).(*AnnotationDict)
	}
	if po.Contents != nil {
		out.Contents = make([]ContentStream, len(po.Contents))
	}
	for i, c := range po.Contents {
		out.Contents[i] = c.Clone()
	}
	return out
}

// ResourcesDict maps name to (indirect) ressources
type ResourcesDict struct {
	ExtGState  map[Name]*GraphicState // optionnal
	ColorSpace map[Name]ColorSpace    // optionnal
	Shading    map[Name]*ShadingDict  // optionnal
	Pattern    map[Name]Pattern       // optionnal
	Font       map[Name]*FontDict     // optionnal
	XObject    map[Name]XObject       // optionnal
}

// IsEmpty returns `true` is the resources pointer is either `nil`
// or all the map are empty; in this case it should not be written in the PDF file.
func (r *ResourcesDict) IsEmpty() bool {
	if r == nil {
		return true
	}
	return len(r.ExtGState) == 0 && len(r.ColorSpace) == 0 &&
		len(r.Shading) == 0 && len(r.Pattern) == 0 &&
		len(r.Font) == 0 && len(r.XObject) == 0
}

func (r *ResourcesDict) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.line("<<")
	if r.ExtGState != nil {
		b.fmt("/ExtGState <<")
		for n, item := range r.ExtGState {
			ref := pdf.addItem(item)
			b.fmt("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.ColorSpace != nil {
		b.fmt("/ColorSpace <<")
		for n, item := range r.ColorSpace {
			if item == nil {
				continue
			}
			b.fmt("%s %s", n, item.colorSpacePDFString(pdf))
		}
		b.line(">>")
	}
	if r.Shading != nil {
		b.fmt("/Shading <<")
		for n, item := range r.Shading {
			ref := pdf.addItem(item)
			b.fmt("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.Pattern != nil {
		b.fmt("/Pattern <<")
		for n, item := range r.Pattern {
			ref := pdf.addItem(item)
			b.fmt("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.Font != nil {
		b.fmt("/Font <<")
		for n, item := range r.Font {
			ref := pdf.addItem(item)
			b.fmt("%s %s", n, ref)
		}
		b.line(">>")
	}
	if r.XObject != nil {
		b.fmt("/XObject <<")
		for n, item := range r.XObject {
			ref := pdf.addItem(item)
			b.fmt("%s %s", n, ref)
		}
		b.line(">>")
	}
	b.fmt(">>")
	return b.String()
}

// clone returns a deep copy
func (r ResourcesDict) clone(cache cloneCache) ResourcesDict {
	var out ResourcesDict
	// to preserve reflect.DeepEqual, we check for nil maps before allocating
	if r.ExtGState != nil {
		out.ExtGState = make(map[Name]*GraphicState, len(r.ExtGState))
	}
	for n, v := range r.ExtGState {
		out.ExtGState[n] = cache.checkOrClone(v).(*GraphicState)
	}
	if r.ColorSpace != nil {
		out.ColorSpace = make(map[Name]ColorSpace, len(r.ColorSpace))
	}
	for n, v := range r.ColorSpace {
		out.ColorSpace[n] = cloneColorSpace(v, cache)
	}
	if r.Shading != nil {
		out.Shading = make(map[Name]*ShadingDict, len(r.Shading))
	}
	for n, v := range r.Shading {
		out.Shading[n] = cache.checkOrClone(v).(*ShadingDict)
	}
	if r.Pattern != nil {
		out.Pattern = make(map[Name]Pattern, len(r.Pattern))
	}
	for n, v := range r.Pattern {
		out.Pattern[n] = cache.checkOrClone(v).(Pattern)
	}
	if r.Font != nil {
		out.Font = make(map[Name]*FontDict, len(r.Font))
	}
	for n, v := range r.Font {
		out.Font[n] = cache.checkOrClone(v).(*FontDict)
	}
	if r.XObject != nil {
		out.XObject = make(map[Name]XObject, len(r.XObject))
	}
	for n, v := range r.XObject {
		out.XObject[n] = cache.checkOrClone(v).(XObject)
	}
	return out
}

// ------------------------------- Bookmarks -------------------------------

//TODO: read

// Outline is the root of the ouline hierarchie
type Outline struct {
	First *OutlineItem
}

// Last returns the last of this item’s immediate children in the outline hierarchy
func (o *Outline) Last() *OutlineItem { return last(o) }

// Count returns the total number of visible outline items
// at all levels of the outline.
func (o *Outline) Count() int {
	c := 0
	for child := o.First; child != nil; child = child.Next {
		c += 1 // child is a top-level item
		if child.Open {
			c += child.Count()
		}
	}
	return c
}

// ref should be the object number of the outline, need for the child
// to reference their parent
func (o *Outline) pdfString(pdf pdfWriter, ref Reference) string {
	firstRef := pdf.createObject()
	pdf.outlines[o.First] = firstRef
	pdf.writeObject(o.First.pdfString(pdf, firstRef, ref), nil, firstRef)
	lastRef := pdf.createObject()
	last := o.Last()
	pdf.outlines[last] = lastRef
	pdf.writeObject(last.pdfString(pdf, lastRef, ref), nil, lastRef)
	return fmt.Sprintf("<</First %s/Last %s/Count %d>>", firstRef, lastRef, o.Count())
}

func (o *Outline) clone(cache cloneCache) *Outline {
	if o == nil {
		return nil
	}
	out := *o
	out.First = o.First.clone(cache, o)
	return &out
}

type OutlineNode interface {
	first() *OutlineItem
}

func (o *Outline) first() *OutlineItem     { return o.First }
func (o *OutlineItem) first() *OutlineItem { return o.First }

// OutlineFlag specify style characteristics for displaying an outline item.
type OutlineFlag uint8

const (
	OutlineItalic OutlineFlag = 1
	OutlineBold   OutlineFlag = 1 << 2
)

// OutlineItem serves as visual table of
// contents to display the document’s structure to the user
type OutlineItem struct {
	Title  string       // text string
	Parent OutlineNode  // parent of this item in the outline hierarchy
	First  *OutlineItem // first of this item’s immediate children in the outline hierarchy
	Next   *OutlineItem // next item at this outline level
	// Prev and Last are deduced

	// indicate if this outline item is open
	// in PDF, it is encoded by the sign of the Count property
	Open bool
	Dest Destination       // optional
	A    Action            // optional
	SE   *StructureElement // optional
	C    [3]Fl             // optional, default to [0 0 0]
	F    OutlineFlag       // optional, default to 0
}

// Prev returns the previous item at this outline level
func (o *OutlineItem) Prev() *OutlineItem {
	elem := o.Parent.first() // start at first sibling
	if elem == o {           // o is the first
		return nil
	}
	for ; elem.Next != o; elem = elem.Next {
	}
	return elem
}

func last(outline OutlineNode) *OutlineItem {
	elem := outline.first()
	if elem == nil {
		return nil
	}
	for ; elem.Next != nil; elem = elem.Next {
	}
	return elem
}

// Last returns the last of this item’s immediate children in the outline hierarchy
func (o *OutlineItem) Last() *OutlineItem { return last(o) }

// Count returns the number of visible descendent outline items at all level
// This is the abolute value of the property Count defined in the PDF spec
func (o *OutlineItem) Count() int {
	if o.First == nil {
		return 0
	}
	c := 0
	// Add to Count the number of immediate children
	for child := o.First; child != nil; child = child.Next {
		c++
		if child.Open { // for each of those immediate children whose Count is positive
			c += child.Count()
		}
	}
	return c
}

// convenience function to write an item only once, and return
// its reference afterwards
func (pdf pdfWriter) addOutlineItem(item *OutlineItem, parent Reference) Reference {
	nextRef, has := pdf.outlines[item]
	if !has {
		nextRef = pdf.createObject()
		pdf.outlines[item] = nextRef
		pdf.writeObject(item.pdfString(pdf, nextRef, parent), nil, nextRef)
	}
	return nextRef
}

// ref should be the object number of the outline item, needed for the child
// to reference it. parent is the parent of the outline item
// since an item will be processed several times (from its siblings)
// we use a cache to keep track of the already written items
func (o *OutlineItem) pdfString(pdf pdfWriter, ref, parent Reference) string {
	b := newBuffer()
	b.fmt("<</Title %s/Parent %s", pdf.EncodeString(o.Title, TextString, ref), parent)
	if o.Next != nil {
		nextRef := pdf.addOutlineItem(o.Next, parent)
		b.fmt("/Next %s", nextRef)
	}
	if prev := o.Prev(); prev != nil {
		prevRef := pdf.addOutlineItem(prev, parent)
		b.fmt("/Prev %s", prevRef)
	}
	if first := o.First; first != nil {
		firstRef := pdf.addOutlineItem(first, ref)
		b.fmt("/First %s", firstRef)
	}
	if last := o.Last(); last != nil {
		lastRef := pdf.addOutlineItem(last, ref)
		b.fmt("/Last %s", lastRef)
	}
	count := o.Count() // absolute value
	if !o.Open {       // closed -> count negative
		count = -count
	}
	b.fmt("/Count %d", count)
	if o.Dest != nil {
		b.fmt("/Dest %s", o.Dest.pdfDestination(pdf, ref))
	}
	if o.A.ActionType != nil {
		b.fmt("/A %s", o.A.pdfString(pdf, ref))
	}
	// TODO: structure element
	if o.C != [3]Fl{} {
		b.fmt("/C %s", writeFloatArray(o.C[:]))
	}
	if o.F != 0 {
		b.fmt("/F %d", o.F)
	}
	b.fmt(">>")
	return b.String()
}

func (o *OutlineItem) clone(cache cloneCache, parent OutlineNode) *OutlineItem {
	if o == nil {
		return o
	}
	out := *o
	out.Parent = parent
	if o.Dest != nil {
		out.Dest = o.Dest.clone(cache)
	}
	out.A = o.A.clone(cache)
	// TODO: Structure element
	out.First = o.First.clone(cache, &out)
	out.Next = o.Next.clone(cache, parent)
	return &out
}
