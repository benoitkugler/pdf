package model

import (
	"fmt"
	"strings"
)

// --------------------- Property list ---------------------

// MetadataStream is a stream containing XML metadata,
// implementing Object
type MetadataStream struct {
	Stream
}

func (m MetadataStream) Clone() Object {
	return MetadataStream{Stream: m.Stream.Clone()}
}

func (m MetadataStream) Write(w PDFWritter, _ Reference) string {
	base := m.Stream.PDFCommonFields(true)
	base.Fields["Type"] = "/Metadata"
	base.Fields["Subtype"] = "/XML"
	ref := w.CreateObject()
	w.WriteStream(base, m.Content, ref)
	return ref.String()
}

// PropertyList is a dictionary of custom values.
// See Metadata for an implementation for the /Metadata key
type PropertyList = ObjDict

// MarkDict provides additional information relevant to
// specialized uses of structured PDF documents.
type MarkDict struct {
	Marked         bool
	UserProperties bool
	Suspects       bool
}

// String returns the PDF dictionary representation
func (m MarkDict) String() string {
	return fmt.Sprintf("<</Marked %v/UserProperties %v/Suspects %v>>",
		m.Marked, m.UserProperties, m.Suspects)
}

// ----------------------- structure -----------------------

// StructureTree is the root of the structure tree.
//
// When read from an existing file, `IDTree` and `ParentTree`
// will be filled.
// However, when creating a new structure tree,
// since the information is mostly redundant (only the shape of the trees are to choose),
// `BuildIDTree` and `BuildParentTree` may be used as a convenience.
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

// BuildIdTree walks through the structure,
// looking for the /ID of the structure elements
// and build a valid ParentTree, updating `s`.
// It should be good enough for most use case,
// but when a custom shape for the tree is needed,
// the `IDTree` attribut may be set directly.
func (s *StructureTree) BuildIDTree() {
	// we use a simple approach, with two passes:
	// 	- a map is used to accumulate the mappings
	//	- the map is then transformed into a tree
	tmp := make(map[string]*StructureElement)

	var walk func(se *StructureElement)
	walk = func(se *StructureElement) {
		if se.ID != "" {
			tmp[se.ID] = se
		}
		for _, kid := range se.K {
			if kidS, ok := kid.(*StructureElement); ok {
				walk(kidS)
			}
		}
	}
	for _, se := range s.K {
		walk(se)
	}
	s.IDTree = NewIDTree(tmp)
}

// BuildParentTree walks through the structure,
// looking for the ∕StructParent and /StructParents
// of the target of the structure elements
// and build a valid ParentTree, updating `s`.
// It should be good enough for most use case,
// but when a custom shape for the tree is needed,
// the `ParentTree` attribut may be set directly.
func (s *StructureTree) BuildParentTree() {
	// we use a simple approach, with two passes:
	// 	- a map is used to accumulate the mappings
	//	- the map is then transformed into a tree
	tmp := make(map[int]NumToParent)

	var walk func(se *StructureElement)
	walk = func(se *StructureElement) {
		for _, kid := range se.K {
			switch kid := kid.(type) {
			case *StructureElement:
				walk(kid) // recursion
			case ContentItemMarkedReference:
				var structParents MaybeInt
				switch ct := kid.Container.(type) {
				case *PageObject:
					structParents = ct.StructParents
				case *XObjectForm:
					structParents = ct.StructParents
				case nil: // default to the structure element
					structParents = se.Pg.StructParents
				}
				if sp, ok := structParents.(ObjInt); ok {
					a := tmp[int(sp)]
					a.Parents = append(a.Parents, se)
					tmp[int(sp)] = a
				}
			case ContentItemObjectReference:
				if kid.Obj != nil {
					if structParent := kid.Obj.GetStructParent(); structParent != nil {
						num := int(structParent.(ObjInt))
						tmp[num] = NumToParent{Parent: se}
					}
				}
			}
		}
	}
	for _, se := range s.K {
		walk(se)
	}

	s.ParentTree = NewParentTree(tmp)
}

