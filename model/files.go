package model

import (
	"fmt"
	"time"
)

// FileSpec is a File Specification Dictionary.
// In PDF, it may be found as a simple File Specification String
// which should be stored in the `UF` field.
type FileSpec struct {
	UF   string // optional
	EF   *EmbeddedFileStream
	Desc string // optional
}

// returns the dictionnay. `pdf` is used
// to create the EmbeddedFileStream object.
func (f *FileSpec) pdfContent(pdf pdfWriter) (string, []byte) {
	b := newBuffer()
	b.fmt("<</Type/Filespec")
	if f.UF != "" {
		b.fmt("/UF %s", pdf.encodeString(f.UF, textString))
	}
	if f.EF != nil {
		ref := pdf.addObject(f.EF.pdfContent(pdf))
		b.fmt("/EF %s", ref)
	}
	if f.Desc != "" {
		b.fmt("/Desc %s", pdf.encodeString(f.Desc, textString))
	}
	b.fmt(">>")
	return b.String(), nil
}

type EmbeddedFileParams struct {
	Size         int       // optional
	CreationDate time.Time // optional
	ModDate      time.Time // optional
	CheckSum     string    // optional, must be hex16 encoded
}

func (params EmbeddedFileParams) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	b.WriteString("<<")
	if params.Size != 0 {
		b.fmt("/Size %d", params.Size)
	}
	if !params.CreationDate.IsZero() {
		b.fmt("/CreationDate %s", pdf.dateString(params.CreationDate))
	}
	if !params.ModDate.IsZero() {
		b.fmt("/ModDate %s", pdf.dateString(params.ModDate))
	}
	if params.CheckSum != "" {
		b.fmt("/CheckSum <%s>", params.CheckSum)
	}
	b.fmt(">>")
	return b.String()
}

type EmbeddedFileStream struct {
	ContentStream
	Params EmbeddedFileParams
}

func (emb EmbeddedFileStream) pdfContent(pdf pdfWriter) (string, []byte) {
	args := emb.PDFCommonFields()
	out := fmt.Sprintf("<</Type/EmbeddedFile %s/Params %s>>", args, emb.Params.pdfString(pdf))
	return out, emb.Content
}
