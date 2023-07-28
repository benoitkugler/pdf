package file

import (
	"fmt"
	"io"
	"os"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/parser"
)

// FDFFile is an in-memory version of a .fdf file,
// used to fill PDF forms.
type FDFFile struct {
	XrefTable

	Version string

	Root model.ObjIndirectRef
}

// ReadFDFFile is the same as ReadFDF, but takes a file name as input.
func ReadFDFFile(file string) (FDFFile, error) {
	f, err := os.Open(file)
	if err != nil {
		return FDFFile{}, err
	}
	defer f.Close()

	return ReadFDF(f)
}

// ReadFDF process a FDF file, loading objects in memory.
func ReadFDF(rs io.ReadSeeker) (FDFFile, error) {
	return processFDFFile(rs)
}

func processFDFFile(rs io.ReadSeeker) (FDFFile, error) {
	version, err := headerVersion(rs, "%FDF-")
	if err != nil {
		return FDFFile{}, err
	}

	ctx, err := newContext(rs, nil)
	if err != nil {
		return FDFFile{}, err
	}

	buf, err := ctx.findStringFromFileEnd(0, "trailer")
	if err != nil {
		return FDFFile{}, err
	}

	trailer, err := parser.ParseObject(buf)
	if err != nil {
		return FDFFile{}, err
	}

	trailerDict, ok := trailer.(model.ObjDict)
	if !ok {
		return FDFFile{}, fmt.Errorf("unexpected type for trailer: %T", trailer)
	}

	root, ok := trailerDict["Root"].(parser.IndirectRef)
	if !ok {
		return FDFFile{}, fmt.Errorf("unexpected type for Root entry: %T", trailerDict["Root"])
	}

	// since FDF files may not have xref section, we simply read the file
	// line by line
	err = ctx.bypassXrefSection()
	if err != nil {
		return FDFFile{}, err
	}

	err = ctx.processAllObjects()
	if err != nil {
		return FDFFile{}, err
	}

	out := FDFFile{Version: version, Root: root, XrefTable: make(XrefTable)}

	for k, v := range ctx.xrefTable.objects {
		if v.free {
			continue
		}

		out.XrefTable[k.ObjectNumber] = v.object
	}

	return out, nil
}
