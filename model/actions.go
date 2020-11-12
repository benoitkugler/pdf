package model

import "fmt"

type FormFielAdditionalActions struct {
	K JavaScriptAction // optional, on update
	F JavaScriptAction // optional, before formating
	V JavaScriptAction // optional, on validate
	C JavaScriptAction // optional, to recalculate
}

type Action interface {
	// ActionDictionary returns the dictionary defining the action
	// as written in PDF
	ActionDictionary(pdfWriter) string
}

// URIAction is a URI which should be ASCII encoded
type URIAction string

func (uri URIAction) ActionDictionary(pdf pdfWriter) string {
	return fmt.Sprintf("<</S /URI /URI (%s)>>", pdf.EncodeString(string(uri), ASCIIString))
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

type JavaScriptAction struct {
	JS string // text string, may be found in PDF as stream object
}

func (j JavaScriptAction) ActionDictionary(pdf pdfWriter) string {
	return fmt.Sprintf("<</S /JavaScript /JS %s>>", pdf.EncodeString(j.JS, TextString))
}
