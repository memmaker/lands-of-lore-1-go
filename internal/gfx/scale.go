package gfx

import "encoding/binary"

// LANDS.EXE's library blits: the intro and the character generation draw on a work page
// and show it through a copy of the screen.

const (
	viewWord      = 4    // Big_Copy_View compares a double word at a time
	scaleBuffer   = 1024 // the scaler's line buffer (C: scale_buffer)
	fractionShift = 8
)

// CopyChangedView copies from page from to page to, and to the visible page, every
// double word that differs between from and to: to is kept as a copy of the screen, so
// only what changed is drawn (C: LANDS.EXE _Big_Copy_View). What was drawn on the screen
// alone stays until the work page changes under it.
func (s *Screen) CopyChangedView(from, to int) {
	src := s.Page(from)[:Width*Height]
	shadow := s.Page(to)[:Width*Height]
	screen := s.Page(0)[:Width*Height]
	le := binary.LittleEndian
	for at := 0; at < Width*Height; at += viewWord {
		v := le.Uint32(src[at:])
		if v != le.Uint32(shadow[at:]) {
			le.PutUint32(shadow[at:], v)
			le.PutUint32(screen[at:], v)
		}
	}
}

// ScaleBlit scales the w x h block at (fromX, fromY) of page from to newW x newH at
// (toX, toY) of page to, clipped to window win, one line at a time with the same
// stepping as the original (C: LANDS.EXE _Scale_Bit_Blit, scale_line, clip_).
func (s *Screen) ScaleBlit(fromX, fromY, toX, toY, w, h, newW, newH, from, to, win int) {
	if newW == 0 || newH == 0 || w == 0 || h == 0 {
		return
	}
	wn := s.Windows[win]
	c, ok := clipScaled(toX, toY, newW, newH, wn.W*ColumnWidth, wn.H)
	if !ok {
		return
	}
	src := s.Page(from)
	dst := s.Page(to)
	line := make([]byte, scaleBuffer)
	lastRow := -1
	dstRow := (c.y + wn.Y) * Width
	// sic: the rows run from the rows cut at the top to the rows left, so a block cut at
	// the top loses rows at its bottom too
	for row := c.skipY; ; row++ {
		srcRow := row * h / newH
		if srcRow != lastRow {
			lastRow = srcRow
			at := fromX + (fromY+srcRow)*Width
			scaleLine(line, src[min(at, len(src)):], w, newW)
		}
		at := dstRow + c.x + wn.X*ColumnWidth
		if at >= 0 && at+c.w <= len(dst) {
			copy(dst[at:at+c.w], line[c.skipX:c.skipX+c.w])
		}
		dstRow += Width
		if row+1 >= c.h {
			break
		}
	}
}

// scaledClip is where a scaled block lands: its visible size and position, and how much
// of it is cut at the left and the top.
type scaledClip struct {
	x, y, w, h   int
	skipX, skipY int
}

// clipScaled clips a block to a window of size winW x winH, as clip_ does with unsigned
// compares; false when nothing is left.
func clipScaled(x, y, w, h, winW, winH int) (scaledClip, bool) {
	c := scaledClip{x: x, y: y, w: w, h: h}
	if right := x + w; right <= 0 {
		return c, false
	} else if uint16(right) <= uint16(x) { // off the left
		c.skipX = w - right
		c.w = right
		c.x = 0
	}
	if room := winW - c.x; room <= 0 {
		return c, false
	} else if room <= c.w {
		c.w = room
	}
	if bottom := y + h; bottom <= 0 {
		return c, false
	} else if uint16(bottom) <= uint16(y) { // off the top
		c.skipY = h - bottom
		c.h = bottom
		c.y = 0
	}
	if room := winH - c.y; room <= 0 {
		return c, false
	} else if room <= c.h {
		c.h = room
	}
	return c, true
}

// scaleLine scales one line of w pixels to newW in an 8.8 fixed point step: shrinking
// skips pixels, growing repeats them (C: scale_line). The buffer is filled to its end at
// most.
func scaleLine(out, src []byte, w, newW int) {
	pixel := func(i int) byte {
		if i < len(src) {
			return src[i]
		}
		return 0
	}
	put := func(o *int, c byte) {
		if *o < len(out) {
			out[*o] = c
		}
		*o++
	}
	o := 0
	switch {
	case newW == w:
		for i := range w {
			put(&o, pixel(i))
		}
	case newW < w:
		if newW == 1 {
			put(&o, pixel(0))
			return
		}
		q := ((w - newW) << fractionShift) / newW
		whole, frac := q>>fractionShift, (q&(1<<fractionShift-1))<<fractionShift
		at, acc := 0, 0
		for range newW {
			put(&o, pixel(at))
			at++
			acc += frac
			at += whole + acc>>16
			acc &= 1<<16 - 1
		}
	default:
		if w == 1 {
			for range newW {
				put(&o, pixel(0))
			}
			return
		}
		q := ((newW-w+1)<<fractionShift)/w + 1<<fractionShift
		whole, frac := q>>fractionShift, (q&(1<<fractionShift-1))<<fractionShift
		acc := 0
		for i := range w {
			acc += frac
			n := whole + acc>>16
			acc &= 1<<16 - 1
			c := pixel(i)
			for range n {
				put(&o, c)
			}
		}
	}
}
