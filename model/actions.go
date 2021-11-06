package model

import (
	"fmt"
	"strconv"
	"strings"
)

// Action defines the characteristics and behaviour of an action.
type Action struct {
	ActionType
	// sequence of actions that shall be performed after
	// the action represented by this dictionary
	Next []Action
}

func (a Action) pdfString(pdf pdfWriter, context Reference) string {
	subtype := a.ActionType.actionParams(pdf, context)
	next := ""
	if len(a.Next) != 0 {
		chunks := make([]string, len(a.Next))
		for i, n := range a.Next {
			chunks[i] = n.pdfString(pdf, context)
		}
		next = fmt.Sprintf("/Next [%s]", strings.Join(chunks, " "))
	}
	return fmt.Sprintf("<<%s %s>>", subtype, next)
}

func (a Action) clone(cache cloneCache) Action {
	var out Action
	if a.ActionType != nil {
		out.ActionType = a.ActionType.clone(cache)
	}
	if a.Next != nil { // preserve nil
		out.Next = make([]Action, len(a.Next))
		for i, n := range a.Next {
			out.Next[i] = n.clone(cache)
		}
	}
	return out
}

// ActionType specialize the action (see Table 198 – Action types).
// Many PDF actions are supported, excepted:
//	- Thread
//	- Sound
//	- Movie
// TODO - SubmitForm
// TODO - ResetForm
// TODO - ImportData
//	- SetOCGState
//	- Rendition
//	- Trans
//	- GoTo3DView
type ActionType interface {
	// actionParams returns the fields of dictionary defining the action
	// as written in PDF
	actionParams(pdfWriter, Reference) string
	clone(cache cloneCache) ActionType
}

type ActionJavaScript struct {
	JS string // text string, may be found in PDF as stream object
}

func (j ActionJavaScript) actionParams(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("/S/JavaScript/JS %s", pdf.EncodeString(j.JS, TextString, ref))
}

func (j ActionJavaScript) clone(cache cloneCache) ActionType { return j }

// ActionURI causes a URI to be resolved
type ActionURI struct {
	URI   string // URI which must be ASCII string
	IsMap bool   // optional
}

func (uri ActionURI) actionParams(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("/S/URI/URI (%s)/IsMap %v",
		pdf.EncodeString(string(uri.URI), ByteString, ref), uri.IsMap)
}

func (uri ActionURI) clone(cache cloneCache) ActionType { return uri }

type ActionGoTo struct {
	D Destination
}

func (ac ActionGoTo) actionParams(pdf pdfWriter, context Reference) string {
	return fmt.Sprintf("/S/GoTo/D %s", ac.D.pdfDestination(pdf, context))
}

func (ac ActionGoTo) clone(cache cloneCache) ActionType {
	return ActionGoTo{D: ac.D.clone(cache)}
}

// DestinationLocation precise where and how to
// display a destination page.
// It is either a Name o
// See Table 151 – Destination syntax in the SPEC.
type DestinationLocation interface {
	locationElements() string // return the elements of the array
}

// DestinationLocationFit is one of /Fit or /FitB
type DestinationLocationFit Name

func (d DestinationLocationFit) locationElements() string {
	return Name(d).String()
}

// DestinationLocationFitDim is one of /FitH, /FitV, /FitBH or ∕FitBV
type DestinationLocationFitDim struct {
	Name Name // one of /FitH /FitV /FitBH ∕FitBV
	Dim  MaybeFloat
}

func (d DestinationLocationFitDim) locationElements() string {
	return d.Name.String() + " " + writeMaybeFloat(d.Dim)
}

// DestinationLocationXYZ is /XYZ
type DestinationLocationXYZ struct {
	Left, Top MaybeFloat
	Zoom      Fl
}

func (d DestinationLocationXYZ) locationElements() string {
	return fmt.Sprintf("/XYZ %s %s %g",
		writeMaybeFloat(d.Left), writeMaybeFloat(d.Top), d.Zoom)
}

// DestinationLocationFitR is /FitR
type DestinationLocationFitR struct {
	Left, Bottom, Right, Top Fl
}

func (d DestinationLocationFitR) locationElements() string {
	return fmt.Sprintf("/FitR %g %g %g %g",
		d.Left, d.Bottom, d.Right, d.Top)
}

type Destination interface {
	// return the PDF content of the destination
	pdfDestination(pdfWriter, Reference) string
	clone(cache cloneCache) Destination
}

// DestinationExplicit is a destination to a page,
// either intern or extern
type DestinationExplicit interface {
	Destination
	isExplicit()
}

func (DestinationExplicitIntern) isExplicit() {}
func (DestinationExplicitExtern) isExplicit() {}

