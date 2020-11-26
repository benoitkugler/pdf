package standardfonts

import "github.com/benoitkugler/pdf/model"

// Metrics provide metrics for the font builtin encoding
type Metrics struct {
	Descriptor model.FontDescriptor
	FirstChar  byte
	Widths     []int
}

type winAnsiMetrics struct {
	FirstChar byte
	Widths    []int
}

// WesternType1Font return a version of the font
// using WinAnsi encoding (except for Symbol and ZapfDingbats)
func (m Metrics) WesternType1Font() model.FontType1 {
	if m.Descriptor.FontName == "ZapfDingbats" || m.Descriptor.FontName == "Symbol" {
		return model.FontType1{
			FirstChar:      m.FirstChar,
			Widths:         m.Widths,
			FontDescriptor: m.Descriptor,
			BaseFont:       m.Descriptor.FontName,
		}
	}

	winAnsi := winAnsiMetricsMap[string(m.Descriptor.FontName)]
	return model.FontType1{
		FirstChar:      winAnsi.FirstChar,
		Widths:         winAnsi.Widths,
		FontDescriptor: m.Descriptor,
		BaseFont:       m.Descriptor.FontName,
		Encoding:       model.WinAnsiEncoding,
	}
}
