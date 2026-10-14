package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// The character screen (C: o2d22): the stats, the skill bars and the equipment over
// the view, opened by a click on a portrait.
const (
	characterStats  = 5 // might, protection and the three skills
	statMight       = 0
	statProtection  = 1
	statFirstSkill  = 2
	ailmentIcons    = 3
	charSkillsShown = charSkillRaised | charSkillRaised<<1 | charSkillRaised<<2

	statLabelX     = 108
	statValueX     = 200
	statRowStep    = 10
	statFirstY     = 22
	skillFirstY    = 62
	statColour     = 158
	skillColour    = 180
	skillLitColour = 254
	statWidth      = 18
	statHeight     = 8
	statPage       = 6
	statBackupDX   = 94 // where the clean background of the values is kept on statPage
	statSteps      = 15
	stringMight    = 0x4014
	stringFighter  = 0x4016

	characterPictureFmt   = "invent%d.cps"
	pictureColumns        = 26
	pictureRows           = 120
	pictureColumn         = 14
	faceColumn            = 10
	faceY                 = 143
	faceColumns           = 29
	faceRows              = 35
	nameX                 = 157
	nameY                 = 9
	nameColour            = 254
	stringAilments        = 0x4033
	ailmentsLabelX        = 182
	ailmentsLabelY        = 103
	ailmentsLabelColour   = 172
	ailmentX              = 108
	ailmentY              = 98
	ailmentGap            = 2
	expBarX               = 154
	expBarY               = 64
	expBarW               = 34
	expBarH               = 5
	expBarColour          = 132
	expBarLimit           = 1 << 15
	viewLineY             = 120
	viewLineX1            = 14
	viewLineX2            = 194
	viewW                 = 208
	viewH                 = 121
	updateCharacterScreen = updateGameMenu | 1<<3

	equipShapeFull     = 4
	equipShapeRingFull = 5
	firstRingSlot      = 9
	ringIconOffset     = 5

	stringCannotEquipHand = 0x418b
	stringCannotEquip     = 0x418a
	stringNothingThere    = 0x418c
	itemTypeNoEquipNote   = 231
	equipNoteOffset       = 8
	eventEquipped         = 1 << 8
	eventUnequipped       = 1 << 7

	ailmentButtonX   = 220
	ailmentButtonW   = 14
	stringPoisonNote = 0x424c
	stringParalyNote = 0x424d
	stringMuteNote   = 0x424e
	messageNoFade    = 1<<15 | 1<<1
)

// characterPictures is the INVENTn.CPS each hero id uses (C: g_0c21).
var characterPictures = [...]int{0, 1, 2, 6, 3, 1, 1, 3, 5, 4}

// emptySlotShapes are the outlines of the empty equipment slots (C: g_0c2b).
var emptySlotShapes = [equipmentSlots]int{48, 52, 48, 52, 46, 47, 50, 51, 49, 53, 53}

// slotNames are the equipment slots' names (C: g_0c36).
var slotNames = [equipmentSlots]int{0x417a, 0x417b, 0x417a, 0x417b, 0x417c, 0x417d, 0x417e, 0x417f, 0x4180, 0x4181, 0x4181}

// ailmentShapes are the ailments the screen shows and their icons (C: g_0c4d).
var ailmentShapes = []struct{ flag, shape int }{{charFaceTinted, 0}, {charMute, 2}, {charParalysed, 1}}

// characterStat is one of the five stats (C: o2d22 stat).
func (g *Game) characterStat(ch, which int) int {
	c := &g.party[ch]
	switch which {
	case statMight:
		v := 0
		for _, m := range c.ItemsMight {
			v += m
		}
		mod := &characterModifiers[c.Modifiers]
		if v == 0 {
			v = mod[8]
		} else {
			v += c.Might
		}
		v = scale(v, mod[1])
		return scale(v, c.TotalMightModifier)
	case statProtection:
		return g.protectionOf(ch)
	case statFirstSkill, statFirstSkill + 1, statFirstSkill + 2:
		k := which - statFirstSkill
		return c.SkillLevels[k] + c.SkillModifiers[k]
	}
	return -1
}

