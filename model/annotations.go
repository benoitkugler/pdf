package model

import (
	"fmt"
	"strings"
	"time"
)

// const (
// 	Text           AnnotationType = "Text"
// 	Link           AnnotationType = "Link"
// 	FreeText       AnnotationType = "FreeText"
// 	Line           AnnotationType = "Line"
// 	Square         AnnotationType = "Square"
// 	Circle         AnnotationType = "Circle"
// 	Polygon        AnnotationType = "Polygon"
// 	PolyLine       AnnotationType = "PolyLine"
// 	Highlight      AnnotationType = "Highlight"
// 	Underline      AnnotationType = "Underline"
// 	Squiggly       AnnotationType = "Squiggly"
// 	StrikeOut      AnnotationType = "StrikeOut"
// 	Stamp          AnnotationType = "Stamp"
// 	Caret          AnnotationType = "Caret"
// 	Ink            AnnotationType = "Ink"
// 	Popup          AnnotationType = "Popup"
// 	FileAttachment AnnotationType = "FileAttachment"
// 	Sound          AnnotationType = "Sound"
// 	Movie          AnnotationType = "Movie"
// 	Widget         AnnotationType = "Widget"
// 	Screen         AnnotationType = "Screen"
// 	PrinterMark    AnnotationType = "PrinterMark"
// 	TrapNet        AnnotationType = "TrapNet"
// 	Watermark      AnnotationType = "Watermark"
// 	ThreeD         AnnotationType = "3D"
// 	Redact         AnnotationType = "Redact"
// )

// Border is written in PDF as an array of 3 or 4 elements
type Border struct {
	HCornerRadius, VCornerRadius, BorderWidth float64
	DashArray                                 []float64 // optional (nil not to specify it)
}

func (b Border) pdfString() string {
	out := fmt.Sprintf("[%3.f %3.f %3.f", b.HCornerRadius, b.VCornerRadius, b.BorderWidth)
	if b.DashArray != nil {
		out += " " + writeFloatArray(b.DashArray)
	}
	return out + "]"
}

// Clone returns a deep copy
func (b *Border) Clone() *Border {
	if b == nil {
		return nil
	}
	out := *b
	out.DashArray = append([]float64(nil), b.DashArray...)
	return &out
}

// BorderStyle specifies the border characteristics for some types of annotations
type BorderStyle struct {
	W MaybeFloat // optional, default to 1
	S Name       // optional
	D []float64  // optional, default to [3], nil not to specify it
}

// String returns the PDF dictionary representing the border style.
func (bo BorderStyle) String() string {
	b := newBuffer()
	b.WriteString("<<")
	if bo.W != nil {
		b.fmt("/W %.3f", bo.W.(Float))
	}
	if bo.S != "" {
		b.fmt("/S %s", bo.S)
	}
	if bo.D != nil {
		b.fmt("/D %s", writeFloatArray(bo.D))
	}
	b.fmt(">>")
	return b.String()
}

// Clone returns a deep copy
func (b *BorderStyle) Clone() *BorderStyle {
	if b == nil {
		return nil
	}
	out := *b
	out.D = append([]float64(nil), b.D...)
	return &out
}

// BorderEffect specifies an effect that shall be applied to the border of the annotations
// See Table 167 – Entries in a border effect dictionary
type BorderEffect struct {
	S Name    // optional
	I float64 // optional
}

// String returns the PDF dictionary .
func (b BorderEffect) String() string {
	return fmt.Sprintf("<</S %s/I %.3f>>", b.S, b.I)
}

func (b *BorderEffect) Clone() *BorderEffect {
	if b == nil {
		return nil
	}
	out := *b
	return &out
}

// AnnotationFlag describe the behaviour of an annotation. See Table 165 – Annotation flags
type AnnotationFlag uint16

