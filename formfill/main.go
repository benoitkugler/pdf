package formfill

import (
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"golang.org/x/text/encoding/unicode"
)

const path = "../goACVE/ressources/ModeleRecuFiscalEditable.pdf"

// const path = "../goACVE/local/recu_fiscal_4568.pdf"

var encoder = unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder()

// Fields map a field name (T in PDF spec) to a value, which
// may be of type Name or (unencoded) StringLitteral
type Fields map[string]pdfcpu.Object
