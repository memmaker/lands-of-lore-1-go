package gfx

// TintThroughMask passes the pixels of page to under a w x h rectangle at (toX, toY)
// through table wherever the rectangle at (fromX, fromY) on page from is not 0,
// clipped to window n; with skew each row is moved right by half the rows still to
// come (C: fn_19c9_1100).
func (s *Screen) TintThroughMask(from, fromX, fromY, to, toX, toY, w, h, n int, skew bool, table []byte) {
	if w == 0 || h == 0 {
		return
	}
	win := s.Windows[n]
	left, top, right, bottom := win.Left(), win.Top(), win.Right(), win.Bottom()
	src, dst := s.Page(from), s.Page(to)
	for row := range h {
		y := toY + row
		if y < top || y >= bottom || y >= Height {
			continue
		}
		dx := 0
		if skew {
			dx = (h - row) >> 1
		}
		for col := range w {
			x := toX + col + dx
			if x < left || x >= right || x >= Width {
				continue
			}
			if src[(fromY+row)*Width+fromX+col] != 0 {
				at := y*Width + x
				dst[at] = table[dst[at]]
			}
		}
	}
}
