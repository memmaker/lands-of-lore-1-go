// Package gfx is the Westwood graphics library the game is built on: 320x200 pages of
// 256-colour pixels, windows, rectangle copies and fills, the shape and buffer blitters,
// fonts, palettes and the mouse cursor.
package gfx

// Screen size in pixels.
const (
	Width  = 320
	Height = 200
)

// ColumnWidth is the unit windows measure x and width in.
const ColumnWidth = 8

// Pages are numbered 0-15 in pairs that share one buffer: pages 0 and 1 are the visible
// screen, 2/3, 4/5 and 6/7 are buffers in memory (C: fn_215f_001d / fn_215f_0003, both of
// which give the same buffer since ovr_2df7_0020 sets up the pair alike).
const (
	pageCount = 16
	pageMask  = pageCount - 1
)

// The page buffer sizes ovr_2db0_0020 sets up: a screen each, a bit more for the pages
// the game also loads files into, and a 320x120 buffer. Pages 8/9 are LANDS.EXE's work
// screen, which the intro and the character generation draw on (C: LANDS.EXE's
// _PageArray); MAIN.EXE has none there.
var pageSizes = [pageCount / 2]int{Width * Height, 64500, 65535, Width * 120, Width * Height}

// Window is one entry of the window table (C: g_1f3b, 16 bytes each). X and W are in
// 8-pixel columns.
type Window struct {
	X, Y, W, H             int
	Foreground, Background int
	Column, Row            int // text cursor
}

// Left, Top, Right, Bottom give the window's edges in pixels; Right and Bottom are
// exclusive.
func (w Window) Left() int   { return w.X * ColumnWidth }
func (w Window) Top() int    { return w.Y }
func (w Window) Right() int  { return (w.X + w.W) * ColumnWidth }
func (w Window) Bottom() int { return w.Y + w.H }

// windowCount is how many windows the game's table holds.
const windowCount = 19

// initialWindows is the game's window table as linked (C: g_1f3b in s005).
var initialWindows = [windowCount]Window{
	{0, 0, 40, 200, 254, 1, 0, 0},
	{8, 72, 24, 56, 254, 1, 0, 0},
	{14, 0, 22, 120, 254, 1, 0, 0},
	{11, 123, 28, 18, 254, 252, 0, 0},
	{11, 123, 28, 45, 254, 252, 0, 0},
	{85, 123, 233, 55, 254, 252, 0, 0},
	{11, 140, 16, 42, 61, 1, 0, 0},
	{4, 89, 32, 60, 0, 0, 0, 0},
	{5, 110, 30, 12, 254, 1, 0, 0},
	{7, 25, 26, 151, 0, 0, 0, 0},
	{3, 30, 34, 140, 0, 0, 0, 0},
	{2, 72, 36, 52, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{14, 0, 22, 120, 254, 1, 0, 0},
	{13, 162, 24, 12, 254, 1, 0, 0},
	{15, 6, 20, 110, 1, 0, 0, 0},
	{26, 190, 10, 7, 254, 1, 0, 0},
	{7, 33, 26, 133, 0, 0, 0, 0},
	{3, 50, 34, 98, 0, 0, 0, 0},
}

// Screen holds the pages and the drawing state shared by the library's routines.
type Screen struct {
	pages   [pageCount / 2][]byte
	Current int // the page drawing goes to (C: g_3649)
	Windows [windowCount]Window
	Palette Palette // the colours on screen (C: g_2fac)

	Font        *Font
	CharSpacing int // extra pixels after each character (C: g_3624)
	LineSpacing int // extra pixels between lines (C: g_3626)
	fontColours [fontColours]byte

	predator predatorState
	scratch  []byte // unpacked shape data (C: the buffer at g_35ac)
	mouse    mouseState
	lineClip clipRect
	overlay  overlay
}

// NewScreen allocates the pages and the window table.
func NewScreen() *Screen {
	s := &Screen{}
	for i, size := range pageSizes {
		s.pages[i] = make([]byte, size)
	}
	s.Windows = initialWindows
	for i := range s.fontColours {
		s.fontColours[i] = byte(i)
	}
	s.predator.offset = predatorOffsets[0]
	s.mouse.init()
	s.lineClip = defaultLineClip
	return s
}

// Page returns the buffer behind page number n.
func (s *Screen) Page(n int) []byte { return s.pages[(n&pageMask)/2] }

// SetPage makes n the current page and returns the previous one (C: fn_1e74_000f).
func (s *Screen) SetPage(n int) int {
	old := s.Current
	s.Current = n
	return old
}

// SetWindow redefines window n (C: fn_1e9a_000b; the text layer re-reads its window
// itself when n is the current text window).
func (s *Screen) SetWindow(n, x, y, w, h int) {
	win := &s.Windows[n]
	win.X, win.Y, win.W, win.H = x, y, w, h
}

// ClearPage fills a page's screen area with colour 0 (C: fn_1e04_0004).
func (s *Screen) ClearPage(n int) {
	clear(s.Page(n)[:Width*Height])
}

// CopyPage copies a whole screen from one page to another unless both are the same
// buffer (C: fn_1dff_0008).
func (s *Screen) CopyPage(from, to int) {
	if (from&pageMask)/2 == (to&pageMask)/2 {
		return
	}
	copy(s.Page(to)[:Width*Height], s.Page(from)[:Width*Height])
}

// Pixel reads a pixel of the current page, 0 off screen (C: fn_1e6f_000a).
func (s *Screen) Pixel(x, y int) int {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return 0
	}
	return int(s.Page(s.Current)[y*Width+x])
}

