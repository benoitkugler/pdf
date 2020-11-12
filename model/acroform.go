package model

import (
	"fmt"
	"time"
)

// type FormType string

// const (
// 	Tx  FormType = "Tx"
// 	Btn FormType = "Btn"
// 	Ch  FormType = "Ch"
// 	Sig FormType = "Sig"
// )

type FormFlag uint32

const (
	ReadOnly          FormFlag = 1
	Required          FormFlag = 1 << 1
	NoExport          FormFlag = 1 << 2
	Multiline         FormFlag = 1 << 12
	Password          FormFlag = 1 << 13
	NoToggleToOff     FormFlag = 1 << 14
	Radio             FormFlag = 1 << 15
	Pushbutton        FormFlag = 1 << 16
	FileSelect        FormFlag = 1 << 20
	DoNotSpellCheck   FormFlag = 1 << 22
	DoNotScroll       FormFlag = 1 << 23
	Comb              FormFlag = 1 << 24
	RadiosInUnison    FormFlag = 1 << 25
	RichText          FormFlag = 1 << 25
	Combo             FormFlag = 1 << 17
	Edit              FormFlag = 1 << 18
	Sort              FormFlag = 1 << 19
	MultiSelect       FormFlag = 1 << 21
	CommitOnSelChange FormFlag = 1 << 26
)

type FormField struct {
	Parent *FormField
	Kids   []*FormField // nil for a leaf node
	// not nil only for a leaf node
	// represented in PDF under the Kids entry,
	// or merged if their is only one widget
	Widgets []Annotation

	Ft FormFieldType

	T  string // optional, text string
	TU string // optional, text string, alternate field name
	TM string // optional, text string, mapping name
	Ff FormFlag
	AA *FormFielAdditionalActions // optional

	// fields for variable text

	DA string // optional
	Q  uint8  // optional, default to 0
	DS string // optional, text string
	RV string // optional, text string, may be written in PDF as a stream
}

// FormFieldType provides additional form attributes,
// depending on the field type.
type FormFieldType interface {
	formFieldAttrs(pdf pdfWriter) string
}

// FormFieldText are boxes or spaces in which the user can enter text from the keyboard.
type FormFieldText struct {
	V      string // text string, may be written in PDF as a stream
	MaxLen int    // optional, Undef when not set
}

func (f FormFieldText) formFieldAttrs(pdf pdfWriter) string {
	out := fmt.Sprintf("/V %s", pdf.EncodeString(f.V, TextString))
	if f.MaxLen != Undef {
		out += fmt.Sprintf(" /MaxLen %d", f.MaxLen)
	}
	return out
}

// FormFieldButton represent interactive controls on the screen
// that the user can manipulate with the mouse.
// They include pushbuttons, check boxes, and radio buttons.
type FormFieldButton struct {
	V   Name     // check box’s appearance state
	Opt []string // optional, text strings, same length as Widgets
}

func (f FormFieldButton) formFieldAttrs(pdf pdfWriter) string {
	out := fmt.Sprintf("/V %s", f.V)
	if len(f.Opt) != 0 {
		out += fmt.Sprintf(" /Opt [%s]", pdf.stringsArray(f.Opt, TextString))
	}
	return out
}

// Option is either a text string representing one of the available options or an
// array consisting of two text strings: the option’s export value and the text that
// shall be displayed as the name of the option.
type Option struct {
	Export string
	Name   string
}

func (o Option) pdfString(pdf pdfWriter) string {
	if o.Export == "" {
		return pdf.EncodeString(o.Name, TextString)
	}
	return fmt.Sprintf("[%s %s]", pdf.EncodeString(o.Export, TextString), pdf.EncodeString(o.Name, TextString))
}

// FormFieldChoice contain several text items,
// at most one of which may be selected as the field value.
// They include scrollable list boxes and combo boxes.
type FormFieldChoice struct {
	// text strings. When empty, it's written in PDF as null
	// If only one item is currently selected, it's written as a text string
	V   []string
	Opt []Option // optional
	TI  int      // optional, default to 0
	I   []int    // optional
}

func (f FormFieldChoice) formFieldAttrs(pdf pdfWriter) string {
	b := newBuffer()
	if len(f.V) == 0 {
		b.fmt("/V null")
	} else if len(f.V) == 1 {
		b.fmt("/V %s", pdf.EncodeString(f.V[0], TextString))
	} else {
		b.fmt("/V %s", pdf.stringsArray(f.V, TextString))
	}
	if len(f.Opt) != 0 {
		b.fmt(" /Opt [")
		for _, o := range f.Opt {
			b.fmt(" " + o.pdfString(pdf))
		}
		b.fmt("]")
	}
	if f.TI != 0 {
		b.fmt(" /TI %d", f.TI)
	}
	if len(f.I) != 0 {
		b.fmt(" /I %s", writeIntArray(f.I))
	}
	return b.String()
}

