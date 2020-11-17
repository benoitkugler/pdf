// Uses pdfcpu library to process a PDF file
// and populate a model.Document object
package reader

import (
	"encoding/hex"
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

	// appearanceEntries map[pdfcpu.IndirectRef]*model.AppearanceEntry
	formFields        map[pdfcpu.IndirectRef]*model.FormFieldDict
	appearanceDicts   map[pdfcpu.IndirectRef]*model.AppearanceDict
	resources         map[pdfcpu.IndirectRef]*model.ResourcesDict
	fonts             map[pdfcpu.IndirectRef]*model.FontDict
	graphicsStates    map[pdfcpu.IndirectRef]*model.GraphicState
	encodings         map[pdfcpu.IndirectRef]*model.SimpleEncodingDict
	annotations       map[pdfcpu.IndirectRef]*model.AnnotationDict
	fileSpecs         map[pdfcpu.IndirectRef]*model.FileSpec
	fileContents      map[pdfcpu.IndirectRef]*model.EmbeddedFileStream
	pages             map[pdfcpu.IndirectRef]*model.PageObject
	shadings          map[pdfcpu.IndirectRef]*model.ShadingDict
	functions         map[pdfcpu.IndirectRef]*model.FunctionDict
	patterns          map[pdfcpu.IndirectRef]model.Pattern
	xObjectForms      map[pdfcpu.IndirectRef]*model.XObjectForm
	images            map[pdfcpu.IndirectRef]*model.XObjectImage
	iccs              map[pdfcpu.IndirectRef]*model.ColorSpaceICCBased
	colorTableStreams map[pdfcpu.IndirectRef]*model.ColorTableStream

	// annotations may reference pages which are not yet processed
	// we store them and update the Page field later
	// see processPages, processDictDests
	destinationsToComplete []incompleteDest
}

type incompleteDest struct {
	destination *model.DestinationExplicit
	ref         pdfcpu.IndirectRef
}

// decodeTextString expects a "text string" as defined in PDF spec,
// that is, either a PDFDocEncoded string or a UTF-16BE string
func decodeTextString(s string) string {
	b, err := pdfcpu.Unescape(s)
	if err != nil {
		log.Printf("error %s unescaping string literal %s\n", err, s)
		return ""
	}

	// Check for Big Endian UTF-16.
	if pdfcpu.IsUTF16BE(b) {
		out, err := pdfcpu.DecodeUTF16String(string(b))
		if err != nil {
			log.Printf("error decoding UTF16 string literal %s \n", err)
		}
		return out
	}

	return encodings.PDFDocEncodingToString(b)
}

// var replacer = strings.NewReplacer("\\\\", "\\", "\\(", ")", "\\)", "(", "\\r", "\r")

// return the string and true if o is a StringLitteral (...) or a HexadecimalLitteral <...>
// the string is unespaced (for StringLitteral) and decoded (for Hexadecimal)
func isString(o pdfcpu.Object) (string, bool) {
	switch o := o.(type) {
	case pdfcpu.StringLiteral:
		return o.Value(), true
	case pdfcpu.HexLiteral:
		out, err := hex.DecodeString(o.Value())
		return string(out), err == nil
	default:
		return "", false
	}
}

func (r resolver) processFloatArray(ar pdfcpu.Array) []float64 {
	out := make([]float64, len(ar))
	for i, v := range ar {
		out[i], _ = r.resolveNumber(v)
	}
	return out
}

func (r resolver) info() model.Info {
	var out model.Info
	if info := r.xref.Info; info != nil {
		d, _ := r.resolve(*info).(pdfcpu.Dict)
		producer, _ := isString(r.resolve(d["Producer"]))
		title, _ := isString(r.resolve(d["Title"]))
		subject, _ := isString(r.resolve(d["Subject"]))
		author, _ := isString(r.resolve(d["Author"]))
		keywords, _ := isString(r.resolve(d["Keywords"]))
		creator, _ := isString(r.resolve(d["Creator"]))
		creationDate, _ := isString(r.resolve(d["CreationDate"]))
		modDate, _ := isString(r.resolve(d["ModDate"]))
		out.Producer = decodeTextString(producer)
		out.Title = decodeTextString(title)
		out.Subject = decodeTextString(subject)
		out.Author = decodeTextString(author)
		out.Keywords = decodeTextString(keywords)
		out.Creator = decodeTextString(creator)
		out.CreationDate, _ = pdfcpu.DateTime(creationDate)
		out.ModDate, _ = pdfcpu.DateTime(modDate)
	}
	return out
}

func (r resolver) encrypt() model.Encrypt {
	var out model.Encrypt
	if enc := r.xref.Encrypt; enc != nil {
		d, _ := r.resolve(*enc).(pdfcpu.Dict)

		out.Filter, _ = r.resolveName(d["Filter"])
		out.SubFilter, _ = r.resolveName(d["SubFilter"])

		v, _ := r.resolveInt(d["V"])
		out.V = model.EncryptionAlgorithm(v)

		out.Length, _ = r.resolveInt(d["Length"])

		cf, _ := r.resolve(d["CF"]).(pdfcpu.Dict)
		out.CF = make(map[model.Name]model.CrypFilter, len(cf))
		for name, c := range cf {
			out.CF[model.Name(name)] = r.processCryptFilter(c)
		}
		out.StmF, _ = r.resolveName(d["StmF"])
		out.StrF, _ = r.resolveName(d["StrF"])
		out.EFF, _ = r.resolveName(d["EFF"])
	}
	return out
}

