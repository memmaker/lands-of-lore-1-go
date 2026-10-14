package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// Casting (C: o2d8a): a spell's word 13 holds its handler in the low byte and its
// flags in the high one.
const (
	spellHandlerMask   = 1<<8 - 1
	spellFlagShift     = 8
	spellNeedsRoom     = 1 << 0 // fails with a wall or monster right ahead
	stringNoRoom       = 0x4257
	stringPickTarget   = 0x4040
	stringSpellCancel  = 0x4041
	stringAlreadyKnown = 0x42d0
	healPicksTarget    = 3 // heal levels below this ask whom to heal
	blockSpellMask     = 1
)

// spellHandlers cast each spell; spellTargetHandlers finish one cast at a picked
// character (C: g_1424, g_1468).
var (
	spellHandlers = [...]func(g *Game) bool{
		0:  func(g *Game) bool { g.sparkSpell(g.spell.level, g.spell.caster); return true }, // (C: ovr_2d8a_0043)
		1:  (*Game).healSpell,
		2:  func(g *Game) bool { g.iceSpell(g.spell.caster, g.spell.level); return true },       // (C: ovr_2d8a_004d)
		3:  func(g *Game) bool { g.lightningSpell(g.spell.caster, g.spell.level); return true }, // (C: ovr_2d8a_0048)
		4:  func(g *Game) bool { g.fireballSpell(g.spell.level); return true },                  // (C: ovr_2d8a_0052)
		5:  func(g *Game) bool { g.mistSpell(g.spell.caster, g.spell.level); return true },      // (C: ovr_2d8a_0057)
		6:  func(g *Game) bool { g.quakeSpell(g.spell.level, g.spell.caster); return true },     // (C: ovr_2d8a_0061)
		8:  func(g *Game) bool { g.wakeSpell(); return true },                                   // (C: ovr_2d8a_005c)
		9:  func(g *Game) bool { g.freezeSpell(10, g.spell.caster); return true },               // (C: ovr_2d8a_0066)
		12: func(g *Game) bool { g.revealSpell(g.spell.caster); return true },                   // (C: ovr_2d8a_006b)
		16: func(g *Game) bool { g.blockSpell(g.spell.caster, g.spell.level); return true },     // (C: ovr_2d8a_0075)
	}
	spellTargetHandlers = map[int]func(g *Game) bool{
		1: func(g *Game) bool { g.healAnimation(g.spell.target, g.spell.level); return true }, // (C: ovr_2d8a_0070)
	}
)

// spellCast is the spell being cast (C: g_4d78).
type spellCast struct {
	id, caster, level, target int
}

func (g *Game) handlerOf(sp int) int { return spells[sp].Flags & spellHandlerMask }

// castSpell casts spell sp at level from caster; a negative caster -n casts it for
// character n-1 without the cost (C: ovr_2d8a_0039).
func (g *Game) castSpell(caster, sp, level int) bool {
	g.spell = spellCast{id: sp, caster: caster, level: abs(level)}
	if spells[sp].Flags>>spellFlagShift&spellNeedsRoom != 0 &&
		g.wallsBlock(g.blockInDirection(g.currentBlock, g.direction), g.direction, blockSpellMask) {
		g.printMessage(messageEffect, g.text(stringNoRoom))
		return false
	}
	if caster < 0 {
		g.spell.caster = -caster - 1
		return g.runSpellHandler(sp) // sic: indexed by the spell, not its handler
	}
	c := &g.party[caster]
	if spells[sp].MPCost[level] > c.MP || spells[sp].HPCost[level] >= c.HP {
		return false
	}
	g.setPoints(caster, true, -spells[sp].MPCost[level], 1)
	g.setPoints(caster, false, -spells[sp].HPCost[level], 1)
	g.drawPortrait(caster)
	g.runSpellHandler(g.handlerOf(sp))
	return true
}

func (g *Game) runSpellHandler(n int) bool {
	if n < len(spellHandlers) && spellHandlers[n] != nil {
		return spellHandlers[n](g)
	}
	return false
}

// healSpell heals a picked character, or the party from level 3 (C: ovr_2d8a_003e).
func (g *Game) healSpell() bool {
	if g.spell.level < healPicksTarget {
		g.printMessage(0, g.text(stringPickTarget))
		g.buildSpellTargetButtons()
	} else {
		g.healAnimation(healAllParty, g.spell.level)
	}
	return true
}

