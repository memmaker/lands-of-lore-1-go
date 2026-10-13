package gfx

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// Palette is 256 colours of red, green, blue, each 0-63.
type Palette [data.PaletteSize]byte

// paletteChunk is how many bytes the original sent to the DAC per vertical retrace.
const paletteChunk = 258

// fixedPoint is 1.0 in the 8.8 fixed point fades count in.
const fixedPoint = 256

const maxFadeStep = 1<<15 - 1

// SetPalette makes pal the screen's colours (C: fn_1ee4_0002). Like the original, which
// sent only the changed part and waited for a vertical retrace before each 258 bytes, it
// lets a host tick pass per chunk.
func (s *Screen) SetPalette(pal []byte) {
	at := 0
	for at < len(s.Palette) {
		for at < len(s.Palette) && at < len(pal) && s.Palette[at] == pal[at] {
			at++
		}
		if at >= len(s.Palette) || at >= len(pal) {
			return
		}
		at -= at % 3 // the DAC is written whole colours at a time
		end := min(at+paletteChunk, len(s.Palette), len(pal))
		host.Yield()
		copy(s.Palette[at:end], pal[at:end])
		at = end
	}
}

// SetPaletteColour changes one colour of pal and shows pal (C: fn_1ef2_002f).
func (s *Screen) SetPaletteColour(pal []byte, colour int, rgb []byte) {
	if pal == nil {
		return
	}
	copy(pal[colour*3:colour*3+3], rgb)
	s.SetPalette(pal)
}

// FadePalette moves the screen's colours to target over speed ticks, running idle while
// it waits (C: fn_1ef2_0075).
func (s *Screen) FadePalette(target []byte, speed int, idle func()) {
	if target == nil {
		return
	}
	step, perStep := s.fadeRate(target, speed)
	fraction := 0
	until := clock.Ticks()
	for {
		fraction += step
		until += fraction / fixedPoint
		fraction %= fixedPoint
		changed := s.FadeStep(target, perStep)
		if changed {
			clock.WaitUntil(until, idle)
		}
		if idle != nil {
			idle()
		}
		if !changed {
			return
		}
	}
}

// fadeRate works out a fade's ticks per step (8.8) and how far each step moves a colour
// component, from the largest difference to the target (C: fn_1ef2_012e).
func (s *Screen) fadeRate(target []byte, speed int) (step, perStep int) {
	largest := 0
	for i, c := range s.Palette {
		largest = max(largest, abs(int(target[i])-int(c)))
	}
	q := speed * fixedPoint
	if largest != 0 {
		q = min(q/largest, maxFadeStep)
	}
	step = q
	unit := step
	for perStep = 1; perStep <= largest && step < fixedPoint; perStep++ {
		step += unit
	}
	return step, perStep
}

// FadeStep moves every colour component up to amount towards target and shows the
// result; false if nothing was left to change (C: fn_1ef2_01f6).
func (s *Screen) FadeStep(target []byte, amount int) bool {
	next := s.Palette
	changed := false
	for i, c := range next {
		t := int(target[i])
		v := int(c)
		if v == t {
			continue
		}
		changed = true
		if v < t {
			v = min(v+amount, t)
		} else {
			v = max(v-amount, t)
		}
		next[i] = byte(v)
	}
	if changed {
		s.SetPalette(next[:])
	}
	return changed
}

// StepColourTowards moves colour by one level per component towards colour target, both
// in pal; false if they were already equal (C: fn_1ee4_0097).
func StepColourTowards(pal []byte, colour, target int) bool {
	changed := false
	for k := range 3 {
		c, t := &pal[colour*3+k], pal[target*3+k]
		switch {
		case *c < t:
			*c++
			changed = true
		case *c > t:
			*c--
			changed = true
		}
	}
	return changed
}

// SetFontPalette uses a palette's first 16 bytes as the font colour map (C: fn_1ef2_0005).
func (s *Screen) SetFontPalette(colours []byte) { s.SetFontColours(colours, 0, nibbleMask) }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
