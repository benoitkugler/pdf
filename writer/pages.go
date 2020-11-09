package writer

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

// since pages should have an indirect ref to their parent,
// we need to choose in advance the object number of
// their parent

func (w *writer) writePageNode(page model.PageNode, parentRef ref) ref {
	switch page := page.(type) {
	case *model.PageObject:
		return w.writePageObject(page, parentRef)
	case model.PageTree:
		return w.writePageTree(page, parentRef)
	default:
		panic("unreachable")
	}
}

// for the tree root, parentRef must be <= 0
func (w *writer) writePageTree(pages model.PageTree, parentRef ref) ref {
	ownRef := w.allocateObject()
	kidRefs := make([]ref, len(pages.Kids))
	for i, page := range pages.Kids {
		kidRefs[i] = w.writePageNode(page, ownRef)
	}
	parent := ""
	if parentRef > 0 {
		parent = fmt.Sprintf(" /Parent %s", parentRef)
	}
	//TODO: resources
	content := []byte(fmt.Sprintf("<</Type /Pages /Kids %s%s>>", writeRefArray(kidRefs), parent))
	w.writeObject(content, ownRef)
	return ownRef
}

func (w *writer) writePageObject(page *model.PageObject, parentRef ref) ref {
	if ref, has := w.pages[page]; has {
		return ref
	}
	// TODO:
	ref := w.defineObject([]byte(fmt.Sprintf("<</Type /Page /Parent %s>>", parentRef)))

	w.pages[page] = ref
	return ref
}

// TODO:
func (w *writer) writePageLabels(labels model.PageLabelsTree) ref {
	return 0
}
