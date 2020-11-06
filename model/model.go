// Implements the in-memory structure of the PDFs object
// Whenever possible, use static types.
// The structure is not directly the one found or written
// in a PDF, but it serves as an intermediate representation
// to facilitate PDF modifications.
package model

import "time"

// Document is the top-level object,
// representing a whole PDF file.
type Document struct {
	Trailer Trailer
	Catalog Catalog
}

type Catalog struct {
	Extensions        Extensions
	Pages             PageTree
	Names             NameDictionnary
	ViewerPreferences ViewerPreferences
	AcroForm          AcroForm
	Dests             DestTree
}

type NameDictionnary struct {
	EmbeddedFiles EmbeddedFileTree
	Dests         DestTree
	// AP
}

type ViewerPreferences struct {
	FitWindow    bool
	CenterWindow bool
}

type Trailer struct {
	Encrypt Encrypt
	Info    Info
}

type Info struct {
	Producer     string
	Title        string
	Subject      string
	Author       string
	Keywords     string
	Creator      string
	CreationDate time.Time
	ModDate      time.Time
}

type EncryptionAlgorithm uint8

const (
	Undocumented EncryptionAlgorithm = iota
	AES
	AESExt // encryption key with length greater than 40
	Unpublished
	InDocument
)

type Encrypt struct {
	Filter    Name
	SubFilter Name
	V         EncryptionAlgorithm
	Length    int
}