// FormFieldSignature represent digital signatures and
// optional data for authenticating the name of the signer and
// the document’s contents.
type FormFieldSignature struct {
	V    *SignatureDict // optional
	Lock *LockDict      // optional
	SV   *SeedDict      // optional
}

func (f FormFieldSignature) formFieldAttrs(pdf pdfWriter) string {
	out := ""
	if f.V != nil {
		out += fmt.Sprintf("/V %s", f.V.pdfString(pdf))
	}
	if lock := f.Lock; lock != nil {
		ref := pdf.addObject(f.Lock.pdfString(pdf), nil)
		out += fmt.Sprintf(" /Lock %s", ref)
	}
	if sv := f.SV; sv != nil {
		ref := pdf.addObject(f.SV.pdfString(pdf), nil)
		out += fmt.Sprintf(" /SV %s", ref)
	}
	return out
}

type SignatureDict struct {
	Filter        Name                     // optional
	SubFilter     Name                     // optional
	Contents      string                   // byte string, written as hexadecimal in PDF
	Cert          []string                 // optional, byte strings. One-element arrays may be written in PDF a a single byte string
	ByteRange     [][2]int                 // optional. Written in PDF as an array of pairs
	Reference     []SignatureRefDict       // optional
	Changes       [3]int                   // optional
	Name          string                   // optional, text string
	M             time.Time                // optional
	Location      string                   // optional, text string
	Reason        string                   // optional, text string
	ContactInfo   string                   // optional, text string
	V             int                      // optional
	Prop_Build    SignatureBuildDictionary // optional
	Prop_AuthTime time.Time                // optional
	Prop_AuthType Name                     // optional
}

func (s SignatureDict) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.WriteString("<<")
	if s.Filter != "" {
		b.fmt("/Filter %s", s.Filter)
	}
	if s.SubFilter != "" {
		b.fmt(" /SubFiler %s", s.SubFilter)
	}
	b.fmt(" /Contents %s", pdf.EncodeString(s.Contents, HexString))
	if len(s.Cert) != 0 {
		b.fmt(" /Cert %s", pdf.stringsArray(s.Cert, TextString))
	}
	if len(s.ByteRange) != 0 {
		b.fmt(" /ByteRange [")
		for _, val := range s.ByteRange {
			b.fmt(" %d %d", val[0], val[1])
		}
		b.fmt("]")
	}
	if len(s.Reference) != 0 {
		b.fmt(" /Reference [")
		for _, val := range s.Reference {
			b.fmt(" %s", val.pdfString(pdf))
		}
		b.fmt("]")
	}
	if s.Changes != [3]int{} {
		b.fmt(" /Changes %s", writeIntArray(s.Changes[:]))
	}
	if s.Name != "" {
		b.fmt(" /Name %s", pdf.EncodeString(s.Name, TextString))
	}
	if s.Location != "" {
		b.fmt(" /Location %s", pdf.EncodeString(s.Location, TextString))
	}
	if s.Reason != "" {
		b.fmt(" /Reason %s", pdf.EncodeString(s.Reason, TextString))
	}
	if s.ContactInfo != "" {
		b.fmt(" /ContactInfo %s", pdf.EncodeString(s.ContactInfo, TextString))
	}
	if !s.M.IsZero() {
		b.fmt(" /M %s", pdf.dateString(s.M))
	}
	if s.V != 0 {
		b.fmt(" /V %d", s.V)
	}
	if s.Prop_Build != nil {
		b.fmt(" /Prop_Build %s", s.Prop_Build.SignatureBuildPDFString())
	}
	if !s.Prop_AuthTime.IsZero() {
		b.fmt(" /Prop_AuthTime %s", pdf.dateString(s.Prop_AuthTime))
	}
	if s.Prop_AuthType != "" {
		b.fmt(" /Prop_AuthType %s", s.Prop_AuthType)
	}
	b.fmt(">>")
	return b.String()
}

