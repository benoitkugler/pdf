package model

type pageNode interface {
	isPageNode()
}

func (PageTree) isPageNode()   {}
func (PageObject) isPageNode() {}

// PageTree describe the page hierarchy
// of a PDF file.
type PageTree struct {
	Parent    *PageTree
	Kids      []pageNode
	Resources *PageResources // if nil, will be inherited from the parent
}

type PageObject struct {
	Parent                    *PageTree
	Resources                 *PageResources // if nil, will be inherited from the parent
	MediaBox                  *Rectangle     // if nil, will be inherited from the parent
	CropBox                   *Rectangle     // if nil, will be inherited. if still nil, default to MediaBox
	BleedBox, TrimBox, ArtBox *Rectangle     // if nil, default to CropBox
	Contents                  Contents
	Rotate                    *Rotation // multiple of 90. if nil, will be inherited from the parent
}

// Contents is an array of stream (often of length 1)
type Contents []ContentStream

type PageResources struct {
	ExtGState  map[Name]GraphicState // optionnal
	ColorSpace map[Name]ColorSpace
	Pattern    map[Name]Pattern
	Shading    map[Name]Shading
	Font       map[Name]Font
	ProcSet    ProcSets
	Properties Properties
}
