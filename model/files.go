package model

import (
	"crypto/md5"
	"encoding/hex"
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

// returns the dictionnay, with a nil content `pdf` is used
// to create the EmbeddedFileStream object.
func (f *FileSpec) pdfContent(pdf pdfWriter, ref Reference) (string, []byte) {
	b := newBuffer()
	b.fmt("<</Type/Filespec")
	if f.UF != "" {
		b.fmt("/UF %s", pdf.EncodeString(f.UF, TextString, ref))
	}
	if f.EF != nil {
		ref := pdf.addObject(f.EF.pdfContent(pdf, ref))
		b.fmt("/EF <</F %s>>", ref)
	}
	if f.Desc != "" {
		b.fmt("/Desc %s", pdf.EncodeString(f.Desc, TextString, ref))
	}
	b.fmt(">>")
	return b.String(), nil
}

func (f *FileSpec) clone(cache cloneCache) Referenceable {
	if f == nil {
		return f
	}
	out := *f
	out.EF = cache.checkOrClone(f.EF).(*EmbeddedFileStream)
	return &out
}

type EmbeddedFileParams struct {
	CreationDate time.Time // optional
	ModDate      time.Time // optional
	Size         int       // optional
	CheckSum     string    // optional, must be hex16 encoded
}

// SetChecksumAndSize compute the size and the checksum of the `content`,
// which must be the original (not encoded) data.
func (f *EmbeddedFileParams) SetChecksumAndSize(content []byte) {
	f.Size = len(content)
	tmp := md5.Sum(content)
	f.CheckSum = hex.EncodeToString(tmp[:])
}

func (params EmbeddedFileParams) pdfString(pdf pdfWriter, ref Reference) string {
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

func (emb *EmbeddedFileStream) pdfContent(pdf pdfWriter, ref Reference) (string, []byte) {
	args := emb.PDFCommonFields(true)
	out := fmt.Sprintf("<</Type/EmbeddedFile %s/Params %s>>", args, emb.Params.pdfString(pdf, ref))
	return out, emb.Content
}

// clone returns a deep copy, with concrete type `*EmbeddedFileStream`
func (emb *EmbeddedFileStream) clone(cloneCache) Referenceable {
	if emb == nil {
		return emb
	}
	out := *emb // shallow copy
	out.Stream = emb.Stream.Clone()
	return &out
}
