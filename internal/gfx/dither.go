package gfx

// DitherRect sets every other pixel of a rectangle on the current page, in a
// checkerboard fixed to the screen (pixels whose column and row are both even or both
// odd), clipped to the screen (C: fn_1b63_0004).
func (s *Screen) DitherRect(x, y, w, h, colour int) {
	if w <= 0 || x >= Width {
		return
	}
	if x < 0 {
		w += x
		if w <= 0 {
			return
		}
		x = 0
	}
	w = min(w, Width-x)
	// fix: the original tested y against the screen's width, so rows 200-319 ran on
	// through memory.
	if h <= 0 || y >= Height {
		return
	}
	if y < 0 {
		h += y
		if h <= 0 {
			return
		}
		y = 0
	}
	h = min(h, Height-y)
	page := s.Page(s.Current)
	for row := y; row < y+h; row++ {
		first := x + (row+x)&1
		for col := first; col < x+w; col += 2 {
			page[row*Width+col] = byte(colour)
		}
	}
}

// CopyRectRemapped copies a w x h rectangle from page from to page to through a colour
// table, unclipped (C: fn_1b63_00e2).
func (s *Screen) CopyRectRemapped(fromX, fromY, toX, toY, w, h, from, to int, table []byte) {
	src, dst := s.Page(from), s.Page(to)
	for row := range h {
		in := (fromY+row)*Width + fromX
		out := (toY+row)*Width + toX
		for i := range w {
			dst[out+i] = table[src[in+i]]
		}
	}
}
