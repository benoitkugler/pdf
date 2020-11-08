package model

type AnnotationType interface {
	isAnnotation()
}

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

type Annotation struct {
	Subtype  AnnotationType
	Rect     Rectangle
	Contents string
	AP       *AppearanceDict
	AS       Name // appearance state (key of the AP.N subdictionnary)
	F        int
}

type AppearanceDict struct {
	N *AppearanceEntry // annotation’s normal appearance
	R *AppearanceEntry // annotation’s rollover appearance, optional, default to N
	D *AppearanceEntry // annotation’s down appearance, optional, default to N
}

// AppearanceEntry is either a dictionnary, or a subdictionnary
// containing multiple appearances
// In the first case, the map is of length 1, with the empty string as key
type AppearanceEntry map[Name]*XObject

func (FileAttachmentAnnotation) isAnnotation() {}
func (LinkAnnotation) isAnnotation()           {}
func (WidgetAnnotation) isAnnotation()         {}

// ------------------------ specializations ------------------------

type FileAttachmentAnnotation struct {
	T  string
	FS *FileSpec // indirect ref
}

// ---------------------------------------------------

// LinkAnnotation either opens an URI (field A)
// or an internal page (field Dest)
type LinkAnnotation struct {
	A    Action      // optional, represented by a dictionary in PDF
	Dest Destination // may only be present is A is nil
}

type Action interface {
	S() string // return the action type
}

type URIAction string // in PDF, ASCII encoded

func (URIAction) S() string { return "URI" }

type GoToAction struct {
	D Destination
}

func (GoToAction) S() string { return "GoTo" }

type Destination interface {
	isDestination()
}

func (ExplicitDestination) isDestination() {}
func (NamedDestination) isDestination()    {}

type NamedDestination string

// ExplicitDestination is an explicit destination to a page
type ExplicitDestination struct {
	Page      *PageObject
	Left, Top *float64 // nil means Don't change the current value
	Zoom      float64
}

// ---------------------------------------------------

type Highlighting Name

const (
	None    Highlighting = "N" // No highlighting.
	Invert  Highlighting = "I" // Invert the contents of the annotation rectangle.
	Outline Highlighting = "O" // Invert the annotation’s border.
	Push    Highlighting = "P" // Display the annotation’s down appearance, if any
	Toggle  Highlighting = "T" // Same as P (which is preferred).
)

// TODO:
type WidgetAnnotation struct {
	H Highlighting
}
