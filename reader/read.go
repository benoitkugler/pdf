// Uses pdfcpu library to process a PDF file
// and populate a model.Document object
package reader

import (
	"fmt"
	"io"
	"log"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/encodings"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

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
	ctx, err := pdfcpu.Read(source, config)
	if err != nil {
		return nil, fmt.Errorf("can't read PDF: %w", err)
	}
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

	return &out, nil
}

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
}
