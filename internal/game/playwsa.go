package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// wsaOverScreen is the mode the effect animations are drawn in: through the level's
// blending tables (C: 0x5000).
const wsaOverScreen = 5 << wsaModeShift

// paletteFade moves the screen's palette from one to another while an animation plays.
type paletteFade struct {
	from, to []byte
	ticks    int
}

// playWSARange plays frames start to end (backwards if end is smaller) of an animation
// over the screen at (x, y), each frame delay ticks, drawn over the screen kept in
// pageBackup2; with fade the palette moves meanwhile, and with restore the screen is
// put back after. Without w only the fade runs, until it is done (C: fn_0845_0d5e).
func (g *Game) playWSARange(w *wsaPlayer, start, end, delay, x, y int, fade *paletteFade, restore bool) {
	s := g.Screen
	var width, height int
	if w != nil {
		width, height = w.width(), w.height()
	}
	px, py, pw, ph := x, y, width, height
	if w != nil {
		if x < 0 {
			px = 0
			pw += px // sic: adds the clipped 0, the width stays
		}
		if y < 0 {
			py = 0
			ph += py
		}
	}
	forward := start <= end
	frame := start
	t0 := clock.Ticks()
	for done := false; !done; {
		clock.Set(clock.Countdown, delay)
		if w != nil {
			copy(s.Page(pageBuild), g.pageBackup2[:])
			g.drawWSA(w, frame%w.frames(), x, y, pageBuild, wsaOverScreen, g.lvl.tlcTable, g.lvl.tlcLevels)
			s.HideMouseIn(px, py, px+pw, py+ph)
			s.CopyRect(px, py, px, py, pw, ph, pageBuild, pageScreen)
			s.ShowMouseIn()
		}
		for {
			if fade != nil {
				if !g.fadeStep(fade, clock.Ticks()-t0) && w == nil {
					return
				}
			}
			if clock.Get(clock.Countdown) == 0 {
				break
			}
			host.Yield()
		}
		if w != nil {
			if forward {
				frame++
				done = frame >= end
			} else {
				frame--
				done = frame < end
			}
		}
	}
	if restore && w != nil {
		copy(s.Page(pageBuild), g.pageBackup2[:])
		s.HideMouseIn(px, py, px+pw, py+ph)
		s.CopyRect(px, py, px, py, px+pw, py+ph, pageBuild, pageScreen) // sic: the corner's coordinates are added to the size
		s.ShowMouseIn()
	}
}

// fadeStep sets the palette t ticks into a fade; once over it keeps the target as the
// level palette. It says whether any colour still moves (C: ovr_2d1b_003e).
func (g *Game) fadeStep(f *paletteFade, t int) bool {
	const sixBits = 1<<6 - 1
	var pal [len(g.darkPalette)]byte
	changed := false
	for i := range pal {
		if t < f.ticks {
			d := int(int8(f.to[i]&sixBits - f.from[i]&sixBits))
			if d != 0 {
				changed = true
			}
			pal[i] = byte(int(f.from[i]&sixBits) + (d<<8)/f.ticks*t>>8)
		} else {
			pal[i] = f.to[i] & sixBits
			g.darkPalette[i] = pal[i]
			changed = false
		}
	}
	g.Screen.SetPalette(pal[:])
	return changed
}