// SetPixel writes a pixel of the current page if it is on screen (C: fn_1f20_0000).
func (s *Screen) SetPixel(x, y, colour int) {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return
	}
	s.Page(s.Current)[y*Width+x] = byte(colour)
}

// FillRect fills the rectangle between two corners (inclusive, in either order) on the
// current page; nothing is drawn if a corner is off screen (C: fn_1e59_001d).
func (s *Screen) FillRect(x1, y1, x2, y2, colour int) {
	if !onScreen(x1, y1) || !onScreen(x2, y2) {
		return
	}
	x1, x2 = min(x1, x2), max(x1, x2)
	y1, y2 = min(y1, y2), max(y1, y2)
	page := s.Page(s.Current)
	for y := y1; y <= y2; y++ {
		row := page[y*Width+x1 : y*Width+x2+1]
		for i := range row {
			row[i] = byte(colour)
		}
	}
}

func onScreen(x, y int) bool { return x >= 0 && x < Width && y >= 0 && y < Height }

// RemapRect passes a rectangle of a page through a colour table (C: fn_1f24_003f).
func (s *Screen) RemapRect(x, y, w, h, page int, table []byte) {
	if table == nil {
		return
	}
	buf := s.Page(page)
	for row := y; row < y+h; row++ {
		for at := row*Width + x; at < row*Width+x+w; at++ {
			if at >= 0 && at < len(buf) {
				buf[at] = table[buf[at]]
			}
		}
	}
}

// CopyRect copies a w x h rectangle from (fromX, fromY) on page from to (toX, toY) on
// page to, clipped to the screen (C: fn_1dbc_0004). Overlapping copies on one page work.
func (s *Screen) CopyRect(fromX, fromY, toX, toY, w, h, from, to int) {
	s.copyRect(fromX, fromY, toX, toY, w, h, from, to, false)
}

// CopyRectColumns is CopyRect with x and w in 8-pixel columns (C: fn_1dbc_0017).
func (s *Screen) CopyRectColumns(fromX, fromY, toX, toY, w, h, from, to int) {
	s.copyRect(fromX*ColumnWidth, fromY, toX*ColumnWidth, toY, w*ColumnWidth, h, from, to, false)
}

// CopyRectTransparent is CopyRect that leaves colour 0 out (C: fn_1dbc_0036).
func (s *Screen) CopyRectTransparent(fromX, fromY, toX, toY, w, h, from, to int) {
	s.copyRect(fromX, fromY, toX, toY, w, h, from, to, true)
}

func (s *Screen) copyRect(fromX, fromY, toX, toY, w, h, from, to int, transparent bool) {
	if fromX >= Width || fromY >= Height || toX >= Width || toY >= Height {
		return
	}
	// A negative corner moves both corners in and shrinks the rectangle.
	if fromX < 0 {
		toX -= fromX
		w += fromX
		fromX = 0
	}
	if fromY < 0 {
		toY -= fromY
		h += fromY
		fromY = 0
	}
	if toX < 0 {
		fromX -= toX
		w += toX
		toX = 0
	}
	if toY < 0 {
		fromY -= toY
		h += toY
		toY = 0
	}
	w = min(w, Width-fromX, Width-toX)
	h = min(h, Height-fromY, Height-toY)
	if fromX >= Width || toX >= Width || fromY >= Height || toY >= Height ||
		w <= 0 || h <= 0 {
		return
	}
	src, dst := s.Page(from), s.Page(to)
	copyRow := func(y int) {
		a := src[(fromY+y)*Width+fromX : (fromY+y)*Width+fromX+w]
		b := dst[(toY+y)*Width+toX : (toY+y)*Width+toX+w]
		if !transparent {
			copy(b, a)
			return
		}
		for i, c := range a {
			if c != 0 {
				b[i] = c
			}
		}
	}
	// The same buffer with the target further on is copied from the end backwards.
	backwards := &src[0] == &dst[0] && toY*Width+toX > fromY*Width+fromX
	if &src[0] == &dst[0] && toY*Width+toX == fromY*Width+fromX {
		return
	}
	if backwards {
		for y := h - 1; y >= 0; y-- {
			copyRowBackwards(src, dst, (fromY+y)*Width+fromX, (toY+y)*Width+toX, w, transparent)
		}
		return
	}
	for y := range h {
		copyRow(y)
	}
}

// copyRowBackwards copies one row from its last pixel to its first, as the original does
// for an overlapping copy to a later position.
func copyRowBackwards(src, dst []byte, from, to, w int, transparent bool) {
	for i := w - 1; i >= 0; i-- {
		c := src[from+i]
		if !transparent || c != 0 {
			dst[to+i] = c
		}
	}
}