// SignatureBuildDictionary is implementation-specific by design.
// It can be used to store audit information that is specific to the software application
// that was used to create the signature.
// The build properties dictionary and all of its contents are required to be direct objects.
type SignatureBuildDictionary interface {
	// SignatureBuildPDFString must return a PDF string representation
	// of the dictionary
	SignatureBuildPDFString() string

	// Clone must return a deep copy of itself
	Clone() SignatureBuildDictionary
}

type SignatureRefDict struct {
	TransformMethod Name // among DocMDP, UR, FieldMDP
	TransformParams TransformParams

	DigestMethod Name
}

func (s SignatureRefDict) pdfString(pdf pdfWriter) string {
	return fmt.Sprintf("<</TransformMethod %s /TransformParams %s /DigestMethod %s>>",
		s.TransformMethod, s.TransformParams.transformParamsDict(pdf), s.DigestMethod)
}

type TransformParams interface {
	transformParamsDict(pdf pdfWriter) string
}

type TransformDocMDP struct {
	P uint // optional; among 1,2,3 ; default to 2
	V Name // optional
}

func (t TransformDocMDP) transformParamsDict(pdfWriter) string {
	out := "<<"
	if t.P != 0 {
		out += fmt.Sprintf("/P %d", t.P)
	}
	if t.V != "" {
		out += fmt.Sprintf(" /V %s", t.V)
	}
	out += ">>"
	return out
}

type TransformUR struct {
	Document  []Name // optional
	Msg       string // optional, text string
	V         Name   // optional
	Annots    []Name // optional
	Form      []Name // optional
	Signature []Name // optional
	EF        []Name // optional
	P         bool   // optional
}

func (t TransformUR) transformParamsDict(pdf pdfWriter) string {
	b := newBuffer()
	b.fmt("<<")
	if len(t.Document) != 0 {
		b.fmt("/Document %s", writeNameArray(t.Document))
	}
	if t.Msg != "" {
		b.fmt(" /Msg %s", pdf.EncodeString(t.Msg, TextString))
	}
	if t.V != "" {
		b.fmt(" /V %s", t.V)
	}
	if len(t.Annots) != 0 {
		b.fmt(" /Annots %s", writeNameArray(t.Annots))
	}
	if len(t.Form) != 0 {
		b.fmt(" /Form %s", writeNameArray(t.Form))
	}
	if len(t.Signature) != 0 {
		b.fmt(" /Signature %s", writeNameArray(t.Signature))
	}
	if len(t.EF) != 0 {
		b.fmt(" /EF %s", writeNameArray(t.EF))
	}
	b.fmt(" /P %v>>", t.P)
	return b.String()
}

// TransformFieldMDP is used to detect changes to the values of a list of form fields.
// BUG(bk) the Data field should be a pointer to
// another item in the Document object. Since the SPEC does not precise it,
// we can't support it in a general way.
type TransformFieldMDP struct {
	Action Name
	Fields []string // text strings, optional is Action == All
	V      Name
	// Data
}

func (t TransformFieldMDP) transformParamsDict(pdf pdfWriter) string {
	out := fmt.Sprintf("<</Action %s", t.Action)
	if len(t.Fields) != 0 {
		out += fmt.Sprintf(" /Fields %s", pdf.stringsArray(t.Fields, TextString))
	}
	out += fmt.Sprintf(" /V %s>>", t.V)
	return out
}

type LockDict struct {
	Action Name     // one of All,Include,Exclude
	Fields []string // field names, text strings, optional when Action == All
}

func (l LockDict) pdfString(pdf pdfWriter) string {
	out := fmt.Sprintf("<</Action %s", l.Action)
	if len(l.Fields) != 0 {
		out += fmt.Sprintf(" /Fields %s", pdf.stringsArray(l.Fields, TextString))
	}
	out += ">>"
	return out
}

type SeedDict struct {
	Ff           SeedFlag  // optional, default to 0
	Filter       Name      // optional
	SubFilter    []Name    // optional
	DigestMethod []Name    // optional, names among SHA1, SHA256, SHA384, SHA512 and RIPEMD160
	V            float64   // optional
	Cert         *CertDict // optional
	Reasons      []string  // optional, text strings
	// optional,  from 0 to 3, default to Undef
	// writen in pdf as a dictionary with entry P
	MDP              int8
	TimeStamp        *TimeStampDict // optional
	LegalAttestation []string       // optional, text strings
	AddRevInfo       bool           // optional, default to false
}

