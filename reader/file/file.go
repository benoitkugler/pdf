// Package file builds upon a parser
// to read an existing PDF file, producing a
// tree of PDF objets.
// See pacakge reader for an higher level of processing.
package file

import "io"

// The logic is adapted from pdfcpu

type Configuration struct{}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{}
}

func Read(rs io.ReadSeeker, conf *Configuration) error {
	ctx, err := newContext(rs, conf)
	if err != nil {
		return err
	}

	o, err := ctx.offsetLastXRefSection(0)
	if err != nil {
		return err
	}

	err = ctx.buildXRefTableStartingAt(o)
	if err != nil {
		return err
	}

	// start by reading object stream and adding them as regular objects
	err = ctx.processObjectStreams()
	if err != nil {
		return err
	}

	return nil
}