const (
	// Do not display the annotation if it does not belong to one of the
	// standard annotation types and no annotation handler is available.
	AInvisible AnnotationFlag = 1 << (1 - 1)
	// Do not display or print the annotation or allow it to
	// interact with the user, regardless of its annotation type or whether an
	// annotation handler is available.
	AHidden AnnotationFlag = 1 << (2 - 1)
	// Print the annotation when the page is printed.
	APrint AnnotationFlag = 1 << (3 - 1)
	// Do not scale the annotation’s appearance to match the
	// magnification of the page.
	ANoZoom AnnotationFlag = 1 << (4 - 1)
	// Do not rotate the annotation’s appearance to match
	// the rotation of the page.
	ANoRotate AnnotationFlag = 1 << (5 - 1)
	// Do not display the annotation on the screen or allow it
	// to interact with the user.
	ANoView AnnotationFlag = 1 << (6 - 1)
	// Do not allow the annotation to interact with the user.
	AReadOnly AnnotationFlag = 1 << (7 - 1)
	// Do not allow the annotation to be deleted or its
	// properties (including position and size) to be modified by the user.
	ALocked AnnotationFlag = 1 << (8 - 1)
	// Invert the interpretation of the NoView flag for certain
	// events.
	AToggleNoView AnnotationFlag = 1 << (9 - 1)
	// Do not allow the contents of the annotation to be
	// modified by the user.
	ALockedContents AnnotationFlag = 1 << (10 - 1)
)

type BaseAnnotation struct {
	Rect     Rectangle
	Contents string          // optional
	NM       string          // optional
	M        time.Time       // optional
	AP       *AppearanceDict // optional
	// Appearance state (key of the AP.N subDictionary).
	// Required if the appearance dictionary AP contains one or more
	// subdictionaries
	AS           Name
	F            AnnotationFlag // optional
	Border       *Border        // optional
	C            []float64      // 0, 1, 3 or 4 numbers in the range 0.0 to 1.0
	StructParent MaybeInt       // required if the annotation is a structural content item
}

func (ba BaseAnnotation) fields(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.fmt("/Rectangle %s", ba.Rect)
	if ba.Contents != "" {
		b.fmt("/Contents %s", pdf.EncodeString(ba.Contents, TextString, ref))
	}
	if ba.NM != "" {
		b.fmt("/NM %s", pdf.EncodeString(ba.NM, TextString, ref))
	}
	if !ba.M.IsZero() {
		b.fmt("/M %s", pdf.dateString(ba.M, ref))
	}
	if ap := ba.AP; ap != nil {
		b.fmt("/AP %s", ap.pdfString(pdf))
	}
	if as := ba.AS; as != "" {
		b.fmt("/AS %s", as)
	}
	if f := ba.F; f != 0 {
		b.fmt("/F %d", f)
	}
	if bo := ba.Border; bo != nil {
		b.fmt("/Border %s", bo.pdfString())
	}
	if len(ba.C) != 0 {
		b.fmt("/C %s", writeFloatArray(ba.C))
	}
	if ba.StructParent != nil {
		b.fmt("/StructParent %d", ba.StructParent.(Int))
	}
	return b.String()
}

func (ba BaseAnnotation) clone(cache cloneCache) BaseAnnotation {
	out := ba
	out.AP = ba.AP.clone(cache)
	out.Border = ba.Border.Clone()
	if ba.C != nil {
		out.C = append([]float64(nil), ba.C...)
	}
	return out
}

// AnnotationMarkup groups attributes common to all markup annotations
// See Table 170 – Additional entries specific to markup annotations
// Markup annotations are:
//	- Text
//	- FreeText
//	- Line
//	- Square
//	- Circle
//	- Polygon
//	- PolyLine
//	- Highlight
//	- Underline
//	- Squiggly
//	- StrikeOut
//	- Stamp
//	- Caret
//	- Ink
//	- FileAttachment
//	- Sound
//	- Redact
type AnnotationMarkup struct {
	T            string           // optional
	Popup        *AnnotationPopup // optional, written as an indirect reference
	CA           MaybeFloat       // optional
	RC           string           // optional, may be written in PDF as a text stream
	CreationDate time.Time        // optional
	Subj         string           // optional
	IT           Name             // optional
	// TODO: reply to
}

