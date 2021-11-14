module github.com/benoitkugler/pdf

go 1.16

require (
	github.com/benoitkugler/pstokenizer v1.0.1
	github.com/benoitkugler/textlayout v0.0.3
	github.com/hhrutter/lzw v0.0.0-20190829144645-6f07a24e8650
	github.com/pdfcpu/pdfcpu v0.3.12
	github.com/phpdave11/gofpdf v1.4.2
	golang.org/x/exp/errors v0.0.0-20211111183329-cb5df436b1a8
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410
	golang.org/x/text v0.3.7
)

replace github.com/pdfcpu/pdfcpu => ../../pdfcpu/pdfcpu
