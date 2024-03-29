package fonts

import (
	"os"
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func TestDefinedEnc(t *testing.T) {
	f := model.FontType1{
		Encoding: &model.SimpleEncodingDict{
			BaseEncoding: model.MacRomanEncoding,
			Differences:  model.Differences{25: "idieresis", 149: "fraction"},
		},
	}

	if enc := ResolveSimpleEncoding(f); enc.RuneToByte()[239] != 25 {
		t.Error()
	}

	content, err := os.ReadFile("test/CalligrapherRegular.pfb")
	if err != nil {
		t.Fatal(err)
	}
	f = model.FontType1{
		Encoding: &model.SimpleEncodingDict{
			// BaseEncoding: MacRomanEncoding,
			Differences: model.Differences{25: "idieresis", 239: "fraction"},
		},
		FontDescriptor: model.FontDescriptor{FontFile: &model.FontFile{Stream: model.Stream{Content: content}}},
	}
	if enc := ResolveSimpleEncoding(f).RuneToByte(); enc[239] != 25 {
		t.Errorf("expected 25, got %d", enc[239])
	}
}