func (a AnnotationMarkup) clone(cache cloneCache) AnnotationMarkup {
	out := a
	if a.Popup != nil {
		out.Popup = a.Popup.clone(cache)
	}
	return out
}

func (a AnnotationMarkup) pdfFields(pdf pdfWriter, context Reference) string {
	b := newBuffer()
	if a.T != "" {
		b.fmt("/T %s", pdf.EncodeString(a.T, TextString, context))
	}
	if a.Popup != nil {
		// the context is also the parent
		ref := pdf.addObject(a.Popup.pdfString(pdf, context), nil)
		b.fmt("/Popup %s", ref)
	}
	if a.CA != nil {
		b.fmt("/CA %.3f", a.CA.(Float))
	}
	if a.RC != "" {
		b.fmt("/RC %s", pdf.EncodeString(a.RC, TextString, context))
	}
	if !a.CreationDate.IsZero() {
		b.fmt("/CreationDate %s", pdf.dateString(a.CreationDate, context))
	}
	if a.Subj != "" {
		b.fmt("/Subj %s", pdf.EncodeString(a.Subj, TextString, context))
	}
	if a.IT != "" {
		b.fmt("/IT %s", a.IT)
	}
	return b.String()
}

// AnnotationPopup is an annotation with a static type of Popup.
// It is not used as a standalone annotation, but in a markup annotation.
// Its Parent field is deduced from its container.
type AnnotationPopup struct {
	BaseAnnotation
	Open bool // optional
}

func (an *AnnotationPopup) clone(cache cloneCache) *AnnotationPopup {
	if an == nil {
		return nil
	}
	out := *an
	out.BaseAnnotation = an.BaseAnnotation.clone(cache)
	return &out
}

func (a AnnotationPopup) pdfString(pdf pdfWriter, parent Reference) string {
	common := a.BaseAnnotation.fields(pdf, parent)
	return fmt.Sprintf("<</Subtype/Popup %s /Open %v/Parent %s>>", common, a.Open, parent)
}

type AnnotationDict struct {
	BaseAnnotation
	Subtype Annotation
}

// GetStructParent implements StructParentObject
func (a *AnnotationDict) GetStructParent() MaybeInt {
	return a.StructParent
}

// pdfContent impements is cachable
func (a *AnnotationDict) pdfContent(pdf pdfWriter, ref Reference) (string, []byte) {
	base := a.BaseAnnotation.fields(pdf, ref)
	subtype := a.Subtype.annotationFields(pdf, ref)
	return fmt.Sprintf("<<%s %s >>", base, subtype), nil
}

func (a *AnnotationDict) clone(cache cloneCache) Referenceable {
	if a == nil {
		return a
	}
	out := *a
	out.BaseAnnotation = a.BaseAnnotation.clone(cache)
	out.Subtype = a.Subtype.clone(cache)
	return &out
}

type AppearanceDict struct {
	N AppearanceEntry // annotation’s normal appearance
	R AppearanceEntry // annotation’s rollover appearance, optional, default to N
	D AppearanceEntry // annotation’s down appearance, optional, default to N
}

func (a AppearanceDict) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.WriteString("<<")
	if a.N != nil {
		b.fmt("/N %s", a.N.pdfString(pdf))
	}
	if a.R != nil {
		b.fmt("/R %s", a.R.pdfString(pdf))
	}
	if a.D != nil {
		b.fmt("/D %s", a.D.pdfString(pdf))
	}
	b.fmt(">>")
	return b.String()
}

func (ap *AppearanceDict) clone(cache cloneCache) *AppearanceDict {
	if ap == nil {
		return nil
	}
	out := *ap
	out.N = ap.N.clone(cache)
	out.R = ap.R.clone(cache)
	out.D = ap.D.clone(cache)
	return &out
}

