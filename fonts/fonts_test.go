package fonts

import (
	"fmt"
	"testing"

	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/standardfonts"
)

func TestStandard(t *testing.T) {
	for name, builtin := range standardfonts.Fonts {
		f := model.FontType1{
			FirstChar:      builtin.FirstChar,
			Widths:         builtin.Widths,
			FontDescriptor: builtin.Descriptor,
		}
		font := BuildFont(&model.FontDict{Subtype: f})
		fmt.Println(name, font.GetWidth('u', 12))
	}
}
