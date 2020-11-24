package formfill

import (
	"bytes"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

type FieldValue interface {
}

type FDFField struct {
	T string // field name
	V FieldValue
}

func ReadFDF(data []byte) error {
	ctx, err := pdfcpu.Read(bytes.NewReader(data), nil)
	if err != nil {
		return err
	}
	fmt.Println(ctx)
	return nil
}
