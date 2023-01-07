// This script decodes the streams of a PDF file.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("missing input file")
	}
	filePath := os.Args[1]

	fi, _, err := reader.ParsePDFFile(filePath, reader.Options{})
	if err != nil {
		log.Fatalf("reading input: %s", err)
	}

	for _, page := range fi.Catalog.Pages.Flatten() {
		for i, ct := range page.Contents {
			decoded, err := ct.Decode()
			if err != nil {
				log.Fatal(err)
			}
			page.Contents[i] = model.ContentStream{
				Stream: model.Stream{
					Content: decoded,
					// no filters
				},
			}
		}
	}

	output := filePath + ".decoded.pdf"
	err = fi.WriteFile(output, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Written in", output)
}
