package fonts

import (
	"fmt"
	"testing"

	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
)

func TestStandard(t *testing.T) {
	for name, builtin := range standardfonts.Fonts {
		f := builtin.WesternType1Font()
		font := BuildFont(&model.FontDict{Subtype: f})
		fmt.Println(name, font.GetWidth('u', 12))
	}
}
