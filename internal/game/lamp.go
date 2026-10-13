package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// The lamp: its icon shows how bright it burns, in pairs of frames it flickers between;
// with no light or no oil it is out (C: fn_0bbc_07ef).
const (
	lampOut          = 8
	noLampLevel      = -1 // no icon drawn yet: draw it and set the brightness
	lampFullEffect   = 100
	lampEffectStep   = 25
	lampBrightest    = 3
	lampFrameBit     = 1 << 0
	lampFlickerMin   = 10
	lampFlickerSides = 30
	lampIconX        = 291
	lampIconY        = 56
	lampIconRight    = 320
	lampIconBottom   = 80
	gameShapeLamp    = 35
)

func (g *Game) updateLamp() {
	if g.updateFlags&updateGameMenu != 0 || !g.flag(flagHasLamp) {
		return
	}
	var frame int
	if g.brightness == 0 || g.lampEffect == 0 {
		frame = lampOut
		if g.lampLevel != frame && g.paletteState == paletteNormal {
			g.setBrightness(g.Palette[:], g.brightness, frame)
		}
	} else {
		effect := min(g.lampEffect, lampFullEffect)
		frame = (lampBrightest - (effect-1)/lampEffectStep) * 2
		flicker := false
		switch {
		case g.lampLevel == noLampLevel:
			if g.paletteState == paletteNormal {
				g.setBrightness(g.Palette[:], g.brightness, frame)
			}
			flicker = true
		case g.lampLevel&^lampFrameBit == frame&^lampFrameBit:
			if clock.Ticks() > g.lampFlickerAt {
				frame = g.lampLevel ^ lampFrameBit
				flicker = true
			} else {
				frame = g.lampLevel
			}
		case g.paletteState == paletteNormal:
			g.setBrightness(g.Palette[:], g.brightness, frame)
		}
		if flicker {
			g.lampFlickerAt = clock.Ticks() + dice(1, lampFlickerSides) + lampFlickerMin
		}
	}
	if g.lampLevel != frame {
		s := g.Screen
		s.HideMouseIn(lampIconX, lampIconY, lampIconRight, lampIconBottom)
		s.DrawShape(s.Current, g.gameShapes[gameShapeLamp+frame], lampIconX, lampIconY, 0, 0, gfx.ShapeOptions{})
		s.ShowMouseIn()
		g.lampLevel = frame
	}
}
