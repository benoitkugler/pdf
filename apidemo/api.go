package apidemo

import (
	"github.com/benoitkugler/pdf/model"
)

func ListAttachments(doc model.Document) []string {
	out := make([]string, len(doc.Catalog.Names.EmbeddedFiles))
	for i, file := range doc.Catalog.Names.EmbeddedFiles {
		out[i] = file.FileSpec.UF
	}
	return out
}
