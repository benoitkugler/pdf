package reader

import (
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestDifferences(t *testing.T) {
	expected := model.Differences{24: "breve", 25: "caron", 39: "quotesingle", 96: "grave", 128: "bullet", 129: "emdash"}
	ar := pdfcpu.Array{
		pdfcpu.Integer(39),
		pdfcpu.Name("quotesingle"),
		pdfcpu.Integer(24),
		pdfcpu.Name("breve"),
		pdfcpu.Name("caron"),
		pdfcpu.Integer(96),
		pdfcpu.Name("grave"),
		pdfcpu.Integer(128),
		pdfcpu.Name("bullet"),
		pdfcpu.Name("emdash"),
	}
	var r resolver
	diff := r.parseDiffArray(ar)
	if !reflect.DeepEqual(diff, expected) {
		t.Errorf("expected %v, got %v", expected, diff)
	}
}
