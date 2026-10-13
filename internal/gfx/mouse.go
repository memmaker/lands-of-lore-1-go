package gfx

import (
	"encoding/binary"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// The original drew the mouse cursor into the screen and restored what was under it; the
// port hands it to the host as an overlay instead, which looks the same and never leaves
// cursor pixels in the page. What is kept exactly is when it is shown: a hide count, and
// a region of the screen in which it is hidden while the game draws there.
type mouseState struct {
	X, Y         int // hot spot position (C: g_2f0c, g_2f0e)
	lastX, lastY int // where the move handler last saw it (C: g_2f28, g_2f2a)
	hidden       int // hide count, visible at 0 (C: g_2f16)

	shape        []byte // the cursor shape (C: g_2f06)
	pixels       []byte
	width        int
	height       int
	hotX, hotY   int
	snapX, snapY int // grid the position snaps to, 0 for none (C: g_3630, g_3632)
	snapOriginX  int // (C: g_3634)
	snapOriginY  int // (C: g_3636)

	// the conditional hide region (C: g_2f2c and g_2f2e..g_2f34)
	regionNesting int
	regionActive  bool
	regionHidIt   bool
	regionLeft    int
	regionTop     int
	regionRight   int
	regionBottom  int
}

// Where the cursor starts and how hidden (C: fn_1c24_04eb).
const (
	mouseStartX      = Width / 2
	mouseStartY      = Height / 2
	mouseStartHidden = 1
	maxHideCount     = 1<<16 - 1
	maxRegionNesting = 1<<8 - 1
)

func (m *mouseState) init() {
	m.X, m.Y = mouseStartX, mouseStartY
	m.hidden = mouseStartHidden
}

// Mouse returns the cursor's hot spot position.
func (s *Screen) Mouse() (x, y int) { return s.mouse.X, s.mouse.Y }

// MouseIn says whether the cursor's hot spot lies in a rectangle, edges included
// (C: fn_1c24_05e3).
func (s *Screen) MouseIn(x1, y1, x2, y2 int) bool {
	m := &s.mouse
	return m.X >= x1 && m.X <= x2 && m.Y >= y1 && m.Y <= y2
}

// SetMouseSnap makes the cursor's position snap to a grid (0: free) laid from an origin.
func (s *Screen) SetMouseSnap(stepX, stepY, originX, originY int) {
	m := &s.mouse
	m.snapX, m.snapY, m.snapOriginX, m.snapOriginY = stepX, stepY, originX, originY
}

// HideMouse hides the cursor; calls nest (C: fn_1ebc_000c, fn_1ebc_00e7).
func (s *Screen) HideMouse() {
	if s.mouse.hidden < maxHideCount {
		s.mouse.hidden++
	}
}

// ShowMouse undoes one HideMouse (C: fn_1ebc_0053, fn_1ebc_00fe).
func (s *Screen) ShowMouse() {
	if s.mouse.hidden > 0 {
		s.mouse.hidden--
	}
}

// HideMouseIn hides the cursor while it would overlap the rectangle (x1, y1)-(x2, y2),
// now or when it moves there, until the matching ShowMouseIn; nested calls narrow the
// region to the overlap (C: fn_1ebc_0111).
func (s *Screen) HideMouseIn(x1, y1, x2, y2 int) {
	m := &s.mouse
	left := max(x1-(m.width-m.hotX), 0)
	top := max(y1-(m.height-m.hotY), 0)
	right := min(x2+m.hotX, Width-1)
	bottom := min(y2+m.hotY, Height-1)
	if !m.regionActive && m.regionNesting == 0 {
		m.regionLeft, m.regionTop, m.regionRight, m.regionBottom = left, top, right, bottom
	}
	m.regionLeft = max(m.regionLeft, left)
	m.regionTop = max(m.regionTop, top)
	m.regionRight = min(m.regionRight, right)
	m.regionBottom = min(m.regionBottom, bottom)
	if !m.regionHidIt && m.inRegion() {
		s.HideMouse()
		m.regionHidIt = true
	}
	m.regionActive = true
	if m.regionNesting < maxRegionNesting {
		m.regionNesting++
	}
}

// ShowMouseIn undoes one HideMouseIn (C: fn_1ebc_01ec).
func (s *Screen) ShowMouseIn() {
	m := &s.mouse
	if m.regionNesting == 0 {
		return
	}
	m.regionNesting--
	if m.regionNesting != 0 {
		return
	}
	if m.regionHidIt {
		s.ShowMouse()
	}
	m.regionActive, m.regionHidIt = false, false
}

func (m *mouseState) inRegion() bool {
	return m.X >= m.regionLeft && m.X <= m.regionRight && m.Y >= m.regionTop && m.Y <= m.regionBottom
}

// MouseMoved takes the host's mouse position, as the original's mouse interrupt handler
// did (C: fn_1c24_034f): snapping, then hiding the cursor if it entered the region the
// game is drawing in. The host calls it every tick.
func (s *Screen) MouseMoved(x, y int) {
	m := &s.mouse
	x = min(x, Width-1)
	if m.snapX != 0 {
		x = (x-m.snapOriginX)/m.snapX*m.snapX + m.snapOriginX
	}
	if m.snapY != 0 {
		y = (y-m.snapOriginY)/m.snapY*m.snapY + m.snapOriginY
	}
	m.X, m.Y = x, y
	if m.hidden == 0 && (x != m.lastX || y != m.lastY) &&
		m.regionActive && !m.regionHidIt && m.inRegion() {
		s.HideMouse()
		m.regionHidIt = true
	}
	m.lastX, m.lastY = x, y
}

// SetMouseShape makes a shape the cursor with its hot spot at (hotX, hotY) and returns
// the previous shape (C: fn_1e9f_0002).
func (s *Screen) SetMouseShape(hotX, hotY int, shape []byte) []byte {
	m := &s.mouse
	old := m.shape
	if shape == nil {
		return old
	}
	m.pixels, m.width, m.height = shapePixels(shape)
	m.hotX, m.hotY = hotX, hotY
	m.shape = shape
	return old
}

// shapePixels unpacks a shape into a picture, 0 transparent, the way the original drew
// the cursor: DrawShape with no flags, so no colour mapping and the pixel data read
// straight after the colour count byte (cursor shapes carry no colour table).
func shapePixels(shape []byte) (pixels []byte, w, h int) {
	if len(shape) < shapeHeaderSize {
		return nil, 0, 0
	}
	w, h = ShapeWidth(shape), ShapeHeight(shape)
	flags := int(binary.LittleEndian.Uint16(shape))
	pos := shapeHeaderSize
	if flags&shapeColourCount != 0 {
		pos++
	}
	src := shape[min(pos, len(shape)):]
	if flags&shapeRaw == 0 {
		unpacked := make([]byte, binary.LittleEndian.Uint16(shape[shapeSizeOffset:]))
		data.DecodeLCW(src, unpacked)
		src = unpacked
	}
	pixels = make([]byte, w*h)
	in := 0
	for y := range h {
		in = decodeShapeRow(src, in, pixels[y*w:(y+1)*w])
	}
	return pixels, w, h
}

// Frame is the picture for the host: the screen page, the palette and the cursor.
func (s *Screen) Frame() *host.Frame {
	m := &s.mouse
	return &host.Frame{
		Pixels:        s.Page(0),
		Palette:       (*[data.PaletteSize]byte)(&s.Palette),
		Pointer:       m.pixels,
		PointerWidth:  m.width,
		PointerHeight: m.height,
		PointerHotX:   m.hotX,
		PointerHotY:   m.hotY,
		PointerX:      m.X,
		PointerY:      m.Y,
		PointerShown:  m.hidden == 0 && m.pixels != nil,
	}
}
