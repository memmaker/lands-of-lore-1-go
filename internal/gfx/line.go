package gfx

// The line clip rectangle, edges included (C: g_2efa, g_2efc, g_2efe, g_2f00).
type clipRect struct{ left, top, right, bottom int }

var defaultLineClip = clipRect{0, 0, Width - 1, Height - 1}

// SetLineClip sets the rectangle lines are clipped to, edges included (C: fn_1e30_000a).
func (s *Screen) SetLineClip(left, top, right, bottom int) {
	s.lineClip = clipRect{left, top, right, bottom}
}

// Outcode bits of a point outside the clip rectangle.
const (
	outAbove = 1 << iota
	outBelow
	outLeft
	outRight
)

func (c clipRect) outcode(x, y int) int {
	code := 0
	if y < c.top {
		code |= outAbove
	}
	if y > c.bottom {
		code |= outBelow
	}
	if x < c.left {
		code |= outLeft
	}
	if x > c.right {
		code |= outRight
	}
	return code
}

// clip moves (x, y), whose outcode is code, onto the edge the original's table picks
// for it, along the line to (ox, oy); false for outcodes the table leaves alone.
// Divisions truncate like idiv.
func (c clipRect) clip(code int, x, y, ox, oy int) (int, int, bool) {
	toRow := func(edge int) (int, int, bool) {
		return x + (ox-x)*(edge-y)/(oy-y), edge, true
	}
	toColumn := func(edge int) (int, int, bool) {
		return edge, y + (edge-x)*(oy-y)/(ox-x), true
	}
	switch code {
	case outAbove, outAbove | outRight:
		return toRow(c.top)
	case outBelow, outBelow | outLeft:
		return toRow(c.bottom)
	case outLeft, outLeft | outAbove:
		return toColumn(c.left)
	case outRight, outRight | outBelow:
		return toColumn(c.right)
	}
	return x, y, false
}

// DrawLine draws a line on the current page, clipped to the line clip rectangle
// (C: fn_1e30_0209).
func (s *Screen) DrawLine(x1, y1, x2, y2, colour int) {
	c := s.lineClip
	for {
		code1 := c.outcode(x1, y1)
		x1, y1, x2, y2 = x2, y2, x1, y1 // the original swaps the ends every pass
		code2 := c.outcode(x1, y1)
		if code1|code2 == 0 {
			break
		}
		if code1&code2 != 0 {
			return
		}
		var moved bool
		if x1, y1, moved = c.clip(code2, x1, y1, x2, y2); moved {
			continue
		}
		x2, y2, _ = c.clip(code1, x2, y2, x1, y1)
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	s.plotLine(x1, y1, x2, y2, byte(colour))
}

// plotLine draws an unclipped line from the upper end down (C: 1e30:0146).
func (s *Screen) plotLine(x1, y1, x2, y2 int, colour byte) {
	page := s.Page(s.Current)
	plot := func(at int) {
		// fix: a clip rectangle reaching past the screen wrote past the page.
		if at >= 0 && at < len(page) {
			page[at] = colour
		}
	}
	dy := y2 - y1
	if dy == 0 {
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		for x := x1; x <= x2; x++ {
			plot(y1*Width + x)
		}
		return
	}
	if dy < 0 {
		dy = -dy
		x1, x2 = x2, x1
		y1 -= dy
	}
	at := y1*Width + x1
	dx := x2 - x1
	if dx == 0 {
		for range dy + 1 {
			plot(at)
			at += Width
		}
		return
	}
	step := 1
	if dx < 0 {
		dx, step = -dx, -1
	}
	major, minor, majorStep, minorStep := dx, dy, step, Width
	if dx < dy {
		major, minor, majorStep, minorStep = dy, dx, Width, step
	}
	err := major >> 1
	for n := major; ; n-- {
		plot(at)
		if n == 0 {
			return
		}
		at += majorStep
		err -= minor
		if err < 0 {
			err += major
			at += minorStep
		}
	}
}

// DrawRectOutline draws a rectangle's outline with four lines (C: fn_1e54_0001).
func (s *Screen) DrawRectOutline(x1, y1, x2, y2, colour int) {
	s.DrawLine(x1, y1, x2, y1, colour)
	s.DrawLine(x1, y2, x2, y2, colour)
	s.DrawLine(x1, y1, x1, y2, colour)
	s.DrawLine(x2, y1, x2, y2, colour)
}