func (s SeedDict) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.fmt("<<")
	if s.Ff != 0 {
		b.fmt("/Ff %d", s.Ff)
	}
	if s.Filter != "" {
		b.fmt(" /Filter %s", s.Filter)
	}
	if len(s.SubFilter) != 0 {
		b.fmt(" /SubFilter %s", writeNameArray(s.SubFilter))
	}
	if len(s.DigestMethod) != 0 {
		b.fmt(" /DigestMethod %s", writeNameArray(s.DigestMethod))
	}
	if s.V != 0 {
		b.fmt(" /V %.3f", s.V)
	}
	if s.Cert != nil {
		b.fmt(" /Cert %s", s.Cert.pdfString(pdf))
	}
	if len(s.Reasons) != 0 {
		b.fmt(" /Reasons %s", pdf.stringsArray(s.Reasons, TextString))
	}
	if s.MDP != Undef {
		b.fmt(" /MDP <</P %d>>", s.MDP)
	}
	if s.TimeStamp != nil {
		b.fmt(" /TimeStamp %s", s.TimeStamp.pdfString(pdf))
	}
	if len(s.LegalAttestation) != 0 {
		b.fmt(" /LegalAttestation %s", pdf.stringsArray(s.LegalAttestation, TextString))
	}
	b.fmt(" /AddRevInfo %v>>", s.AddRevInfo)
	return b.String()
}

type TimeStampDict struct {
	URL string // ASCII string
	Ff  uint8  // 0 or 1, default to 0
}

func (s TimeStampDict) pdfString(pdf pdfWriter) string {
	return fmt.Sprintf("<</URL %s /Ff %d>>", pdf.EncodeString(s.URL, ASCIIString), s.Ff)
}

// CertDict contains characteristics of the certificate that shall be used when signing
type CertDict struct {
	Ff        uint8             // optional, default to 0
	Subject   []string          // optional byte strings
	SubjectDN []map[Name]string // optional, each map values are text strings
	KeyUsage  []string          // optional, ASCII strings
	Issuer    []string          // optional, byte strings
	OID       []string          // optional, byte strings
	URL       string            // optional, ASCII string
	URLType   Name              // optional
}

func (c CertDict) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.WriteString("<<")
	if c.Ff != 0 {
		b.fmt("/Ff %d", c.Ff)
	}
	if len(c.Subject) != 0 {
		b.fmt(" /Subject %s", pdf.stringsArray(c.Subject, ByteString))
	}
	if len(c.SubjectDN) != 0 {
		b.fmt(" /SubjectDN [")
		for _, dn := range c.SubjectDN {
			b.fmt("<<")
			for name, value := range dn {
				b.fmt("%s %s ", name, pdf.EncodeString(value, TextString))
			}
			b.fmt(">> ")
		}
		b.fmt("]")
	}
	if len(c.KeyUsage) != 0 {
		b.fmt(" /KeyUsage %s", pdf.stringsArray(c.KeyUsage, ASCIIString))
	}
	if len(c.Issuer) != 0 {
		b.fmt(" /Issuer %s", pdf.stringsArray(c.Issuer, ByteString))
	}
	if len(c.OID) != 0 {
		b.fmt(" /OID %s", pdf.stringsArray(c.OID, ByteString))
	}
	if c.URL != "" {
		b.fmt(" /URL %s", pdf.EncodeString(c.URL, ASCIIString))
	}
	if c.URLType != "" {
		b.fmt(" /URLType %s", c.URLType)
	}
	b.fmt(">>")
	return b.String()
}

type CertFlag uint8

const (
	CertSubject CertFlag = 1 << iota
	CertIssuer
	CertOID
	CertSubjectDN
	CertReserved
	CertKeyUsage
	CertURL
)

type SeedFlag int8

const (
	SeedFilter SeedFlag = 1 << iota
	SeedSubFilter
	SeedV
	SeedReasons
	SeedLegalAttestation
	SeedAddRevInfo
	SeedDigestMethod
)

type SignatureFlag uint32

const (
	SignaturesExist SignatureFlag = 1
	AppendOnly      SignatureFlag = 1 << 1
)

type AcroForm struct {
	Fields          []*FormField
	NeedAppearances bool
	SigFlags        SignatureFlag // optional, default to 0

	// (optional) references to field dictionaries with calculation actions, defining
	// the calculation order in which their values will be recalculated
	// when the value of any field changes
	CO []*FormField
	DR *ResourcesDict // optional
	DA string         // optional
	Q  int            // optional

	// TODO: support XFA forms
}

// TODO: AcroForm
func (a AcroForm) pdfString(pdf pdfWriter) string {
	return "<<>>"
}
