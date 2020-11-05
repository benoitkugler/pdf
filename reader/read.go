// Uses pdfcpu library to process a PDF file
// and populate a model.Document object
package reader

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/encodings"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// maintain tables mapping PDF indirect object numbers
// to model objects
type resolver struct {
	xref *pdfcpu.XRefTable

	formFields        map[pdfcpu.IndirectRef]*model.FormField
	appearanceDicts   map[pdfcpu.IndirectRef]*model.AppearanceDict
	appearanceEntries map[pdfcpu.IndirectRef]*model.AppearanceEntry
	xObjects          map[pdfcpu.IndirectRef]*model.XObject
	resources         map[pdfcpu.IndirectRef]*model.ResourcesDict
	fonts             map[pdfcpu.IndirectRef]*model.Font
	encodings         map[pdfcpu.IndirectRef]*model.EncodingDict
	annotations       map[pdfcpu.IndirectRef]*model.Annotation
	fileSpecs         map[pdfcpu.IndirectRef]*model.FileSpec
	fileContents      map[pdfcpu.IndirectRef]*model.EmbeddedFileStream
	pages             map[pdfcpu.IndirectRef]*model.PageObject

	// annotations may reference pages which are not yet processed
	// we store them and update the Page field later
	// see processPages, processDictDests
	destinationsToComplete []incompleteDest
}

type incompleteDest struct {
	destination *model.ExplicitDestination
	ref         pdfcpu.IndirectRef
}

func decodeStringLit(s pdfcpu.StringLiteral) string {
	b, err := pdfcpu.Unescape(s.Value())
	if err != nil {
		log.Printf("error decoding string literal %s : %s", s, err)
		return ""
	}

	s1 := string(b)

	// Check for Big Endian UTF-16.
	if pdfcpu.IsStringUTF16BE(s1) {
		out, err := pdfcpu.DecodeUTF16String(s1)
		if err != nil {
			log.Printf("error decoding string literal %s : %s", s, err)
		}
		return out
	}

	return encodings.PDFDocEncodingToString([]byte(s))
}

// return the number and true is o is an int or a number
func isNumber(o pdfcpu.Object) (float64, bool) {
	switch o := o.(type) {
	case pdfcpu.Float:
		return o.Value(), true
	case pdfcpu.Integer:
		return float64(o.Value()), true
	default:
		return 0, false
	}
}

func info(xref *pdfcpu.XRefTable) (model.Info, error) {
	var info model.Info
	if xref.Info != nil {
		d, err := xref.DereferenceDict(*xref.Info)
		if err != nil {
			return info, fmt.Errorf("can't resolve Info dictionnary: %w", err)
		}
		producer, _ := d["Producer"].(pdfcpu.StringLiteral)
		title, _ := d["Title"].(pdfcpu.StringLiteral)
		subject, _ := d["Subject"].(pdfcpu.StringLiteral)
		author, _ := d["Author"].(pdfcpu.StringLiteral)
		keywords, _ := d["Keywords"].(pdfcpu.StringLiteral)
		creator, _ := d["Creator"].(pdfcpu.StringLiteral)
		creationDate, _ := d["CreationDate"].(pdfcpu.StringLiteral)
		modDate, _ := d["ModDate"].(pdfcpu.StringLiteral)
		info.Producer = decodeStringLit(producer)
		info.Title = decodeStringLit(title)
		info.Subject = decodeStringLit(subject)
		info.Author = decodeStringLit(author)
		info.Keywords = decodeStringLit(keywords)
		info.Creator = decodeStringLit(creator)
		info.CreationDate, _ = pdfcpu.DateTime(string(creationDate))
		info.ModDate, _ = pdfcpu.DateTime(string(modDate))
	}
	return info, nil
}

