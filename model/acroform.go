package model

import (
	"fmt"
	"sort"
	"time"
)

// See Table 221 – Field flags common to all field types
// Table 226 – Field flags specific to button fields
// Table 228 – Field flags specific to text fields
// Table 230 – Field flags specific to choice fields
type FormFlag uint32

const (
	ReadOnly          FormFlag = 1 << (1 - 1)
	Required          FormFlag = 1 << (2 - 1)
	NoExport          FormFlag = 1 << (3 - 1)
	Multiline         FormFlag = 1 << (13 - 1)
	Password          FormFlag = 1 << (14 - 1)
	NoToggleToOff     FormFlag = 1 << (15 - 1)
	Radio             FormFlag = 1 << (16 - 1)
	Pushbutton        FormFlag = 1 << (17 - 1)
	FileSelect        FormFlag = 1 << (21 - 1)
	DoNotSpellCheck   FormFlag = 1 << (23 - 1)
	DoNotScroll       FormFlag = 1 << (24 - 1)
	Comb              FormFlag = 1 << (25 - 1)
	RadiosInUnison    FormFlag = 1 << (26 - 1)
	RichText          FormFlag = 1 << (26 - 1)
	Combo             FormFlag = 1 << (18 - 1)
	Edit              FormFlag = 1 << (19 - 1)
	Sort              FormFlag = 1 << (20 - 1)
	MultiSelect       FormFlag = 1 << (22 - 1)
	CommitOnSelChange FormFlag = 1 << (27 - 1)
)

type Quadding uint8

const (
	LeftJustified Quadding = iota
	Centered
	RightJustified
)

type FormFieldInheritable struct {
	FT FormField // inheritable, so might be nil
	Ff FormFlag  // optional
	Q  Quadding  // inheritable, optional, default to 0
	DA string    // inheritable, required
}

// use `parent` field if f one is empty
func (f FormFieldInheritable) merge(parent FormFieldInheritable) FormFieldInheritable {
	if f.FT == nil {
		f.FT = parent.FT
	}
	if f.Ff == 0 {
		f.Ff = parent.Ff
	}
	if f.Q == 0 {
		f.Q = parent.Q
	}
	if f.DA == "" {
		f.DA = parent.DA
	}
	return f
}

// FormFieldInherited associate to a field
// the values resolved from its ancestors
type FormFieldInherited struct {
	Field  *FormFieldDict
	Merged FormFieldInheritable
}

// FormFields are organized hierarchically into one or more tree structures.
// Many field attributes are inheritable, meaning that if they are not explicitly
// specified for a given field, their values are taken from those of its parent in the field hierarchy.
//
// This tree only defines the logic of the forms, not their appearance on the page.
// This is the purpose of the AnnotationWidget, defined in the Annots list of a
// page object. FormFields refer to them via the Widgets list.
//
// We depart from the SPEC in that all fields related to the specialisation
// of a field (attribut FT) are inherited (or not) at the same time.
// This should not be a problem in pratice, because if a parent
// changes its type, the values related to it should change as well.
// TODO: fix this
type FormFieldDict struct {
	FormFieldInheritable

	Parent *FormFieldDict   // nil for the top level fields
	Kids   []*FormFieldDict // nil for a leaf node

	// Widgets is not nil only for a leaf node,
	// and is represented in PDF under the /Kids entry,
	// or merged if their is only one widget.
	// *AnnotationDict must also be registered in
	// a PageObject.Annots list. See the PageObject.AddFormFieldWidget
	// for a convenient way of doing so.
	Widgets []FormFieldWidget

	// Partial field name (optional, text string)
	T string

	TU string                    // optional, text string, alternate field name
	TM string                    // optional, text string, mapping name
	AA FormFielAdditionalActions // optional

	// fields for variable text

	DS string // optional, text string
	RV string // optional, text string, may be written in PDF as a stream
}

