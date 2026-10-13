package gfx

// Buffer drawing modes (C: the low bits of fn_1ddc_0026's flag word).
const (
	BufferRemapped    = 1 << 0 // through Levels/LevelTables, like ShapeLevels
	BufferTransparent = 1 << 2 // colour 0 is not drawn
)

// DrawBuffer draws a w x h block of pixels onto the current page at screen position
// (x, y), clipped to a window (C: fn_1ddc_0026). With BufferRemapped, levels gives each
// colour a row of tables (255 = draw as is) indexed by the pixel underneath.
//
// The original's modes 8-15 composite against a priority and background buffer whose
// pointers the game never sets; they are left out.
func (s *Screen) DrawBuffer(x, y, w, h, window, mode int, pixels, levels, tables []byte) {
	win := s.Windows[window]
	stride := w
	skip := 0
	if y < win.Top() {
		cut := win.Top() - y
		h -= cut
		if h <= 0 {
			return
		}
		y = win.Top()
		skip = cut * stride
	}
	h = min(h, win.Bottom()-y)
	if h <= 0 {
		return
	}
	left := 0
	if x < win.Left() {
		left = win.Left() - x
		x = win.Left()
		w -= left
	}
	w = min(w, win.Right()-x)
	// fix: a block entirely left of the window drew garbage in the original.
	if w <= 0 {
		return
	}
	page := s.Page(s.Current)
	for row := range h {
		src := pixels[skip+row*stride+left:]
		dst := page[(y+row)*Width+x:]
		for i := range w {
			c := src[i]
			if mode&BufferTransparent != 0 && c == 0 {
				continue
			}
			if mode&BufferRemapped != 0 {
				if level := levels[c]; level != 255 {
					c = tables[int(level)*256+int(dst[i])]
				}
			}
			dst[i] = c
		}
	}
}
