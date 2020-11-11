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
		b.fmt(" /W %.3f", bo.W)
	}
	if bo.S != "" {
		b.fmt(" /S %s", bo.S)
	}
	if bo.D != nil {
		b.fmt(" /D %s", writeFloatArray(bo.D))
	}
	b.fmt(">>")
	return b.String()
}

type Annotation struct {
	Subtype  AnnotationType
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

func (a *Annotation) pdfContent(pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	subtype := a.Subtype.annotationFields(pdf)
	b.fmt("<<%s /Rectangle %s", subtype, a.Rect.PDFstring())
	if a.Contents != "" {
		b.fmt(" /Contents %s", pdf.EncodeTextString(a.Contents))
	}
	if ap := a.AP; ap != nil {
		b.fmt(" /AP %s", ap.pdfString(pdf))
	}
	if as := a.AS; as != "" {
		b.fmt(" /AS %s", as)
	}
	if f := a.F; f != 0 {
		b.fmt(" /F %d", f)
	}
	if bo := a.Border; bo != nil {
		b.fmt(" /Border %s", bo.pdfString())
	}
	b.fmt(">>")
	return b.String(), nil
}

type AppearanceDict struct {
	N AppearanceEntry // annotation’s normal appearance
	R AppearanceEntry // annotation’s rollover appearance, optional, default to N
	D AppearanceEntry // annotation’s down appearance, optional, default to N
}

func (a AppearanceDict) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.fmt("<<")
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

type AnnotationType interface {
	// return the specialized fields (including Subtype)
	annotationFields(pdf pdfWriter) string
}

// ------------------------ specializations ------------------------

type FileAttachmentAnnotation struct {
	T  string
	FS *FileSpec
}

func (f FileAttachmentAnnotation) annotationFields(pdf pdfWriter) string {
	ref := pdf.addItem(f.FS)
	return fmt.Sprintf("/Subtype /FileAttachment /T %s /FS %s", pdf.EncodeTextString(f.T), ref)
}

// ---------------------------------------------------

// LinkAnnotation either opens an URI (field A)
// or an internal page (field Dest)
type LinkAnnotation struct {
	A    Action      // optional, represented by a dictionary in PDF
	Dest Destination // may only be present is A is nil
}

func (l LinkAnnotation) annotationFields(pdf pdfWriter) string {
	out := "/Subtype /Link"
	if l.A != nil {
		out += " /A " + l.A.ActionDictionary(pdf)
	} else if l.Dest != nil {
		out += " /Dest " + l.Dest.pdfDestination(pdf)
	}
	return out
}

type Action interface {
	// ActionDictionary returns the dictionary defining the action
	// as written in PDF
	ActionDictionary(pdfWriter) string
}

// URIAction is a URI which should be ASCII encoded
type URIAction string

func (uri URIAction) ActionDictionary(pdf pdfWriter) string {
	return fmt.Sprintf("<</S /URI /URI (%s)>>", pdf.ASCIIString(string(uri)))
}

type GoToAction struct {
	D Destination
}

func (ac GoToAction) ActionDictionary(pdf pdfWriter) string {
	return fmt.Sprintf("<</S /GoTo /D %s>>", ac.D.pdfDestination(pdf))
}

type Destination interface {
	// return the PDF content of the destination
	pdfDestination(pdfWriter) string
}

// ExplicitDestination is an explicit destination to a page
type ExplicitDestination struct {
	Page      *PageObject
	Left, Top *float64 // nil means Don't change the current value
	Zoom      float64
}

func (d ExplicitDestination) pdfDestination(pdf pdfWriter) string {
	pageRef := pdf.pages[d.Page]
	left, top := "null", "null"
	if d.Left != nil {
		left = fmt.Sprintf("%.3f", *d.Left)
	}
	if d.Top != nil {
		top = fmt.Sprintf("%.3f", *d.Top)
	}
	return fmt.Sprintf("[%s /XYZ %s %s %.3f]", pageRef, left, top, d.Zoom)
}

type DestinationName Name

func (n DestinationName) pdfDestination(pdfWriter) string {
	return Name(n).String()
}

type DestinationString string

func (s DestinationString) pdfDestination(pdf pdfWriter) string { return fmt.Sprintf("(%s)", s) }

// ---------------------------------------------------

type Highlighting Name

const (
	HNone    Highlighting = "N" // No highlighting.
	HInvert  Highlighting = "I" // Invert the contents of the annotation rectangle.
	HOutline Highlighting = "O" // Invert the annotation’s border.
	HPush    Highlighting = "P" // Display the annotation’s down appearance, if any
	HToggle  Highlighting = "T" // Same as P (which is preferred).
)

// TODO:
type WidgetAnnotation struct {
	H  Highlighting
	A  Action
	BS *BorderStyle
}

func (w WidgetAnnotation) annotationFields(pdf pdfWriter) string {
	out := fmt.Sprintf("/Subtype /Widget")
	if w.H != "" {
		out += fmt.Sprintf(" /H %s", w.H)
	}
	if w.A != nil {
		out += fmt.Sprintf(" /A %s", w.A.ActionDictionary(pdf))
	}
	if w.BS != nil {
		out += fmt.Sprintf(" /BS %s", w.BS.pdfString())
	}
	return out
}
