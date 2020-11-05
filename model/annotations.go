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

// ------------------------ specializations ------------------------

type FileAttachmentAnnotation struct {
	T  string
	FS *FileSpec // indirect ref
}

func (FileAttachmentAnnotation) isAnnotation() {}

// ---------------------------------------------------

// LinkAnnotation either opens an URI (field A)
// or an internal page (field Dest)
type LinkAnnotation struct {
	A    *ActionDict
	Dest *Destination
}

// TODO: only URI is supported
type ActionDict struct {
	// S   Name
	URI string
}

// Destination is an explicit destination to a page
type Destination struct {
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
