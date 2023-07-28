// Package reader leverage a PDF file reader
// to read a file, analyze its structure and build a
// high level, in-memory representation as a `model.Document`.
package reader

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
	"golang.org/x/text/encoding/unicode"
)

type Fl = model.Fl

const debug = false

// maintain tables mapping PDF indirect object numbers
// to model objects
type resolver struct {
	file file.PDFFile

	// appearanceEntries map[model.ObjIndirectRef]*model.AppearanceEntry
	formFields        map[model.ObjIndirectRef]*model.FormFieldDict
	appearanceDicts   map[model.ObjIndirectRef]*model.AppearanceDict
	resources         map[model.ObjIndirectRef]model.ResourcesDict
	fonts             map[model.ObjIndirectRef]*model.FontDict
	graphicsStates    map[model.ObjIndirectRef]*model.GraphicState
	encodings         map[model.ObjIndirectRef]*model.SimpleEncodingDict
	annotations       map[model.ObjIndirectRef]*model.AnnotationDict
	fileSpecs         map[model.ObjIndirectRef]*model.FileSpec
	fileContents      map[model.ObjIndirectRef]*model.EmbeddedFileStream
	pages             map[model.ObjIndirectRef]*model.PageObject
	shadings          map[model.ObjIndirectRef]*model.ShadingDict
	functions         map[model.ObjIndirectRef]*model.FunctionDict
	patterns          map[model.ObjIndirectRef]model.Pattern
	xObjectForms      map[model.ObjIndirectRef]*model.XObjectForm
	images            map[model.ObjIndirectRef]*model.XObjectImage
	xObjectsGroups    map[model.ObjIndirectRef]*model.XObjectTransparencyGroup
	imageSMasks       map[model.ObjIndirectRef]*model.ImageSMask
	iccs              map[model.ObjIndirectRef]*model.ColorSpaceICCBased
	colorTableStreams map[model.ObjIndirectRef]*model.ColorTableStream
	structure         map[model.ObjIndirectRef]*model.StructureElement
	fontFiles         map[model.ObjIndirectRef]*model.FontFile

	customResolve CustomObjectResolver // optional, default is nil
}

func newResolver() resolver {
	return resolver{
		formFields:        make(map[model.ObjIndirectRef]*model.FormFieldDict),
		appearanceDicts:   make(map[model.ObjIndirectRef]*model.AppearanceDict),
		resources:         make(map[model.ObjIndirectRef]model.ResourcesDict),
		fonts:             make(map[model.ObjIndirectRef]*model.FontDict),
		graphicsStates:    make(map[model.ObjIndirectRef]*model.GraphicState),
		encodings:         make(map[model.ObjIndirectRef]*model.SimpleEncodingDict),
		annotations:       make(map[model.ObjIndirectRef]*model.AnnotationDict),
		fileSpecs:         make(map[model.ObjIndirectRef]*model.FileSpec),
		fileContents:      make(map[model.ObjIndirectRef]*model.EmbeddedFileStream),
		pages:             make(map[model.ObjIndirectRef]*model.PageObject),
		functions:         make(map[model.ObjIndirectRef]*model.FunctionDict),
		shadings:          make(map[model.ObjIndirectRef]*model.ShadingDict),
		patterns:          make(map[model.ObjIndirectRef]model.Pattern),
		xObjectForms:      make(map[model.ObjIndirectRef]*model.XObjectForm),
		images:            make(map[model.ObjIndirectRef]*model.XObjectImage),
		xObjectsGroups:    make(map[model.ObjIndirectRef]*model.XObjectTransparencyGroup),
		imageSMasks:       make(map[model.ObjIndirectRef]*model.ImageSMask),
		iccs:              make(map[model.ObjIndirectRef]*model.ColorSpaceICCBased),
		colorTableStreams: make(map[model.ObjIndirectRef]*model.ColorTableStream),
		structure:         make(map[model.ObjIndirectRef]*model.StructureElement),
		fontFiles:         make(map[model.ObjIndirectRef]*model.FontFile),
	}
}

 

func isUTF16(b []byte) bool {
	if len(b) < 2 {
		return false
	}
	// Check BOM
	return b[0] == 0xFE && b[1] == 0xFF || b[0] == 0xff && b[1] == 0xfe
}

var (
	utf16Dec  = unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder()
)

// DecodeTextString expects a "text string" as defined in PDF spec,
// that is, either a PDFDocEncoded string or a UTF-16BE string,
// and returns the UTF-8 corresponding string.
// Note that encryption, escaping or hex-encoding should already
// have been taken care of.
func DecodeTextString(s string) string {
	b := []byte(s)

	// Check for UTF-16: we also accept LE, since text/encoding handles it
	if isUTF16(b) {
		out, err := utf16Dec.Bytes(b)
		if err != nil {
			log.Printf("error decoding UTF16 string literal %v: %s \n", b, err)
		}
		return string(out)
	}

	return model.PdfDocEncodingToString(b)
}

// var replacer = strings.NewReplacer("\\\\", "\\", "\\(", ")", "\\)", "(", "\\r", "\r")

