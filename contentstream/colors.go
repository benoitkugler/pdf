package contentstream

import "image/color"

func clamp(ch, a uint32) Fl {
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
