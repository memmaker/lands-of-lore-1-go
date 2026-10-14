package gfx

import "image"

// The overlay is an addition of the port: a box drawn over the screen page in the frame
// the host shows only, so the game's own pages stay as the original drew them (the
// spell tooltips).
type overlay struct {
	page     []byte          // a screen-sized page the box is drawn on
	box      image.Rectangle // the part shown; empty when none
	composed []byte          // the screen page with the box over it
}

// SetOverlay shows box over the screen, its pixels drawn by draw onto a page of its own
// that stands in for page 0 meanwhile (the box keeps the drawing's coordinates).
func (s *Screen) SetOverlay(box image.Rectangle, draw func()) {
	o := &s.overlay
	if o.page == nil {
		o.page = make([]byte, Width*Height)
		o.composed = make([]byte, Width*Height)
	}
	screen := s.pages[0]
	s.pages[0] = o.page
	pg := s.SetPage(0)
	draw()
	s.SetPage(pg)
	s.pages[0] = screen
	o.box = box.Intersect(image.Rect(0, 0, Width, Height))
}

// ClearOverlay takes the overlay away.
func (s *Screen) ClearOverlay() { s.overlay.box = image.Rectangle{} }

// framePixels is the screen page with the overlay, if any, over it.
func (s *Screen) framePixels() []byte {
	o := &s.overlay
	screen := s.Page(0)
	if o.box.Empty() {
		return screen
	}
	copy(o.composed, screen[:Width*Height])
	for y := o.box.Min.Y; y < o.box.Max.Y; y++ {
		row := y * Width
		copy(o.composed[row+o.box.Min.X:row+o.box.Max.X], o.page[row+o.box.Min.X:row+o.box.Max.X])
	}
	return o.composed
}
