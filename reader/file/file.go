// Package file builds upon a parser
// to read an existing PDF file, producing a
// tree of PDF objets.
// See pacakge reader for an higher level of processing.
package file

import (
	"errors"
	"io"
	"os"

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

	// Optionnal reference to the Info dictionnary, containing metadata.
	Info *parser.IndirectRef

	// ID is found in the trailer, and used for encryption
	ID [2]string

	// Encryption dictionary found in the trailer. Optionnal.
	Encrypt *model.Encrypt
}

// ResolveObject use the xref table to resolve indirect reference.
// If the reference is invalid, the ObjNull{} is returned.
// As convenience, direct objects may also be passed and
// will be returned as it is.
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

// IsString return the string and true if o is a StringLitteral (...) or a HexadecimalLitteral <...>.
// Note that the string is unespaced (for StringLitteral) or decoded (for HexadecimalLitteral),
// but is not always UTF-8.
func IsString(o model.Object) (string, bool) {
	switch o := o.(type) {
	case model.ObjStringLiteral:
		return string(o), true
	case model.ObjHexLiteral:
		return string(o), true
	default:
		return "", false
	}
}

type Configuration struct {
	// Either owner or user password.
	// TODO: We don't support changing permissions,
	// so both password acts the same.
	Password string
}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{}
}

// ReadFile is the same as Read, but takes a file name as input.
func ReadFile(file string, conf *Configuration) (File, error) {
	f, err := os.Open(file)
	if err != nil {
		return File{}, err
	}
	defer f.Close()

	return Read(f, conf)
}

// Read process a PDF file, reading the xref table and loading
// objects in memory.
func Read(rs io.ReadSeeker, conf *Configuration) (File, error) {
	ctx, err := processFile(rs, conf)
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
		Info:              ctx.trailer.info,
	}

	for k, v := range ctx.xrefTable.objects {
		out.xRefTable[k.ObjectNumber] = v.object
	}

	if ctx.enc != nil {
		out.ID = ctx.enc.ID
		out.Encrypt = &ctx.enc.enc
	}

	return out, nil
}

func processFile(rs io.ReadSeeker, conf *Configuration) (*context, error) {
	ctx, err := newContext(rs, conf)
	if err != nil {
		return nil, err
	}

	o, err := ctx.offsetLastXRefSection(0)
	if err != nil {
		return nil, err
	}

	err = ctx.buildXRefTableStartingAt(o)
	if err != nil {
		return nil, err
	}

	err = ctx.setupEncryption()
	if err != nil {
		return nil, err
	}

	return ctx, nil
}