// clickSpellTargetCharacter finishes the cast at the clicked character
// (C: ovr_2d8a_0034).
func (g *Game) clickSpellTargetCharacter(b *gui.Button) bool {
	g.printMessage(0, "%s.\r", g.party[b.Arg].Name)
	g.spell.target = b.Arg
	if h, ok := spellTargetHandlers[g.handlerOf(g.spell.id)]; ok {
		h(g)
	}
	g.buildButtons()
	return true
}

// clickSpellTargetScene cancels the pick and gives the points back
// (C: ovr_2d8a_002f).
func (g *Game) clickSpellTargetScene(*gui.Button) bool {
	c := &g.party[g.spell.caster]
	g.printMessage(0, g.text(stringSpellCancel))
	c.MP = min(c.MP+spells[g.spell.id].MPCost[g.spell.level], c.MaxMP)
	c.HP = min(c.HP+spells[g.spell.id].HPCost[g.spell.level], c.MaxHP)
	g.drawPortrait(g.spell.caster)
	g.buildButtons()
	return true
}

// opSpellTooCostly: character, spell, level (C: fn_0fbe_3244).
func (g *Game) opSpellTooCostly(s *emc.State) int {
	return boolInt(g.spellTooCostly(s.Arg(0), s.Arg(1), s.Arg(2)))
}

// opLearnSpell: spell, character (C: ovr_2d86_0034).
func (g *Game) opLearnSpell(s *emc.State) int {
	return boolInt(g.learnSpell(s.Arg(0), s.Arg(1)))
}

// learnSpell adds a spell to the party's list, with the spell book's animation for
// any but the first two (C: ovr_2d86_002f).
func (g *Game) learnSpell(sp, ch int) bool {
	slot, found := 0, false
	for i, have := range g.availableSpells {
		if !found && have == noSpell {
			found, slot = true, i
		}
		if have == sp {
			g.printMessage(messageEffect, g.text(stringAlreadyKnown))
			return false
		}
	}
	if sp > 1 {
		g.spellBookAnimation(ch, sp, slot-1)
	}
	g.availableSpells[slot] = sp
	g.buildButtons()
	return true
}

// The spell book: the spell list unrolls a line, the new spell flies from the
// character into it and is written in (C: o2d86).
const (
	spellBookPicture   = "PLAYFLD.CPS"
	spellWritingFmt    = "write%0d%c.wsa"
	spellBookFlyFile   = "getspell.wsa"
	spellBookFlashFile = "spellexp.wsa"
	spellBookPenFile   = "writing.wsa"
	spellBookRows      = 9
	spellBookFlySteps  = 16
	spellBookFlyFrame  = 51
	spellBookFlyY      = 160
	spellBookFlyDX     = 16
	spellBookFlyMargin = 16
	spellBookCastY     = 148
	spellBookLangs     = "EFG"
)

// spellWriting is how each spell is written into the book: frames first to last, a
// frame every delay ticks, and the sound (C: g_12c4).
var spellWriting = [...]struct{ delay, first, last, sound int }{
	{}, {}, {4, 0, 19, 182}, {4, 0, 19, 77}, {4, 0, 19, 185}, {4, 0, 19, 184}, {4, 0, 19, 186},
}

// spellBookLines are where each spell's line of the list is written (C: g_12fc).
var spellBookLines = [...][2]int{{22, 22}, {22, 31}, {22, 40}, {22, 49}, {22, 58}, {22, 67}}

