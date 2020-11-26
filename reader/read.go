// Uses pdfcpu library to process a PDF file
// and populate a model.Document object
package reader

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"golang.org/x/text/encoding/unicode"
)

type Fl = model.Fl

// maintain tables mapping PDF indirect object numbers
// to model objects
type resolver struct {
	xref *pdfcpu.XRefTable

	// appearanceEntries map[pdfcpu.IndirectRef]*model.AppearanceEntry
	formFields        map[pdfcpu.IndirectRef]*model.FormFieldDict
	appearanceDicts   map[pdfcpu.IndirectRef]*model.AppearanceDict
	resources         map[pdfcpu.IndirectRef]model.ResourcesDict
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
	structure         map[pdfcpu.IndirectRef]*model.StructureElement
}

type incompleteDest struct {
	destination *model.DestinationExplicitIntern
	ref         pdfcpu.IndirectRef
}

func isUTF16(b []byte) bool {
	if len(b) < 2 {
		return false
	}
	// Check BOM
	return b[0] == 0xFE && b[1] == 0xFF || b[0] == 0xff && b[1] == 0xfe
}

var utf16Dec = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)

// decodeTextString expects a "text string" as defined in PDF spec,
// that is, either a PDFDocEncoded string or a UTF-16BE string
func decodeTextString(s string) string {
	b, err := pdfcpu.Unescape(s)
	if err != nil {
		log.Printf("error %s unescaping string literal %s\n", err, s)
		return ""
	}

	// Check for UTF-16: we also accept LE, since text/encoding handles it
	if isUTF16(b) {
		out, err := utf16Dec.NewDecoder().Bytes(b)
		if err != nil {
			log.Printf("error decoding UTF16 string literal %s \n", err)
		}
		return string(out)
	}

	return model.PDFDocEncodingToString(b)
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

// output as same length as input
func (r resolver) processFloatArray(ar pdfcpu.Array) []Fl {
	out := make([]Fl, len(ar))
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

func (r resolver) encrypt() (*model.Encrypt, error) {
	var out model.Encrypt
	enc := r.xref.Encrypt
	if enc == nil {
		return nil, nil
	}

	d, _ := r.resolve(*enc).(pdfcpu.Dict)

	out.Filter, _ = r.resolveName(d["Filter"])
	out.SubFilter, _ = r.resolveName(d["SubFilter"])

	v, _ := r.resolveInt(d["V"])
	out.V = model.EncryptionAlgorithm(v)

	length, _ := r.resolveInt(d["Length"])
	if length%8 != 0 {
		return nil, fmt.Errorf("field Length must be a multiple of 8")
	}
	out.Length = uint8(length / 8)

	cf, _ := r.resolve(d["CF"]).(pdfcpu.Dict)
	out.CF = make(map[model.Name]model.CrypFilter, len(cf))
	for name, c := range cf {
		out.CF[model.Name(name)] = r.processCryptFilter(c)
	}
	out.StmF, _ = r.resolveName(d["StmF"])
	out.StrF, _ = r.resolveName(d["StrF"])
	out.EFF, _ = r.resolveName(d["EFF"])

	p, _ := r.resolveInt(d["P"])
	out.P = model.UserPermissions(p)

	// subtypes
	var err error
	if out.Filter == "Standard" {
		out.EncryptionHandler, err = r.processStandardSecurityHandler(d)
		if err != nil {
			return nil, err
		}
	} else {
		out.EncryptionHandler = r.processPublicKeySecurityHandler(d)
	}

	return &out, nil
}

func (r resolver) processStandardSecurityHandler(dict pdfcpu.Dict) (model.EncryptionStandard, error) {
	var out model.EncryptionStandard
	r_, _ := r.resolveInt(dict["R"])
	out.R = uint8(r_)

	o, _ := isString(r.resolve(dict["O"]))
	if len(o) != 32 {
		return out, fmt.Errorf("expected 32-length byte string for entry O, got %v", o)
	}
	for i := 0; i < len(o); i++ {
		out.O[i] = o[i]
	}

	u, _ := isString(r.resolve(dict["U"]))
	if len(u) != 32 {
		return out, fmt.Errorf("expected 32-length byte string for entry U, got %v", u)
	}
	for i := 0; i < len(u); i++ {
		out.U[i] = u[i]
	}
	if meta, ok := r.resolveBool(dict["EncryptMetadata"]); ok {
		out.DontEncryptMetadata = !meta
	}
	return out, nil
}

func (r resolver) processPublicKeySecurityHandler(dict pdfcpu.Dict) model.EncryptionPublicKey {
	rec, _ := r.resolveArray(dict["Recipients"])
	out := make(model.EncryptionPublicKey, len(rec))
	for i, re := range rec {
		out[i], _ = isString(r.resolve(re))
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
	if enc, ok := r.resolveBool(cryptDict["EncryptMetadata"]); ok {
		out.DontEncryptMetadata = !enc
	}
	return out
}

func ParseFile(filename string, userPassword string) (model.Document, *model.Encrypt, error) {
	f, err := os.Open(filename)
	if err != nil {
		return model.Document{}, nil, fmt.Errorf("can't open file: %w", err)
	}
	defer f.Close()

	return ParsePDF(f, userPassword)
}

func ParsePDF(source io.ReadSeeker, userPassword string) (model.Document, *model.Encrypt, error) {
	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = userPassword
	config.DecodeAllStreams = true

	ti := time.Now()

	ctx, err := pdfcpu.Read(source, config)
	if err != nil {
		return model.Document{}, nil, fmt.Errorf("can't read PDF: %w", err)
	}

	fmt.Printf("pdfcpu processing: %s\n", time.Since(ti))
	ti = time.Now()

	out, enc, err := ProcessContext(ctx)

	fmt.Printf("model processing: %s\n", time.Since(ti))

	return out, enc, err
}

// ProcessContext walk through a parsed PDF to build a model.
// It also returns the potential encryption information.
func ProcessContext(ctx *pdfcpu.Context) (model.Document, *model.Encrypt, error) {
	r := resolver{
		xref:            ctx.XRefTable,
		formFields:      make(map[pdfcpu.IndirectRef]*model.FormFieldDict),
		appearanceDicts: make(map[pdfcpu.IndirectRef]*model.AppearanceDict),
		// appearanceEntries: make(map[pdfcpu.IndirectRef]*model.AppearanceEntry),
		resources:         make(map[pdfcpu.IndirectRef]model.ResourcesDict),
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
		structure:         make(map[pdfcpu.IndirectRef]*model.StructureElement),
	}
	var (
		out model.Document
		err error
	)

	out.Trailer.Info = r.info()

	enc, err := r.encrypt()
	if err != nil {
		return out, nil, err
	}

	out.Catalog, err = r.catalog()
	return out, enc, err
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

	out.Dests, err = r.resolveDests(d["Dests"])
	if err != nil {
		return out, err
	}

	out.Names, err = r.processNameDict(d["Names"])
	if err != nil {
		return out, err
	}
	// TODO: annotations
	out.PageLayout, _ = r.resolveName(d["PageLayout"])
	out.PageMode, _ = r.resolveName(d["PageMode"])

	if pl := d["PageLabels"]; pl != nil {
		out.PageLabels = new(model.PageLabelsTree)
		err = r.resolveNumberTree(pl, pageLabelTree{out: out.PageLabels})
		if err != nil {
			return out, err
		}
	}

	// pages, annotations, xforms need to be resolved
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

	uriDict, ok := r.resolve(d["URI"]).(pdfcpu.Dict)
	if ok {
		out.URI, _ = isString(r.resolve(uriDict["Base"]))
	}

	out.OpenAction, err = r.resolveDestinationOrAction(d["OpenAction"])
	if err != nil {
		return out, err
	}

	return out, nil
}

// might return nil, since, (PDF spec, clause 7.3.10)
// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
// it shall be treated as a reference to the null object.
func (r resolver) resolve(o pdfcpu.Object) pdfcpu.Object {
	o, _ = r.xref.Dereference(o) // return no error despite its signature
	return o
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
func (r resolver) resolveNumber(o pdfcpu.Object) (Fl, bool) {
	switch o := r.resolve(o).(type) {
	case pdfcpu.Float:
		return Fl(o.Value()), true
	case pdfcpu.Integer:
		return Fl(o.Value()), true
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
