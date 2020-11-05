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
	total := 0
	for _, kid := range p.Kids {
		switch kid := kid.(type) {
		case *PageTree:
			total += kid.Count()
		case *PageObject:
			total += 1
		}
	}
	return total
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
