package game

import (
	"errors"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// takeCharacterItem takes the first item a character wears in the slots of mask and
// runs its unequip script; it is the item, or 0 (C: ovr_2d46_0048).
func (g *Game) takeCharacterItem(ch, mask int) int {
	c := &g.party[ch]
	for i := range equipmentSlots {
		if mask&(1<<i) == 0 {
			continue
		}
		if it := c.Items[i]; it != 0 {
			c.Items[i] = 0
			g.runItemScript(ch, it, eventEquipped, 0, 0)
			return it
		}
	}
	return 0
}

// Vipers: a monster's spell sends snakes at the party from the nearest monster or
// wall ahead; one character is bitten for each in the party (C: o2da0).
const (
	viperFile       = "viper.wsa"
	viperSound      = 148
	viperBiteSound  = 172
	viperReach      = 3
	viperFrameTicks = 5
	viperDamageAdd  = 10 // plus the level number
	viperSource     = monsterFlag
	viperKind       = 2
	viperHow        = 1<<7 | 1<<2 | 1<<1
	viewBackupSize  = viewHeight * gfx.Width // the rows of the screen the view covers
)

// viperRuns are the frames by how many blocks away the snakes come from, and the
// frame the bite sounds on (C: f_26a2_0000).
var viperRuns = [...]struct{ start, end, sound int }{
	{15, 25, 20}, {10, 25, 20}, {5, 25, 20}, {0, 25, 20},
}

// viperSpell (C: ovr_2da0_002a).
func (g *Game) viperSpell() {
	g.sceneReady = true
	b := g.currentBlock
	dist := 0
	for ; dist < viperReach; dist++ {
		next := g.blockInDirection(b, g.direction)
		if g.lvl.blocks[b].Objects&monsterFlag != 0 || g.lvl.wallFlags[g.lvl.blocks[next].Walls[g.direction^2]]&wallBlocksAll != 0 {
			break
		}
		b = next
	}
	s := g.Screen
	old := s.SetPage(pageBuild)
	s.HideMouse()
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	g.playSound(viperSound, 0)
	w := g.openWSA(viperFile, wsaNoBuffer, nil)
	if w == nil {
		g.scriptErr = errors.New("Viper: Unable to load viper.wsa")
		return
	}
	run := viperRuns[dist]
	for i := run.start; i <= run.end; i++ {
		t := clock.Ticks()
		copy(s.Page(pageBuild)[:viewBackupSize], g.pageBackup2[:viewBackupSize])
		if run.sound == i {
			g.playSound(viperBiteSound, 0)
		}
		g.drawWSA(w, i%w.frames(), spellViewX, 0, pageBuild, wsaOverScreen, g.lvl.tlcTable, g.lvl.tlcLevels)
		s.HideMouseIn(spellViewX, 0, spellViewMouseX, viewHeight)
		s.CopyRect(spellViewX, 0, spellViewX, 0, viewWidth, viewHeight, pageBuild, pageScreen)
		s.ShowMouseIn()
		for t+viperFrameTicks > clock.Ticks() {
			host.Yield()
		}
	}
	s.SetPage(old)
	s.HideMouse()
	copy(s.Page(pageScreen)[:viewBackupSize], g.pageBackup2[:viewBackupSize])
	s.ShowMouse()
	copy(s.Page(pageBuild)[:viewBackupSize], g.pageBackup2[:viewBackupSize])
	who := dice(1, partySize) - 1
	for i := range partySize {
		// sic: it tests one character and bites another
		present := g.party[who%partySize].Flags&charPresent != 0 // fix: the original read past the party
		who++
		if !present {
			who %= partySize
		} else {
			g.damageFighter(i, g.level+viperDamageAdd, viperSource, viperKind, viperHow)
		}
	}
}
