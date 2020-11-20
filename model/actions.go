package model

import (
	"fmt"
	"strconv"
	"strings"
)

type FormFielAdditionalActions struct {
	K Action // JavaScript action, optional, on update
	F Action // JavaScript action, optional, before formating
	V Action // JavaScript action, optional, on validate
	C Action // JavaScript action, optional, to recalculate
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

func (ff *FormFielAdditionalActions) Clone() *FormFielAdditionalActions {
	if ff == nil {
		return nil
	}
	a := *ff
	return &a
}

// Action defines the characteristics and behaviour of an action.
type Action struct {
	ActionType
	// sequence of actions that shall be performed after
	// the action represented by this dictionary
	Next []Action
}

func (a Action) pdfString(pdf pdfWriter, context Reference) string {
	subtype := a.ActionType.actionParams(pdf, context)
	chunks := make([]string, len(a.Next))
	for i, n := range a.Next {
		chunks[i] = n.pdfString(pdf, context)
	}
	return fmt.Sprintf("<<%s /Next [%s]>>", subtype, strings.Join(chunks, " "))
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

// ActionType specialize the action
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
// display a destination page
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
	Zoom      float64
}

func (d DestinationLocationXYZ) locationElements() string {
	return fmt.Sprintf("/XYZ %s %s %.3f",
		writeMaybeFloat(d.Left), writeMaybeFloat(d.Top), d.Zoom)
}

// DestinationLocationFitR is /FitR
type DestinationLocationFitR struct {
	Left, Bottom, Right, Top float64
}

func (d DestinationLocationFitR) locationElements() string {
	return fmt.Sprintf("/FitR %.3f %.3f %.3f %.3f",
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

type DestinationName Name

func (n DestinationName) pdfDestination(pdfWriter, Reference) string {
	return Name(n).String()
}

func (d DestinationName) clone(cloneCache) Destination { return d }

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
		fs, _ := ac.F.pdfContent(pdf, ref)
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
		fs, _ := ac.F.pdfContent(pdf, ref)
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
	return ac
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
