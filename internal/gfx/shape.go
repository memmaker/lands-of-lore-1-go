package gfx

import (
	"encoding/binary"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
)

// A shape starts with a 10-byte header:
//
//	flags (2), height (1), width (2), original height (1), packed size (2), size (2)
//
// then, if the flags say so, a colour count byte and a colour table, then the pixel
// rows: LCW packed unless the flags say raw. Each row is a run of bytes where a nonzero
// byte is a pixel and 0 is followed by a count of transparent pixels.
const (
	shapeHeaderSize     = 10
	defaultShapeColours = 16
)

// Shape header flags.
const (
	shapeHasColours   = 1 << 0
	shapeRaw          = 1 << 1
	shapeColourCount  = 1 << 2
	shapeHeightOffset = 2
	shapeWidthOffset  = 3
	shapeOrigOffset   = 5
	shapeSizeOffset   = 8
	wordMask          = 1<<16 - 1
)

// Shape drawing flags (C: the flag word of fn_1fd6_000a). The options that go with a
// flag were the original's variable arguments, in this order: Colours, Fade+FadeSteps,
// Levels+LevelTables, PredatorStep, Priority, ScaleX+ScaleY, Overlay.
const (
	ShapeFlipX          = 1 << 0
	ShapeFlipY          = 1 << 1
	ShapeScaled         = 1 << 2  // ScaleX, ScaleY: 256 = full size
	ShapeWindowRelative = 1 << 4  // x and y count from the window's corner
	ShapeCentred        = 1 << 5  // x and y name the shape's centre
	ShapeFaded          = 1 << 8  // pass pixels FadeSteps times through Fade
	ShapePredator       = 1 << 9  // shimmer: show the pixels a little to the right instead
	ShapeOwnColours     = 1 << 10 // set by the shape itself when it carries a colour table
	ShapePriority       = 1 << 11 // takes Priority; draws nothing in this game's library
	ShapeLevels         = 1 << 12 // blend with what is underneath through Levels/LevelTables
	ShapeOverlay        = 1 << 13 // colour 255 shows Overlay[pixel underneath]
	ShapePredatorStep   = 1 << 14 // PredatorStep instead of 256
	ShapeRecoloured     = 1 << 15 // Colours instead of the shape's own table
)

// ShapeOptions carries the values the drawing flags ask for.
type ShapeOptions struct {
	Colours      []byte
	Fade         []byte
	FadeSteps    int
	Levels       []byte // per colour: a level (row of LevelTables), or negative to stay opaque
	LevelTables  []byte // 256 bytes per level, indexed by the pixel underneath
	PredatorStep int
	Priority     int
	ScaleX       int
	ScaleY       int
	Overlay      []byte
}

// scaleUnit is the scale factor of an unscaled shape (8.8 fixed point).
const scaleUnit = 256

// predatorOffsets are how far right each successive predator shape samples
// (C: g_35b8); every predator draw moves to the next entry.
var predatorOffsets = [...]int{1, 3, 2, 5, 4, 3, 2, 1}

type predatorState struct {
	index  int
	offset int
	acc    int
	step   int
}

// ShapeCount is the number of shapes in a shape file (C: fn_1f36_0001).
func ShapeCount(file []byte) int {
	if len(file) < 2 {
		return 0
	}
	return int(binary.LittleEndian.Uint16(file))
}

// ShapeFrame returns shape i of a shape file, nil if there is none (C: fn_1f36_0064).
// The file is a shape count and a table of offsets counted from the end of the count.
func ShapeFrame(file []byte, i int) []byte {
	if i < 0 || i >= ShapeCount(file) || 2+4*i+4 > len(file) {
		return nil
	}
	off := int(binary.LittleEndian.Uint32(file[2+4*i:])) + 2
	if off >= len(file) {
		return nil
	}
	return file[off:]
}

