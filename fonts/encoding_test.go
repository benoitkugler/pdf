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
	if resolveCharMapType1(f, nil)[239] != 25 {
		t.Error()
	}

	content, err := ioutil.ReadFile("type1font/test/CalligrapherRegular.pfb")
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
	if b := resolveCharMapType1(f, nil)[239]; b != 25 {
		t.Errorf("expected 25, got %d", b)
	}
}
