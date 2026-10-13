package game

import "github.com/memmaker/lands-of-lore-1-go/internal/clock"

// barDef is one of a portrait's two bars (C: struct BarDef, g_2256).
type barDef struct {
	x          int // offset from the portrait's left edge
	colour     int
	textColour int
	warns      bool // turns yellow below half and red below a quarter
	label      int  // string id
}

var (
	hitPointBar  = barDef{x: 33, colour: 162, textColour: 160, warns: false, label: 0x4253}
	magicBar     = barDef{x: 39, colour: 154, textColour: 152, warns: true, label: 0x4254}
	portraitBars = [2]barDef{magicBar, hitPointBar} // indexed by "not magic"
)

// Bar geometry and colours (C: fn_0a21_06fd, fn_0845_00ac).
const (
	barBottom     = 175
	barWidth      = 5
	barHeight     = 32
	barLabelY     = 144
	barBackground = 1
	barBorder     = 1
	barHalfColour = 144
	barLowColour  = 132
	barScaleOne   = 1 << 13 // the bar is drawn in 1/32 steps of the maximum
	barScaleShift = 8
	barStep       = 2
)

// How setPoints combines the value with the current one.
const (
	pointsSet = iota
	pointsAdd
	pointsAddToMax
)

// updateNoPortraits keeps the portrait area from being redrawn (C: g_7fd6 bit 1).
const updateNoPortraits = 1 << 1

// drawBar draws a vertical bar growing upwards from (x, y): value of max filled in
// colour, the rest in background, and a second value as an outline (C: fn_0845_00ac).
func (g *Game) drawBar(x, y, value, second, maximum, w, h, colour, background int, warns bool) {
	s := g.Screen
	w--
	h--
	if maximum < 1 {
		return
	}
	value = min(max(value, 0), maximum)
	length := h * value / maximum
	if length <= 0 && value > 0 {
		length = 1
	}
	s.DrawLine(x-1, y-h, x-1, y, barBorder)
	if warns {
		if maximum>>1 > value {
			colour = barHalfColour
		}
		if maximum>>2 > value {
			colour = barLowColour
		}
	}
	if length > 0 {
		s.FillRect(x, y-length, x+w, y, colour)
	}
	if length < h {
		s.FillRect(x, y-h, x+w, y-length, background)
	}
	if second > 0 && second <= maximum {
		length = max(h*second/maximum, 1)
		s.DrawRectOutline(x, y-length, x+w, y, colour-2)
	}
}

// setPoints changes a character's hit points (magic false) or magic points and moves the
// bar to the new value two steps at a time, overshooting by a step and settling back
// (C: fn_0a21_06fd).
func (g *Game) setPoints(n int, magic bool, value, mode int) {
	c := &g.party[n]
	if c.Flags&charPresent == 0 {
		return
	}
	old, maximum, bar := c.HP, c.MaxHP, portraitBars[1]
	if magic {
		old, maximum, bar = c.MP, c.MaxMP, portraitBars[0]
	}
	var v int
	switch mode {
	case pointsSet:
		v = value
	case pointsAdd:
		v = old + value
	case pointsAddToMax:
		v = maximum + value
	}
	v = max(min(v, maximum), 0)
	if magic {
		c.MP = v
	} else {
		c.HP = v
		if c.HP < 1 {
			c.Flags |= charDead
		}
	}
	if g.updateFlags&updateNoPortraits != 0 {
		return
	}
	s := g.Screen
	font := g.Text.SetFont(g.SmallFont)
	page := s.SetPage(0)
	s.HideMouse()
	// fix: a maximum of 0 divided by zero in the original.
	scale := barScaleOne / max(maximum, 1)
	maximum = scale * maximum >> barScaleShift
	old = scale * old >> barScaleShift
	v = scale * v >> barScaleShift
	end := max(min(v, maximum), 0)
	if v <= old {
		v -= barStep
	} else {
		v += barStep
	}
	v = max(min(v, maximum), 0)
	if v != old {
		step := barStep
		if v < old {
			step = -barStep
		}
		x := bar.x + g.portraitX[n]
		for cur := old; cur != v || v != end; {
			if abs(cur-v) < abs(step) {
				step >>= 1
			}
			cur += step
			g.drawBar(x, barBottom, cur, 0, maximum, barWidth, barHeight, bar.colour, barBackground, bar.warns)
			s.DrawText(g.text(bar.label), x, barLabelY, bar.textColour, 0)
			if cur == v {
				v = end
				step = -step
			}
			clock.WaitCountdown(1)
		}
	}
	s.ShowMouse()
	g.Text.SetFont(font)
	s.SetPage(page)
}