// output has same length as input
func (r resolver) processFloatArray(ar model.ObjArray) []Fl {
	out := make([]Fl, len(ar))
	for i, v := range ar {
		out[i], _ = r.resolveNumber(v)
	}
	return out
}

func (r resolver) info() model.Info {
	var out model.Info
	if info := r.file.Info; info != nil {
		d, _ := r.resolve(*info).(model.ObjDict)
		producer, _ := file.IsString(r.resolve(d["Producer"]))
		title, _ := file.IsString(r.resolve(d["Title"]))
		subject, _ := file.IsString(r.resolve(d["Subject"]))
		author, _ := file.IsString(r.resolve(d["Author"]))
		keywords, _ := file.IsString(r.resolve(d["Keywords"]))
		creator, _ := file.IsString(r.resolve(d["Creator"]))
		creationDate, _ := file.IsString(r.resolve(d["CreationDate"]))
		modDate, _ := file.IsString(r.resolve(d["ModDate"]))
		out.Producer = DecodeTextString(producer)
		out.Title = DecodeTextString(title)
		out.Subject = DecodeTextString(subject)
		out.Author = DecodeTextString(author)
		out.Keywords = DecodeTextString(keywords)
		out.Creator = DecodeTextString(creator)
		out.CreationDate, _ = DateTime(creationDate)
		out.ModDate, _ = DateTime(modDate)
	}
	return out
}

// Options enables greater control on the processing.
// The zero value is a valid default configuration.
type Options struct {
	CustomObjectResolver CustomObjectResolver
	UserPassword         string
}

// ParsePDFFile opens a file and calls `ParsePDFReader`,
// see the latter for details.
func ParsePDFFile(filename string, options Options) (model.Document, *model.Encrypt, error) {
	f, err := os.Open(filename)
	if err != nil {
		return model.Document{}, nil, fmt.Errorf("can't open file: %w", err)
	}
	defer f.Close()

	return ParsePDFReader(f, options)
}

// ParsePDFReader reads a PDF file and builds a model.
// This is done in two steps:
//   - a first parsing step (involving lexing and parsing) builds a tree object
//   - this tree is then interpreted according to the PDF specification,
//     resolving indirect objects and transforming dynamic, opaque types
//     into statically typed objects, building the returned `Document`.
//
// Information about encryption are returned separately, and will be needed
// if you want to encrypt the document back.
func ParsePDFReader(source io.ReadSeeker, options Options) (model.Document, *model.Encrypt, error) {
	config := file.Configuration{Password: options.UserPassword}

	ti := time.Now()

	ctx, err := file.Read(source, &config)
	if err != nil {
		return model.Document{}, nil, fmt.Errorf("can't read PDF: %w", err)
	}

	if debug {
		fmt.Printf("raw file processing: %s\n", time.Since(ti))
	}
	ti = time.Now()

	r := newResolver()
	r.file = ctx
	r.customResolve = options.CustomObjectResolver

	out, enc, err := r.processPDF()

	if debug {
		fmt.Printf("model processing: %s\n", time.Since(ti))
	}

	return out, enc, err
}

// ProcessContext walks through an already parsed PDF to build a model.
// This function is exposed for debug purposes; you should probably use
// one of `ParsePDFFile` or `ParsePDFReader` methods.
func ProcessContext(ctx file.PDFFile) (model.Document, *model.Encrypt, error) {
	r := newResolver()
	r.file = ctx
	return r.processPDF()
}

// processPDF walk through a parsed PDF to build a model.
// It also returns the potential encryption information.
func (r resolver) processPDF() (model.Document, *model.Encrypt, error) {
	var (
		out model.Document
		err error
	)

	out.Trailer.Info = r.info()

	enc := r.file.Encrypt

	out.Catalog, err = r.catalog()
	return out, enc, err
}

// might return ObjNull{}, since, (PDF spec, clause 7.3.10)
// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
// it shall be treated as a reference to the null object.
func (r resolver) resolve(o model.Object) model.Object {
	return r.file.ResolveObject(o)
}

func (r resolver) resolveBool(o model.Object) (bool, bool) {
	b, ok := r.resolve(o).(model.ObjBool)
	return bool(b), ok
}

func (r resolver) resolveInt(o model.Object) (int, bool) {
	b, ok := r.resolve(o).(model.ObjInt)
	return int(b), ok
}

// accepts both integer and float
func (r resolver) resolveNumber(o model.Object) (Fl, bool) {
	switch o := r.resolve(o).(type) {
	case model.ObjFloat:
		return Fl(o), true
	case model.ObjInt:
		return Fl(o), true
	default:
		return 0, false
	}
}

func (r resolver) resolveName(o model.Object) (model.ObjName, bool) {
	b, ok := r.resolve(o).(model.ObjName)
	return model.ObjName(b), ok
}

func (r resolver) resolveArray(o model.Object) (model.ObjArray, bool) {
	b, ok := r.resolve(o).(model.ObjArray)
	return b, ok
}

func errType(label string, o model.Object) error {
	return fmt.Errorf("unexpected type for %s: %T", label, o)
}