func (r resolver) processCryptFilter(crypt pdfcpu.Object) model.CrypFilter {
	cryptDict, _ := r.resolve(crypt).(pdfcpu.Dict)
	var out model.CrypFilter
	out.CFM, _ = r.resolveName(cryptDict["CFM"])
	out.AuthEvent, _ = r.resolveName(cryptDict["AuthEvent"])
	out.Length, _ = r.resolveInt(cryptDict["AuthEvent"])
	recipients := r.resolve(cryptDict["Recipients"])
	if rec, ok := isString(recipients); ok {
		out.Recipients = []string{rec}
	} else if ar, ok := recipients.(pdfcpu.Array); ok {
		out.Recipients = make([]string, len(ar))
		for i, re := range ar {
			out.Recipients[i], _ = isString(r.resolve(re))
		}
	}
	out.EncryptMetadata = true
	if enc, ok := r.resolveBool(cryptDict["EncryptMetadata"]); ok {
		out.EncryptMetadata = enc
	}
	return out
}

func ParsePDF(source io.ReadSeeker, userPassword string) (model.Document, error) {
	var out model.Document
	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = userPassword
	config.DecodeAllStreams = true
	ti := time.Now()
	ctx, err := pdfcpu.Read(source, config)
	if err != nil {
		return out, fmt.Errorf("can't read PDF: %w", err)
	}
	fmt.Printf("pdfcpu processing: %s\n", time.Since(ti))

	r := resolver{
		xref:            ctx.XRefTable,
		formFields:      make(map[pdfcpu.IndirectRef]*model.FormFieldDict),
		appearanceDicts: make(map[pdfcpu.IndirectRef]*model.AppearanceDict),
		// appearanceEntries: make(map[pdfcpu.IndirectRef]*model.AppearanceEntry),
		resources:         make(map[pdfcpu.IndirectRef]*model.ResourcesDict),
		fonts:             make(map[pdfcpu.IndirectRef]*model.FontDict),
		graphicsStates:    make(map[pdfcpu.IndirectRef]*model.GraphicState),
		encodings:         make(map[pdfcpu.IndirectRef]*model.SimpleEncodingDict),
		annotations:       make(map[pdfcpu.IndirectRef]*model.AnnotationDict),
		fileSpecs:         make(map[pdfcpu.IndirectRef]*model.FileSpec),
		fileContents:      make(map[pdfcpu.IndirectRef]*model.EmbeddedFileStream),
		pages:             make(map[pdfcpu.IndirectRef]*model.PageObject),
		functions:         make(map[pdfcpu.IndirectRef]*model.FunctionDict),
		shadings:          make(map[pdfcpu.IndirectRef]*model.ShadingDict),
		patterns:          make(map[pdfcpu.IndirectRef]model.Pattern),
		xObjectForms:      make(map[pdfcpu.IndirectRef]*model.XObjectForm),
		images:            make(map[pdfcpu.IndirectRef]*model.XObjectImage),
		iccs:              make(map[pdfcpu.IndirectRef]*model.ColorSpaceICCBased),
		colorTableStreams: make(map[pdfcpu.IndirectRef]*model.ColorTableStream),
	}

	ti = time.Now()

	out.Trailer.Info = r.info()

	out.Trailer.Encrypt = r.encrypt()

	out.Catalog, err = r.catalog()
	if err != nil {
		return out, err
	}

	fmt.Printf("model processing: %s\n", time.Since(ti))

	return out, nil
}

func (r resolver) catalog() (model.Catalog, error) {
	var out model.Catalog
	d, err := r.xref.Catalog()
	if err != nil {
		return out, fmt.Errorf("can't resolve Catalog: %w", err)
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

	out.PageLayout, _ = r.resolveName(d["PageLayout"])
	out.PageMode, _ = r.resolveName(d["PageMode"])

	if pl := d["PageLabels"]; pl != nil {
		out.PageLabels = new(model.PageLabelsTree)
		err = r.resolvePageLabelsTree(pl, out.PageLabels)
		if err != nil {
			return out, err
		}
	}

	out.StructTreeRoot, err = r.resolveStructureTree(d["StructTreeRoot"])
	if err != nil {
		return out, err
	}

	// may need pages
	out.Outlines, err = r.resolveOutline(d["Outlines"])
	if err != nil {
		return out, err
	}

	out.ViewerPreferences, err = r.resolveViewerPreferences(d["ViewerPreferences"])
	if err != nil {
		return out, err
	}

	out.MarkInfo, err = r.resolveMarkDict(d["MarkInfo"])
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

func (r resolver) resolveBool(o pdfcpu.Object) (bool, bool) {
	b, ok := r.resolve(o).(pdfcpu.Boolean)
	return bool(b), ok
}

func (r resolver) resolveInt(o pdfcpu.Object) (int, bool) {
	b, ok := r.resolve(o).(pdfcpu.Integer)
	return int(b), ok
}

// accepts both integer and float
func (r resolver) resolveNumber(o pdfcpu.Object) (float64, bool) {
	switch o := r.resolve(o).(type) {
	case pdfcpu.Float:
		return o.Value(), true
	case pdfcpu.Integer:
		return float64(o.Value()), true
	default:
		return 0, false
	}
}

func (r resolver) resolveName(o pdfcpu.Object) (model.Name, bool) {
	b, ok := r.resolve(o).(pdfcpu.Name)
	return model.Name(b), ok
}

func (r resolver) resolveArray(o pdfcpu.Object) (pdfcpu.Array, bool) {
	b, ok := r.resolve(o).(pdfcpu.Array)
	return b, ok
}

func errType(label string, o pdfcpu.Object) error {
	return fmt.Errorf("unexpected type for %s: %T", label, o)
}
