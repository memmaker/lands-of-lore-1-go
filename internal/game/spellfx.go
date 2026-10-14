package game

import (
	"errors"
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
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

// fogSpell plays the fog over the view and has it hit the monsters right ahead
// (C: ovr_2d07_0025).
func (g *Game) fogSpell() {
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	// The original kept the pool the animation loads into in pageBackup meanwhile.
	w := g.openWSA(fogFile, wsaNoBuffer, nil)
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

// guardianSpell (C: ovr_2d12_002a); it says whether a monster was there. The original
// also looked for the first monster or wall in three blocks ahead (fn_0845_0c7e)
// and never used it.
func (g *Game) guardianSpell(caster int) bool {
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	s.CopyPage(pageScreen, pageBuild)
	s.ShowMouse()
	copy(g.pageBackup2[:], s.Page(pageBuild))
	w := g.openWSA(guardianFile, wsaNoBuffer, nil)
	if w == nil {
		g.scriptErr = errors.New("Guardian: Unable to load guardian.wsa")
		return false
	}
	g.playSound(guardianSound, 0)
	g.playWSARange(w, 0, guardianStrikeEnd, guardianTicks, spellViewX, 0, nil, nil, false)
	copy(g.pageBackup[:], s.Page(pageIcon))
	copy(g.pageBackup2[:], s.Page(pageBuild))
	found := g.lvl.blocks[g.currentBlock].Objects&monsterFlag != 0
	g.damageBlockMonsters(g.currentBlock, caster, guardianMight, guardianKind)
	copy(s.Page(pageBuild), g.pageBackup2[:])
	g.copyViewFrame()
	g.drawScene(pageBuild)
	if g.level == levelGuardianAgain {
		if w = g.openWSA(guardianFile, wsaNoBuffer, nil); w == nil {
			return found // fix: the original went on without the animation
		}
	} else {
		copy(s.Page(pageIcon), g.pageBackup[:])
	}
	copy(g.pageBackup2[:], s.Page(pageBuild))
	g.playSound(guardianLeaveSound, 0)
	g.playWSARange(w, guardianLeaveStart, guardianLeaveEnd, guardianLeaveTicks, spellViewX, 0, nil, nil, false)
	s.SetPage(old)
	s.HideMouse()
	g.drawGameScreen()
	s.ShowMouse()
	g.copyViewFrame()
	return found
}

// Mists: the monsters ahead are hit, then the mist plays over the view (C: o2d4d).
const (
	mistFile        = "mists%0d.wsa"
	mistSound       = 155
	mistFirstTicks  = 7
	mistSecondTicks = 14
	mistKind        = 1 << 7
)

// mistMight is each level's might (C: g_0ff8).
var mistMight = [...]int{30, 70, 110, 200}

// mistParts are each level's two frame ranges and its sound (C: g_0fc4).
var mistParts = [...]struct{ start1, end1, start2, end2, sound int }{
	{0, 7, 7, 13, 155}, {0, 16, 16, 17, 155}, {0, 24, 24, 24, 174}, {0, 19, 19, 19, 174}, {0, 16, 16, 17, 175},
}

// mistSpell (C: ovr_2d4d_002a).
func (g *Game) mistSpell(caster, level int) {
	g.delaySounds = true
	g.damageBlockMonsters(g.blockInDirection(g.currentBlock, g.direction), caster, mistMight[level], mistKind)
	g.delaySounds = false
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	s.CopyPage(pageScreen, pageBuild)
	s.ShowMouse()
	g.drawScene(pageBuild)
	copy(g.pageBackup2[:], s.Page(pageBuild))
	g.playSound(mistSound, 0)
	w := g.openWSA(fmt.Sprintf(mistFile, level+1), wsaNoBuffer, nil)
	if w == nil {
		g.scriptErr = errors.New("Mist: Unable to load mists.wsa")
		return
	}
	p := mistParts[level]
	g.playSound(p.sound, 0)
	g.playWSARange(w, p.start1, p.end1, mistFirstTicks, spellViewX, 0, nil, nil, false)
	g.playWSARange(w, p.start2, p.end2, mistSecondTicks, spellViewX, 0, nil, nil, false)
	s.SetPage(old)
	s.HideMouse()
	copy(s.Page(pageScreen), g.pageBackup2[:])
	s.ShowMouse()
	g.copyViewFrame()
	g.playDelayedSounds()
}

// Spark: sparks fly from the caster's portrait, the nearest monster within four
// blocks ahead is hit and the spark bursts (C: o2d83).
const (
	sparkCastFile   = "spark1.wsa"
	sparkBurstFile  = "spark2.wsa"
	sparkCastSound  = 72
	sparkBurstSound = 162
	sparkCastEnd    = 7
	sparkCastTicks  = 4
	sparkCastDX     = -2
	sparkCastY      = 138
	sparkReach      = 4
	sparkKind       = 5
	sparkBursts     = 6
	sparkSpreadX    = 64
	sparkSpreadY    = 32
	sparkBurstLen   = 13
	sparkTicks      = 4
	sparkFramesDist = 16 // each block of distance has its own frames
)

// sparkMight is each level's might (C: g_1264).
var sparkMight = [...]int{7, 15, 25, 60}

// sparkSpell (C: ovr_2d83_0020).
func (g *Game) sparkSpell(level, caster int) {
	s := g.Screen
	s.HideMouse()
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	w := g.openWSA(sparkCastFile, 0, nil)
	if w == nil {
		g.scriptErr = errors.New("SPARK: Unable to load SPARK1.WSA")
		return
	}
	g.playSound(sparkCastSound, 0)
	g.playWSARange(w, 0, sparkCastEnd, sparkCastTicks, g.portraitX[caster]+sparkCastDX, sparkCastY, nil, nil, false)
	s.HideMouse()
	copy(s.Page(pageScreen), g.pageBackup2[:])
	s.ShowMouse()
	dist, hit := g.firstStopAhead(sparkReach)
	if m := g.nearestMonsterIn(hit, caster); m != noTarget {
		g.damageMonster(m, caster, sparkMight[level], sparkKind, false)
		g.copyViewFrame()
		g.drawScene(pageScreen)
		copy(g.pageBackup2[:], s.Page(pageScreen))
	}
	if w = g.openWSA(sparkBurstFile, 0, nil); w == nil {
		g.scriptErr = errors.New("SPARK: Unable to load SPARK2.WSA")
		return
	}
	var xs, ys, starts [sparkBursts]int
	for i := range sparkBursts {
		xs[i] = (viewWidth-w.width())>>1 + (random.Rand()%sparkSpreadX - sparkSpreadX/2) + spellViewX
		ys[i] = (viewHeight-w.height())>>1 + (random.Rand()%sparkSpreadY - sparkSpreadY/2)
		starts[i] = i * 2
	}
	steps := level*2 + sparkBurstLen - 1
	for j := range steps {
		clock.Set(clock.Countdown, sparkTicks)
		copy(s.Page(pageBuild), g.pageBackup2[:])
		for i := 0; i <= level; i++ {
			if starts[i] < j && starts[i]+sparkBurstLen > j {
				if j-starts[i] == 1 {
					g.playSound(sparkBurstSound, 0)
				}
				g.drawWSA(w, (j-starts[i]+dist*sparkFramesDist)%w.frames(), xs[i], ys[i], pageBuild, wsaOverScreen, g.lvl.tlcTable, g.lvl.tlcLevels)
				s.HideMouseIn(xs[i], ys[i], xs[i]+w.width(), ys[i]+w.height())
				s.CopyRect(xs[i], ys[i], xs[i], ys[i], w.width(), w.height(), pageBuild, pageScreen)
				s.ShowMouseIn()
			}
		}
		if j < steps-1 {
			for clock.Get(clock.Countdown) != 0 {
				host.Yield()
			}
		}
	}
	copy(s.Page(pageBuild), g.pageBackup2[:])
	g.copyViewFrame()
	g.sceneUpdateRequired = true
}

// firstStopAhead walks up to n blocks ahead of the party and gives how many it
// passed and the block with a monster or before a wall, or -1 (C: fn_0845_0c7e).
func (g *Game) firstStopAhead(n int) (steps, stop int) {
	const wallStops = 1<<3 - 1
	cur, dir := g.currentBlock, g.direction
	next := g.blockInDirection(cur, dir)
	stop = -1
	for steps = 0; steps < n; steps++ {
		if g.lvl.blocks[cur].Objects&monsterFlag != 0 {
			return steps, cur
		}
		if g.lvl.wallFlags[g.lvl.blocks[next].Walls[dir^2]]&wallStops != 0 {
			return steps, next
		}
		cur, next = next, g.blockInDirection(next, dir)
	}
	return steps, stop
}

// Swarm: the monsters ahead are hit and shown frozen while the swarm plays, the
// view blinking between them and the empty view (C: o2d99).
const (
	swarmFile       = "swarm.wsa"
	swarmSound      = 74
	swarmEnd        = 37
	swarmTicks      = 2
	swarmBlinkStart = 38
	swarmBlinkEnd   = 41
	swarmBlinkTicks = 8
	swarmBlinks     = 1
	monsterUnharmed = 1 << 13 // a swarm does not hurt it
	swarmPoseMode   = 8       // the pose the monsters are drawn in
)

// swarmSpell (C: ovr_2d99_002a).
func (g *Game) swarmSpell(might, caster int) {
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	g.playSound(swarmSound, 0)
	var hit []*Monster
	for o := g.lvl.blocks[g.blockInDirection(g.currentBlock, g.direction)].Objects; o&monsterFlag != 0; {
		m := &g.monsters[o&monsterIndex]
		next := m.Next
		if m.Mode < monsterDead {
			hit = append(hit, m)
			if m.Type.flags&monsterUnharmed == 0 {
				g.delaySounds = true
				g.damageMonster(o, caster, might, 0, false)
				g.delaySounds = false
				m.Flags &^= monsterWasHit
			}
		}
		o = next
	}
	type pose struct{ mode, counter int }
	saved := make([]pose, len(hit))
	for i, m := range hit {
		saved[i] = pose{m.Mode, m.Counter}
		m.Mode, m.Counter = swarmPoseMode, 0
	}
	g.drawScene(s.Current)
	frozen := s.SaveRect(spellViewX, 0, viewWidth, viewHeight, s.Current)
	for i, m := range hit {
		m.Mode, m.Counter = saved[i].mode, saved[i].counter
	}
	w := g.openWSA(swarmFile, 0, nil)
	if w == nil {
		g.scriptErr = errors.New("Swarm: Unable to load swarm.wsa")
		return
	}
	s.HideMouse()
	g.playWSARange(w, 0, swarmEnd, swarmTicks, 0, 0, nil, nil, false)
	for range swarmBlinks {
		g.playWSARange(w, swarmBlinkStart, swarmBlinkEnd, swarmBlinkTicks, 0, 0, func() {
			if g.swarmBlink {
				// fix: the original put back its page 6 pool, not the frozen view it had
				// kept in the page 7 pool
				s.RestoreRect(spellViewX, 0, viewWidth, viewHeight, frozen, s.Current)
			}
			g.swarmBlink = !g.swarmBlink
		}, nil, false)
	}
	s.ShowMouse()
	s.SetPage(old)
	s.HideMouse()
	copy(s.Page(pageScreen), g.pageBackup2[:])
	g.copyViewFrame()
	s.ShowMouse()
	g.playDelayedSounds()
}

// Hand of Fate: a hand pushes the monsters ahead a block back, or from level 2
// crushes them (C: o2d15).
const (
	handFile         = "hand.wsa"
	handOpenSound    = 173
	handActSound     = 151
	handCloseSound   = 18
	handOpenEnd      = 10
	handTicks        = 3
	handPushLevels   = 2 // levels below this push
	handPushSteps    = 8 // steps of a monster's position in a block
	handKind         = 1 << 7
	eventMonsterPush = 1 << 11
	levelHandRedraw  = 29 // the view is put back from the screen kept before
)

// handParts are each level's frames of the hand at work (C: g_0a54).
var handParts = [...][2]int{{17, 26}, {11, 16}, {27, 35}, {27, 35}}

// handMight is each crushing level's might (C: g_0a62).
var handMight = [...]int{35, 0, 75, 125, 175}

// handOfFateSpell (C: ovr_2d15_002a).
func (g *Game) handOfFateSpell(level int) {
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	w := g.openWSA(handFile, wsaNoBuffer, nil)
	if w == nil {
		g.scriptErr = errors.New("Hand: Unable to load HAND.WSA")
		return
	}
	g.playSound(handOpenSound, 0)
	g.playWSARange(w, 0, handOpenEnd, handTicks, spellViewX, 0, nil, nil, false)
	g.playSound(handActSound, 0)
	g.playWSARange(w, handParts[level][0], handParts[level][1], handTicks, spellViewX, 0, nil, nil, false)
	g.playSound(handCloseSound, 0)
	g.playWSARange(w, handOpenEnd, 0, handTicks, spellViewX, 0, nil, nil, false)
	s.SetPage(old)
	copy(s.Page(pageBuild), g.pageBackup2[:])
	g.drawScene(pageBuild)
	ahead := g.blockInDirection(g.currentBlock, g.direction)
	if level < handPushLevels {
		behind := g.blockInDirection(ahead, g.direction)
		if !g.wallsBlock(behind, 0, wallStopsMonsters) && g.lvl.blocks[behind].Objects&monsterFlag == 0 {
			g.noteBlockChanged(ahead)
			for o := g.lvl.blocks[ahead].Objects; o&monsterFlag != 0; {
				m := &g.monsters[o&monsterIndex]
				next := *g.objectNext(o)
				x, y := m.X, m.Y
				for range handPushSteps {
					x, y = nextStep(x, y, g.direction*2)
				}
				g.moveMonster(m, x, y)
				g.runBlockScriptWith(behind, eventMonsterPush, -1, o, 0, 0)
				o = next
			}
		}
	} else {
		g.noteBlockChanged(ahead)
		for o := g.lvl.blocks[ahead].Objects; o&monsterFlag != 0; {
			next := *g.objectNext(o)
			g.damageMonster(o, noTarget, handMight[level], handKind, true)
			o = next
		}
	}
	if g.level == levelHandRedraw {
		copy(s.Page(pageBuild), g.pageBackup2[:])
	}
	s.HideMouse()
	s.CopyPage(pageBuild, pageScreen)
	s.ShowMouse()
	g.drawScene(pageBuild)
	g.copyViewFrame()
}

// Lightning: bolts strike over the view while the screen flashes and shakes, then
// the monsters ahead are hit (C: o2d28).
const (
	lightningFile       = "litning%d.wsa"
	lightningStrikes    = 4
	lightningTicks      = 3
	lightningX          = 93
	lightningKind       = 5
	lightningShakeEvery = 2 // with this period the view shakes and the sound plays once
	lightningBright     = 120
	lightningScale      = 64
	lightningFirstLit   = 2   // colours 0 and 1 stay
	lightningLitColours = 128 // the level's colours
	lightningMaxLevel   = 63  // the brightest a VGA colour gets
	lightningShakeTicks = 1
	lightningShakeX     = 2
	lightningShakeY     = 3
)

// lightningParts are each level's last frame, flash period and sound (C: g_0c70).
var lightningParts = [...]struct{ end, period, sound int }{
	{6, 5, 171}, {6, 4, 171}, {8, 4, 170}, {6, 3, 170}, {6, 2, 169},
}

// lightningMight is each level's might (C: g_0c8a).
var lightningMight = [...]int{18, 35, 50, 72}

// lightningSpell (C: ovr_2d28_002a).
func (g *Game) lightningSpell(level, caster int) {
	s := g.Screen
	s.HideMouse()
	s.CopyPage(pageScreen, pageBuild)
	g.drawScene(pageBuild)
	old := s.SetPage(pageBuild)
	copy(g.pageBackup2[:], s.Page(pageBuild))
	p := lightningParts[level]
	played := false
	w := g.openWSA(fmt.Sprintf(lightningFile, level+1), wsaNoBuffer, nil)
	if w == nil {
		g.scriptErr = errors.New("Litning: Unable to load litning.wsa")
		return
	}
	// flash lights the screen every period frames, shakes the view and plays the
	// thunder (C: ovr_2d28_002f).
	flash := func() {
		if p.period == lightningShakeEvery {
			g.shakeScreen(lightningShakeTicks, lightningShakeX, lightningShakeY, 0)
		}
		if g.lightningFlashes%p.period == 0 {
			pal := g.darkPalette
			for i := lightningFirstLit * 3; i < lightningLitColours*3; i++ {
				pal[i] = byte(min(int(int8(pal[i]))*lightningBright/lightningScale, lightningMaxLevel))
			}
			s.SetPalette(pal[:])
		} else {
			s.SetPalette(g.darkPalette[:])
		}
		if p.period == lightningShakeEvery {
			if !played {
				g.playSound(p.sound, 0)
				played = true
			}
		} else if g.lightningFlashes&7 == 0 {
			g.playSound(p.sound, 0)
		}
		g.lightningFlashes++
	}
	for range lightningStrikes {
		g.playWSARange(w, 0, p.end, lightningTicks, lightningX, 0, flash, nil, false)
	}
	s.SetPalette(g.darkPalette[:])
	s.SetPage(old)
	copy(s.Page(pageBuild), g.pageBackup2[:])
	copy(s.Page(pageScreen), g.pageBackup2[:])
	g.copyViewFrame()
	g.damageBlockMonsters(g.blockInDirection(g.currentBlock, g.direction), caster, lightningMight[level], lightningKind)
	g.sceneUpdateRequired = true
	g.drawScene(pageScreen)
	s.ShowMouse()
}