// DestinationExplicitIntern is an explicit destination to a page
type DestinationExplicitIntern struct {
	Page     *PageObject
	Location DestinationLocation // required
}

func (d DestinationExplicitIntern) pdfDestination(pdf pdfWriter, _ Reference) string {
	pageRef := pdf.pages[d.Page]
	var location string
	if d.Location != nil {
		location = d.Location.locationElements()
	}
	return fmt.Sprintf("[%s %s]", pageRef, location)
}

// concrete type is DestinationExplicitIntern
func (d DestinationExplicitIntern) clone(cache cloneCache) Destination {
	out := d
	if d.Page != nil {
		out.Page = cache.pages[d.Page].(*PageObject)
	}
	return d
}

// DestinationExplicitExtern is an explicit destination to a page
// in an other PDF file.
type DestinationExplicitExtern struct {
	Page     int                 // 0 based
	Location DestinationLocation // required
}

func (d DestinationExplicitExtern) pdfDestination(pdfWriter, Reference) string {
	var location string
	if d.Location != nil {
		location = d.Location.locationElements()
	}
	return fmt.Sprintf("[%d %s]", d.Page, location)
}

// concrete type is DestinationExplicitExtern
func (d DestinationExplicitExtern) clone(cache cloneCache) Destination { return d }

// DestinationString refers to a destination in the Dests entry of
// the document catalog
type DestinationName Name

func (n DestinationName) pdfDestination(pdfWriter, Reference) string {
	return Name(n).String()
}

func (d DestinationName) clone(cloneCache) Destination { return d }

// DestinationString refers to a destination in the Dests entry of
// the document name dictionnary
type DestinationString string

func (s DestinationString) pdfDestination(pdf pdfWriter, context Reference) string {
	return pdf.EncodeString(string(s), ByteString, context)
}

func (d DestinationString) clone(cloneCache) Destination { return d }

// ActionRemoteGoTo is either an RemoteGoTo action
// or a Launch action, when D is nil
type ActionRemoteGoTo struct {
	D         Destination // DestinationExplicitIntern are not allowed
	F         *FileSpec   // must not be embedded
	NewWindow bool
}

func (ac ActionRemoteGoTo) actionParams(pdf pdfWriter, ref Reference) string {
	fs := ""
	if ac.F != nil {
		_, fs, _ := ac.F.pdfContent(pdf, ref)
		fs = "/F " + fs
	}
	name, dest := Name("Launch"), ""
	if ac.D != nil {
		name = Name("GoToR")
		dest = "/D " + ac.D.pdfDestination(pdf, ref)
	}
	return fmt.Sprintf("/S%s/D %s %s/NewWindow %v",
		name, dest, fs, ac.NewWindow)
}

func (ac ActionRemoteGoTo) clone(cache cloneCache) ActionType {
	out := ac
	out.D = ac.D.clone(cache)
	out.F = ac.F.clone(cache).(*FileSpec)
	return out
}

type EmbeddedTarget struct {
	R Name
	N string
	P EmbeddedTargetDest  // optional
	A EmbeddedTargetAnnot // optional
	T *EmbeddedTarget     // optional
}

func (r EmbeddedTarget) pdfString(pdf pdfWriter, context Reference) string {
	out := fmt.Sprintf("<</R %s ", r.R)
	if r.N != "" {
		out += "/N " + pdf.EncodeString(r.N, ByteString, context)
	}
	if r.P != nil {
		out += "/P " + r.P.embeddedTargetDestString(pdf, context)
	}
	if r.A != nil {
		out += "/A " + r.A.embeddedTargetAnnotString(pdf, context)
	}
	if r.T != nil {
		out += "/T " + r.pdfString(pdf, context)
	}
	return out + ">>"
}

func (r *EmbeddedTarget) clone() *EmbeddedTarget {
	if r == nil {
		return nil
	}
	out := r
	out.T = r.T.clone()
	return out
}

// EmbeddedTargetDest is either the page number (zero-based) in the current
// document containing the file attachment annotation or a string that specifies
// a named destination in the current document that provides the page number of
// the file attachment annotation.
type EmbeddedTargetDest interface {
	embeddedTargetDestString(pdf pdfWriter, context Reference) string
}

type EmbeddedTargetDestPage int

func (e EmbeddedTargetDestPage) embeddedTargetDestString(pdfWriter, Reference) string {
	return strconv.Itoa(int(e))
}

type EmbeddedTargetDestNamed DestinationString

func (e EmbeddedTargetDestNamed) embeddedTargetDestString(pdf pdfWriter, context Reference) string {
	return DestinationString(e).pdfDestination(pdf, context)
}