// rememberStats keeps a character's stats as shown (C: ovr_2d22_0020).
func (g *Game) rememberStats(ch int) {
	for i := range g.shownStats {
		g.shownStats[i] = g.characterStat(ch, i)
	}
}

// drawStat draws a stat's value and, with full, its name (C: o2d22 drawstat).
func (g *Game) drawStat(which int, full bool, value int) {
	dx := 0
	if g.Screen.Current == pageScreen {
		dx = viewLeft
	}
	var label, y, colour int
	switch which {
	case statMight, statProtection:
		label, y, colour = stringMight+which, which*statRowStep+statFirstY, statColour
	case statFirstSkill, statFirstSkill + 1, statFirstSkill + 2:
		k := which - statFirstSkill
		label, y, colour = stringFighter+k, k*statRowStep+skillFirstY, skillColour
		if g.party[g.selectedCharacter].Flags&(charSkillRaised<<k) != 0 {
			colour = skillLitColour
		}
	default:
		return
	}
	if full {
		g.printAt(g.text(label), statLabelX+dx, y, colour, 0, printShadow)
	}
	if dx != 0 {
		g.Screen.CopyRect(statValueX+statBackupDX, y, statValueX+dx-statWidth, y, statWidth, statHeight, statPage, g.Screen.Current)
	}
	g.printAt(fmt.Sprintf("%d", value), statValueX+dx, y, colour, 0, printRight|printShadow)
}

// countStats counts the shown stats to the new values (C: ovr_2d22_0025).
func (g *Game) countStats(ch int) {
	var now, step [characterStats]int
	changed := false
	for i := range now {
		now[i] = g.characterStat(ch, i)
		d := now[i] - g.shownStats[i]
		step[i] = d / statSteps
		switch {
		case d < 0:
			if step[i] == 0 {
				step[i] = -1
			}
			changed = true
		case d > 0:
			if step[i] == 0 {
				step[i] = 1
			}
			changed = true
		}
	}
	for changed {
		changed = false
		for i := range now {
			if now[i] == g.shownStats[i] {
				continue
			}
			g.shownStats[i] += step[i]
			if (step[i] < 0 && now[i] > g.shownStats[i]) || (step[i] > 0 && now[i] < g.shownStats[i]) {
				g.shownStats[i] = now[i]
			}
			g.drawStat(i, false, g.shownStats[i])
			changed = true
		}
		g.waitTicks(1)
	}
}

// drawStats draws all five (C: ovr_2d22_002a).
func (g *Game) drawStats(ch int) {
	for i := range characterStats {
		g.drawStat(i, true, g.characterStat(ch, i))
	}
	g.statsShown |= 1 << ch
}