// AppearanceEntry is either a Dictionary, or a subDictionary
// containing multiple appearances
// In the first case, the map is of length 1, with the empty string as key
type AppearanceEntry map[Name]*XObjectForm

// pdfString returns the Dictionary for the appearance
// `pdf` is used to write the form XObjects
func (ap AppearanceEntry) pdfString(pdf pdfWriter) string {
	chunks := make([]string, 0, len(ap))
	for n, f := range ap {
		ref := pdf.addItem(f)
		chunks = append(chunks, fmt.Sprintf("%s %s", n, ref))
	}
	return fmt.Sprintf("<<%s>>", strings.Join(chunks, " "))
}

func (ap AppearanceEntry) clone(cache cloneCache) AppearanceEntry {
	if ap == nil { // preserve reflect.DeepEqual
		return nil
	}
	out := make(AppearanceEntry, len(ap))
	for name, form := range ap {
		out[name] = cache.checkOrClone(form).(*XObjectForm)
	}
	return out
}

// Annotation associates an object such as a note, sound, or movie
// with a location on a page of a PDF document,
// or provides a way to interact with the user by means of the mouse and keyboard.
type Annotation interface {
	// return the specialized fields (including Subtype)
	annotationFields(pdf pdfWriter, ref Reference) string
	clone(cloneCache) Annotation
}

// ------------------------ specializations ------------------------

// AnnotationText represents a “sticky note” attached to a point in the PDF document.
// See Table 172 – Additional entries specific to a text annotation.
type AnnotationText struct {
	AnnotationMarkup
	Open       bool   // optional
	Name       Name   // optional
	State      string // optional
	StateModel string // optional
}

func (f AnnotationText) annotationFields(pdf pdfWriter, ref Reference) string {
	out := "/Subtype/Text " + f.AnnotationMarkup.pdfFields(pdf, ref)
	if f.Open {
		out += fmt.Sprintf("/Open %v", f.Open)
	}
	if f.Name != "" {
		out += fmt.Sprintf("/Name %s", f.Name)
	}
	if f.State != "" {
		out += fmt.Sprintf("/State %s", pdf.EncodeString(f.State, TextString, ref))
	}
	if f.StateModel != "" {
		out += fmt.Sprintf("/StateModel %s", pdf.EncodeString(f.StateModel, TextString, ref))
	}
	return out
}

func (f AnnotationText) clone(cache cloneCache) Annotation {
	out := f
	out.AnnotationMarkup = f.AnnotationMarkup.clone(cache)
	return out
}

// ----------------------------------------------------------

// AnnotationLink either opens an URI (field A)
// or an internal page (field Dest)
// See Table 173 – Additional entries specific to a link annotation
type AnnotationLink struct {
	A          Action       // optional, represented by a dictionary in PDF
	Dest       Destination  // may only be present is A is nil
	H          Highlighting // optional
	PA         Action       // optional, of type ActionURI
	QuadPoints []float64    // optional, length 8 x n
	BS         *BorderStyle // optional
}

func (l AnnotationLink) annotationFields(pdf pdfWriter, ref Reference) string {
	out := "/Subtype/Link"
	if l.A.ActionType != nil {
		out += "/A " + l.A.pdfString(pdf, ref)
	} else if l.Dest != nil {
		out += "/Dest " + l.Dest.pdfDestination(pdf, ref)
	}
	if l.H != "" {
		out += "/H " + Name(l.H).String()
	}
	if l.PA.ActionType != nil {
		out += "/PA " + l.PA.pdfString(pdf, ref)
	}
	if len(l.QuadPoints) != 0 {
		out += "/QuadPoints " + writeFloatArray(l.QuadPoints)
	}
	if l.BS != nil {
		out += "/BS " + l.BS.String()
	}
	return out
}

