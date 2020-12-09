package fonts

import (
	"io/ioutil"
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

	if enc := resolveSimpleEncoding(f); enc.RuneToByte()[239] != 25 {
		t.Error()
	}

	content, err := ioutil.ReadFile("type1/test/CalligrapherRegular.pfb")
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
	if enc := resolveSimpleEncoding(f).RuneToByte(); enc[239] != 25 {
		t.Errorf("expected 25, got %d", enc[239])
	}
}
