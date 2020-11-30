package formfill

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/fonts/standardfonts"
	"github.com/benoitkugler/pdf/model"
)

func TestBreaks(t *testing.T) {
	s := strings.Repeat("84'(à)ç,lkfiiiiiiiiiiiiiiiiiiiiii", 10)
	font, err := fonts.BuildFont(&model.FontDict{Subtype: standardfonts.Times_BoldItalic.WesternType1Font()})
	if err != nil {
		t.Errorf("can't built standard font: %s", err)
	}
	fmt.Println(strings.Join(breakLines(getHardBreaks(s), font, 8, 50), "\n"))

	fmt.Println(font.GetWidth('i', 10), font.GetWidth('8', 10))
}

func TestEncoding(t *testing.T) {
	fmt.Println(defaultFont.Subtype.(model.FontType1).Widths[160-32])
}
