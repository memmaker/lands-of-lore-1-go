package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// The game screen's buttons (C: s015 fn_0ca0_*).
const (
	charCannotAct = charHitOnly | charDead | charFaceOverlay | charParalysed | charMute | charLocked // (C: 0x314c)
	blockShatters = 1 << 4                                                                           // a crystal wall stands in the block: a hit shatters it

	eventAttack       = 1 << 10 // an item script and the block's on an attack (C: 0x400)
	attackHands       = 4       // the first four equipment slots are the hands and the like
	attackSlotStep    = 10      // each further weapon hits a little later
	recoveryWhenHit   = 4
	recoveryBonus     = 4
	itemTypeSpeedRing = 230 // carried, it halves the time to recover

	spellCastTicks = 8
	mageSkill      = 2
	spellLevelRow  = 144 // the submenu's first level's row
	spellLevelH    = 8

	keyCastLeft  = 0x41 // (C: the key put back to repeat a cast with the other hand)
	keyCastRight = 0x42
	buttonRight  = 1 << 8
	buttonLeft   = 1 << 7

	stringCharacterStats  = 0x4047
	stringCannotUseDead   = 0x402c
	stringNoUseWeapon     = 0x4029
	stringNoUseArmour     = 0x402a
	stringNoUseOther      = 0x402b
	stringOneCrown        = 0x402d
	stringCrowns          = 0x402e
	stringCompassBroken   = 0x425b
	stringFacingNorth     = 0x402f
	stringLampFull        = 0x4061
	stringLampFilled      = 0x4062
	stringLampState       = 0x405b
	stringLampEmpty       = 0x405c
	stringLampThirds      = 0x405d
	stringLampFullState   = 0x4060
	stringNotEnoughMagic  = 0x4043
	stringNotEnoughHealth = 0x4179

	itemTypeUsable   = 1 << 0
	itemTypeWeapon   = 1 << 3
	itemTypeArmour   = 1 << 4
	itemTypeDeadUse  = 1 << 5
	wallBlocksThrow  = 1 << 1
	itemTypeLampOil  = 248
	lampOilAmount    = 100
	lampOilMost      = 200
	lampThird        = 33
	soundLampFilled  = 181
	soundThrow       = 18
	throwFlightKind  = 6
	blockCentreUnits = 128
	pickUpSpots      = 21
	pickInBlock      = 1 << 7 // the colour of items in the party's block has this bit
	pickNumber       = pickInBlock - 1
)

// dropSpots is where an item put down at each of the four drop spots lands in a block
// (C: g_23e6, g_23ea); dropSpotOf picks the spot by direction and button (C: f_27dd_0000).
var (
	dropSpotX  = [4]int{64, 192, 64, 192}
	dropSpotY  = [4]int{64, 64, 192, 192}
	dropSpotOf = [16]int{0, 1, 2, 3, 1, 3, 0, 2, 3, 2, 1, 0, 2, 0, 3, 1}
	// pickUpOffsets are the pixels around the click a picked item is looked for at
	// (C: g_23bc, g_23d1).
	pickUpDX = [pickUpSpots]int{0, 0, 1, 0, -1, -1, 1, 1, -1, 0, 2, 0, -2, -1, 1, 2, 2, 1, -1, -2, -2}
	pickUpDY = [pickUpSpots]int{0, -1, 0, 1, 0, -1, -1, 1, 1, -2, 0, 2, 0, -2, -2, -1, 1, 2, 2, 1, -1}
)

