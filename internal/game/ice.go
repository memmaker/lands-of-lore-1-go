package game

import (
	"errors"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// Ice: the view turns icy while snow or ice plays over it; the monsters ahead are
// hurt, or from level 3 frozen and shattered. On level 11 it freezes the swamp
// (C: o2d1b).
const (
	iceColoursFile    = "swampice.col"
	iceSnowFile       = "snow.wsa"
	iceFile           = "ice.wsa"
	shatterFile       = "shatter.wsa"
	iceSound          = 71
	iceFreezeSound    = 165
	shatterSound      = 166
	iceFadeTicks      = 40
	iceFrameTicks     = 2
	shatterFrameTicks = 1
	shatterFadeTicks  = 20
	shatterX          = 58
	iceSnowX          = 112
	iceX              = 136
	iceY              = 12
	iceFreezeLevels   = 3 // from this level the monsters are frozen
	iceKind           = 3
	iceSkill          = 2
	sixBitMax         = 63
	halfPalette       = levelPaletteSize // the colours the swamp palette replaces
	monsterModeScript = monsterModeStill // a scripted death starts from standing
)

// iceMight and the frozen damage dice (C: g_0b46, g_0b4e, g_0b53, g_0b58, g_0b5d).
var (
	iceMight      = [...]int{10, 20, 30, 55}
	iceSwampTicks = [...]int{20, 28, 40, 60, 80}
	iceDice       = [...]int{1, 2, 15, 20, 35}
	iceDiceSides  = [...]int{10, 10, 3, 4, 4}
	iceDiceAdd    = [...]int{5, 10, 30, 10, 10}
)

// iceSpell (C: ovr_2d1b_0039).
func (g *Game) iceSpell(caster, level int) {
	s := g.Screen
	old := s.SetPage(pageBuild)
	clock.Disable(playClockSlot)
	s.HideMouse()
	g.drawScene(pageScreen)
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	if g.level == swapPaletteLevel && !g.flag(flagSwampIce) {
		g.swapPalettes(halfPalette)
		g.setFlag(flagSwampIce, true)
		g.addStatusEffect(caster, effectLevelPalette, iceSwampTicks[level], true)
	}
	icy, normal, ok := g.icePalettes()
	if !ok {
		return
	}
	g.darkenPalette(g.Palette[:], g.brightness, g.lampLevel)
	var w *wsaPlayer
	x, y := 0, 0
	switch level {
	case 1, 2:
		if w = g.openWSA(iceSnowFile, wsaNoBuffer, nil); w == nil {
			g.scriptErr = errors.New("Ice: Unable to load snow.wsa")
			return
		}
		x = iceSnowX
	case 3:
		if w = g.openWSA(iceFile, wsaNoBuffer, nil); w == nil {
			g.scriptErr = errors.New("Ice: Unable to load ice.wsa")
			return
		}
		x, y = iceX, iceY
	}
	g.playSound(iceSound, 0)
	dark := g.darkPalette
	g.playWSARange(nil, 0, 0, iceFrameTicks, 0, 0, nil, &paletteFade{from: dark[:], to: icy[:], ticks: iceFadeTicks}, false)
	g.fadeStep(&paletteFade{from: dark[:], to: icy[:], ticks: 1}, int(clock.Ticks())) // sic: a time far past the fade's end
	if w != nil {
		restore := true
		if level >= iceFreezeLevels {
			g.lvl.blocks[g.blockInDirection(g.currentBlock, g.direction)].Flags |= blockShatters
			restore = false
			g.playSound(iceFreezeSound, 0)
		}
		g.playWSARange(w, 0, w.frames(), iceFrameTicks, x, y, nil, nil, restore)
	}
	ahead := g.blockInDirection(g.currentBlock, g.direction)
	shattered := false
	if level < iceFreezeLevels {
		g.damageBlockMonsters(ahead, caster, iceMight[level], iceKind)
	} else {
		shattered = g.freezeMonsters(ahead, caster, iceDice[level], iceDiceSides[level], iceDiceAdd[level], iceKind)
	}
	g.copyViewFrame()
	g.drawScene(pageScreen)
	clock.Enable(playClockSlot)
	if g.level != swapPaletteLevel {
		g.darkenPaletteInto(normal[:], g.Palette[:], g.brightness, g.lampLevel)
	}
	g.playWSARange(nil, 0, 0, iceFrameTicks, 0, 0, nil, &paletteFade{from: icy[:], to: normal[:], ticks: iceFadeTicks}, false)
	g.fadeStep(&paletteFade{from: icy[:], to: normal[:], ticks: 1}, int(clock.Ticks()))
	if shattered {
		g.shatterWall(icy[:], normal[:])
	}
	// The original added the swamp's effect again for a level 11 cast here; its
	// spell levels never reach 11.
	s.SetPage(old)
}

// icePalettes are the icy palette (the level's colours as blue-greys) and the
// normal one with the swamp's colours, both at the current brightness
// (C: ovr_2d1b_002f).
func (g *Game) icePalettes() (icy, normal gfx.Palette, ok bool) {
	file, err := g.Files.Read(iceColoursFile)
	if err != nil {
		g.scriptErr = errors.New("Ice: Unable to load swampice.col")
		return icy, normal, false
	}
	copy(normal[:], file)
	for i := halfPalette; i < len(normal); i++ {
		normal[i] = g.darkPalette[i] & sixBitMax
		icy[i] = g.darkPalette[i] & sixBitMax
	}
	for i := 1; i < halfPalette/3; i++ {
		grey := (int(g.darkPalette[i*3+1]) + int(g.darkPalette[i*3+2]) + int(g.darkPalette[i*3])) / 3
		icy[i*3] = 0
		icy[i*3+1] = byte(grey)
		icy[i*3+2] = byte(min(grey*2, sixBitMax))
	}
	g.darkenPaletteInto(icy[:], icy[:], g.brightness, g.lampLevel)
	g.darkenPaletteInto(normal[:], normal[:], g.brightness, g.lampLevel)
	clear(icy[:3])
	clear(normal[:3])
	return icy, normal, true
}

// freezeMonsters freezes the monsters on block b: those it would kill die at once,
// the rest are hurt; it says whether any was left standing (C: ovr_2d1b_0048).
func (g *Game) freezeMonsters(b, att, n, sides, add, kind int) bool {
	hit := false
	for o := g.lvl.blocks[b].Objects; o&monsterFlag != 0; {
		r := g.damageOfKind(att, o, dice(n, sides)+add, kind, 2)
		m := &g.monsters[o&monsterIndex]
		if m.HitPoints <= r {
			g.addExperience(att, iceSkill, m.HitPoints)
			o = m.Next
			if m.Flags&monsterDropItems != 0 {
				g.scriptedMonsterDeath(m, o) // sic: the death script gets the next monster
			} else {
				g.killMonster(m)
			}
		} else {
			hit = true
			g.damageFighter(o, r, att, iceSkill, kind)
			m.Damage = 0
			o = m.Next
		}
		if m.Flags&monsterDropItems != 0 {
			break
		}
	}
	return hit
}

// scriptedMonsterDeath has m die through the level's death script, called with who.
func (g *Game) scriptedMonsterDeath(m *Monster, who int) {
	m.Mode = monsterModeScript
	g.dropMonsterItems(m)
	if g.level != finalLevel {
		g.setMonsterMode(m, monsterModeRemove)
	}
	g.runBlockScriptWith(monsterDeathFunction, allEvents, who, who, 0, 0)
	g.noteBlockChanged(m.Block)
	if m.Mode == monsterModeRemove {
		g.moveMonster(m, 0, 0)
	}
}

// shatterWall shatters the frozen block ahead, the palette moving from one to
// another meanwhile if given (C: ovr_2d1b_0052).
func (g *Game) shatterWall(from, to []byte) {
	s := g.Screen
	s.HideMouse()
	old := s.SetPage(pageBuild)
	g.lvl.blocks[g.blockInDirection(g.currentBlock, g.direction)].Flags &^= blockShatters
	s.CopyPage(pageScreen, pageBuild)
	g.drawScene(pageBuild)
	copy(g.pageBackup2[:], s.Page(pageBuild))
	w := g.openWSA(shatterFile, wsaNoBuffer, nil)
	if w == nil {
		g.scriptErr = errors.New("Shatter: Unable to load shatter.wsa")
		return
	}
	g.playSound(shatterSound, 0)
	var fade *paletteFade
	if from != nil && to != nil {
		fade = &paletteFade{from: from, to: to, ticks: shatterFadeTicks}
	}
	g.playWSARange(w, 0, w.frames(), shatterFrameTicks, shatterX, 0, nil, fade, true)
	s.SetPage(old)
	g.copyViewFrame()
	g.drawScene(pageScreen)
	s.ShowMouse()
}

// swapPalettes swaps the first n bytes of the palette with the swamp's, from colour
// 0's green on (sic: its red stays).
func (g *Game) swapPalettes(n int) {
	for i := 1; i < n; i++ {
		g.Palette[i], g.swampPalette[i] = g.swampPalette[i], g.Palette[i]
	}
}

// swapLevelPalette loads level 11's frozen swamp palette and swaps it in if the
// swamp is frozen (C: ovr_2d1b_002a).
func (g *Game) swapLevelPalette() {
	if g.level != swapPaletteLevel {
		return
	}
	if file, err := g.Files.Read(swapPaletteFile); err == nil {
		copy(g.swampPalette[:halfPalette], file)
	}
	copy(g.swampPalette[halfPalette:], g.Palette[halfPalette:])
	if g.flag(flagSwampIce) {
		g.swapPalettes(len(g.Palette))
	}
}

// swapLevelPaletteBack thaws the swamp (C: ovr_2d1b_0043).
func (g *Game) swapLevelPaletteBack() {
	g.setFlag(flagSwampIce, false)
	if g.level != swapPaletteLevel {
		return
	}
	g.swapPalettes(len(g.Palette))
	g.darkenPalette(g.Palette[:], g.brightness, g.lampLevel)
	g.keepInterfaceColours(g.swampPalette[:])
	g.keepInterfaceColours(g.darkPalette[:])
	from, to := g.swampPalette, g.darkPalette
	g.playWSARange(nil, 0, 0, iceFrameTicks, 0, 0, nil, &paletteFade{from: from[:], to: to[:], ticks: iceFadeTicks}, false)
}