func encrypt(xref *pdfcpu.XRefTable) (model.Encrypt, error) {
	var out model.Encrypt
	if xref.Encrypt != nil {
		d, err := xref.DereferenceDict(*xref.Encrypt)
		if err != nil {
			return out, fmt.Errorf("can't resolve Encrypt dictionnary: %w", err)
		}
		filter, _ := d["Filter"].(pdfcpu.Name)
		out.Filter = model.Name(filter)
		subFilter, _ := d["SubFilter"].(pdfcpu.Name)
		out.SubFilter = model.Name(subFilter)
		out.V = model.EncryptionAlgorithm(xref.E.V)
		length, _ := d["Length"].(pdfcpu.Integer)
		out.Length = int(length)
	}
	return out, nil
}

func ParsePDF(source io.ReadSeeker, userPassword string) (*model.Document, error) {
	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = userPassword
	config.DecodeAllStreams = true
	ti := time.Now()
	ctx, err := pdfcpu.Read(source, config)
	if err != nil {
		return nil, fmt.Errorf("can't read PDF: %w", err)
	}
	fmt.Printf("pdfcpu processing: %s\n", time.Since(ti))
	ti = time.Now()
	var out model.Document
	xref := ctx.XRefTable

	out.Trailer.Info, err = info(xref)
	if err != nil {
		return nil, err
	}
	out.Trailer.Encrypt, err = encrypt(xref)
	if err != nil {
		return nil, err
	}

	out.Catalog, err = catalog(xref)
	if err != nil {
		return nil, err
	}

	fmt.Printf("model processing: %s\n", time.Since(ti))

	return &out, nil
}

func catalog(xref *pdfcpu.XRefTable) (model.Catalog, error) {
	var out model.Catalog
	d, err := xref.Catalog()
	if err != nil {
		return out, fmt.Errorf("can't resolve Catalog: %w", err)
	}
	r := resolver{
		xref:              xref,
		formFields:        make(map[pdfcpu.IndirectRef]*model.FormField),
		appearanceDicts:   make(map[pdfcpu.IndirectRef]*model.AppearanceDict),
		appearanceEntries: make(map[pdfcpu.IndirectRef]*model.AppearanceEntry),
		xObjects:          make(map[pdfcpu.IndirectRef]*model.XObject),
		resources:         make(map[pdfcpu.IndirectRef]*model.ResourcesDict),
		fonts:             make(map[pdfcpu.IndirectRef]*model.Font),
		encodings:         make(map[pdfcpu.IndirectRef]*model.EncodingDict),
		annotations:       make(map[pdfcpu.IndirectRef]*model.Annotation),
		fileSpecs:         make(map[pdfcpu.IndirectRef]*model.FileSpec),
		fileContents:      make(map[pdfcpu.IndirectRef]*model.EmbeddedFileStream),
		pages:             make(map[pdfcpu.IndirectRef]*model.PageObject),
	}

	out.AcroForm, err = r.processAcroForm(d["AcroForm"])
	if err != nil {
		return out, err
	}
	out.Pages, err = r.processPages(d["Pages"])
	if err != nil {
		return out, err
	}

	out.Dests, err = r.processDictDests(d["Dests"])
	if err != nil {
		return out, err
	}

	out.Names, err = r.processNameDict(d["Names"])
	if err != nil {
		return out, err
	}

	// complete the destinations
	for _, dest := range r.destinationsToComplete {
		po := r.pages[dest.ref]
		if po == nil {
			return out, fmt.Errorf("reference %s not found in pages: ignoring destination", dest.ref)
		}
		dest.destination.Page = po
	}

	return out, nil
}

// might return nil, since, (PDF spec, clause 7.3.10)
// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
// it shall be treated as a reference to the null object.
func (r resolver) resolve(o pdfcpu.Object) pdfcpu.Object {
	// despite it's signature, Dereference always return a nil error
	out, _ := r.xref.Dereference(o)
	return out
}

func errType(label string, o pdfcpu.Object) error {
	return fmt.Errorf("unexpected type for %s: %T", label, o)
}