// clickAttack has a character hit with what it holds (C: fn_0ca0_0012).
func (g *Game) clickAttack(b *gui.Button) bool {
	ch := b.Arg
	c := &g.party[ch]
	if c.Flags&charCannotAct != 0 {
		return true
	}
	if front := g.blockInDirection(g.currentBlock, g.direction); g.lvl.blocks[front].Flags&blockShatters != 0 {
		g.shatterWall(nil, nil)
		return true
	}
	target := g.monsterInFront(ch)
	delay := 0
	for i := range attackHands {
		if it := c.Items[i]; it != 0 {
			g.runItemScript(ch, it, eventAttack, target, delay)
			g.runBlockScriptWith(g.currentBlock, eventAttack, ch, it, target, delay)
			delay -= attackSlotStep
		}
	}
	if delay == 0 {
		g.runItemScript(ch, 0, eventAttack, target, delay)
		g.runBlockScriptWith(g.currentBlock, eventAttack, ch, 0, target, delay)
	}
	recovery := recoveryWhenHit
	if c.WeaponHit == 0 {
		recovery = g.attackRecovery(ch, spellCastTicks) + recoveryBonus
	}
	if g.carries(ch, itemTypeSpeedRing) {
		recovery >>= 1
	}
	c.Flags |= charHitOnly
	g.selectCharacter(ch)
	g.addStatusEffect(ch, effectAttackRecovery, recovery, true)
	return true
}

// spellTooCostly says, and tells, when a character lacks the points for a spell's
// level (C: fn_0845_1280).
func (g *Game) spellTooCostly(ch, sp, level int) bool {
	c := &g.party[ch]
	if sp == noSpell {
		return true // fix: the original read the row before the table
	}
	if spells[sp].MPCost[level] > c.MP {
		if g.characterSays(stringNotEnoughMagic, c.ID, true) {
			g.printMessage(messagePoison, g.text(stringNotEnoughMagic), c.Name)
		}
		return true
	}
	if spells[sp].HPCost[level] >= c.HP {
		g.printMessage(messageEffect, g.text(stringNotEnoughHealth), c.Name)
		return true
	}
	return false
}

// clickMagic opens the spell level submenu over a character (C: fn_0ca0_0662).
func (g *Game) clickMagic(b *gui.Button) bool {
	ch := b.Arg
	c := &g.party[ch]
	if c.Flags&charCannotAct == 0 && !g.spellTooCostly(ch, g.availableSpells[g.selectedSpell], 0) {
		c.Flags ^= charSelected
		g.drawPortrait(ch)
		g.buildMagicSubmenuButtons(ch)
		g.spellTarget = b.Arg
	}
	return true
}

// clickMagicSubmenu casts the selected spell at the clicked level (C: fn_0ca0_06cb).
func (g *Game) clickMagicSubmenu(b *gui.Button) bool {
	_, y := input.Click()
	level := (y - spellLevelRow) >> 3
	ch := b.Arg
	c := &g.party[ch]
	g.buildButtons()
	sp := g.availableSpells[g.selectedSpell]
	if !g.spellTooCostly(ch, sp, level) {
		c.Flags |= charHitOnly
		c.Flags &^= charSelected
		if g.castSpell(ch, sp, level) {
			g.addStatusEffect(ch, effectAttackRecovery, spellCastTicks, true)
			g.addExperience(ch, mageSkill, level*level)
			g.spellTarget = noCharacter
			return true
		}
		c.Flags &^= charHitOnly
	} else {
		c.Flags &^= charSelected
	}
	g.drawPortrait(ch)
	g.spellTarget = noCharacter
	return true
}

// clickScreen closes the spell submenu; a key click repeats it with the other button
// (C: fn_0ca0_079a).
func (g *Game) clickScreen(b *gui.Button) bool {
	g.party[g.spellTarget].Flags &^= charSelected
	g.drawPortrait(g.spellTarget)
	g.spellTarget = noCharacter
	if b.Flags&buttonLeft == 0 {
		if b.Flags&buttonRight != 0 {
			g.putKeyBack(keyCastLeft)
		} else {
			g.putKeyBack(keyCastRight)
		}
	}
	g.buildButtons()
	return true
}

// clickLiveMagicBarsLeft selects the character and tells its points
// (C: fn_0ca0_030a).
func (g *Game) clickLiveMagicBarsLeft(b *gui.Button) bool {
	g.selectCharacter(b.Arg)
	c := &g.party[b.Arg]
	g.printMessage(0, g.text(stringCharacterStats), c.Name, c.HP, c.MaxHP, c.MP, c.MaxMP)
	return true
}

