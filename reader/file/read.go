package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Context represents an environment for processing PDF files.
type Context struct {
	rs       io.ReadSeeker
	fileSize int64

	*Configuration
	// *XRefTable
}

type Configuration struct{}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{}
}

func newContext(rs io.ReadSeeker, conf *Configuration) (*Context, error) {

	rdCtx := &Context{
		rs: rs,
		// ObjectStreams: intSet{},
		// XRefStreams:   intSet{},
		Configuration: conf,
	}

	fileSize, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	rdCtx.fileSize = fileSize

	return rdCtx, nil
}

// NewContext initializes a new Context.
func NewContext(rs io.ReadSeeker, conf *Configuration) (*Context, error) {

	if conf == nil {
		conf = NewDefaultConfiguration()
	}

	ctx, err := newContext(rs, conf)
	if err != nil {
		return nil, err
	}

	// ctx := &Context{
	// 	conf,
	// 	newXRefTable(conf.ValidationMode),
	// 	rdCtx,
	// }

	return ctx, nil
}

func (ctx *Context) readAt(p []byte, offset int64) error {
	_, err := ctx.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = ctx.rs.Read(p)
	return err
}

// Get the file offset of the last XRefSection.
// Go to end of file and search backwards for the first occurrence of startxref {offset} %%EOF
// xref at 114172
func (ctx *Context) offsetLastXRefSection(skip int64) (int64, error) {

	rs := ctx.rs

	var (
		prevBuf, workBuf []byte
		bufSize          int64 = 512
		offset           int64
	)

	// guard for very small files
	if ctx.fileSize < bufSize {
		bufSize = ctx.fileSize
	}

	for i := 1; offset == 0; i++ {

		_, err := rs.Seek(-int64(i)*bufSize-skip, io.SeekEnd)
		if err != nil {
			return 0, fmt.Errorf("can't find last xref section: %s", err)
		}

		curBuf := make([]byte, bufSize)

		_, err = rs.Read(curBuf)
		if err != nil {
			return 0, fmt.Errorf("can't read last xref section: %s", err)
		}

		workBuf = append(curBuf, prevBuf...)

		j := bytes.LastIndex(workBuf, []byte("startxref"))
		if j == -1 {
			prevBuf = curBuf
			continue
		}

		p := workBuf[j+len("startxref"):]
		posEOF := bytes.Index(p, []byte("%%EOF"))
		if posEOF == -1 {
			return 0, errors.New("no matching %%EOF for startxref")
		}

		p = p[:posEOF]
		offset, err = strconv.ParseInt(string(bytes.TrimSpace(p)), 10, 64)
		if err != nil || offset >= ctx.fileSize {
			return 0, errors.New("corrupted last xref section")
		}
	}
	return offset, nil
}
