package formfill

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/standardfonts"
)

func TestBreaks(t *testing.T) {
	s := strings.Repeat("84'(à)ç,lkfiiiiiiiiiiiiiiiiiiiiii", 10)
	font := fonts.BuildFont(&model.FontDict{Subtype: model.FontType1{
		FirstChar:      standardfonts.Times_BoldItalic.FirstChar,
		Widths:         standardfonts.Times_BoldItalic.Widths,
		FontDescriptor: standardfonts.Times_BoldItalic.Descriptor,
	}})
	fmt.Println(strings.Join(breakLines(getHardBreaks(s), font, 8, 50), "\n"))

	fmt.Println(font.GetWidth('i', 10), font.GetWidth('8', 10))
}

func TestEncoding(t *testing.T) {
	fmt.Println(defaultFont.Subtype.(model.FontType1).Widths[160-32])
}
