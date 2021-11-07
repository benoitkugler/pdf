// Package file builds upon a parser
// to read an existing PDF file, producing a
// tree of PDF objets.
// See pacakge reader for an higher level of processing.
package file

import (
	"errors"
	"io"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/parser"
)

// The logic is adapted from pdfcpu

// File represents a parsed PDF file.
type File struct {
	xRefTable map[int]parser.Object

	// The PDF version the source is claiming to us as per its header.
	HeaderVersion string

	// AdditionalStreams (array of IndirectRef) is not described in the spec,
	// but may be found in the trailer :e.g., Oasis "Open Doc"
	AdditionalStreams parser.Array

	// Reference to the Catalog root dictionnary
	Root parser.IndirectRef
}

// ResolveObject use the xref table to resolve indirect reference.
func (f *File) ResolveObject(o parser.Object) parser.Object {
	ref, ok := o.(parser.IndirectRef)
	if !ok {
		return o // return the direct object as it is
	}

	if o, has := f.xRefTable[ref.ObjectNumber]; has {
		return o
	}

	// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
	// it shall be treated as a reference to the null object.
	return model.ObjNull{}
}

type Configuration struct{}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{}
}

// Read process a PDF file, reading the xref table and loading
// objects in memory.
func Read(rs io.ReadSeeker, conf *Configuration) (File, error) {
	ctx, err := newContext(rs, conf)
	if err != nil {
		return File{}, err
	}

	o, err := ctx.offsetLastXRefSection(0)
	if err != nil {
		return File{}, err
	}

	err = ctx.buildXRefTableStartingAt(o)
	if err != nil {
		return File{}, err
	}

	err = ctx.processAllObjects()
	if err != nil {
		return File{}, err
	}

	if ctx.trailer.root == nil {
		return File{}, errors.New("missing Root entry")
	}

	out := File{
		HeaderVersion:     ctx.HeaderVersion,
		Root:              *ctx.trailer.root,
		AdditionalStreams: ctx.additionalStreams,
		xRefTable:         make(map[int]model.Object, len(ctx.xrefTable.objects)),
	}

	for k, v := range ctx.xrefTable.objects {
		out.xRefTable[k] = v.object
	}

	return out, nil
}
