package model

type PageNode interface {
	isPageNode()
}

func (*PageTree) isPageNode()   {}
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
func (p *PageTree) Count() int {
	return len(p.Flatten())
}

// Flatten returns all the leaf of the tree,
// respecting the indexing convention for pages (0-based):
// the page with index i is Flatten()[i].
// Be aware that inherited resource are not resolved
func (p *PageTree) Flatten() []*PageObject {
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
	Rotate                    *Rotation      // multiple of 90. if nil, will be inherited from the parent
	Annots                    []*Annotation
	Contents                  Contents
}

// Contents is an array of stream (often of length 1)
type Contents []ContentStream

type ResourcesDict struct {
	ExtGState  map[Name]*GraphicState // optionnal
	ColorSpace map[Name]*ColorSpace
	Pattern    map[Name]*Pattern
	Shading    map[Name]*Shading
	Font       map[Name]*Font
}