// drawCharacterScreen builds the screen on the back page and copies it over the view
// (C: ovr_2d22_002f).
func (g *Game) drawCharacterScreen(ch int) {
	s := g.Screen
	old := s.SetPage(pageBuild)
	c := &g.party[ch]
	if id := abs(c.ID); id != g.characterPicture {
		g.loadPicture(fmt.Sprintf(characterPictureFmt, characterPictures[id]), pageScratch, nil)
		s.CopyRectColumns(0, 0, pictureColumn, 0, pictureColumns, pictureRows, pageBuild, statPage)
	} else {
		s.CopyRectColumns(pictureColumn, 0, 0, 0, pictureColumns, pictureRows, statPage, pageBuild)
	}
	s.HideMouse()
	s.CopyRectColumns(faceColumn, faceY, faceColumn, faceY, faceColumns, faceRows, pageScreen, pageBuild)
	s.ShowMouse()
	g.drawPortraits()
	g.printAt(c.Name, nameX, nameY, nameColour, 0, printCentred|printShadow)
	g.drawStats(ch)
	for i := range equipmentSlots {
		g.drawEquipmentSlot(i)
	}
	g.printAt(g.text(stringAilments), ailmentsLabelX, ailmentsLabelY, ailmentsLabelColour, 0, printCentred|printShadow)
	for i := range g.shownAilments {
		g.shownAilments[i] = -1
	}
	x, n := 0, 0
	for _, a := range ailmentShapes {
		if a.flag&c.Flags != 0 {
			shape := shapeAt(g.gameShapes, a.shape)
			s.DrawShape(s.Current, shape, x+ailmentX, ailmentY, 0, 0, gfx.ShapeOptions{})
			x += gfx.ShapeWidth(shape) + ailmentGap
			g.shownAilments[n] = a.shape
			n++
		}
	}
	for i := range skillCount {
		level := c.SkillLevels[i]
		have := c.Experience[i] - experienceNeeded[level-1]
		need := experienceNeeded[level] - experienceNeeded[level-1]
		for need >= expBarLimit {
			need >>= 1
			t := have
			have = t >> 1
			if t != 0 && have == 0 {
				have = 1
			}
		}
		g.drawStatBar(expBarX, i*statRowStep+expBarY, expBarW, expBarH, have, need, expBarColour, 0)
	}
	s.HideMouse()
	s.DrawLine(viewLineX1, viewLineY, viewLineX2, viewLineY, shadowColour)
	s.CopyRect(0, 0, viewLeft, 0, viewW, viewH, pageBuild, pageScreen)
	s.CopyRectColumns(faceColumn, faceY, faceColumn, faceY, faceColumns, faceRows, pageBuild, pageScreen)
	s.ShowMouse()
	s.SetPage(old)
}

// drawStatBar fills a bar to value of maximum (C: ovr_2d22_0052).
func (g *Game) drawStatBar(x, y, w, h, value, maximum, colour, background int) {
	if maximum < 1 {
		return
	}
	value = min(max(value, 0), maximum)
	w--
	h--
	if w == 0 || h == 0 {
		return
	}
	n := w * value / maximum
	if n == 0 && value != 0 {
		n = 1
	}
	if n != 0 {
		g.Screen.FillRect(x, y, x+n-1, y+h, colour)
	}
	if n < w && background != 0 {
		g.Screen.FillRect(x+n, y, x+w, y+h, background)
	}
}

// drawEquipmentSlot draws an equipment slot of the selected character
// (C: ovr_2d22_003e).
func (g *Game) drawEquipmentSlot(slot int) {
	c := &g.party[g.selectedCharacter]
	at := inventoryLayouts[inventoryLayoutOf[c.RaceClassSex]][slot]
	x, y := at[0], at[1]
	if x == noInventorySlot || y == noInventorySlot {
		return
	}
	s := g.Screen
	if s.Current == pageScreen {
		x += viewLeft
	}
	it := c.Items[slot]
	frame := emptySlotShapes[slot]
	if it != 0 {
		frame = equipShapeFull
		if slot >= firstRingSlot {
			frame = equipShapeRingFull
		}
	}
	s.HideMouse()
	s.DrawShape(s.Current, shapeAt(g.gameShapes, frame), x, y, 0, 0, gfx.ShapeOptions{})
	if it != 0 {
		if slot >= firstRingSlot {
			x -= ringIconOffset
			y -= ringIconOffset
		}
		g.drawItemIcon(s.Current, it, x+1, y+1)
	}
	s.ShowMouse()
}

