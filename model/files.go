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
func (f *FileSpec) pdfContent(pdf pdfWriter, ref reference) (string, []byte) {
	b := newBuffer()
	b.fmt("<</Type/Filespec")
	if f.UF != "" {
		b.fmt("/UF %s", pdf.EncodeString(f.UF, TextString, ref))
	}
	if f.EF != nil {
		ref := pdf.addObject(f.EF.pdfContent(pdf, ref))
		b.fmt("/EF %s", ref)
	}
	if f.Desc != "" {
		b.fmt("/Desc %s", pdf.EncodeString(f.Desc, TextString, ref))
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

func (params EmbeddedFileParams) pdfString(pdf pdfWriter, ref reference) string {
	b := newBuffer()
	b.WriteString("<<")
	if params.Size != 0 {
		b.fmt("/Size %d", params.Size)
	}
	if !params.CreationDate.IsZero() {
		b.fmt("/CreationDate %s", pdf.dateString(params.CreationDate, ref))
	}
	if !params.ModDate.IsZero() {
		b.fmt("/ModDate %s", pdf.dateString(params.ModDate, ref))
	}
	if params.CheckSum != "" {
		b.fmt("/CheckSum <%s>", params.CheckSum)
	}
	b.fmt(">>")
	return b.String()
}

type EmbeddedFileStream struct {
	Stream
	Params EmbeddedFileParams
}

func (emb *EmbeddedFileStream) pdfContent(pdf pdfWriter, ref reference) (string, []byte) {
	args := emb.PDFCommonFields()
	out := fmt.Sprintf("<</Type/EmbeddedFile %s/Params %s>>", args, emb.Params.pdfString(pdf, ref))
	return out, emb.Content
}
