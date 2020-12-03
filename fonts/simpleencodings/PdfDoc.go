package simpleencodings

import "github.com/benoitkugler/pdf/model"

// PdfDoc is the PdfDoc encoding.
// It should not be used in fonts, but
// is exposed here for the sake of completeness.
var PdfDoc = Encoding{
	Names: model.PdfDocNames,
	Runes: model.PdfDocRunes,
}