// EmbeddedTargetAnnot is either the index (zero-based) of the annotation in the
// Annots array of the page specified by P or a text string that specifies
// the value of NM in the annotation dictionary
type EmbeddedTargetAnnot interface {
	embeddedTargetAnnotString(pdf pdfWriter, context Reference) string
}

type EmbeddedTargetAnnotIndex int

func (e EmbeddedTargetAnnotIndex) embeddedTargetAnnotString(pdfWriter, Reference) string {
	return strconv.Itoa(int(e))
}

type EmbeddedTargetAnnotNamed string

func (e EmbeddedTargetAnnotNamed) embeddedTargetAnnotString(pdf pdfWriter, context Reference) string {
	return pdf.EncodeString(string(e), TextString, context)
}

type ActionEmbeddedGoTo struct {
	D         Destination // DestinationExplicitIntern are not allowed
	F         *FileSpec   // optional
	NewWindow bool
	T         *EmbeddedTarget // optional if F is present
}

func (ac ActionEmbeddedGoTo) actionParams(pdf pdfWriter, ref Reference) string {
	out := fmt.Sprintf("/S/GoToE/D %s/NewWindow %v",
		ac.D.pdfDestination(pdf, ref), ac.NewWindow)
	if ac.F != nil {
		_, fs, _ := ac.F.pdfContent(pdf, ref)
		out += "/F " + fs
	}
	if ac.T != nil {
		out += "/T " + ac.T.pdfString(pdf, ref)
	}
	return out
}

func (ac ActionEmbeddedGoTo) clone(cache cloneCache) ActionType {
	out := ac
	out.D = ac.D.clone(cache)
	out.F = ac.F.clone(cache).(*FileSpec)
	out.T = ac.T.clone()
	return out
}

// ActionHide hides or shows one or more annotations
type ActionHide struct {
	T    []ActionHideTarget
	Show bool // optional, written in a PDF file as H (hide)
}

func (a ActionHide) actionParams(pdf pdfWriter, ref Reference) string {
	chunks := make([]string, len(a.T))
	for i, ta := range a.T {
		chunks[i] = ta.hideTargetString(pdf, ref)
	}
	return fmt.Sprintf("/S /Hide /T %s /H %v", strings.Join(chunks, " "), !a.Show)
}

func (ac ActionHide) clone(cache cloneCache) ActionType {
	out := ac
	out.T = make([]ActionHideTarget, len(ac.T))
	for i, t := range ac.T {
		out.T[i] = t.cloneHT(cache)
	}
	return out
}

// ActionHideTarget is either an annotation or
// a form field
type ActionHideTarget interface {
	hideTargetString(pdf pdfWriter, context Reference) string
	cloneHT(cloneCache) ActionHideTarget
}

func (an *AnnotationDict) hideTargetString(pdf pdfWriter, _ Reference) string {
	return pdf.addItem(an).String()
}

func (an *AnnotationDict) cloneHT(cache cloneCache) ActionHideTarget {
	return an.clone(cache).(*AnnotationDict)
}

// HideTargetFormName is the fully qualified field name of an interactive form field
type HideTargetFormName string

func (h HideTargetFormName) hideTargetString(pdf pdfWriter, context Reference) string {
	return pdf.EncodeString(string(h), TextString, context)
}

func (h HideTargetFormName) cloneHT(cloneCache) ActionHideTarget { return h }

// ActionNamed is one of NextPage,PrevPage,FirstPage,LastPage
type ActionNamed Name

func (a ActionNamed) actionParams(pdfWriter, Reference) string {
	return fmt.Sprintf("/S/Named/N%s", Name(a))
}

func (ac ActionNamed) clone(cache cloneCache) ActionType { return ac }

// All actions are optional and must be JavaScript actions.
// See Table 196 – Entries in a form field’s additional-actions dictionary
type FormFielAdditionalActions struct {
	K Action // on update
	F Action // before formating
	V Action // on validate
	C Action // to recalculate
}

// IsEmpty returns `true` if it contains no action.
func (f FormFielAdditionalActions) IsEmpty() bool {
	return f.K.ActionType == nil && f.F.ActionType == nil &&
		f.V.ActionType == nil && f.C.ActionType == nil
}

func (f FormFielAdditionalActions) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if f.K.ActionType != nil {
		b.line("/K %s", f.K.pdfString(pdf, ref))
	}
	if f.F.ActionType != nil {
		b.line("/F %s", f.F.pdfString(pdf, ref))
	}
	if f.V.ActionType != nil {
		b.line("/V %s", f.V.pdfString(pdf, ref))
	}
	if f.C.ActionType != nil {
		b.line("/C %s", f.C.pdfString(pdf, ref))
	}
	b.fmt(">>")
	return b.String()
}

