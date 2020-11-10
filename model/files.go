package model

import (
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

// PDFString returns the dictionnay. `pdf` is used
// to create the EmbeddedFileStream object.
func (f *FileSpec) PDFBytes(pdf PDFWriter) []byte {
	b := newBuffer()
	b.fmt("<</Type /Filespec")
	if f.UF != "" {
		b.fmt(" /UF %s", pdf.EncodeTextString(f.UF))
	}
	if f.EF != nil {
		ref := pdf.addObject(f.EF.PDFBytes())
		b.fmt(" /EF %s", ref)
	}
	if f.Desc != "" {
		b.fmt(" /Desc %s", pdf.EncodeTextString(f.Desc))
	}
	b.fmt(">>")
	return b.Bytes()
}

type EmbeddedFileParams struct {
	Size         int       // optional
	CreationDate time.Time // optional
	ModDate      time.Time // optional
	CheckSum     string    // optional, should be hex16 encoded
}

func (params EmbeddedFileParams) PDFBytes() []byte {
	b := newBuffer()
	b.fmt("<<")
	if params.Size != 0 {
		b.fmt("/Size %d", params.Size)
	}
	if !params.CreationDate.IsZero() {
		b.fmt("/CreationDate %s", dateString(params.CreationDate))
	}
	if !params.ModDate.IsZero() {
		b.fmt("/ModDate %s", dateString(params.ModDate))
	}
	if params.CheckSum != "" {
		b.fmt("/CheckSum <%s>", params.CheckSum)
	}
	b.fmt(">>")
	return b.Bytes()
}

type EmbeddedFileStream struct {
	ContentStream
	Params EmbeddedFileParams
}

func (emb EmbeddedFileStream) PDFBytes() []byte {
	args := emb.PDFCommonFields()
	b := newBuffer()
	b.line("<</Type /EmbeddedFile %s /Params ", args)
	b.Write(emb.Params.PDFBytes())
	b.line(">>")
	b.line("stream")
	b.Write(emb.Content)
	b.WriteString("\nendstream")
	return b.Bytes()
}
