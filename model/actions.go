package model

import "fmt"

type FormFielAdditionalActions struct {
	K ActionJavaScript // optional, on update
	F ActionJavaScript // optional, before formating
	V ActionJavaScript // optional, on validate
	C ActionJavaScript // optional, to recalculate
}

func (f FormFielAdditionalActions) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if f.K != (ActionJavaScript{}) {
		b.line("/K %s", f.K.ActionDictionary(pdf, ref))
	}
	if f.F != (ActionJavaScript{}) {
		b.line("/F %s", f.F.ActionDictionary(pdf, ref))
	}
	if f.V != (ActionJavaScript{}) {
		b.line("/V %s", f.V.ActionDictionary(pdf, ref))
	}
	if f.C != (ActionJavaScript{}) {
		b.line("/C %s", f.C.ActionDictionary(pdf, ref))
	}
	b.fmt(">>")
	return b.String()
}

// TODO: support more action type
type Action interface {
	// actionDictionary returns the dictionary defining the action
	// as written in PDF
	actionDictionary(pdfWriter, Reference) string
	clone(cache cloneCache) Action
}

type ActionJavaScript struct {
	JS string // text string, may be found in PDF as stream object
}

func (j ActionJavaScript) ActionDictionary(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("<</S/JavaScript/JS %s>>", pdf.EncodeString(j.JS, TextString, ref))
}

// ActionURI is a URI which should be ASCII encoded
type ActionURI string

func (uri ActionURI) actionDictionary(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("<</S/URI/URI (%s)>>", pdf.EncodeString(string(uri), ASCIIString, ref))
}

func (uri ActionURI) clone(cache cloneCache) Action { return uri }

type ActionGoTo struct {
	D Destination
}

func (ac ActionGoTo) actionDictionary(pdf pdfWriter, _ Reference) string {
	return fmt.Sprintf("<</S/GoTo/D %s>>", ac.D.pdfDestination(pdf))
}

func (ac ActionGoTo) clone(cache cloneCache) Action {
	return ActionGoTo{D: ac.D.clone(cache)}
}

type Destination interface {
	// return the PDF content of the destination
	pdfDestination(pdfWriter) string
	clone(cache cloneCache) Destination
}

// DestinationExplicit is an explicit destination to a page
type DestinationExplicit struct {
	Page      *PageObject
	Left, Top float64 // Undef for null value
	Zoom      float64
}

func (d DestinationExplicit) pdfDestination(pdf pdfWriter) string {
	pageRef := pdf.pages[d.Page]
	left, top := "null", "null"
	if d.Left != Undef {
		left = fmt.Sprintf("%.3f", d.Left)
	}
	if d.Top != Undef {
		top = fmt.Sprintf("%.3f", d.Top)
	}
	return fmt.Sprintf("[%s/XYZ %s %s %.3f]", pageRef, left, top, d.Zoom)
}

func (d DestinationExplicit) clone(cache cloneCache) Destination {
	out := d
	out.Page = cache.pages[d.Page].(*PageObject)
	return d
}

type DestinationName Name

func (n DestinationName) pdfDestination(pdfWriter) string {
	return Name(n).String()
}

func (d DestinationName) clone(cloneCache) Destination { return d }

type DestinationString string

func (s DestinationString) pdfDestination(pdf pdfWriter) string { return fmt.Sprintf("(%s)", s) }

func (d DestinationString) clone(cloneCache) Destination { return d }
