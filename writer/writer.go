package writer

import (
	"fmt"
	"io"

	"github.com/benoitkugler/pdf/model"
)

type writer struct {
	dst     io.Writer
	err     error // internal error, to defer error checking
	written int   // total number of bytes written to dst

	objNumber  int   // current object number (starts at 1)
	objOffsets []int // byte offset of objects (starts at 1, [0] is unused)

	doc model.Document

	// caches to avoid duplication of indirect object: ptr -> object number
	formFields        map[*model.FormField]int
	appearanceDicts   map[*model.AppearanceDict]int
	appearanceEntries map[*model.AppearanceEntry]int
	xObjects          map[*model.XObject]int
	resources         map[*model.ResourcesDict]int
	fonts             map[*model.Font]int
	graphicsStates    map[*model.GraphicState]int
	encodings         map[*model.EncodingDict]int
	annotations       map[*model.Annotation]int
	fileSpecs         map[*model.FileSpec]int
	fileContents      map[*model.EmbeddedFileStream]int
	pages             map[*model.PageObject]int
	shadings          map[*model.ShadingDict]int
	functions         map[*model.Function]int
	iccs              map[*model.ICCBasedColorSpace]int
	patterns          map[model.Pattern]int
}

func newWriter(dest io.Writer) *writer {
	return &writer{dst: dest, objNumber: 1, objOffsets: []int{0},
		formFields:        make(map[*model.FormField]int),
		appearanceDicts:   make(map[*model.AppearanceDict]int),
		appearanceEntries: make(map[*model.AppearanceEntry]int),
		xObjects:          make(map[*model.XObject]int),
		resources:         make(map[*model.ResourcesDict]int),
		fonts:             make(map[*model.Font]int),
		graphicsStates:    make(map[*model.GraphicState]int),
		encodings:         make(map[*model.EncodingDict]int),
		annotations:       make(map[*model.Annotation]int),
		fileSpecs:         make(map[*model.FileSpec]int),
		fileContents:      make(map[*model.EmbeddedFileStream]int),
		pages:             make(map[*model.PageObject]int),
		shadings:          make(map[*model.ShadingDict]int),
		functions:         make(map[*model.Function]int),
		iccs:              make(map[*model.ICCBasedColorSpace]int),
		patterns:          make(map[model.Pattern]int),
	}
}

func (w *writer) fmt(s string, args ...interface{}) {
	if w.err != nil { // Write is now a no-op
		return
	}
	n, err := fmt.Fprintf(w.dst, s, args...)
	if err != nil {
		w.err = err
		return
	}
	w.written += n
}
