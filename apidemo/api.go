package apidemo

import (
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

// AddAttachments embeds files into the document and writes the result to w.
// A file is either a file name or a file name and a description separated by a comma.
func AddAttachments(doc *model.Document, enc *model.Encrypt, w io.Writer, files []string) error {
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

		content, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("can't read file : %w", err)
		}

		var emb model.EmbeddedFileStream
		emb.Params.SetChecksumAndSize(content)
		emb.Params.ModDate = fi.ModTime()

		// compression with flate, optional
		emb.Stream, err = model.NewStream(content, model.Filter{Name: model.Flate}, model.Filter{Name: model.ASCIIHex})
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

	err := doc.Write(w, enc)
	return err
}

// ExtractContent dumps "PDF source" files from `doc` into `outDir` for selected pages.
// Passing `nil` for `pageNumbers` extracts all pages. Invalid page numbers are ignored.
func ExtractContent(doc model.Document, outDir string, pageNumbers []int) error {
	// Note: the parsing of the page selection must have been done previously
	pages := doc.Catalog.Pages.Flatten()

	if pageNumbers == nil {
		pageNumbers = make([]int, len(pages))
		for i := 0; i < len(pages); i++ {
			pageNumbers[i] = i
		}
	}

	seen := map[int]bool{}
	for _, pageNumber := range pageNumbers {
		if seen[pageNumber] { // avoid duplicate
			continue
		}
		if pageNumber >= len(pages) { // Handle overflow gracefully
			continue
		}
		seen[pageNumber] = true

		var totalPageContent []byte
		for _, ct := range pages[pageNumber].Contents {
			ctContent, err := ct.Decode()
			if err != nil {
				return err
			}
			totalPageContent = append(totalPageContent, ctContent...)
		}
		outPath := filepath.Join(outDir, fmt.Sprintf("Content_page_%d.txt", pageNumber))
		err := ioutil.WriteFile(outPath, totalPageContent, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