func (l AnnotationLink) clone(cache cloneCache) Annotation {
	out := l
	out.A = l.A.clone(cache)
	if l.Dest != nil {
		out.Dest = l.Dest.clone(cache)
	}
	if l.PA.ActionType != nil {
		out.PA = l.PA.clone(cache)
	}
	out.QuadPoints = append([]float64(nil), l.QuadPoints...)
	out.BS = l.BS.Clone()
	return out
}

// -----------------------------------------------------------

// AnnotationFreeText displays text directly on the page.
// See Table 174 – Additional entries specific to a free text annotation
type AnnotationFreeText struct {
	AnnotationMarkup
	DA string        // required
	Q  uint8         // optional
	RC string        // optional, may be written in PDF as a text stream
	DS string        // optional
	CL []float64     // optional
	BE *BorderEffect // optional
	RD Rectangle     // optional
	BS *BorderStyle  // optional
	LE Name          // optional
}

func (f AnnotationFreeText) annotationFields(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("/Subtype/FreeText " + f.AnnotationMarkup.pdfFields(pdf, ref))
	b.WriteString("/DA " + pdf.EncodeString(f.DA, ByteString, ref))
	if f.Q != 0 {
		b.WriteString(fmt.Sprintf("/Q %d", f.Q))
	}
	if f.RC != "" {
		b.WriteString("/RC " + pdf.EncodeString(f.RC, TextString, ref))
	}
	if f.DS != "" {
		b.WriteString("/DS " + pdf.EncodeString(f.DS, TextString, ref))
	}
	if len(f.CL) != 0 {
		b.WriteString("/CL " + writeFloatArray(f.CL))
	}
	if f.BE != nil {
		b.WriteString("/BE " + f.BE.String())
	}
	if f.RD != (Rectangle{}) {
		b.WriteString("/RD " + f.RD.String())
	}
	if f.BS != nil {
		b.WriteString("/BS " + f.BS.String())
	}
	if f.LE != "" {
		b.WriteString("/LE " + f.LE.String())
	}
	return b.String()
}

func (f AnnotationFreeText) clone(cache cloneCache) Annotation {
	out := f
	out.AnnotationMarkup = f.AnnotationMarkup.clone(cache)
	out.CL = append([]float64(nil), f.CL...)
	out.BE = f.BE.Clone()
	out.BS = f.BS.Clone()
	return out
}

// ------------------------------------------------------------------------------

// AnnotationLine displays a single straight line on the page.
// See Table 175 – Additional entries specific to a line annotation
type AnnotationLine struct {
	AnnotationMarkup
	L   [4]float64   // required
	BS  *BorderStyle // optional
	LE  [2]Name      // optional
	IC  []float64    // optional
	LL  float64      // optional
	LLE float64      // optional
	Cap bool         // optional
	LLO MaybeFloat   // optional
	CP  Name         // optional
	CO  [2]float64   // optional
	// TODO: support measure dictionary
	// Measure *MeasureDict // optional
}

func (f AnnotationLine) annotationFields(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("/Subtype/Line " + f.AnnotationMarkup.pdfFields(pdf, ref))
	b.WriteString("/L " + writeFloatArray(f.L[:]))
	if f.BS != nil {
		b.WriteString("/BS " + f.BS.String())
	}
	if f.LE != ([2]Name{}) {
		b.WriteString(fmt.Sprintf("/LE %s", writeNameArray(f.LE[:])))
	}
	if len(f.IC) != 0 {
		b.WriteString("/IC " + writeFloatArray(f.IC))
	}
	if f.LL != 0 {
		b.fmt("/LL %.3f", f.LL)
	}
	if f.LLE != 0 {
		b.fmt("/LLE %.3f", f.LLE)
	}
	b.fmt("/Cap %v", f.Cap)
	if f.LLO != nil {
		b.fmt("/LLO %.3f", f.LLO.(Float))
	}
	if f.CP != "" {
		b.fmt("/CP %s", f.CP)
	}
	if f.CO != ([2]float64{}) {
		b.WriteString(fmt.Sprintf("/CO %s", writeFloatArray(f.CO[:])))
	}
	return b.String()
}

