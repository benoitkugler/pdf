package model

// PageNode is either a `PageTree` or a `PageObject`
type PageNode interface {
	isPageNode()
}

func (PageTree) isPageNode()    {}
func (*PageObject) isPageNode() {}

// PageTree describe the page hierarchy
// of a PDF file.
type PageTree struct {
	Parent    *PageTree
	Kids      []PageNode
	Resources *ResourcesDict // if nil, will be inherited from the parent
}

// Count returns the number of Page objects (leaf node)
// in all the descendants of `p` (not only in its direct children)
func (p PageTree) Count() int {
	return len(p.Flatten())
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

type PageObject struct {
	Parent                    *PageTree
	Resources                 *ResourcesDict // if nil, will be inherited from the parent
	MediaBox                  *Rectangle     // if nil, will be inherited from the parent
	CropBox                   *Rectangle     // if nil, will be inherited. if still nil, default to MediaBox
	BleedBox, TrimBox, ArtBox *Rectangle     // if nil, default to CropBox
	Rotate                    *Rotation      // if nil, will be inherited from the parent. Only multiple of 90 are allowed
	Annots                    []*Annotation
	Contents                  Contents
}

// Contents is an array of stream (often of length 1)
type Contents []ContentStream

type ResourcesDict struct {
	ExtGState  map[Name]*GraphicState // optionnal
	ColorSpace map[Name]ColorSpace
	Shading    map[Name]*ShadingDict
	Pattern    map[Name]Pattern
	Font       map[Name]*Font
	XObject    map[Name]XObject
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