// spellBookAnimation (C: ovr_2d86_002a).
func (g *Game) spellBookAnimation(ch, sp, slot int) {
	s := g.Screen
	toX, toY := g.portraitX[ch]+spellBookFlyDX, spellBookFlyY
	if slot != 1 {
		g.loadSpellBookFrame()
		for i := range spellBookRows {
			g.waitTicks(1)
			g.unrollSpellBook(slot, i)
		}
	}
	s.HideMouse()
	copy(g.pageBackup2[:], s.Page(pageScreen))
	s.ShowMouse()
	line := spellBookLines[max(slot, 0)]
	x0, y0 := line[0]+32, line[1]+5
	writing := fmt.Sprintf(spellWritingFmt, sp, spellBookLangs[min(g.language, len(spellBookLangs)-1)])
	g.playSound(95, 0)
	fly := g.openWSA(spellBookFlyFile, wsaDirect, nil)
	if fly == nil {
		g.scriptErr = fmt.Errorf("SpellBook: Unable to load getspell anim")
		return
	}
	g.playWSARange(fly, 0, 25, 5, g.portraitX[ch], spellBookCastY, nil, true)
	g.playSound(128, 0)
	g.playWSARange(fly, 26, 52, 5, g.portraitX[ch], spellBookCastY, nil, true)
	x, y := toX, toY
	for i := spellBookFlySteps; i >= 0; i-- {
		clock.Set(clock.Countdown, 1)
		copy(s.Page(pageBuild), g.pageBackup2[:])
		x = x0 + int(uint32(((toX-x0)<<8)/spellBookFlySteps*i)>>8)
		y = y0 + int(uint32(((toY-y0)<<8)/spellBookFlySteps*i)>>8)
		g.drawWSA(fly, spellBookFlyFrame, x-spellBookFlyMargin, y-spellBookFlyMargin, pageBuild, wsaOverScreen, g.lvl.tlcTable, g.lvl.tlcLevels)
		s.CopyRect(x-spellBookFlyMargin, y-spellBookFlyMargin, x-spellBookFlyMargin, y-spellBookFlyMargin, fly.width()+3*spellBookFlyMargin, fly.height()+3*spellBookFlyMargin, pageBuild, pageScreen)
		for clock.Get(clock.Countdown) != 0 {
			host.Yield()
		}
	}
	flash := g.openWSA(spellBookFlashFile, wsaDirect, nil)
	if flash == nil {
		g.scriptErr = fmt.Errorf("SpellBook: Unable to load spellexp anim")
		return
	}
	g.playSound(168, 0)
	g.playWSARange(flash, 0, 8, 3, x0-44, y0-38, nil, true)
	pen := g.openWSA(spellBookPenFile, wsaDirect, nil)
	if pen == nil {
		g.scriptErr = fmt.Errorf("SpellBook: Unable to load writing anim")
		return
	}
	g.playWSARange(pen, 0, 6, 5, line[0], line[1], nil, false)
	w := g.openWSA(writing, wsaDirect, nil)
	if w == nil {
		g.scriptErr = fmt.Errorf("SpellBook: Unable to load spellbook anim")
		return
	}
	sw := spellWriting[sp]
	g.playSound(sw.sound, 0)
	g.playWSARange(w, sw.first, sw.last, sw.delay, line[0], line[1], nil, false)
	g.drawScene(pageBuild)
	g.copyViewFrame()
}

// loadSpellBookFrame keeps the spell list's frame from the play field picture in
// pageBackup (C: ovr_2d86_0020).
func (g *Game) loadSpellBookFrame() {
	g.loadPicture(spellBookPicture, pageScratch, nil)
	g.Screen.CopyRect(8, 0, 216, 0, 96, 120, pageScratch, pageScratch)
	copy(g.pageBackup[:], g.Screen.Page(pageScratch))
}

// unrollSpellBook draws the spell list a row longer for the new slot (C: ovr_2d86_0025).
func (g *Game) unrollSpellBook(slot, row int) {
	s := g.Screen
	row = min(row, spellBookRows)
	copy(s.Page(pageScratch), g.pageBackup[:])
	s.CopyRect(216, 0, 8, 0, 96, 120, pageScratch, pageScratch)
	h := (slot+1)*9 + row + 1
	s.CopyRect(112, 0, 12, 0, 87, 15, pageBuild, pageBuild)
	s.CopyRect(201, 1, 17, 15, 6, h, pageBuild, pageBuild)
	s.CopyRect(208, 1, 89, 15, 6, h, pageBuild, pageBuild)
	old := s.SetPage(pageBuild)
	s.FillRect(21, 15, 89, h+15, 206)
	s.CopyRect(112, 16, 12, h+15, 87, 14, pageBuild, pageBuild)
	y := 15
	for i, sp := range g.availableSpells {
		if sp == noSpell {
			continue
		}
		colour := 1
		if g.selectedSpell == i {
			colour = 132
		}
		g.printAt(g.text(spells[sp].Name), 24, y, colour, 0, 0)
		y += 9
	}
	s.SetPage(old)
	s.HideMouse()
	s.CopyRect(8, 0, 8, 0, 96, 120, pageScratch, pageScreen)
	s.ShowMouse()
}