// ShapeWidth, ShapeHeight read a shape's header (C: fn_1f36_01e2, fn_1f36_0224).
func ShapeWidth(shape []byte) int {
	if len(shape) < shapeHeaderSize {
		return 0
	}
	return int(binary.LittleEndian.Uint16(shape[shapeWidthOffset:]))
}

func ShapeHeight(shape []byte) int {
	if len(shape) < shapeHeaderSize {
		return 0
	}
	return int(shape[shapeHeightOffset])
}

// SetShapeHeight changes how many rows of a shape are drawn and returns the old value
// (C: fn_1f36_02b0); RestoreShapeHeight puts the original back (C: fn_1f36_018e).
func SetShapeHeight(shape []byte, h int) int {
	old := ShapeHeight(shape)
	if len(shape) >= shapeHeaderSize {
		shape[shapeHeightOffset] = byte(h)
	}
	return old
}

func RestoreShapeHeight(shape []byte) int {
	old := ShapeHeight(shape)
	if len(shape) >= shapeHeaderSize {
		shape[shapeHeightOffset] = shape[shapeOrigOffset]
	}
	return old
}

// ScaledShapeWidth, ScaledShapeHeight are a shape's size at a scale (C: fn_1fd1_0024,
// fn_1fd1_003d).
func ScaledShapeWidth(shape []byte, scale int) int { return ShapeWidth(shape) * scale >> 8 & wordMask }
func ScaledShapeHeight(shape []byte, scale int) int {
	return ShapeHeight(shape) * scale >> 8 & wordMask
}

// RecolourShape passes a shape's own colour table through table (C: fn_1f24_0000).
func RecolourShape(shape, table []byte) {
	if len(shape) < shapeHeaderSize+defaultShapeColours || shape[0]&shapeHasColours == 0 {
		return
	}
	colours := shape[shapeHeaderSize : shapeHeaderSize+defaultShapeColours]
	for i, c := range colours {
		colours[i] = table[c]
	}
}

