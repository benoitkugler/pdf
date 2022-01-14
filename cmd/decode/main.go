// This tools reads a PDF file and decode all the streams.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader"
)

func check(err error) {
	if err != nil {
		fmt.Println("fatal error", err)
		os.Exit(1)
	}
}

func decodeStream(c *model.Stream) {
	b, err := c.Decode()
	check(err)

	*c = model.Stream{Content: b}
}

func decodeResources(res model.ResourcesDict) {
	for _, xo := range res.XObject {
		if form, ok := xo.(*model.XObjectForm); ok {
			decodeStream(&form.Stream)
			decodeResources(form.Resources)
		}
	}

	for _, gs := range res.ExtGState {
		if gs.SMask.G != nil {
			decodeStream(&gs.SMask.G.Stream)
			decodeResources(gs.SMask.G.Resources)
		}
	}
}

func main() {
	flag.Parse()
	input := flag.Arg(0)

	fmt.Println(input)
	doc, _, err := reader.ParsePDFFile(input, reader.Options{})
	check(err)

	for _, p := range doc.Catalog.Pages.Flatten() {
		for i := range p.Contents {
			decodeStream(&p.Contents[i].Stream)
		}
		if p.Resources != nil {
			decodeResources(*p.Resources)
		}
	}

	err = doc.WriteFile(input+".dec.pdf", nil)
	check(err)
	fmt.Println("Done")
}
