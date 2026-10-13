package host

import "testing"

// TestSquarePixels checks that a 16:10 window fits 320x200 with square pixels, and a wider
// screen at the CRT's shape.
func TestSquarePixels(t *testing.T) {
	defer func() { SquarePixels = false }()
	const windowW, windowH = 1920, 1200
	for _, c := range []struct {
		square bool
		width  int
	}{{false, 384}, {true, ScreenWidth}} {
		SquarePixels = c.square
		if w, h, _ := fitScreen(windowW, windowH); w != c.width || h != ScreenHeight {
			t.Errorf("square %v: the window fits %dx%d, want %dx%d", c.square, w, h, c.width, ScreenHeight)
		}
	}
}
