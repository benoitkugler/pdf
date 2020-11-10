package model

type FormType string

const (
	Tx  FormType = "Tx"
	Btn FormType = "Btn"
	Ch  FormType = "Ch"
	Sig FormType = "Sig"
)

type FormFlag uint32

const (
	Multiline       FormFlag = 1 << 13
	Password        FormFlag = 1 << 14
	FileSelect      FormFlag = 1 << 21
	DoNotSpellCheck FormFlag = 1 << 23
	DoNotScroll     FormFlag = 1 << 24
	Comb            FormFlag = 1 << 25
	RichText        FormFlag = 1 << 26
)

type FormField struct {
	Ft         FormType
	T          string
	Ff         FormFlag
	MaxLen     int // optional, -1 when not set
	DA         string
	Annotation // might be merged
}

type AcroForm struct {
	Fields          []*FormField
	NeedAppearances bool
}

func (a AcroForm) pdfBytes(pdf PDFWriter) []byte {
	return nil
}
