package gfx

// clipToScreen cuts a rectangle to the screen; ok is false if nothing is left
// (C: the clipping that opens fn_2096_0000 and fn_2096_014b).
func clipToScreen(x, y, w, h int) (int, int, int, int, bool) {
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	w = min(w, Width-x)
	h = min(h, Height-y)
	if w <= 0 || h <= 0 || x >= Width || y >= Height {
		return 0, 0, 0, 0, false
	}
	return x, y, w, h, true
}

// SaveRect copies what a rectangle of a page shows into a new buffer, rows packed at the
// clipped width (C: fn_2096_014b).
func (s *Screen) SaveRect(x, y, w, h, page int) []byte {
	x, y, w, h, ok := clipToScreen(x, y, w, h)
	if !ok {
		return nil
	}
	src := s.Page(page)
	buf := make([]byte, w*h)
	for row := range h {
		copy(buf[row*w:(row+1)*w], src[(y+row)*Width+x:])
	}
	return buf
}

// RestoreRect puts a buffer from SaveRect back into a page (C: fn_2096_0000).
func (s *Screen) RestoreRect(x, y, w, h int, buf []byte, page int) {
	x, y, w, h, ok := clipToScreen(x, y, w, h)
	if !ok || buf == nil {
		return
	}
	dst := s.Page(page)
	for row := range h {
		copy(dst[(y+row)*Width+x:(y+row)*Width+x+w], buf[row*w:])
	}
}