// DrawShape draws a shape onto a page, clipped to a window (C: fn_1fd6_000a). Without
// ShapeWindowRelative, x and y are screen coordinates.
func (s *Screen) DrawShape(page int, shape []byte, x, y, window, flags int, opt ShapeOptions) {
	if len(shape) < shapeHeaderSize {
		return
	}
	shapeFlags := int(binary.LittleEndian.Uint16(shape))
	if shapeFlags&shapeHasColours != 0 {
		flags |= ShapeOwnColours
	}
	if flags&ShapeFaded != 0 && opt.FadeSteps == 0 {
		flags &^= ShapeFaded
	}
	p := &s.predator
	if flags&ShapePredator != 0 {
		p.index = (p.index + 1) % len(predatorOffsets)
		p.offset = predatorOffsets[p.index]
		p.acc = 0
		p.step = scaleUnit
	}
	if flags&ShapePredatorStep != 0 {
		p.step = opt.PredatorStep
	}
	scaleX, scaleY := scaleUnit, scaleUnit
	if flags&ShapeScaled != 0 {
		scaleX, scaleY = opt.ScaleX, opt.ScaleY
	}

	win := s.Windows[window]
	if flags&ShapeWindowRelative == 0 {
		x -= win.Left()
	} else {
		y += win.Top()
	}
	width := ShapeWidth(shape)
	outH := ShapeHeight(shape)
	outW := width
	if flags&ShapeScaled != 0 {
		outH = outH * scaleY >> 8 & 0xffff
		outW = outW * scaleX >> 8 & 0xffff
		if outH == 0 || outW == 0 {
			return
		}
	}
	if flags&ShapeCentred != 0 {
		y -= outH / 2
		x -= outW / 2
	}

	pos := shapeHeaderSize
	colourCount := defaultShapeColours
	if shapeFlags&shapeColourCount != 0 {
		colourCount = int(shape[pos])
		pos++
	}
	colours := opt.Colours
	if flags&ShapeRecoloured == 0 && shapeFlags&shapeHasColours != 0 {
		colours = shape[pos:min(pos+colourCount, len(shape))]
	}
	if flags&ShapeOwnColours != 0 {
		pos += colourCount
	}
	if pos > len(shape) {
		return
	}
	pixels := shape[pos:]
	if shapeFlags&shapeRaw == 0 {
		size := int(binary.LittleEndian.Uint16(shape[shapeSizeOffset:]))
		if cap(s.scratch) < size {
			s.scratch = make([]byte, size)
		}
		s.scratch = s.scratch[:size]
		data.DecodeLCW(pixels, s.scratch)
		pixels = s.scratch
	}

	screenRow := func(o int) int {
		if flags&ShapeFlipY != 0 {
			return y + outH - 1 - o
		}
		return y + o
	}
	screenColumn := func(o int) int {
		if flags&ShapeFlipX != 0 {
			return x + outW - 1 - o
		}
		return x + o
	}
	// Nothing at all is drawn if the shape misses the window.
	if top, bottom := min(screenRow(0), screenRow(outH-1)), max(screenRow(0), screenRow(outH-1)); bottom < win.Top() || top >= win.Bottom() {
		return
	}
	if left, right := min(screenColumn(0), screenColumn(outW-1)), max(screenColumn(0), screenColumn(outW-1)); right < 0 || left >= win.W*ColumnWidth {
		return
	}

	buf := s.Page(page)
	plot := s.pixelOp(buf, flags, opt, colours)
	row := make([]byte, width)
	in := 0
	accY := 0
	for out := 0; out < outH; {
		in = decodeShapeRow(pixels, in, row)
		accY = (accY + scaleY) & 0xffff
		repeats := accY >> 8
		accY &= 0xff
		for ; repeats > 0 && out < outH; repeats-- {
			sy := screenRow(out)
			out++
			if sy < win.Top() || sy >= win.Bottom() {
				continue
			}
			base := sy*Width + win.Left()
			col, accX := 0, 0
			for _, c := range row {
				accX += scaleX
				n := accX >> 8
				accX &= 0xff
				for ; n > 0 && col < outW; n-- {
					sx := screenColumn(col)
					col++
					if c != 0 && sx >= 0 && sx < win.W*ColumnWidth {
						plot(base+sx, c)
					}
				}
			}
		}
	}
}

// decodeShapeRow unpacks the row starting at src[in] into row (0 = transparent) and
// returns where the next row starts. A transparent run past the row's end is cut off.
func decodeShapeRow(src []byte, in int, row []byte) int {
	clear(row)
	for at := 0; at < len(row); {
		if in >= len(src) {
			return in
		}
		c := src[in]
		in++
		if c != 0 {
			row[at] = c
			at++
			continue
		}
		if in >= len(src) {
			return in
		}
		at += int(src[in])
		in++
	}
	return in
}

