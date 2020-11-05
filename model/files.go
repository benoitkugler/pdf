package model

import "time"

type EmbeddedFile struct {
	Name     string
	FileSpec *FileSpec // indirect
}

type FileSpec struct {
	UF   string
	EF   *EmbeddedFileStream
	Desc string
}

type EmbeddedFileParams struct {
	Size         int
	CreationDate time.Time
	ModDate      time.Time
	CheckSum     string // should be wrote as hex16 encoded
}

type EmbeddedFileStream struct {
	ContentStream
	Params EmbeddedFileParams
}
