package game

import (
	"errors"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// The healing sparkles over the portraits (C: o2d18).
const (
	healShapesFile   = "heal.shp"
	healGlowFile     = "heali.shp"
	healShapes       = 4
	healGlows        = 16
	healSteps        = 16
	healStepTicks    = 4
	healSound        = 68
	healBoxW         = 77
	healBoxH         = 44
	healBoxDX        = -6
	healBoxY         = 138
	healSaveY        = 32
	healGlowSize     = 32
	healGlowDX       = 7
	healGlowDY       = 6
	healTintMul      = 52 // the glow brightens what is under it (C: ovr_2ce1_0043's arguments)
	healTintRed      = 22
	healTintGreen    = 20
	healTintColours  = 256
	healAllParty     = -1
	healCuresPoison  = 1 // kinds 2 and up also end the poison
	healSmall        = 25
	healMedium       = 45
	healUnlimited    = 10000
	healFractionBits = 8
)

// The sparkle frames of each step for the kinds of healing (C: f_25fd_0000-0040).
var (
	healSparklesSmall = [healSteps]int{0, 0, 0, 1, 1, 1, 2, 2, 2, 2, 1, 1, 1}
	healSparklesLarge = [healSteps]int{0, 0, 1, 1, 2, 2, 3, 3, 3, 3, 2, 2, 1, 1}
	healGlowSmall     = [healSteps]int{0, 0, 0, 1, 2, 3, 4, 5, 5, 4, 3, 2, 1}
	healGlowMedium    = [healSteps]int{0, 1, 2, 3, 4, 5, 6, 7, 7, 6, 5, 4, 3, 2, 1}
	healGlowLarge     = [healSteps]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
)

// loadHealShapes loads the sparkles and builds the glow's table (C: ovr_2d18_0020).
func (g *Game) loadHealShapes() bool {
	if g.healSparkles == nil {
		var err error
		if g.healSparkles, err = g.loadShapesUpTo(healShapesFile, healShapes); err != nil {
			return false
		}
		if g.healGlows, err = g.loadShapesUpTo(healGlowFile, healGlows); err != nil {
			g.healSparkles = nil
			return false
		}
	}
	g.healTint = tintedTable(g.darkPalette[:], healTintMul, healTintRed, healTintGreen, 0, healTintColours, true)
	return true
}

// freeHealShapes (C: ovr_2d18_0025).
func (g *Game) freeHealShapes() { g.healSparkles, g.healGlows = nil, nil }

// healAnimation heals character who, or with -1 the party, while sparkles play over
// the portraits; kind picks the amount: 0 and 1 a little, 2 and 3 all and the poison,
// more than 3 that much and the poison (C: ovr_2d18_002a).
func (g *Game) healAnimation(who, kind int) {
	opened := false
	if g.healSparkles == nil {
		if !g.loadHealShapes() {
			g.scriptErr = errors.New("Process_Heal_FX: Can't load _Heal FX") // C: ovr_2ce1_007f, the fatal exit
			return
		}
		opened = true
	}
	sparkles, glows, cures, most := healSparklesLarge[:], healGlowLarge[:], true, kind
	switch kind {
	case 0:
		sparkles, glows, cures, most = healSparklesSmall[:], healGlowSmall[:], false, healSmall
	case 1:
		glows, cures, most = healGlowMedium[:], false, healMedium
	case 2, 3:
		most = healUnlimited
	}
	first, end := who, who+1
	if who == healAllParty {
		first, end = 0, partySize
	}
	s := g.Screen
	var amount, x, y, fraction [partySize]int // fix: the original began with whatever the stack held in the fractions
	for ch := first; ch < end; ch++ {
		c := &g.party[ch]
		if c.Flags&charPresent == 0 {
			continue
		}
		x[ch], y[ch] = g.portraitX[ch]+healBoxDX, healBoxY
		c.DamageSuffered = 0
		amount[ch] = min(c.MaxHP-c.HP, most)
		s.HideMouseIn(x[ch], y[ch], x[ch]+healBoxW, y[ch]+healBoxH)
		s.CopyRect(x[ch], y[ch], ch*healBoxW, healSaveY, healBoxW, healBoxH, pageScreen, pageBuild)
		s.ShowMouseIn()
	}
	old := s.SetPage(pageBuild)
	g.playSound(healSound, 0)
	for step := range healSteps {
		clock.Set(clock.Countdown, healStepTicks)
		for ch := first; ch < end; ch++ {
			if g.party[ch].Flags&charPresent == 0 {
				continue
			}
			s.CopyRect(ch*healBoxW, healSaveY, x[ch], y[ch], healBoxW, healBoxH, pageBuild, pageBuild)
			fraction[ch] &= 1<<healFractionBits - 1
			fraction[ch] += amount[ch] << healFractionBits / healSteps
			g.heal(ch, fraction[ch]>>healFractionBits, true)
			g.drawPortrait(ch)
			s.DrawShape(pageBuild, shapeAt(g.healSparkles, sparkles[step]), x[ch], y[ch], 0, gfx.ShapeLevels,
				gfx.ShapeOptions{Levels: g.lvl.tlcTable, LevelTables: g.lvl.tlcLevels})
			s.FillRect(0, 0, healGlowSize-1, healGlowSize-1, 0)
			s.DrawShape(s.Current, shapeAt(g.healGlows, glows[step]), 0, 0, 0, 0, gfx.ShapeOptions{})
			s.TintThroughMask(s.Current, 0, 0, pageBuild, x[ch]+healGlowDX, y[ch]+healGlowDY, healGlowSize, healGlowSize, 0, false, g.healTint)
			s.HideMouseIn(x[ch], y[ch], x[ch]+healBoxW, y[ch]+healBoxH)
			s.CopyRect(x[ch], y[ch], x[ch], y[ch], healBoxW, healBoxH, pageBuild, pageScreen)
			s.ShowMouseIn()
		}
		for clock.Get(clock.Countdown) != 0 {
			host.Yield()
		}
	}
	for ch := first; ch < end; ch++ {
		c := &g.party[ch]
		if c.Flags&charPresent == 0 {
			continue
		}
		s.CopyRect(ch*healBoxW, healSaveY, x[ch], y[ch], healBoxW, healBoxH, pageBuild, pageBuild)
		if cures {
			g.clearStatusEffects(c, effectPoison, effectPoison)
		}
		g.drawPortrait(ch)
		s.HideMouseIn(x[ch], y[ch], x[ch]+healBoxW, y[ch]+healBoxH)
		s.CopyRect(x[ch], y[ch], x[ch], y[ch], healBoxW, healBoxH, pageBuild, pageScreen)
		s.ShowMouseIn()
	}
	s.SetPage(old)
	if opened {
		g.freeHealShapes()
	}
	g.copyViewFrame()
}