func (f AnnotationLine) clone(cache cloneCache) Annotation {
	out := f
	out.AnnotationMarkup = f.AnnotationMarkup.clone(cache)
	out.BS = f.BS.Clone()
	out.IC = append([]float64(nil), f.IC...)
	return out
}

// -------------------------------------------------------------------

// AnnotationSquare displays a rectangle on the page.
// See Table 177 – Additional entries specific to a square or circle annotation
type AnnotationSquare struct {
	AnnotationMarkup
	BS *BorderStyle  // optional
	IC []float64     // optional
	BE *BorderEffect // optional
	RD Rectangle     // optional
}

// shared with AnnorationCircle
func (f AnnotationSquare) annotationFieldsExt(pdf pdfWriter, ref Reference, isSquare bool) string {
	b := newBuffer()
	subtype := Name("Circle")
	if isSquare {
		subtype = "Square"
	}
	b.fmt("/Subtype%s %s", subtype, f.AnnotationMarkup.pdfFields(pdf, ref))
	if f.BS != nil {
		b.WriteString("/BS " + f.BS.String())
	}
	if len(f.IC) != 0 {
		b.WriteString("/IC " + writeFloatArray(f.IC))
	}
	if f.BE != nil {
		b.WriteString("/BE " + f.BE.String())
	}
	if f.RD != (Rectangle{}) {
		b.WriteString("/RD " + f.RD.String())
	}
	return b.String()
}

func (f AnnotationSquare) annotationFields(pdf pdfWriter, ref Reference) string {
	return f.annotationFieldsExt(pdf, ref, true)
}

func (f AnnotationSquare) clone(cache cloneCache) Annotation {
	out := f
	out.AnnotationMarkup = f.AnnotationMarkup.clone(cache)
	out.BS = f.BS.Clone()
	out.IC = append([]float64(nil), f.IC...)
	out.BE = f.BE.Clone()
	return out
}

// AnnotationCircle displays an ellipse on the page
type AnnotationCircle AnnotationSquare

func (f AnnotationCircle) annotationFields(pdf pdfWriter, ref Reference) string {
	return AnnotationSquare(f).annotationFieldsExt(pdf, ref, false)
}

func (f AnnotationCircle) clone(cache cloneCache) Annotation {
	return AnnotationCircle(AnnotationSquare(f).clone(cache).(AnnotationSquare))
}

// -------------------------------------------------------------------------

//TODO: add and check the remaining annotation

type AnnotationFileAttachment struct {
	T  string
	FS *FileSpec
}

func (f AnnotationFileAttachment) annotationFields(pdf pdfWriter, ref Reference) string {
	fsRef := pdf.addItem(f.FS)
	return fmt.Sprintf("/Subtype/FileAttachment/T %s/FS %s", pdf.EncodeString(f.T, TextString, ref), fsRef)
}

func (f AnnotationFileAttachment) clone(cache cloneCache) Annotation {
	out := f
	out.FS = cache.checkOrClone(f.FS).(*FileSpec)
	return out
}

// ---------------------------------------------------

// AnnotationWidget is an annotation widget,
// primarily for form fields
// The Parent field is deduced from the containing form field.
type AnnotationWidget struct {
	H  Highlighting                 // optional
	MK *AppearanceCharacteristics   // optional
	A  Action                       // optional
	BS *BorderStyle                 // optional
	AA *AnnotationAdditionalActions // optional
}