func (ff FormFielAdditionalActions) clone(cache cloneCache) FormFielAdditionalActions {
	var a FormFielAdditionalActions
	a.K = ff.K.clone(cache)
	a.F = ff.F.clone(cache)
	a.V = ff.V.clone(cache)
	a.C = ff.C.clone(cache)
	return a
}

// All actions are optional
// See Table 194 – Entries in an annotation’s additional-actions dictionary.
type AnnotationAdditionalActions struct {
	E  Action // cursor enters the annotation’s active area.
	X  Action // cursor exits the annotation’s active area.
	D  Action // mouse button is pressed inside the annotation’s active area.
	U  Action // mouse button is released inside the annotation’s active area.
	Fo Action // the annotation receives the input focus.
	Bl Action // the annotation loses the input focus.
	PO Action // the page containing the annotation is opened.
	PC Action // the page containing the annotation is closed.
	PV Action // the page containing the annotation becomes visible.
	PI Action // the page containing the annotation is no longer visible in the conforming reader’s user interface.
}

// IsEmpty return `true` if it contains no actions.
func (a AnnotationAdditionalActions) IsEmpty() bool {
	return a.E.ActionType == nil && a.X.ActionType == nil &&
		a.D.ActionType == nil && a.U.ActionType == nil &&
		a.Fo.ActionType == nil && a.Bl.ActionType == nil &&
		a.PO.ActionType == nil && a.PC.ActionType == nil &&
		a.PV.ActionType == nil && a.PI.ActionType == nil
}

func (ann AnnotationAdditionalActions) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if ann.E.ActionType != nil {
		b.line("/E %s", ann.E.pdfString(pdf, ref))
	}
	if ann.X.ActionType != nil {
		b.line("/X %s", ann.X.pdfString(pdf, ref))
	}
	if ann.D.ActionType != nil {
		b.line("/D %s", ann.D.pdfString(pdf, ref))
	}
	if ann.U.ActionType != nil {
		b.line("/U %s", ann.U.pdfString(pdf, ref))
	}
	if ann.Fo.ActionType != nil {
		b.line("/Fo %s", ann.Fo.pdfString(pdf, ref))
	}
	if ann.Bl.ActionType != nil {
		b.line("/Bl %s", ann.Bl.pdfString(pdf, ref))
	}
	if ann.PO.ActionType != nil {
		b.line("/PO %s", ann.PO.pdfString(pdf, ref))
	}
	if ann.PC.ActionType != nil {
		b.line("/PC %s", ann.PC.pdfString(pdf, ref))
	}
	if ann.PV.ActionType != nil {
		b.line("/PV %s", ann.PV.pdfString(pdf, ref))
	}
	if ann.PI.ActionType != nil {
		b.line("/PI %s", ann.PI.pdfString(pdf, ref))
	}
	b.fmt(">>")
	return b.String()
}

func (ann AnnotationAdditionalActions) clone(cache cloneCache) AnnotationAdditionalActions {
	var a AnnotationAdditionalActions
	a.E = ann.E.clone(cache)
	a.X = ann.X.clone(cache)
	a.D = ann.D.clone(cache)
	a.U = ann.U.clone(cache)
	a.Fo = ann.Fo.clone(cache)
	a.Bl = ann.Bl.clone(cache)
	a.PO = ann.PO.clone(cache)
	a.PC = ann.PC.clone(cache)
	a.PV = ann.PV.clone(cache)
	a.PI = ann.PI.clone(cache)
	return a
}

// --------------------------------------------------------------

const (
	RenditionPlay = iota
	RenditionStop
	RenditionPause
	RenditionResume
	RenditionPlayWithAN
)

// ActionRendition controls the playing of multimedia content
// See 12.6.4.13 - Rendition Actions
type ActionRendition struct {
	R  RenditionDict   // optional
	AN *AnnotationDict // optional, must be with type Screen
	OP MaybeInt        // optional, see the RenditionXxx constants
	JS string          // optional, maybe written in PDF a text stream
}

func (a ActionRendition) actionParams(pdf pdfWriter, ref Reference) string {
	out := "/S/Rendition"
	if a.R.Subtype != nil {
		out += "/R " + a.R.pdfString(pdf, ref)
	}
	if a.AN != nil {
		ref := pdf.addItem(a.AN)
		out += "/AN " + ref.String()
	}
	if a.OP != nil {
		out += fmt.Sprintf("/OP %d", a.OP.(ObjInt))
	}
	if a.JS != "" {
		out += "/JS " + pdf.EncodeString(a.JS, TextString, ref)
	}
	return out
}

func (ac ActionRendition) clone(cache cloneCache) ActionType {
	out := ac
	out.R = ac.R.clone(cache)
	out.AN = ac.AN.clone(cache).(*AnnotationDict)
	return out
}