func (s *StructureTree) clone(cache cloneCache) *StructureTree {
	if s == nil {
		return nil
	}
	out := *s
	if s.K != nil { // preserve nil
		out.K = make([]*StructureElement, len(s.K))
		for i, k := range s.K {
			out.K[i] = k.clone(cache).(*StructureElement)
		}
	}

	out.IDTree = s.IDTree.clone(cache)         // here, cache.structure is field
	out.ParentTree = s.ParentTree.clone(cache) // same

	if s.RoleMap != nil {
		out.RoleMap = make(map[Name]Name, len(s.RoleMap))
		for k, v := range s.RoleMap {
			out.RoleMap[k] = v
		}
	}
	if s.ClassMap != nil {
		out.ClassMap = make(map[Name][]AttributeObject, len(s.ClassMap))
		for k, v := range s.ClassMap {
			if v != nil {
				vc := make([]AttributeObject, len(v))
				for i, a := range v {
					vc[i] = a.Clone()
				}
				out.ClassMap[k] = vc
			}
		}
	}
	return &out
}

func (s StructureTree) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()

	// start by walking the structure elements tree,
	// so that pdf.structure is filled
	refs := make([]Reference, len(s.K))
	for i, k := range s.K {
		kidRef := pdf.CreateObject()
		pdf.WriteObject(k.pdfString(pdf, kidRef, 0), kidRef)
		refs[i] = kidRef
	}

	roleChunks := make([]string, 0, len(s.RoleMap))
	for k, v := range s.RoleMap {
		roleChunks = append(roleChunks, k.String()+v.String())
	}
	classChunks := make([]string, 0, len(s.ClassMap))
	for k, attrs := range s.ClassMap {
		attrChunks := make([]string, len(attrs))
		for i, a := range attrs {
			attrChunks[i] = a.pdfString(pdf, ref)
		}
		classChunks = append(classChunks, fmt.Sprintf("%s [%s]", k, strings.Join(attrChunks, " ")))
	}

	idTreeRef := pdf.CreateObject()
	pdf.WriteObject(s.IDTree.pdfString(pdf, idTreeRef), idTreeRef)

	b.line("<</Type/StructTreeRoot/K %s/IDTree %s/ParentTree %s/ParentTreeNextKey %d",
		writeRefArray(refs), idTreeRef, s.ParentTree.pdfString(pdf), s.ParentTreeNextKey())
	b.line("/RoleMap <<%s>>", strings.Join(roleChunks, ""))
	b.line("/ClassMap<<%s>> ", strings.Join(classChunks, ""))
	b.WriteString(">>")
	return b.String()
}

type ClassName struct {
	Name           Name
	RevisionNumber int // optional, default to 0
}

// String returns one or two elements
func (c ClassName) String() string {
	out := c.Name.String()
	if c.RevisionNumber != 0 {
		out += fmt.Sprintf(" %d", c.RevisionNumber)
	}
	return out
}

type StructureElement struct {
	S          Name
	P          *StructureElement // parent
	ID         string            // byte string, optional
	Pg         *PageObject       // optional
	K          []ContentItem     // 1-array may be written in PDF directly
	A          []AttributeObject // 1-array may be written in PDF directly
	C          []ClassName       // 1-array may be written in PDF directly
	R          int               // optional, revision number
	T          string            // optional, text string
	Lang       string            // optional, text string
	Alt        string            // optional, text string
	E          string            // optional, text string
	ActualText string            // optional, text string
}

// `own` reference is needed to encrypt, and for the kids
func (s *StructureElement) pdfString(pdf pdfWriter, own, parent Reference) string {
	b := newBuffer()
	b.fmt("<</S%s", s.S)
	if s.P != nil {
		b.fmt("/P %s", parent)
	}
	if s.ID != "" {
		b.fmt("/ID %s", pdf.EncodeString(s.ID, ByteString, own))
	}
	if s.Pg != nil {
		ref := pdf.pages[s.Pg]
		b.fmt("/Pg %s", ref)
	}
	b.WriteString("/K [")
	for _, k := range s.K {
		if k != nil {
			s := writeContentItem(k, pdf, own)
			b.WriteString(s + " ")
		}
	}
	b.line("]")
	chunks := make([]string, len(s.A))
	for i, at := range s.A {
		chunks[i] = at.pdfString(pdf, own)
	}
	b.line("/A [%s]", strings.Join(chunks, " "))
	chunks = make([]string, len(s.C))
	for i, c := range s.C {
		chunks[i] = c.String()
	}
	b.line("/C [%s]", strings.Join(chunks, " "))
	if s.R != 0 {
		b.fmt("/R %d", s.R)
	}
	if s.T != "" {
		b.fmt("/T %s", pdf.EncodeString(s.T, TextString, own))
	}
	if s.Lang != "" {
		b.fmt("/Lang %s", pdf.EncodeString(s.Lang, TextString, own))
	}
	if s.Alt != "" {
		b.fmt("/Alt %s", pdf.EncodeString(s.Alt, TextString, own))
	}
	if s.E != "" {
		b.fmt("/E %s", pdf.EncodeString(s.E, TextString, own))
	}
	if s.ActualText != "" {
		b.fmt("/ActualText %s", pdf.EncodeString(s.ActualText, TextString, own))
	}
	b.WriteString(">>")
	return b.String()
}

