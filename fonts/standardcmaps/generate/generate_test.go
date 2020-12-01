package generate

import (
	"testing"
)

func TestGeneratePredefined(t *testing.T) {
	for _, file := range [...]string{
		"data/Adobe-CNS1-UCS2.txt",
		"data/Adobe-GB1-UCS2.txt",
		"data/Adobe-Japan1-UCS2.txt",
		"data/Adobe-Korea1-UCS2.txt",
		"data/Adobe-KR-UCS2.txt",
	} {
		err := generatedPredefined(file)
		if err != nil {
			t.Fatal(err)
		}
	}
}
