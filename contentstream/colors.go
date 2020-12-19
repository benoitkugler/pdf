package contentstream

import "image/color"

// return the more precise representation of the color
func colorToArray(col color.Color) []Fl {
	switch col := col.(type) {
	case color.Gray:
		return []Fl{Fl(col.Y) / 255}
	case color.Gray16:
		return []Fl{Fl(col.Y) / 0xFFFF}
	case color.RGBA:
		return []Fl{Fl(col.R) / Fl(col.A), Fl(col.G) / Fl(col.A), Fl(col.B) / Fl(col.A)}
	case color.RGBA64:
		return []Fl{Fl(col.R) / Fl(col.A), Fl(col.G) / Fl(col.A), Fl(col.B) / Fl(col.A)}
	case color.NRGBA:
		return []Fl{Fl(col.R) / 255, Fl(col.G) / 255, Fl(col.B) / 255}
	case color.NRGBA64:
		return []Fl{Fl(col.R) / 0xFFFF, Fl(col.G) / 0xFFFF, Fl(col.B) / 0xFFFF}
	case color.CMYK:
		return []Fl{Fl(col.C) / 255, Fl(col.M) / 255, Fl(col.Y) / 255, Fl(col.K) / 255}
	default: // default to interface method
		r, g, b := colorRGB(col)
		return []Fl{r, g, b}
	}
}

func clamp(ch, a uint32) Fl {
	if ch < 0 {
		return 0
	}
	if ch > a {
		return 1
	}
	return Fl(ch) / Fl(a)
}

func colorRGB(c color.Color) (r, g, b Fl) {
	if c == nil {
		return 0, 0, 0
	}
	cr, cg, cb, ca := c.RGBA()
	return clamp(cr, ca), clamp(cg, ca), clamp(cb, ca)
}