// return a concrete type *StructureElement, and register the clone in the cache
func (s *StructureElement) clone(cache cloneCache) ContentItem {
	if s == nil {
		return s // typed nil
	}
	out := *s

	cache.structure[s] = &out

	out.P = cache.structure[s.P]
	if s.Pg != nil {
		out.Pg = cache.pages[s.Pg].(*PageObject)
	}
	if s.K != nil {
		out.K = make([]ContentItem, len(s.K))
		for i, k := range s.K {
			out.K[i] = k.clone(cache)
		}
	}
	if s.A != nil {
		out.A = make([]AttributeObject, len(s.A))
		for i, k := range s.A {
			out.A[i] = k.Clone()
		}
	}
	out.C = append([]ClassName(nil), s.C...)
	return &out
}

// ContentItem may be one the three following item
// 	- A structure element dictionary denoting another structure element
//	- A marked-content reference dictionary denoting a marked-content sequence
//		In PDF, it may be written directly as an integer, when marked-content
//		sequence is contained in the content stream of the page
//		that is specified in the Pg entry of the structure element dictionary
//	- An object reference dictionary denoting a PDF object
type ContentItem interface {
	isContentItem()
	clone(cache cloneCache) ContentItem
}

func (*StructureElement) isContentItem()          {}
func (ContentItemMarkedReference) isContentItem() {}
func (ContentItemObjectReference) isContentItem() {}

func writeContentItem(c ContentItem, pdf pdfWriter, parent Reference) string {
	switch c := c.(type) {
	case *StructureElement:
		ownRef := pdf.CreateObject()
		pdf.WriteObject(c.pdfString(pdf, ownRef, parent), ownRef)
		return ownRef.String()
	case ContentItemMarkedReference:
		return c.pdfString(pdf)
	case ContentItemObjectReference:
		return c.pdfString(pdf)
	default:
		panic("ContentItem can't be nil")
	}
}

// ContentItemMarkedReference is a marked-content reference dictionary
type ContentItemMarkedReference struct {
	// marked-content identifier marked-content identifier sequence
	// within its content stream sequence within its content stream
	MCID int

	// optional. This entry should be an XObjectForm only if the marked-content sequence
	// resides in a content stream other than the content stream for the page
	Container ContentMarkedContainer

	// TODO: StmOwn
}

// ContentMarkedContainer is either *PageObject
// or *XObjectForm (found for exemple in appearance streams)
type ContentMarkedContainer interface {
	isContentMarkedContainer()
}

func (*PageObject) isContentMarkedContainer()  {}
func (*XObjectForm) isContentMarkedContainer() {}

func (c ContentItemMarkedReference) pdfString(pdf pdfWriter) string {
	out := fmt.Sprintf("<</Type/MCR/MCID %d", c.MCID)
	switch ct := c.Container.(type) {
	case *PageObject:
		ref := pdf.pages[ct]
		out += fmt.Sprintf("/Pg %s", ref)
	case *XObjectForm:
		ref := pdf.cache[ct]
		out += fmt.Sprintf("/Stm %s", ref)
	}
	return out + ">>"
}

// required that pages and XobjectForm have been cloned
func (c ContentItemMarkedReference) clone(cache cloneCache) ContentItem {
	out := c
	switch ct := c.Container.(type) {
	case *PageObject:
		out.Container = cache.pages[ct].(*PageObject)
	case *XObjectForm:
		out.Container = cache.refs[ct].(*XObjectForm)
	}
	return out
}