func (w AnnotationWidget) annotationFields(pdf pdfWriter, ref Reference) string {
	out := fmt.Sprintf("/Subtype/Widget")
	if w.H != "" {
		out += fmt.Sprintf("/H %s", w.H)
	}
	if w.MK != nil {
		out += fmt.Sprintf("/MK %s", w.MK.pdfString(pdf, ref))
	}
	if w.A.ActionType != nil {
		out += fmt.Sprintf("/A %s", w.A.pdfString(pdf, ref))
	}
	if w.BS != nil {
		out += fmt.Sprintf("/BS %s", w.BS.String())
	}
	if w.AA != nil {
		out += fmt.Sprintf("/AA %s", w.AA.pdfString(pdf, ref))
	}
	return out
}

func (w AnnotationWidget) clone(cache cloneCache) Annotation {
	out := w
	out.MK = w.MK.clone(cache)
	out.A = w.A.clone(cache)
	out.BS = w.BS.Clone()
	out.AA = w.AA.clone(cache)
	return out
}

// AppearanceCharacteristics contains additional information
// for constructing the annotation’s appearance stream.
// See Table 189 – Entries in an appearance characteristics dictionary
type AppearanceCharacteristics struct {
	R          Rotation     // optional
	BC, BG     []float64    // optional
	CA, RC, AC string       // optional
	I, RI, IX  *XObjectForm // optional
	IF         *IconFit     // optional
	TP         uint8        // optional
}

func (a AppearanceCharacteristics) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if a.R != 0 {
		b.fmt("/R %d", a.R.Degrees())
	}
	if len(a.BC) != 0 {
		b.fmt("/BC %s", writeFloatArray(a.BC))
	}
	if len(a.BG) != 0 {
		b.fmt("/BG %s", writeFloatArray(a.BG))
	}
	if a.CA != "" {
		b.fmt("/CA %s", pdf.EncodeString(a.CA, TextString, ref))
	}
	if a.RC != "" {
		b.fmt("/RC %s", pdf.EncodeString(a.RC, TextString, ref))
	}
	if a.AC != "" {
		b.fmt("/AC %s", pdf.EncodeString(a.AC, TextString, ref))
	}
	if a.I != nil {
		ref := pdf.addItem(a.I)
		b.fmt("/I %s", ref)
	}
	if a.RI != nil {
		ref := pdf.addItem(a.RI)
		b.fmt("/RI %s", ref)
	}
	if a.IX != nil {
		ref := pdf.addItem(a.IX)
		b.fmt("/IX %s", ref)
	}
	if a.IF != nil {
		b.fmt("/IF %s", a.IF)
	}
	if a.TP != 0 {
		b.fmt("/TP %d", a.TP)
	}
	b.WriteString(">>")
	return b.String()
}

func (a *AppearanceCharacteristics) clone(cache cloneCache) *AppearanceCharacteristics {
	if a == nil {
		return nil
	}
	out := *a
	out.BC = append([]float64(nil), a.BC...)
	out.BG = append([]float64(nil), a.BG...)
	out.I = a.I.clone(cache).(*XObjectForm)
	out.RI = a.RI.clone(cache).(*XObjectForm)
	out.IX = a.IX.clone(cache).(*XObjectForm)
	out.IF = a.IF.Clone()
	return &out
}

// IconFit specifies how to display the button’s icon
// within the annotation rectangle of its widget annotation.
// See Table 247 – Entries in an icon fit dictionary
type IconFit struct {
	SW Name        // optional
	S  Name        // optional
	A  *[2]float64 // optional
	FB bool        // optional
}

// String return a PDF dictionary.
func (i IconFit) String() string {
	out := ""
	if i.SW != "" {
		out += "/SW " + i.SW.String()
	}
	if i.S != "" {
		out += "/S " + i.S.String()
	}
	if i.A != nil {
		out += "/A " + writeFloatArray((*i.A)[:])
	}
	if i.FB {
		out += fmt.Sprintf("/FB %v", i.FB)
	}
	return "<<" + out + ">>"
}

// Clone returns a deep copy.
func (i *IconFit) Clone() *IconFit {
	if i == nil {
		return nil
	}
	out := *i
	if i.A != nil {
		cp := *i.A
		out.A = &cp
	}
	return &out
}