// clickPortraitLeft opens the character screen, or shows another character on it
// (C: ovr_2d22_0039).
func (g *Game) clickPortraitLeft(b *gui.Button) bool {
	clock.Disable(playClockSlot)
	if !g.screenFrozen {
		s := g.Screen
		s.HideMouse()
		copy(g.pageBackup[:], s.Page(pageBuild))
		s.CopyRectColumns(0, 0, 0, 0, gfx.Width/gfx.ColumnWidth, gfx.Height, pageScreen, pageBuild)
		copy(g.pageBackup2[:], s.Page(pageBuild))
		s.ShowMouse()
		g.updateFlags |= updateCharacterScreen
		g.lockControls(controlsLockedAll)
	}
	g.selectedCharacter = b.Arg
	g.screenFrozen = true
	g.drawCharacterScreen(g.selectedCharacter)
	g.buildInventoryButtons(g.selectedCharacter)
	return true
}

// clickExitCharacterInventory closes the character screen (C: ovr_2d22_0034).
func (g *Game) clickExitCharacterInventory(*gui.Button) bool {
	g.updateFlags &^= updateCharacterScreen
	g.buildButtons()
	g.screenFrozen = false
	for i := range partySize {
		if g.statsShown&(1<<i) != 0 {
			g.party[i].Flags &^= charSkillsShown
		}
	}
	s := g.Screen
	s.HideMouse()
	copy(s.Page(pageBuild), g.pageBackup2[:])
	old := s.SetPage(pageBuild)
	g.drawPortraits()
	g.drawInventory()
	s.SetPage(old)
	s.CopyPage(pageBuild, pageScreen)
	g.unlockControls()
	copy(s.Page(pageBuild), g.pageBackup[:])
	s.ShowMouse()
	g.characterPicture = -1
	g.copyViewFrame()
	clock.Enable(playClockSlot)
	return true
}

// cannotEquip tells why an item does not fit the slot (C: ovr_2d22_0043).
func (g *Game) cannotEquip(it, mask int) {
	t := &g.itemTypes[g.items[it].Type]
	for i := range equipmentSlots {
		if mask&(1<<i) != 0 {
			k := stringCannotEquip
			if i <= 3 {
				k = stringCannotEquipHand
			}
			g.printMessage(0, g.text(k), g.text(t.Name), g.text(slotNames[i]))
			return
		}
	}
	kind := 0
	if g.items[it].Type == itemTypeNoEquipNote {
		kind = messageEffect
	}
	g.printMessage(kind, g.text(stringNothingThere))
}

// clickCharacterInventorySlot swaps the held item with an equipment slot's
// (C: ovr_2d22_004d).
func (g *Game) clickCharacterInventorySlot(b *gui.Button) bool {
	ch := g.selectedCharacter
	c := &g.party[ch]
	slot := b.Arg
	if held := g.heldItem; held != 0 {
		m := g.itemTypes[g.items[held].Type].Slot
		if m&(1<<slot) == 0 {
			g.cannotEquip(held, m)
			return true
		}
	} else if c.Items[slot] == 0 {
		g.printMessage(0, g.text(slotNames[slot]+equipNoteOffset))
		return true
	}
	it := g.heldItem
	g.setHeldItem(c.Items[slot])
	c.Items[slot] = it
	g.drawEquipmentSlot(slot)
	g.rememberStats(ch)
	if g.heldItem != 0 {
		g.runItemScript(ch, g.heldItem, eventEquipped, 0, 0)
	}
	if it != 0 {
		g.runItemScript(ch, it, eventUnequipped, 0, 0)
	}
	g.drawEquipmentSlot(slot)
	g.drawPortrait(ch)
	g.countStats(ch)
	return true
}

// clickStatsButtons explains the ailment icon clicked (C: ovr_2d22_0057).
func (g *Game) clickStatsButtons(*gui.Button) bool {
	x, _ := input.Click()
	i := min(max(x-ailmentButtonX, 0)/ailmentButtonW, ailmentIcons-1)
	var m int
	switch g.shownAilments[i] {
	case -1:
		return true
	case 0:
		m = stringPoisonNote
	case 2:
		m = stringParalyNote
	case 1:
		m = stringMuteNote
	default:
		m = g.shownAilments[i] // sic: an unknown icon prints its own number as a string id
	}
	g.printMessage(messageNoFade, g.text(m))
	return true
}