// StructParentObject identifies the PDF object
// that may be reference in the Structure Tree,
// that is, the objects that have a StructParent entry.
// It may be one of *AnnotationDict, *XObjectForm or *XObjectImage
type StructParentObject interface {
	Referenceable
	// isStructParentObject()
	GetStructParent() MaybeInt
}

// func (*AnnotationDict) isStructParentObject() {}
// func (*XObjectForm) isStructParentObject()    {}
// func (*XObjectImage) isStructParentObject()   {}

// ContentItemObjectReference identifies an entire PDF object, such as an XObject or an annotation, that is
// associated with a page but not directly included in the page’s content stream
type ContentItemObjectReference struct {
	Pg  *PageObject        // optional
	Obj StructParentObject // required
}

// requires that pages and all other objects have been written
func (c ContentItemObjectReference) pdfString(pdf pdfWriter) string {
	ref := pdf.cache[c.Obj]
	out := fmt.Sprintf("<</Type/OBJR/Obj %s", ref)
	if c.Pg != nil {
		ref := pdf.pages[c.Pg]
		out += fmt.Sprintf("/Pg %s", ref)
	}
	return out + ">>"
}

func (c ContentItemObjectReference) clone(cache cloneCache) ContentItem {
	out := c
	if c.Pg != nil {
		out.Pg = cache.pages[c.Pg].(*PageObject)
	}
	if c.Obj != nil {
		// since refs preserve the concrete types
		// we can safely cast back to StructParentObject
		out.Obj = cache.refs[c.Obj].(StructParentObject)
	}
	return out
}

// AttributeObject is represented by a single or a pair of array
// elements, the first or only element shall contain the attribute object itself
// and the second (when present) shall contain the integer revision number
// associated with it in this structure element.
// We only support dictionary (via the Attributes field), not streams.
type AttributeObject struct {
	RevisionNumber int // optional, default to 0

	O Name // required

	// Other keys and values defining the attributes
	Attributes map[Name]Object
}

// return one or to element, suitable to be included in an array
func (a AttributeObject) pdfString(pdf pdfWriter, ref Reference) string {
	chunks := make([]string, 0, len(a.Attributes))
	for name, attr := range a.Attributes {
		chunks = append(chunks, name.String()+" "+attr.Write(pdf, ref))
	}
	out := fmt.Sprintf("<</O%s%s>>", a.O, strings.Join(chunks, " "))
	if a.RevisionNumber != 0 {
		out += fmt.Sprintf(" %d", a.RevisionNumber)
	}
	return out
}

func (a AttributeObject) Clone() AttributeObject {
	out := a
	if a.Attributes != nil {
		out.Attributes = make(map[Name]Object, len(a.Attributes))
		for k, v := range a.Attributes {
			out.Attributes[k] = v.Clone()
		}
	}
	return out
}

// AttributeUserProperties is a predefined kind of Attribute
// It may be used only if an AttributeObject with O = UserProperties
// and Attributes = map[P: AttributeUserProperties]
type AttributeUserProperties []UserProperty

// Clone implements Attribute
func (us AttributeUserProperties) Clone() Object {
	var out AttributeUserProperties
	if us != nil {
		out = make(AttributeUserProperties, len(us))
		for i, a := range out {
			out[i] = a.Clone()
		}
	}
	return out
}

// PDFString implements Attribute
func (us AttributeUserProperties) Write(enc PDFWritter, context Reference) string {
	chunks := make([]string, len(us))
	for i, u := range us {
		chunks[i] = u.Write(enc, context)
	}
	return "[" + strings.Join(chunks, " ") + "]"
}

type UserProperty struct {
	N string // required
	V Object // required
	F string // optional
	H bool   // optional
}

func (u UserProperty) Write(enc PDFWritter, context Reference) string {
	v := ""
	if u.V != nil {
		v = "/V " + u.V.Write(enc, context)
	}
	return fmt.Sprintf("<</N %s%s/F %s/H %v>>",
		enc.EncodeString(u.N, TextString, context), v,
		enc.EncodeString(u.F, TextString, context), u.H)
}

// Clone returns a deep copy
func (u UserProperty) Clone() UserProperty {
	out := u
	if u.V != nil {
		out.V = u.V.Clone()
	}
	return out
}