// clickPortraitRight uses the held item on a character (C: fn_0ca0_04dc).
func (g *Game) clickPortraitRight(b *gui.Button) bool {
	held := g.heldItem
	if held == 0 {
		return true
	}
	c := &g.party[b.Arg]
	flags := g.itemTypes[g.items[held].Type].Flags
	switch {
	case flags&itemTypeUsable != 0:
		if c.Flags&charDead == 0 || flags&itemTypeDeadUse != 0 {
			g.runItemScript(b.Arg, held, eventAttack, 0, 0)
			g.runBlockScriptWith(g.currentBlock, eventAttack, b.Arg, held, 0, 0)
		} else {
			g.printMessage(messageEffect, g.text(stringCannotUseDead), c.Name)
		}
	case flags&itemTypeWeapon != 0:
		g.printMessage(messageEffect, g.text(stringNoUseWeapon))
	case flags&itemTypeArmour != 0:
		g.printMessage(messageEffect, g.text(stringNoUseArmour))
	default:
		g.printMessage(messageEffect, g.text(stringNoUseOther))
	}
	return true
}

// clickSceneDropItem puts the held item down in the party's block or, through a wall
// one can see through, the next (C: fn_0ca0_08af).
func (g *Game) clickSceneDropItem(b *gui.Button) bool {
	if g.updateFlags&updateInScene != 0 || g.heldItem == 0 {
		return false
	}
	block := g.currentBlock
	if b.Arg > 1 {
		block = g.blockInDirection(g.currentBlock, g.direction)
		f := g.lvl.wallFlags[g.lvl.blocks[block].Walls[g.direction^2]]
		if f&wallSeeThrough == 0 || f&wallBlocksThrow != 0 {
			return true
		}
	}
	spot := dropSpotOf[g.direction<<2+b.Arg]
	x, y := blockPosition(block, dropSpotX[spot], dropSpotY[spot])
	g.dropItem(g.heldItem, x, y, 0, true)
	g.setHeldItem(0)
	return true
}

// clickScenePickUpItem picks up the item clicked in the view: the floor items are
// drawn in their own colours to find it (C: fn_0ca0_07e9).
func (g *Game) clickScenePickUpItem(*gui.Button) bool {
	if g.updateFlags&updateInScene != 0 || g.heldItem != 0 {
		return false
	}
	old := g.Screen.SetPage(g.viewPage)
	g.highlightFloorItems()
	x, y := input.Click()
	pick := 0
	for i := range pickUpSpots {
		if pick = g.Screen.Pixel(x+pickUpDX[i], y+pickUpDY[i]); pick != 0 {
			break
		}
	}
	g.Screen.SetPage(old)
	if pick == 0 {
		return false
	}
	block := g.blockInDirection(g.currentBlock, g.direction)
	if pick > pickInBlock {
		block = g.currentBlock
	}
	if it := g.nthDrawnItem(g.lvl.blocks[block].DrawObjects, pick&pickNumber); it != -1 {
		g.takeItemFromBlock(it, block)
		g.setHeldItem(it)
	}
	g.sceneUpdateRequired = true
	return true
}

// nthDrawnItem is the n-th item, from 1, of a draw list, -1 without one
// (C: fn_080e_01a6).
func (g *Game) nthDrawnItem(id, n int) int {
	for id != 0 {
		if !isMonster(id) {
			n--
			if n == 0 {
				return id
			}
			id = g.items[id].DrawNext
		} else {
			id = g.monsters[id&monsterIndex].DrawNext
		}
	}
	return -1
}

// clickSpellListEntry selects a spell (C: fn_0ca0_0630).
func (g *Game) clickSpellListEntry(b *gui.Button) bool {
	if g.selectedSpell != b.Arg {
		g.markSelectedSpell(false)
		g.selectedSpell = b.Arg
		g.markSelectedSpell(true)
		g.drawPortraits()
	}
	return true
}

