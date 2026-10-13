package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
)

// The view's rectangle as the whole-view copies give it.
const (
	viewLeftX   = 112
	viewCopyW   = 176
	viewCopyH   = 120
	shakeFrame  = 2 // ticks a shaken frame shows
	pageShake   = 6
	shakeRedraw = 1
)

// shakeScreen shakes the view for ticks by up to dx and dy pixels; with restore the
// view is put back after (C: ovr_2d28_0034). The mouse sensitivity call of the
// original (int 33h 5330h) has nothing to do here.
func (g *Game) shakeScreen(ticks, dx, dy, restore int) {
	s := g.Screen
	g.hideMouseInWindow(viewWindow)
	s.CopyRect(viewLeftX, 0, viewLeftX, 0, viewCopyW, viewCopyH, pageScreen, pageShake)
	end := clock.Ticks() + ticks
	for clock.Ticks() < end {
		clock.Set(clock.Countdown, shakeFrame)
		x, y := shakeOffset(dx), shakeOffset(dy)
		fx, tx, fy, ty := viewLeftX, viewLeftX+x, 0, y
		if x < 0 {
			fx, tx = viewLeftX-x, viewLeftX
		}
		if y < 0 {
			fy, ty = -y, 0
		}
		s.CopyRect(fx, fy, tx, ty, viewCopyW-abs(x), viewCopyH-abs(y), pageShake, pageScreen)
		for clock.Get(clock.Countdown) != 0 {
			host.Yield()
		}
	}
	if restore == shakeRedraw {
		s.CopyRect(viewLeftX, 0, viewLeftX, 0, viewCopyW, viewCopyH, pageShake, pageScreen)
		g.copyViewFrame()
	}
	s.ShowMouseIn()
}

// shakeOffset is a random offset from -n to n-1, 0 for no n (C: o2d28's RND).
func shakeOffset(n int) int {
	if n == 0 {
		return 0
	}
	return random.Byte()%(n*2) - n
}