// pixelOp returns how one shape pixel goes onto the page (C: the routines in the table
// at g_2b90, chosen by flag bits 8-13). Some combinations behave oddly in the original:
// fading together with predator darkens what is underneath instead, and overlay without
// a colour table and without fading ignores colour 255; both are kept.
func (s *Screen) pixelOp(buf []byte, flags int, opt ShapeOptions, colours []byte) func(at int, c byte) {
	p := &s.predator
	fade := func(v byte) byte {
		for range opt.FadeSteps {
			v = opt.Fade[v]
		}
		return v
	}
	blend := func(v, under byte) byte {
		level := int8(opt.Levels[v])
		if level < 0 {
			return v
		}
		return opt.LevelTables[int(level)*256+int(under)]
	}
	overlay := func(v, under byte) byte {
		if v == 255 {
			return opt.Overlay[under]
		}
		return v
	}
	// displaced says whether this pixel shows the page further right instead.
	displaced := func() bool {
		next := (p.acc + p.step) & 0xffff
		if next>>8 == 0 {
			p.acc = next
			return false
		}
		p.acc = next & 0xff
		return true
	}
	at := func(i int) byte {
		if i >= 0 && i < len(buf) {
			return buf[i]
		}
		return 0
	}
	colour := func(c byte) byte {
		if c < byte(len(colours)) || len(colours) == 256 {
			return colours[c]
		}
		return c
	}
	put := func(i int, v byte) {
		if i >= 0 && i < len(buf) {
			buf[i] = v
		}
	}
	putVisible := func(i int, v byte) {
		if v != 0 {
			put(i, v)
		}
	}

	const (
		faded     = ShapeFaded >> 8
		predator  = ShapePredator >> 8
		coloured  = ShapeOwnColours >> 8
		priority  = ShapePriority >> 8
		levels    = ShapeLevels >> 8
		overlayed = ShapeOverlay >> 8
		opMask    = 1<<6 - 1
	)
	op := flags >> 8 & opMask
	if op&priority != 0 {
		return func(int, byte) {}
	}
	// Fading with predator fades the pixel underneath, whatever else is set, except that
	// overlay still lets colour 255 through.
	if op&(faded|predator) == faded|predator {
		switch {
		case op&(overlayed|levels) == overlayed:
			return func(i int, c byte) {
				if c == 255 {
					putVisible(i, opt.Overlay[at(i)])
					return
				}
				putVisible(i, fade(at(i)))
			}
		case op&(overlayed|levels) == overlayed|levels:
			return func(i int, _ byte) { d := at(i); putVisible(i, blend(fade(d), d)) }
		default:
			return func(i int, _ byte) { putVisible(i, fade(at(i))) }
		}
	}
	source := func(c byte) byte {
		if op&coloured != 0 {
			return colour(c)
		}
		return c
	}
	switch op &^ coloured {
	case 0:
		return func(i int, c byte) { put(i, source(c)) }
	case faded:
		return func(i int, c byte) { putVisible(i, fade(source(c))) }
	case predator:
		return func(i int, c byte) {
			if displaced() {
				put(i, at(i+p.offset))
				return
			}
			put(i, source(c))
		}
	case levels:
		return func(i int, c byte) { put(i, blend(source(c), at(i))) }
	case levels | faded:
		if op&coloured == 0 {
			return func(i int, c byte) {
				if int8(opt.Levels[c]) >= 0 {
					putVisible(i, blend(c, at(i)))
					return
				}
				putVisible(i, fade(c))
			}
		}
		// fix: the original's routine left two words on the stack here and crashed.
		return func(i int, c byte) { putVisible(i, fade(blend(colour(c), at(i)))) }
	case levels | predator:
		return func(i int, c byte) {
			if displaced() {
				put(i, blend(at(i+p.offset), at(i)))
				return
			}
			if op&coloured != 0 {
				put(i, blend(colour(c), at(i)))
				return
			}
			put(i, c)
		}
	case overlayed:
		if op&coloured == 0 {
			return func(i int, c byte) { put(i, c) }
		}
		return func(i int, c byte) { put(i, overlay(colour(c), at(i))) }
	case overlayed | faded:
		return func(i int, c byte) {
			v := source(c)
			if v == 255 {
				under := opt.Overlay[at(i)]
				if op&coloured != 0 {
					putVisible(i, under)
				} else {
					put(i, under)
				}
				return
			}
			putVisible(i, fade(v))
		}
	case overlayed | predator:
		return func(i int, c byte) {
			v := source(c)
			if displaced() {
				v = at(i + p.offset)
			}
			put(i, overlay(v, at(i)))
		}
	case overlayed | levels:
		return func(i int, c byte) { put(i, blend(source(c), at(i))) }
	case overlayed | levels | faded:
		return func(i int, c byte) { putVisible(i, blend(fade(source(c)), at(i))) }
	case overlayed | levels | predator:
		return func(i int, c byte) {
			if displaced() {
				put(i, blend(at(i+p.offset), at(i)))
				return
			}
			if op&coloured != 0 {
				put(i, blend(colour(c), at(i)))
				return
			}
			put(i, c)
		}
	}
	return func(int, byte) {}
}