func (f *FormFieldDict) resolve(parentName string, parentFields FormFieldInheritable, currentMap map[string]FormFieldInherited) {
	fullName := f.T
	if f.Parent != nil {
		fullName = parentName + "." + f.T
	}
	merged := f.FormFieldInheritable.merge(parentFields)
	currentMap[fullName] = FormFieldInherited{Field: f, Merged: merged}
	// recursion
	for _, kid := range f.Kids {
		kid.resolve(fullName, merged, currentMap)
	}
}

// AppearanceKeys returns the (sorted, unique) keys used in widgets appearances, usually used to check a checkbox.
//
// It returns an empty list if the field is not a [FormFieldButton].
//
// See 12.7.4.2.3 Check Boxes
func (f *FormFieldDict) AppearanceKeys() (keys []Name) {
	if _, ok := f.FT.(FormFieldButton); !ok {
		return nil
	}

	uniq := map[Name]bool{}
	for _, widget := range f.Widgets {
		if widget.AP == nil {
			continue
		}
		for key := range widget.AP.N {
			uniq[key] = true
		}
	}
	out := make([]Name, 0, len(uniq))
	for k := range uniq {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// FullFieldName returns the fully qualified field name, which is not explicitly defined
// but is constructed from the partial field names of the field
// and all of its ancestors.
// This is a convenient function, but not efficient if called
// on all the fields of the tree.
func (f *FormFieldDict) FullFieldName() string {
	if f.Parent == nil {
		return f.T
	}
	return f.Parent.FullFieldName() + "." + f.T
}

func (f *FormFieldDict) shouldBeMerged() (*AnnotationDict, bool) {
	if len(f.Kids) == 0 && len(f.Widgets) == 1 {
		return f.Widgets[0].AnnotationDict, true
	}
	return nil, false
}

// returns the fields to be included in the annotation dictionnary
func (f *FormFieldDict) mergedFields(pdf pdfWriter, ownRef Reference) string {
	b := newBuffer()
	if f.FT != nil { // might be nil if inherited
		b.WriteString(f.FT.formFieldAttrs(pdf, ownRef))
	}
	if f.Parent != nil {
		parent := pdf.fields[f.Parent]
		b.fmt("/Parent %s", parent)
	}
	if f.T != "" {
		b.fmt("/T %s", pdf.EncodeString(f.T, TextString, ownRef))
	}
	if f.TU != "" {
		b.fmt("/TU %s", pdf.EncodeString(f.TU, TextString, ownRef))
	}
	if f.TM != "" {
		b.fmt("/TM %s", pdf.EncodeString(f.TM, TextString, ownRef))
	}
	if f.Ff != 0 {
		b.fmt("/Ff %d", f.Ff)
	}
	if !f.AA.IsEmpty() {
		b.fmt("/AA %s", f.AA.pdfString(pdf, ownRef))
	}
	if f.Q != 0 {
		b.fmt("/Q %d", f.Q)
	}
	if f.DA != "" {
		b.line("/DA %s", pdf.EncodeString(f.DA, ByteString, ownRef))
	}
	if f.DS != "" {
		b.line("/DS %s", pdf.EncodeString(f.DS, TextString, ownRef))
	}
	if f.RV != "" {
		b.line("/RV %s", pdf.EncodeString(f.RV, TextString, ownRef))
	}
	return b.String()
}

// also allocate an object number for itself and stores it into pdf.fields
// pages annotations must have been written
func (f *FormFieldDict) pdfString(pdf pdfWriter) (content string, writeObject bool) {
	if annot, ok := f.shouldBeMerged(); ok {
		// do not create a new object : use the annotation ref
		ref := pdf.cache[annot]
		pdf.fields[f] = ref
		return ref.String(), false
	}

	// before recursing, first register it's own ref
	// so that it is accessible by the kids
	ownRef := pdf.CreateObject()
	pdf.fields[f] = ownRef // register to the cache

	fields := f.mergedFields(pdf, ownRef)

	var kids string
	if len(f.Kids) != 0 {
		refs := make([]Reference, len(f.Kids))
		for i, kid := range f.Kids {
			kidS, write := kid.pdfString(pdf)
			kidRef := pdf.fields[kid] // now valid
			if write {
				pdf.WriteObject(kidS, kidRef)
			}
			refs[i] = kidRef
		}
		kids = writeRefArray(refs)
	} else if len(f.Widgets) != 0 {
		// we use the annotations previously written
		refs := make([]Reference, len(f.Widgets))
		for i, w := range f.Widgets {
			refs[i] = pdf.cache[w.AnnotationDict]
		}
		kids = writeRefArray(refs)
	}

	return fmt.Sprintf("<<%s /Kids %s>>", fields, kids), true
}

// also stores into pdf.fields
func (f *FormFieldDict) clone(cache cloneCache) *FormFieldDict {
	if f == nil {
		return nil
	}
	out := *f
	out.Parent = cache.fields[f.Parent]
	// before recursing, store the clone
	cache.fields[f] = &out
	if f.FT != nil {
		out.FT = f.FT.clone(cache)
	}
	if f.Widgets != nil { // preserve nil
		out.Widgets = make([]FormFieldWidget, len(f.Widgets))
		for i, w := range f.Widgets {
			out.Widgets[i] = w.clone(cache)
		}
	}
	if f.Kids != nil { // preserve nil
		out.Kids = make([]*FormFieldDict, len(f.Kids))
		for i, k := range f.Kids {
			out.Kids[i] = k.clone(cache)
		}
	}
	out.AA = f.AA.clone(cache)
	return &out
}

// ---------------------------------------------------

// FormFieldWidget is an annotation
// which must have a type of FormFieldWidget
type FormFieldWidget struct {
	*AnnotationDict
}

func (w FormFieldWidget) clone(cache cloneCache) FormFieldWidget {
	if w.AnnotationDict == nil {
		return FormFieldWidget{}
	}
	out := cache.checkOrClone(w.AnnotationDict).(*AnnotationDict)
	return FormFieldWidget{AnnotationDict: out}
}

// FormField provides additional form attributes,
// depending on the field type.
type FormField interface {
	// must include the type entry/FT
	// `pdf.catalog` is used by FieldSignature
	// `fieldRef` is the reference of the field dict object
	formFieldAttrs(pdf pdfWriter, fieldRef Reference) string
	// return a deep copy, preserving concrete type
	clone(cache cloneCache) FormField
}

// FormFieldText are boxes or spaces in which the user can enter text from the keyboard.
type FormFieldText struct {
	V      string   // text string, may be written in PDF as a stream
	MaxLen MaybeInt // optional
}

func (f FormFieldText) formFieldAttrs(pdf pdfWriter, fieldRef Reference) string {
	out := fmt.Sprintf("/FT/Tx/V %s", pdf.EncodeString(f.V, TextString, fieldRef))
	if f.MaxLen != nil {
		out += fmt.Sprintf("/MaxLen %d", f.MaxLen.(ObjInt))
	}
	return out
}

func (f FormFieldText) clone(cloneCache) FormField { return f }

// FormFieldButton represent interactive controls on the screen
// that the user can manipulate with the mouse.
// They include pushbuttons, check boxes, and radio buttons.
type FormFieldButton struct {
	V   Name     // check box’s appearance state
	Opt []string // optional, text strings, same length as Widgets
}

func (f FormFieldButton) formFieldAttrs(pdf pdfWriter, fieldRef Reference) string {
	out := "/FT/Btn"
	if f.V != "" {
		out += "/V " + f.V.String()
	}
	if len(f.Opt) != 0 {
		out += fmt.Sprintf("/Opt [%s]", writeStringsArray(f.Opt, pdf, TextString, fieldRef))
	}
	return out
}

func (f FormFieldButton) clone(cloneCache) FormField {
	out := f
	out.Opt = append([]string(nil), f.Opt...)
	return out
}

// Option is either a text string representing one of the available options or an
// array consisting of two text strings: the option’s export value and the text that
// shall be displayed as the name of the option.
type Option struct {
	Export string // optional
	Name   string
}

func (o Option) pdfString(pdf pdfWriter, context Reference) string {
	if o.Export == "" {
		return pdf.EncodeString(o.Name, TextString, context)
	}
	return fmt.Sprintf("[%s %s]", pdf.EncodeString(o.Export, TextString, context),
		pdf.EncodeString(o.Name, TextString, context))
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

func (f FormFieldChoice) formFieldAttrs(pdf pdfWriter, fieldRef Reference) string {
	b := newBuffer()
	b.fmt("/FT/Ch")
	if len(f.V) == 0 {
		b.fmt("/V null")
	} else if len(f.V) == 1 {
		b.fmt("/V %s", pdf.EncodeString(f.V[0], TextString, fieldRef))
	} else {
		b.fmt("/V %s", writeStringsArray(f.V, pdf, TextString, fieldRef))
	}
	if len(f.Opt) != 0 {
		b.fmt("/Opt [")
		for _, o := range f.Opt {
			b.fmt(" " + o.pdfString(pdf, fieldRef))
		}
		b.fmt("]")
	}
	if f.TI != 0 {
		b.fmt("/TI %d", f.TI)
	}
	if len(f.I) != 0 {
		b.fmt("/I %s", writeIntArray(f.I))
	}
	return b.String()
}

func (f FormFieldChoice) clone(cloneCache) FormField {
	out := f
	out.V = append([]string(nil), f.V...)
	out.Opt = append([]Option(nil), f.Opt...)
	out.I = append([]int(nil), f.I...)
	return out
}

// FormFieldSignature represent digital signatures and
// optional data for authenticating the name of the signer and
// the document’s contentstream.
type FormFieldSignature struct {
	V    *SignatureDict // optional
	Lock *LockDict      // optional
	SV   *SeedDict      // optional
}

func (f FormFieldSignature) formFieldAttrs(pdf pdfWriter, fieldRef Reference) string {
	out := "/FT/Sig"
	if f.V != nil {
		out += fmt.Sprintf("/V %s", f.V.pdfString(pdf, fieldRef))
	}
	if lock := f.Lock; lock != nil {
		ref := pdf.addObject(f.Lock.pdfString(pdf, fieldRef))
		out += fmt.Sprintf("/Lock %s", ref)
	}
	if sv := f.SV; sv != nil {
		ref := pdf.addObject(f.SV.pdfString(pdf, fieldRef))
		out += fmt.Sprintf("/SV %s", ref)
	}
	return out
}

func (f FormFieldSignature) clone(cache cloneCache) FormField {
	out := f
	out.V = f.V.Clone()
	out.Lock = f.Lock.Clone()
	out.SV = f.SV.Clone()
	return out
}

type SignatureDict struct {
	Filter      Name               // optional
	SubFilter   Name               // optional
	Contents    string             // byte string, written as hexadecimal in PDF
	Cert        []string           // optional, byte strings. One-element arrays may be written in PDF a a single byte string
	ByteRange   [][2]int           // optional. Written in PDF as an array of pairs
	Reference   []SignatureRefDict // optional
	Changes     [3]int             // optional
	Name        string             // optional, text string
	M           time.Time          // optional
	Location    string             // optional, text string
	Reason      string             // optional, text string
	ContactInfo string             // optional, text string
	V           int                // optional

	// Prop_Build is implementation-specific by design.
	// It can be used to store audit information that is specific to the software application
	// that was used to create the signature.
	Prop_Build    Object    // optional
	Prop_AuthTime time.Time // optional
	Prop_AuthType Name      // optional
}

func (s SignatureDict) pdfString(pdf pdfWriter, fieldRef Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if s.Filter != "" {
		b.fmt("/Filter %s", s.Filter)
	}
	if s.SubFilter != "" {
		b.fmt("/SubFiler %s", s.SubFilter)
	}
	b.fmt("/Contents %s", pdf.EncodeString(s.Contents, HexString, fieldRef))
	if len(s.Cert) != 0 {
		b.fmt("/Cert %s", writeStringsArray(s.Cert, pdf, TextString, fieldRef))
	}
	if len(s.ByteRange) != 0 {
		b.fmt("/ByteRange [")
		for _, val := range s.ByteRange {
			b.fmt(" %d %d", val[0], val[1])
		}
		b.fmt("]")
	}
	if len(s.Reference) != 0 {
		b.fmt("/Reference [")
		for _, val := range s.Reference {
			b.fmt(" %s", val.pdfString(pdf, fieldRef))
		}
		b.fmt("]")
	}
	if s.Changes != [3]int{} {
		b.fmt("/Changes %s", writeIntArray(s.Changes[:]))
	}
	if s.Name != "" {
		b.fmt("/Name %s", pdf.EncodeString(s.Name, TextString, fieldRef))
	}
	if s.Location != "" {
		b.fmt("/Location %s", pdf.EncodeString(s.Location, TextString, fieldRef))
	}
	if s.Reason != "" {
		b.fmt("/Reason %s", pdf.EncodeString(s.Reason, TextString, fieldRef))
	}
	if s.ContactInfo != "" {
		b.fmt("/ContactInfo %s", pdf.EncodeString(s.ContactInfo, TextString, fieldRef))
	}
	if !s.M.IsZero() {
		b.fmt("/M %s", pdf.dateString(s.M, fieldRef))
	}
	if s.V != 0 {
		b.fmt("/V %d", s.V)
	}
	if s.Prop_Build != nil {
		b.fmt("/Prop_Build %s", s.Prop_Build.Write(pdf, fieldRef))
	}
	if !s.Prop_AuthTime.IsZero() {
		b.fmt("/Prop_AuthTime %s", pdf.dateString(s.Prop_AuthTime, fieldRef))
	}
	if s.Prop_AuthType != "" {
		b.fmt("/Prop_AuthType %s", s.Prop_AuthType)
	}
	b.fmt(">>")
	return b.String()
}

// Clone returns a deep copy
func (s *SignatureDict) Clone() *SignatureDict {
	if s == nil {
		return nil
	}
	out := *s
	out.Cert = append([]string(nil), s.Cert...)
	out.ByteRange = append([][2]int(nil), s.ByteRange...)
	if s.Reference != nil { // preserve nil
		out.Reference = make([]SignatureRefDict, len(s.Reference))
	}
	for i, r := range s.Reference {
		out.Reference[i] = r.Clone()
	}
	if s.Prop_Build != nil {
		out.Prop_Build = s.Prop_Build.Clone()
	}
	return &out
}

// SignatureRefDict is a signature reference dictionary
// Note: The SPEC does not restrict the Data attribute, but
// we, as other libraries, do: we only allow it to point to the Catalog.
type SignatureRefDict struct {
	TransformMethod Name // among DocMDP, UR, FieldMDP
	TransformParams Transform

	DigestMethod Name
}

func (s SignatureRefDict) pdfString(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("<</TransformMethod %s/TransformParams %s/DigestMethod %s ∕Data %s>>",
		s.TransformMethod, s.TransformParams.transformParamsDict(pdf, ref), s.DigestMethod, pdf.catalog)
}

// Clone returns a deep copy
func (s SignatureRefDict) Clone() SignatureRefDict {
	out := s
	if s.TransformParams != nil {
		out.TransformParams = s.TransformParams.Clone()
	}
	return out
}

// Transform determines which objects are included and excluded
// in revision comparison
type Transform interface {
	transformParamsDict(pdf pdfWriter, ref Reference) string
	// Clone returns a deep copy of the transform, preserving the concrete type.
	Clone() Transform
}

type TransformDocMDP struct {
	P uint // optional; among 1,2,3 ; default to 2
	V Name // optional
}

func (t TransformDocMDP) transformParamsDict(pdfWriter, Reference) string {
	out := "<<"
	if t.P != 0 {
		out += fmt.Sprintf("/P %d", t.P)
	}
	if t.V != "" {
		out += fmt.Sprintf("/V %s", t.V)
	}
	out += ">>"
	return out
}

func (t TransformDocMDP) Clone() Transform { return t }

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

func (t TransformUR) transformParamsDict(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if len(t.Document) != 0 {
		b.fmt("/Document %s", writeNameArray(t.Document))
	}
	if t.Msg != "" {
		b.fmt("/Msg %s", pdf.EncodeString(t.Msg, TextString, ref))
	}
	if t.V != "" {
		b.fmt("/V %s", t.V)
	}
	if len(t.Annots) != 0 {
		b.fmt("/Annots %s", writeNameArray(t.Annots))
	}
	if len(t.Form) != 0 {
		b.fmt("/Form %s", writeNameArray(t.Form))
	}
	if len(t.Signature) != 0 {
		b.fmt("/Signature %s", writeNameArray(t.Signature))
	}
	if len(t.EF) != 0 {
		b.fmt("/EF %s", writeNameArray(t.EF))
	}
	b.fmt("/P %v>>", t.P)
	return b.String()
}

func (t TransformUR) Clone() Transform {
	out := t
	out.Document = append([]Name(nil), t.Document...)
	out.Annots = append([]Name(nil), t.Annots...)
	out.Form = append([]Name(nil), t.Form...)
	out.Signature = append([]Name(nil), t.Signature...)
	out.EF = append([]Name(nil), t.EF...)
	return out
}

// TransformFieldMDP is used to detect changes to the values of a list of form fields.
type TransformFieldMDP struct {
	Action Name
	Fields []string // text strings, optional is Action == All
	V      Name
}

func (t TransformFieldMDP) transformParamsDict(pdf pdfWriter, ref Reference) string {
	out := fmt.Sprintf("<</Action %s", t.Action)
	if len(t.Fields) != 0 {
		out += fmt.Sprintf("/Fields %s", writeStringsArray(t.Fields, pdf, TextString, ref))
	}
	out += fmt.Sprintf("/V %s>>", t.V)
	return out
}

func (t TransformFieldMDP) Clone() Transform {
	out := t
	out.Fields = append([]string(nil), t.Fields...)
	return out
}

type LockDict struct {
	Action Name     // one of All,Include,Exclude
	Fields []string // field names, text strings, optional when Action == All
}

func (l LockDict) pdfString(pdf pdfWriter, ref Reference) string {
	out := fmt.Sprintf("<</Action %s", l.Action)
	if len(l.Fields) != 0 {
		out += fmt.Sprintf("/Fields %s", writeStringsArray(l.Fields, pdf, TextString, ref))
	}
	out += ">>"
	return out
}

func (l *LockDict) Clone() *LockDict {
	if l == nil {
		return nil
	}
	out := *l
	out.Fields = append([]string(nil), l.Fields...)
	return &out
}

type SeedDict struct {
	Ff           SeedFlag  // optional, default to 0
	Filter       Name      // optional
	SubFilter    []Name    // optional
	DigestMethod []Name    // optional, names among SHA1, SHA256, SHA384, SHA512 and RIPEMD160
	V            Fl        // optional
	Cert         *CertDict // optional
	Reasons      []string  // optional, text strings
	// optional,  from 0 to 3
	// writen in pdf as a dictionary with entry P
	MDP              MaybeInt
	TimeStamp        *TimeStampDict // optional
	LegalAttestation []string       // optional, text strings
	AddRevInfo       bool           // optional, default to false
}

func (s SeedDict) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if s.Ff != 0 {
		b.fmt("/Ff %d", s.Ff)
	}
	if s.Filter != "" {
		b.fmt("/Filter %s", s.Filter)
	}
	if len(s.SubFilter) != 0 {
		b.fmt("/SubFilter %s", writeNameArray(s.SubFilter))
	}
	if len(s.DigestMethod) != 0 {
		b.fmt("/DigestMethod %s", writeNameArray(s.DigestMethod))
	}
	if s.V != 0 {
		b.fmt("/V %s", FmtFloat(s.V))
	}
	if s.Cert != nil {
		b.fmt("/Cert %s", s.Cert.pdfString(pdf, ref))
	}
	if len(s.Reasons) != 0 {
		b.fmt("/Reasons %s", writeStringsArray(s.Reasons, pdf, TextString, ref))
	}
	if s.MDP != nil {
		b.fmt("/MDP <</P %d>>", s.MDP.(ObjInt))
	}
	if s.TimeStamp != nil {
		b.fmt("/TimeStamp %s", s.TimeStamp.pdfString(pdf, ref))
	}
	if len(s.LegalAttestation) != 0 {
		b.fmt("/LegalAttestation %s", writeStringsArray(s.LegalAttestation, pdf, TextString, ref))
	}
	b.fmt("/AddRevInfo %v>>", s.AddRevInfo)
	return b.String()
}

func (s *SeedDict) Clone() *SeedDict {
	if s == nil {
		return nil
	}
	out := *s
	out.SubFilter = append([]Name(nil), s.SubFilter...)
	out.DigestMethod = append([]Name(nil), s.DigestMethod...)
	out.Reasons = append([]string(nil), s.Reasons...)
	out.LegalAttestation = append([]string(nil), s.LegalAttestation...)
	out.Cert = s.Cert.Clone()
	out.TimeStamp = s.TimeStamp.Clone()
	return &out
}

type TimeStampDict struct {
	URL string // URL must be ASCII string
	Ff  uint8  // 0 or 1, default to 0
}

func (s TimeStampDict) pdfString(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("<</URL %s/Ff %d>>", pdf.EncodeString(s.URL, ByteString, ref), s.Ff)
}

func (t *TimeStampDict) Clone() *TimeStampDict {
	if t == nil {
		return t
	}
	out := *t
	return &out
}

// CertDict contains characteristics of the certificate that shall be used when signing
type CertDict struct {
	Ff        uint8             // optional, default to 0
	Subject   []string          // optional byte strings
	SubjectDN []map[Name]string // optional, each map values are text strings
	KeyUsage  []string          // optional, must be ASCII strings
	Issuer    []string          // optional, byte strings
	OID       []string          // optional, byte strings
	URL       string            // optional, must be ASCII string
	URLType   Name              // optional
}

func (c CertDict) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if c.Ff != 0 {
		b.fmt("/Ff %d", c.Ff)
	}
	if len(c.Subject) != 0 {
		b.fmt("/Subject %s", writeStringsArray(c.Subject, pdf, ByteString, ref))
	}
	if len(c.SubjectDN) != 0 {
		b.fmt("/SubjectDN [")
		for _, dn := range c.SubjectDN {
			b.WriteString("<<")
			for name, value := range dn {
				b.fmt("%s %s ", name, pdf.EncodeString(value, TextString, ref))
			}
			b.fmt(">> ")
		}
		b.fmt("]")
	}
	if len(c.KeyUsage) != 0 {
		b.fmt("/KeyUsage %s", writeStringsArray(c.KeyUsage, pdf, ByteString, ref))
	}
	if len(c.Issuer) != 0 {
		b.fmt("/Issuer %s", writeStringsArray(c.Issuer, pdf, ByteString, ref))
	}
	if len(c.OID) != 0 {
		b.fmt("/OID %s", writeStringsArray(c.OID, pdf, ByteString, ref))
	}
	if c.URL != "" {
		b.fmt("/URL %s", pdf.EncodeString(c.URL, ByteString, ref))
	}
	if c.URLType != "" {
		b.fmt("/URLType %s", c.URLType)
	}
	b.fmt(">>")
	return b.String()
}

// Clone returns a deep copy
func (c *CertDict) Clone() *CertDict {
	if c == nil {
		return nil
	}
	out := *c
	out.Subject = append([]string(nil), c.Subject...)
	out.KeyUsage = append([]string(nil), c.KeyUsage...)
	out.Issuer = append([]string(nil), c.Issuer...)
	out.OID = append([]string(nil), c.OID...)
	if c.SubjectDN != nil { // preserve nil
		out.SubjectDN = make([]map[Name]string, len(c.SubjectDN))
		for i, m := range c.SubjectDN {
			var newM map[Name]string
			if m != nil { // preserve nil
				newM = make(map[Name]string, len(newM))
				for n, v := range m {
					newM[n] = v
				}
			}
			out.SubjectDN[i] = newM
		}
	}
	return &out
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
	Fields          []*FormFieldDict
	NeedAppearances bool
	SigFlags        SignatureFlag // optional, default to 0

	// (optional) references to field dictionaries with calculation actions, defining
	// the calculation order in which their values will be recalculated
	// when the value of any field changes
	CO []*FormFieldDict
	DR ResourcesDict // optional
	DA string        // optional
	Q  Quadding      // optional, default to 0

	// TODO: support XFA forms
}

// Flatten walk the tree of form fields and accumulate them
// in a map, resolving the inheritance and forming the fully qualified names
// used as keys of the returned map.
func (a AcroForm) Flatten() map[string]FormFieldInherited {
	out := make(map[string]FormFieldInherited)
	for _, kid := range a.Fields {
		kid.resolve("", FormFieldInheritable{DA: a.DA, Q: a.Q}, out)
	}
	return out
}

func (a AcroForm) toBeMerged() map[*AnnotationDict]*FormFieldDict {
	out := make(map[*AnnotationDict]*FormFieldDict)

	var aux func(field *FormFieldDict)
	aux = func(field *FormFieldDict) {
		if annot, ok := field.shouldBeMerged(); ok {
			out[annot] = field
			return
		}

		// recurse on Kids
		for _, kid := range field.Kids {
			aux(kid)
		}
	}

	for _, fi := range a.Fields {
		aux(fi)
	}

	return out
}

func (a AcroForm) pdfString(pdf pdfWriter, acroRef Reference) string {
	b := newBuffer()
	refs := make([]Reference, len(a.Fields))
	for i, f := range a.Fields {
		s, write := f.pdfString(pdf)
		fieldRef := pdf.fields[f]
		if write {
			pdf.WriteObject(s, fieldRef)
		}
		refs[i] = fieldRef
	}
	b.fmt("<</Fields %s", writeRefArray(refs))
	b.fmt("/NeedAppearances %v", a.NeedAppearances)
	if a.SigFlags != 0 {
		b.fmt("/SigFlags %d", a.SigFlags)
	}
	if len(a.CO) != 0 { // wil use the ref from the cache
		refs := make([]Reference, len(a.CO))
		for i, co := range a.CO {
			refs[i] = pdf.fields[co]
		}
		b.fmt("/CO %s", writeRefArray(refs))
	}
	if !a.DR.IsEmpty() {
		ref := pdf.addObject(a.DR.pdfString(pdf, acroRef))
		b.fmt("/DR %s", ref)
	}
	if a.DA != "" {
		b.fmt("/DA %s", pdf.EncodeString(a.DA, ByteString, acroRef))
	}
	if a.Q != 0 {
		b.fmt("/Q %d", a.Q)
	}
	b.fmt(">>")
	return b.String()
}

func (a AcroForm) clone(cache cloneCache) AcroForm {
	out := a
	if a.Fields != nil { // preserve nil
		out.Fields = make([]*FormFieldDict, len(a.Fields))
		for i, f := range a.Fields {
			out.Fields[i] = f.clone(cache)
		}
	}
	if a.CO != nil { // preserve nil
		out.CO = make([]*FormFieldDict, len(a.CO))
		for i, c := range a.CO {
			out.CO[i] = cache.fields[c]
		}
	}
	out.DR = a.DR.clone(cache)
	return out
}
