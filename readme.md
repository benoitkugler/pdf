# Golang PDF toolbox

## Why yet another PDF processing library ?

There are already numerous good PDF libraries for Go, and this one deliberatly takes inspiration from them. However, it is based on a slighty different approach : instead of working with a PDF as a tree of dynamic objects, it starts by modeling the whole SPEC (at least a good portion of it) with static types: see the package [model](model).

## Overview

The package model is the corner stone of this library. Then, packages may be divided in two parts:

- [reader](reader) imports a PDF file into memory

- [fonts](fonts) provides support to use embeded PDF fonts

- [contentstream](contentstream) and [formfill](formfill) provides tools to create PDF models

## Scope

The idea is possibly to provide a complete support of the PDF spec, but more importantly to exposes the differents layers (such as parser or content stream operators) so that it can be reusable by other libraries.
As such, the first target of this library would be higher levels libraries (such as pdfcpu, gofpdf, oksvg, etc...).

## Code example

A standard workflow to modify an existing PDF would look like

```[go]
// load the existing file in memory
fi, _, err := reader.ParsePDFFile(filePath, reader.Options{})
// error handling ...

// process the document model as you wish

err = fi.WriteFile(output, nil)
// error handling ...
```

See [decompress](cmd/decompress/decompress.go) and [api](apidemo/api.go) for more examples.
