package game

import (
	"errors"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// The view's area, where the spell animations play.
const (
	spellViewX      = 112
	spellViewMouseX = 288
	spellViewMouseY = 128
)

// damageMonster has a spell of kind with might base hit monster m (C: ovr_2d8a_0020).
func (g *Game) damageMonster(m, att, base, kind int, withProtection bool) {
	mode := 2
	if withProtection {
		mode = 1
	}
	g.damageFighter(m, g.damageOfKind(att, m, base, kind, mode), att, 2, kind)
}

// damageBlockMonsters has a spell hit the monsters on block b, stopping after one
// that drops its items (C: ovr_2d8a_0025).
func (g *Game) damageBlockMonsters(b, att, base, kind int) {
	for o := g.lvl.blocks[b].Objects; o&monsterFlag != 0; {
		r := g.damageOfKind(att, o, base, kind, 2)
		m := &g.monsters[o&monsterIndex]
		next := m.Next
		g.damageFighter(o, r, att, 2, kind)
		if m.Flags&monsterDropItems != 0 && g.level != levelNoOverlay {
			break
		}
		o = next
	}
}

// Fog (C: o2d07).
const (
	fogFile       = "fog.wsa"
	fogFrameTicks = 3
	fogSound      = 145
	fogMight      = 15
	fogKind       = 6
)

// wakeSpell plays the fog over the view and has it hit the monsters right ahead
// (C: ovr_2d07_0025).
func (g *Game) wakeSpell() {
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	// The original kept the pool the animation loads into in pageBackup meanwhile.
	w := g.openWSA(fogFile, wsaDirect, nil)
	if w == nil {
		g.scriptErr = errors.New("Fog: Unable to load fog.wsa")
		return
	}
	g.playSound(fogSound, 0)
	for i := 0; i < w.frames(); i++ {
		clock.Set(clock.Countdown, fogFrameTicks)
		copy(s.Page(pageBuild), g.pageBackup2[:])
		g.drawWSA(w, i, spellViewX, 0, pageBuild, wsaOverScreen, g.lvl.tlcTable, g.lvl.tlcLevels)
		s.HideMouseIn(spellViewX, 0, spellViewMouseX, spellViewMouseY)
		s.CopyRect(spellViewX, 0, spellViewX, 0, viewWidth, viewHeight, pageBuild, pageScreen)
		s.ShowMouseIn()
		for clock.Get(clock.Countdown) != 0 {
			host.Yield()
		}
	}
	copy(s.Page(pageBuild), g.pageBackup2[:])
	s.SetPage(old)
	g.copyViewFrame()
	for o := g.lvl.blocks[g.blockInDirection(g.currentBlock, g.direction)].Objects; o&monsterFlag != 0; o = g.monsters[o&monsterIndex].Next {
		g.damageMonster(o, noTarget, fogMight, fogKind, false)
	}
	g.drawScene(pageScreen)
}

// Reveal: the palette turns bluish, false walls behind the party vanish and
// illusions are hit (C: o2d9d).
const (
	revealTicks       = 70
	revealSound       = 146
	revealRed         = 16
	revealMaxRed      = 60
	revealBlue        = 19
	revealMaxBlue     = 63
	revealColours     = 128
	revealDamage      = 100
	revealHow         = 1 << 7
	wallUnrevealable  = 1 << 0
	revealVMPFalse1   = 1
	revealVMPFalse2   = 2
	revealWall        = 47 // the wall the reveal finds on levels 17 and 24
	levelRevealWall1  = 17
	levelRevealWall2  = 24
	monsterIsIllusion = monsterPassesMonsters // the same type flag
)

// revealSpell (C: ovr_2d9d_0020); it says whether anything was revealed.
func (g *Game) revealSpell() bool {
	found := false
	s := g.Screen
	s.HideMouse()
	normal := g.darkPalette
	bright := g.darkPalette
	for i := 1; i < revealColours; i++ {
		bright[i*3] = byte(min(int(normal[i*3])+revealRed, revealMaxRed))
		bright[i*3+2] = byte(min(int(normal[i*3+2])+revealBlue, revealMaxBlue))
	}
	g.playSound(revealSound, 0)
	g.fadeFor(normal[:], bright[:])
	b := g.currentBlock
	w := g.lvl.blocks[b].Walls[g.direction^2]
	flags := g.lvl.wallFlags[w]
	if (g.level == levelRevealWall1 || g.level == levelRevealWall2) && w == revealWall {
		found = true
	}
	if v := g.lvl.wallVMP[w]; (v == revealVMPFalse1 || v == revealVMPFalse2) && flags&wallUnrevealable == 0 && g.level != levelNoOverlay {
		g.lvl.blocks[b].Walls = [directionCount]int{}
		g.drawScene(pageScreen)
		found = true
	}
	for o := g.lvl.blocks[b].Objects; o&monsterFlag != 0; {
		m := &g.monsters[o&monsterIndex]
		next := m.Next
		if m.Type.flags&monsterIsIllusion != 0 {
			g.damageFighter(o, revealDamage, noTarget, 0, revealHow)
			found = true
		}
		o = next
	}
	g.fadeFor(bright[:], normal[:])
	s.ShowMouse()
	return found
}

// fadeFor moves the palette from one to another over revealTicks.
func (g *Game) fadeFor(from, to []byte) {
	f := paletteFade{from: from, to: to, ticks: revealTicks}
	t0 := clock.Ticks()
	clock.Set(clock.Countdown, revealTicks)
	for clock.Get(clock.Countdown) != 0 {
		g.fadeStep(&f, clock.Ticks()-t0)
		host.Yield()
	}
}

// Guardian: a guardian strikes the monsters on the party's block (C: o2d12).
const (
	guardianFile       = "guardian.wsa"
	guardianSound      = 156
	guardianLeaveSound = 176
	guardianStrikeEnd  = 37
	guardianLeaveStart = 38
	guardianLeaveEnd   = 48
	guardianTicks      = 2
	guardianLeaveTicks = 8
	guardianMight      = 200
	guardianKind       = 1 << 7
	levelGuardianAgain = 24 // its scene drawing needs the animation loaded again
)

// blockSpell (C: ovr_2d12_002a); it says whether a monster was there. The original
// also looked for the first monster or wall in three blocks ahead (fn_0845_0c7e)
// and never used it.
func (g *Game) blockSpell(caster int) bool {
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	s.CopyPage(pageScreen, pageBuild)
	s.ShowMouse()
	copy(g.pageBackup2[:], s.Page(pageBuild))
	w := g.openWSA(guardianFile, wsaDirect, nil)
	if w == nil {
		g.scriptErr = errors.New("Guardian: Unable to load guardian.wsa")
		return false
	}
	g.playSound(guardianSound, 0)
	g.playWSARange(w, 0, guardianStrikeEnd, guardianTicks, spellViewX, 0, nil, false)
	copy(g.pageBackup[:], s.Page(pageIcon))
	copy(g.pageBackup2[:], s.Page(pageBuild))
	found := g.lvl.blocks[g.currentBlock].Objects&monsterFlag != 0
	g.damageBlockMonsters(g.currentBlock, caster, guardianMight, guardianKind)
	copy(s.Page(pageBuild), g.pageBackup2[:])
	g.copyViewFrame()
	g.drawScene(pageBuild)
	if g.level == levelGuardianAgain {
		if w = g.openWSA(guardianFile, wsaDirect, nil); w == nil {
			return found // fix: the original went on without the animation
		}
	} else {
		copy(s.Page(pageIcon), g.pageBackup[:])
	}
	copy(g.pageBackup2[:], s.Page(pageBuild))
	g.playSound(guardianLeaveSound, 0)
	g.playWSARange(w, guardianLeaveStart, guardianLeaveEnd, guardianLeaveTicks, spellViewX, 0, nil, false)
	s.SetPage(old)
	s.HideMouse()
	g.drawGameScreen()
	s.ShowMouse()
	g.copyViewFrame()
	return found
}