// markSelectedSpell draws the selected spell's name lit or not (C: fn_0ca0_05bb).
func (g *Game) markSelectedSpell(on bool) {
	const (
		spellListX     = 24
		spellListY     = 15
		spellListStep  = 9
		spellListLit   = 132
		spellListPlain = 1
	)
	colour := spellListPlain
	if on {
		colour = spellListLit
	}
	y := spellListY
	for i := range spellSlots {
		if g.availableSpells[i] == noSpell {
			break
		}
		if g.selectedSpell == i {
			g.Screen.HideMouse()
			g.Screen.DrawText(g.text(spells[g.availableSpells[i]].Name), spellListX, y, colour, 0)
			g.Screen.ShowMouse()
			break
		}
		y += spellListStep
	}
}

// clickSceneThrow throws the held item ahead (C: fn_0ca0_01cf).
func (g *Game) clickSceneThrow(*gui.Button) bool {
	if g.updateFlags&updateInScene != 0 {
		return false
	}
	front := g.blockInDirection(g.currentBlock, g.direction)
	if g.lvl.wallFlags[g.lvl.blocks[front].Walls[g.direction^2]]&wallBlocksThrow != 0 || g.heldItem == 0 {
		return false
	}
	x, y := blockPosition(g.currentBlock, blockCentreUnits, blockCentreUnits)
	if g.launchObject(0, g.heldItem, x, y, throwPower, g.direction<<1, throwFlightKind, g.selectedCharacter, throwFlags) {
		g.playSound(soundThrow, 0)
		g.setHeldItem(0)
	}
	g.sceneUpdateRequired = true
	return true
}

// clickMoneyBox tells the party's money (C: fn_0ca0_0281).
func (g *Game) clickMoneyBox(*gui.Button) bool {
	if g.credits == 1 {
		g.printMessage(0, g.text(stringOneCrown), g.credits)
	} else {
		g.printMessage(0, g.text(stringCrowns), g.credits)
	}
	return true
}

// clickCompass tells the way the party faces (C: fn_0ca0_02b2).
func (g *Game) clickCompass(*gui.Button) bool {
	if !g.flag(flagHasCompass) {
		return false
	}
	if g.compassBroken != 0 {
		if g.characterSays(stringCompassBroken, -1, true) {
			g.printMessage(messageKeepTalk, g.text(stringCompassBroken))
		}
	} else {
		g.printMessage(0, g.text(stringFacingNorth+g.direction))
	}
	return true
}

// clickAutomap opens the map (C: fn_0ca0_039a).
func (g *Game) clickAutomap(*gui.Button) bool {
	if !g.flag(flagHasAutomap) {
		return false
	}
	g.showAutomap()
	g.drawGameScreen()
	g.setBrightness(g.Palette[:], g.brightness, g.lampLevel)
	return true
}

// clickLamp fills the lamp with held oil or tells how full it is (C: fn_0ca0_03cf).
func (g *Game) clickLamp(*gui.Button) bool {
	if !g.flag(flagHasLamp) {
		return false
	}
	held := g.heldItem
	if g.items[held].Type == itemTypeLampOil {
		if g.lampEffect+lampOilAmount > lampOilMost {
			g.printMessage(0, g.text(stringLampFull))
		} else {
			g.printMessage(0, g.text(stringLampFilled))
			g.freeItem(held)
			g.playSound(soundLampFilled, 0)
			g.setHeldItem(0)
			g.lampEffect += lampOilAmount
			if g.brightness != 0 {
				g.setBrightness(g.Palette[:], g.brightness, g.lampLevel)
			}
		}
		return true
	}
	m := g.lampEffect/lampThird + stringLampThirds
	if g.lampEffect == 0 {
		m = stringLampEmpty
	}
	if g.lampEffect >= lampOilAmount {
		m = stringLampFullState
	}
	g.printMessage(0, g.text(stringLampState), g.text(m))
	if g.brightness != 0 {
		g.setBrightness(g.Palette[:], g.brightness, g.lampLevel)
	}
	return true
}
