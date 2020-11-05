package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) processPages(catalog pdfcpu.Dict) (model.PageTree, error) {
	pages := r.resolve(catalog["Pages"])
	pagesDict, isDict := pages.(pdfcpu.Dict)
	if !isDict {
		return model.PageTree{}, errType("Pages", pages)
	}
	root, err := r.resolvePageTree(pagesDict, nil)
	if err != nil {
		return model.PageTree{}, err
	}
	return *root, err
}

func (r resolver) resolvePageObject(node pdfcpu.Dict, parent *model.PageTree) (*model.PageObject, error) {
	resources, err := r.resolveResources(node["Resources"])
	if err != nil {
		return nil, err
	}
	var page model.PageObject
	page.Parent = parent
	page.Resources = resources
	if ar, isArray := node["MediaBox"].(pdfcpu.Array); isArray {
		page.MediaBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["CropBox"].(pdfcpu.Array); isArray {
		page.CropBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["BleedBox"].(pdfcpu.Array); isArray {
		page.BleedBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["TrimBox"].(pdfcpu.Array); isArray {
		page.TrimBox = rectangleFromArray(ar)
	}
	if ar, isArray := node["ArtBox"].(pdfcpu.Array); isArray {
		page.ArtBox = rectangleFromArray(ar)
	}
	if rot := node.IntEntry("Rotate"); rot != nil {
		page.Rotate = model.NewRotation(*rot)
	}

	annots := node.ArrayEntry("Annots")
	for _, annot := range annots {
		annotRef, isRef := annot.(pdfcpu.IndirectRef)
		if annotModel := r.annotations[annotRef]; isRef && annotModel != nil {
			page.Annots = append(page.Annots, annotModel)
			continue
		}
		annot = r.resolve(annot)
		annotDict, isDict := annot.(pdfcpu.Dict)
		if !isDict {
			return nil, errType("Annotation", annot)
		}
		var annotModel model.Annotation
		if rect := rectangleFromArray(annotDict.ArrayEntry("Rect")); rect != nil {
			annotModel.Rect = *rect
		}

		contents, _ := annotDict["Contents"].(pdfcpu.StringLiteral)
		annotModel.Contents = decodeStringLit(contents)

		if f := annotDict.IntEntry("F"); f != nil {
			annotModel.F = *f
		}
		if name := annotDict.NameEntry("Name"); name != nil {
			annotModel.AS = model.Name(*name)
		}

		annotModel.AP, err = r.resolveAppearanceDict(annotDict["AP"])
		if err != nil {
			return nil, err
		}

		annotModel.Subtype, err = r.resolveAnnotationSubType(annotDict)
		if err != nil {
			return nil, err
		}

		if isRef { // write the annotation back into the cache
			r.annotations[annotRef] = &annotModel
		}
		page.Annots = append(page.Annots, &annotModel)
	}
	return &page, nil
}

// node, possibly root
func (r resolver) resolvePageTree(node pdfcpu.Dict, parent *model.PageTree) (*model.PageTree, error) {
	resources, err := r.resolveResources(node["Resources"])
	if err != nil {
		return nil, err
	}
	var page model.PageTree
	page.Parent = parent
	page.Resources = resources
	kids, _ := r.resolve(node["Kids"]).(pdfcpu.Array)
	for _, node := range kids {
		// each page should be distinct, don't bother
		// tracking the refs
		node = r.resolve(node)
		nodeDict, ok := node.(pdfcpu.Dict)
		if !ok {
			return nil, errType("PageNode", node)
		}
		kid, err := r.processPageNode(nodeDict, &page)
		if err != nil {
			return nil, err
		}
		page.Kids = append(page.Kids, kid)
	}
	return &page, nil
}

func (r resolver) processPageNode(node pdfcpu.Dict, parent *model.PageTree) (model.PageNode, error) {
	switch node["Type"] {
	case pdfcpu.Name("Pages"):
		return r.resolvePageTree(node, parent)
	case pdfcpu.Name("Page"):
		return r.resolvePageObject(node, parent)
	default:
		return nil, fmt.Errorf("unexpected value for Type field of page node: %s", node["Type"])
	}
}

// TODO:
func (r resolver) resolveAnnotationSubType(annot pdfcpu.Dict) (model.AnnotationType, error) {
	fmt.Println(annot)
	return nil, nil
}
