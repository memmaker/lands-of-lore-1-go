package data

import (
	"encoding/binary"
	"fmt"
)

// A WSA file starts with seven words: frame count, x, y, width, height, largest delta
// size, flags. Then come frameCount+2 file offsets (frame i's delta runs from offset i to
// offset i+1; offset frameCount is the delta from the last frame back to the first and is
// 0 if the animation does not loop), the palette if the flags say so, and the LCW-packed
// XOR deltas. The offsets do not count the palette.
const (
	wsaHeaderSize = 14
	wsaOffsetSize = 4
)

// WSA header flags.
const (
	wsaHasPalette   = 1
	wsaXORFirstOnto = 2 // frame 0 is XORed onto the screen instead of replacing it
)

// The frame count's top bit is a flag the player never reads.
const wsaFrameCountMask = 1<<15 - 1

// The original sizes its delta buffer as largest+6 and then subtracts the size of its own
// 43-byte bookkeeping header from it, so no delta may unpack to more than largest-37 bytes.
const (
	wsaDeltaExtra    = 6
	wsaPlayerRecSize = 43
)

// Animation is an opened WSA animation (C: o2dfd, SysAnimHeaderType).
type Animation struct {
	Frames        int
	X, Y          int // where the animation sits relative to the position it is drawn at
	Width, Height int
	Palette       []byte // nil if the file has none

	file          []byte
	offsets       []int
	paletteSize   int
	xorFirstFrame bool
	hasFirstFrame bool
	loops         bool
	current       int
	delta         []byte
}

// DecodeWSA opens a WSA animation from its file contents (C: ovr_2dfd_0020).
func DecodeWSA(file []byte) (*Animation, error) {
	if len(file) < wsaHeaderSize {
		return nil, fmt.Errorf("wsa: %d bytes is too short", len(file))
	}
	word := func(i int) int { return int(binary.LittleEndian.Uint16(file[2*i:])) }
	a := &Animation{
		Frames: word(0) & wsaFrameCountMask,
		X:      int(int16(word(1))),
		Y:      int(int16(word(2))),
		Width:  word(3),
		Height: word(4),
		file:   file,
	}
	largest, flags := word(5), word(6)
	a.xorFirstFrame = flags&wsaXORFirstOnto != 0

	tableEnd := wsaHeaderSize + (a.Frames+2)*wsaOffsetSize
	if tableEnd > len(file) {
		return nil, fmt.Errorf("wsa: offset table beyond the end")
	}
	for i := 0; i < a.Frames+2; i++ {
		at := wsaHeaderSize + i*wsaOffsetSize
		a.offsets = append(a.offsets, int(binary.LittleEndian.Uint32(file[at:])))
	}
	if flags&wsaHasPalette != 0 {
		if tableEnd+PaletteSize > len(file) {
			return nil, fmt.Errorf("wsa: palette beyond the end")
		}
		a.paletteSize = PaletteSize
		a.Palette = append([]byte(nil), file[tableEnd:tableEnd+PaletteSize]...)
	}
	a.hasFirstFrame = a.offsets[0] != 0
	a.loops = a.offsets[a.Frames+1] != 0
	a.delta = make([]byte, max(0, largest+wsaDeltaExtra-wsaPlayerRecSize))
	if a.hasFirstFrame {
		a.unpack(0)
	}
	a.current = a.Frames
	return a, nil
}

// unpack decodes frame i's delta into a.delta; false if the frame has none.
func (a *Animation) unpack(i int) bool {
	start, end := a.offsets[i], a.offsets[i+1]
	if start == 0 || end <= start {
		return false
	}
	start += a.paletteSize
	end = min(end+a.paletteSize, len(a.file))
	if start >= end {
		return false
	}
	clear(a.delta)
	DecodeLCW(a.file[start:end], a.delta)
	return true
}

// step lays frame i's delta onto dst (C: ovr_2dfd_005c).
func (a *Animation) step(i int, dst []byte, stride int) {
	if a.unpack(i) {
		ApplyDelta(dst, a.delta, a.Width, stride, false)
	}
}

// Seek brings dst to frame (C: ovr_2dfd_002a without the final draw). dst holds the
// animation's top-left pixel at index 0 and its rows are stride bytes apart. onScreen says
// dst is the screen itself, where frame 0 replaces what is there; otherwise dst is the
// animation's own zeroed Width*Height buffer. Like the original it walks the shorter way
// round when the animation loops, and returns false for a frame out of range.
func (a *Animation) Seek(frame int, dst []byte, stride int, onScreen bool) bool {
	total := a.Frames
	if frame < 0 || frame >= total {
		return false
	}
	if a.current == total {
		if a.hasFirstFrame {
			ApplyDelta(dst, a.delta, a.Width, stride, onScreen && !a.xorFirstFrame)
		}
		a.current = 0
	}
	current := a.current
	distance := abs(current - frame)
	forward := true
	var steps int
	if frame > current {
		steps = total - frame + current
		if steps < distance && a.loops {
			forward = false
		} else {
			steps = distance
		}
	} else {
		steps = total - current + frame
		if steps >= distance || !a.loops {
			forward = false
			steps = distance
		}
	}
	for range steps {
		if forward {
			current++
			a.step(current, dst, stride)
			if current == total {
				current = 0
			}
		} else {
			if current == 0 {
				current = total
			}
			a.step(current, dst, stride)
			current--
		}
	}
	a.current = frame
	return true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
