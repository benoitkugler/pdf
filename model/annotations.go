package model

import (
	"fmt"
	"strings"
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
	W float64   // optional, default to 1, Undef not to specify it
	S Name      // optional
	D []float64 // optional, default to [3], nil not to specify it
}

func (bo BorderStyle) pdfString() string {
	b := newBuffer()
	b.WriteString("<<")
	if bo.W != Undef {
		b.fmt("/W %.3f", bo.W)
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

type BaseAnnotation struct {
	Rect     Rectangle
	Contents string
	AP       *AppearanceDict // optional
	// Appearance state (key of the AP.N subDictionary).
	// Required if the appearance dictionary AP contains one or more
	// subdictionaries
	AS     Name
	F      int     // optional
	Border *Border // optional
}

func (ba BaseAnnotation) fields(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.fmt("/Rectangle %s", ba.Rect)
	if ba.Contents != "" {
		b.fmt("/Contents %s", pdf.EncodeString(ba.Contents, TextString, ref))
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
	return b.String()
}

func (ba BaseAnnotation) clone(cache cloneCache) BaseAnnotation {
	out := ba
	out.AP = ba.AP.clone(cache)
	out.Border = ba.Border.Clone()
	return out
}

type AnnotationDict struct {
	BaseAnnotation
	Subtype Annotation
}

// pdfContent impements is cachable
func (a *AnnotationDict) pdfContent(pdf pdfWriter, ref Reference) (string, []byte) {
	base := a.BaseAnnotation.fields(pdf, ref)
	subtype := a.Subtype.annotationFields(pdf, ref)
	return fmt.Sprintf("<<%s %s >>", base, subtype), nil
}

func (a *AnnotationDict) clone(cache cloneCache) Referencable {
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

type Annotation interface {
	// return the specialized fields (including Subtype)
	annotationFields(pdf pdfWriter, ref Reference) string
	clone(cloneCache) Annotation
}

// ------------------------ specializations ------------------------

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

// AnnotationLink either opens an URI (field A)
// or an internal page (field Dest)
type AnnotationLink struct {
	// TODO: complete fields
	A    Action      // optional, represented by a dictionary in PDF
	Dest Destination // may only be present is A is nil
}

func (l AnnotationLink) annotationFields(pdf pdfWriter, ref Reference) string {
	out := "/Subtype/Link"
	if l.A != nil {
		out += "/A " + l.A.actionDictionary(pdf, ref)
	} else if l.Dest != nil {
		out += "/Dest " + l.Dest.pdfDestination(pdf)
	}
	return out
}

func (l AnnotationLink) clone(cache cloneCache) Annotation {
	out := l
	if l.A != nil {
		out.A = l.A.clone(cache)
	}
	if l.Dest != nil {
		out.Dest = l.Dest.clone(cache)
	}
	return out
}

// AnnotationWidget is an annotation widget,
// primarily for form fields
type AnnotationWidget struct {
	// TODO: complete fields
	H  Highlighting
	A  Action // optional
	BS *BorderStyle
}

func (w AnnotationWidget) annotationFields(pdf pdfWriter, ref Reference) string {
	out := fmt.Sprintf("/Subtype/Widget")
	if w.H != "" {
		out += fmt.Sprintf("/H %s", w.H)
	}
	if w.A != nil {
		out += fmt.Sprintf("/A %s", w.A.actionDictionary(pdf, ref))
	}
	if w.BS != nil {
		out += fmt.Sprintf("/BS %s", w.BS.pdfString())
	}
	return out
}

func (w AnnotationWidget) clone(cache cloneCache) Annotation {
	out := w
	if w.A != nil {
		out.A = w.A.clone(cache)
	}
	out.BS = w.BS.Clone()
	return out
}
