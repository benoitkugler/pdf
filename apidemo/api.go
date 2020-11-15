package apidemo

import (
	"bytes"
	"compress/zlib"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/benoitkugler/pdf/model"
)

// ListAttachments returns a list of embedded file attachment names of `doc`.
func ListAttachments(doc model.Document) []string {
	out := make([]string, len(doc.Catalog.Names.EmbeddedFiles))
	for i, file := range doc.Catalog.Names.EmbeddedFiles {
		out[i] = file.FileSpec.UF
	}
	return out
}

// return the hex encoded checksum of `data`
func checksum(data []byte) string {
	tmp := md5.Sum(data)
	sl := make([]byte, len(tmp))
	for i, v := range tmp {
		sl[i] = v
	}
	return hex.EncodeToString(sl)
}

// sliceCompress returns a zlib-compressed copy of the specified byte array
func sliceCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	cmp, _ := zlib.NewWriterLevel(&buf, zlib.BestSpeed)
	_, err := cmp.Write(data)
	if err != nil {
		return nil, err
	}
	err = cmp.Close()
	return buf.Bytes(), err
}

// AddAttachments embeds files into the document and writes the result to w.
// A file is either a file name or a file name and a description separated by a comma.
func AddAttachments(doc *model.Document, w io.Writer, files []string) error {
	for _, fn := range files {
		s := strings.Split(fn, ",")
		if len(s) == 0 || len(s) > 2 {
			return fmt.Errorf("invalid file description : %s", fn)
		}

		fileName := s[0]
		desc := ""
		if len(s) == 2 {
			desc = s[1]
		}

		f, err := os.Open(fileName)
		if err != nil {
			return err
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			return err
		}
		mt := fi.ModTime()

		var emb model.EmbeddedFileStream
		emb.Content, err = ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("can't read file : %w", err)
		}
		emb.Params.ModDate = mt
		emb.Params.Size = int(fi.Size())
		emb.Params.CheckSum = checksum(emb.Content)

		// compression with flate, optional
		emb.Stream.Filter = []model.Filter{model.Flate}
		emb.Content, err = sliceCompress(emb.Content)
		if err != nil {
			return fmt.Errorf("can't compress file : %w", err)
		}

		fs := model.FileSpec{
			UF:   filepath.Base(fileName),
			EF:   &emb,
			Desc: desc,
		}

		att := model.NameToFile{Name: fs.UF, FileSpec: &fs}

		doc.Catalog.Names.EmbeddedFiles = append(doc.Catalog.Names.EmbeddedFiles, att)
	}

	err := doc.Write(w)
	return err
}
