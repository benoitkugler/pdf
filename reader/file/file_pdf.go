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

// PDFFile represents a parsed PDF file.
// It is mainly composed of a store of objects (model.Object), identified
// by their reference (model.ObjIndirectRef).
// It usually requires more processing to be really useful: see the packages 'reader'
// and 'model'.
type PDFFile struct {
	XrefTable

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
func ReadFile(file string, conf *Configuration) (PDFFile, error) {
	f, err := os.Open(file)
	if err != nil {
		return PDFFile{}, err
	}
	defer f.Close()

	return Read(f, conf)
}

// Read process a PDF file, reading the xref table and loading
// objects in memory.
func Read(rs io.ReadSeeker, conf *Configuration) (PDFFile, error) {
	ctx, err := processPDFFile(rs, conf)
	if err != nil {
		return PDFFile{}, err
	}

	err = ctx.processAllObjects()
	if err != nil {
		return PDFFile{}, err
	}

	if ctx.trailer.root == nil {
		return PDFFile{}, errors.New("missing Root entry")
	}

	out := PDFFile{
		HeaderVersion:     ctx.HeaderVersion,
		Root:              *ctx.trailer.root,
		AdditionalStreams: ctx.additionalStreams,
		XrefTable:         make(XrefTable, len(ctx.xrefTable.objects)),
		Info:              ctx.trailer.info,
	}

	for k, v := range ctx.xrefTable.objects {
		// ignore free objects
		if v.free {
			continue
		}
		out.XrefTable[k.ObjectNumber] = v.object
	}

	if ctx.enc != nil {
		out.ID = ctx.enc.ID
		out.Encrypt = &ctx.enc.enc
	}

	return out, nil
}

func processPDFFile(rs io.ReadSeeker, conf *Configuration) (*context, error) {
	ctx, err := newContext(rs, conf)
	if err != nil {
		return nil, err
	}

	ctx.HeaderVersion, err = headerVersion(ctx.rs, "%PDF-")
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
