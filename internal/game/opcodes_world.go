package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
)

// setWalls sets block b's wall dir to type v, or with allSides all four; a whole door
// frame block shows on the automap as one (C: fn_130a_0360).
func (g *Game) setWalls(b, dir, v int) {
	blk := &g.lvl.blocks[b]
	if dir == allSides {
		blk.Walls = [directionCount]int{v, v, v, v}
		if g.lvl.automapData[v] == automapDoorType {
			blk.Flags &^= blockSeen
			blk.Flags |= blockAutomapMarker
		} else {
			blk.Flags &^= blockAutomapMarker
		}
	} else {
		blk.Walls[dir] = v
	}
	g.noteBlockChanged(b)
}

const allSides = -1

// wallKillsMonsters marks wall types that crush the monsters in their block
// (C: g_531e & 4).
const wallKillsMonsters = 1 << 2

// opSetWall sets a block's wall: block, side or -1 for all, type (C: fn_0fbe_007f).
func (g *Game) opSetWall(s *emc.State) int {
	b, dir, v := s.Arg(0), s.Arg(1), s.Arg(2)
	if g.lvl.wallFlags[v]&wallKillsMonsters != 0 {
		g.killMonstersOn(b)
	}
	g.setWalls(b, dir, v)
	return 1
}

// opToggleWall switches a block's wall between two types: block, side or -1, a, b; a
// wall that is a becomes b, any other becomes a (C: fn_0fbe_1063).
func (g *Game) opToggleWall(s *emc.State) int {
	b, dir, a, other := s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3)
	side := dir
	if dir == allSides {
		side = 0
	}
	v := a
	if g.lvl.blocks[b].Walls[side] == a {
		v = other
	}
	g.setWalls(b, dir, v)
	return 0
}

// opDrawScene draws the view on a page (C: fn_0fbe_014b).
func (g *Game) opDrawScene(s *emc.State) int {
	g.drawScene(s.Arg(0))
	return 1
}

// opPrintMessage prints a string of the script's on the message line in white
// (C: fn_0fbe_021a).
func (g *Game) opPrintMessage(s *emc.State) int {
	const white = 254
	g.Text.SelectWindow(textWindowMessage)
	g.Text.Foreground = white
	g.Text.Printf(s.StringArg(0))
	return 1
}

// opWait lets ticks pass, the game idling meanwhile (C: fn_0fbe_0273).
func (g *Game) opWait(s *emc.State) int {
	g.waitTicks(s.Arg(0))
	return 1
}

// waitTicks (C: fn_0845_0bb6).
func (g *Game) waitTicks(ticks int) {
	clock.WaitUntil(clock.Ticks()+ticks, g.idleWork)
}

// opTakeHeldItem takes the item off the cursor and frees it; it returns the item
// (C: fn_0fbe_0322). With an empty hand it frees item 0 (sic).
func (g *Game) opTakeHeldItem(s *emc.State) int {
	it := g.heldItem
	g.freeItem(it)
	g.setHeldItem(0)
	return it
}

// Item type flags and events of setHeldItem.
const (
	itemTypeScriptOnTake = 1 << 7
	itemEventTake        = 1 << 10
	stringTaken          = 0x403e
)

// setHeldItem puts item on the cursor; an item type with a script runs it first, and
// may use the item up. Picking an item up says so on the message line
// (C: fn_0845_0acf).
func (g *Game) setHeldItem(item int) {
	it := item
	if it != 0 && g.itemTypes[g.items[it].Type].Flags&itemTypeScriptOnTake != 0 {
		g.runItemScript(noCharacter, it, itemEventTake, 0, 0)
		if g.items[it].Flags&itemFree != 0 {
			it = 0
		}
	}
	hot := 0
	if it != 0 && !g.flag(flagBusy) {
		hot = heldCursorHotSpot
		if g.controlMode == 0 || g.textEnabled {
			g.printMessage(0, g.text(stringTaken), g.text(g.itemTypes[g.items[it].Type].Name))
		}
	}
	g.heldItem = it
	g.Screen.SetMouseShape(hot, hot, g.itemIcon(it))
}

// placeItem puts item it on block of level at (x, y) inside it, height sub
// (C: fn_130a_0052).
func (g *Game) placeItem(it, level, block, x, y, sub int) {
	item := &g.items[it]
	item.X, item.Y = blockPosition(block, x, y)
	if item.Block != 0 {
		g.takeItemFromBlock(it, item.Block)
	}
	if g.level == level {
		g.dropItem(it, item.X, item.Y, sub, true)
	} else {
		item.Level = level
		item.Block = block
		item.Sub = sub
		item.Flags |= itemOnFloor
	}
}

// eventItemTaken is the block script event of an item leaving the block.
const eventItemTaken = 1 << 8

// takeItemFromBlock takes item it off block b (C: fn_07bc_0367).
func (g *Game) takeItemFromBlock(it, b int) {
	blk := &g.lvl.blocks[b]
	g.removeObject(&blk.Objects, it)
	g.removeFromDrawList(blk, it)
	g.runBlockScriptWith(b, eventItemTaken, -1, it, 0, 0)
	g.items[it].Block = 0
	g.items[it].Level = 0
}

// removeFromDrawList takes id out of a block's draw list (C: fn_080e_02f5). The
// original printed an error when it was not there.
func (g *Game) removeFromDrawList(b *Block, id int) {
	if b.Direction == noDirection {
		return
	}
	for list := &b.DrawObjects; *list != 0; {
		next := g.drawNextOf(*list)
		if *list == id {
			*list = *next
			*next = 0
			return
		}
		list = next
	}
}

// opPlaceItem moves an item: item, level, block, x, y, height (C: fn_0fbe_037f).
func (g *Game) opPlaceItem(s *emc.State) int {
	g.placeItem(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3)&lowByteMask, s.Arg(4)&lowByteMask, s.Arg(5))
	return 1
}

// opCreateItem makes an item and places it: type, flags, more flags, level, block, x,
// y, height; it returns the item or -1 (C: fn_0fbe_03fc).
func (g *Game) opCreateItem(s *emc.State) int {
	it := g.newItem(s.Arg(0), s.Arg(1), s.Arg(2))
	if it == 0 {
		return -1
	}
	g.placeItem(it, s.Arg(3), s.Arg(4), s.Arg(5), s.Arg(6), s.Arg(7))
	return it
}

// opPlayAnimationFrames shows frames from to to of a scene animation, a delay apart
// (C: fn_0fbe_08cb).
func (g *Game) opPlayAnimationFrames(s *emc.State) int {
	a := &g.animations[s.Arg(0)]
	from, to, delay := s.Arg(1), s.Arg(2), s.Arg(3)
	step := 1
	if to < from {
		step = -1
	}
	for fr := from; fr != to+step; fr += step {
		t := clock.Ticks() + delay
		g.Screen.HideMouseIn(a.x, a.y, a.x+a.wsa.width(), a.y+a.wsa.height())
		if a.flags&animOnBuildPage != 0 {
			mode := gfx.BufferTransparent
			if a.flags&animRemapped != 0 {
				mode |= gfx.BufferRemapped
			}
			mode <<= wsaModeShift
			g.Screen.CopyRect(viewLeft, viewTop, viewLeft, viewTop, viewWidth, viewHeight, pageOtherView, pageBuild)
			g.drawWSA(a.wsa, fr-1, a.x, a.y, pageBuild, mode, g.lvl.tlcTable, g.lvl.tlcLevels)
			g.Screen.CopyRect(viewLeft, viewTop, viewLeft, viewTop, viewWidth, viewHeight, pageBuild, pageScreen)
		} else {
			g.drawWSA(a.wsa, fr-1, a.x, a.y, pageScreen, 0, nil, nil)
		}
		g.Screen.ShowMouseIn()
		clock.WaitUntil(t, g.idleWork)
	}
	return 1
}

// opPartySound plays the first hero's sound (C: fn_0fbe_319d). Only the first
// character can match: the sounds for the others were for slots 5, 8 and 9 (sic).
func (g *Game) opPartySound(s *emc.State) int {
	snd := -1
	for i := range partySize {
		c := &g.party[i]
		if c.Flags&charPresent != 0 && c.ID < 0 {
			if i == 0 {
				snd = 136
			}
			break
		}
	}
	g.playSound(snd, 1)
	return 1
}

// opMouseIn says whether the cursor is in a rectangle (C: fn_0fbe_0a52, fn_0845_03b4).
func (g *Game) opMouseIn(s *emc.State) int {
	x, y := g.Screen.Mouse()
	return boolInt(rect{s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3)}.has(x, y))
}

// opSetAnimationSequence sets a scene animation's sequence: slot, sequence, first,
// last, loops, next, delay, a word, sound, sound frame (C: fn_0fbe_0b0f).
func (g *Game) opSetAnimationSequence(s *emc.State) int {
	p := &g.animations[s.Arg(0)].sequences[s.Arg(1)]
	for i := range p {
		p[i] = s.Arg(2 + i)
	}
	return 0
}

// opStartAnimationSequence (C: fn_0fbe_0c11).
func (g *Game) opStartAnimationSequence(s *emc.State) int {
	g.startAnimationSequence(s.Arg(0), s.Arg(1))
	return 1
}

// startAnimationSequence starts sequence seq of slot n and shows its first frame
// (C: fn_130a_0c28).
func (g *Game) startAnimationSequence(n, seq int) {
	a := &g.animations[n]
	a.sequence = int(int8(seq))
	a.playing = true
	a.next = clock.Ticks() + a.delay
	a.frame = a.sequences[a.sequence][0]
	a.loops = 0
	g.Screen.HideMouseIn(a.x, a.y, a.x+a.wsa.width(), a.y+a.wsa.height())
	g.drawWSA(a.wsa, a.frame-1, a.x, a.y, pageScreen, 0, nil, nil)
	g.Screen.ShowMouseIn()
}

// opFadeToScenePalette fades to the scene animations' palette (C: fn_0fbe_0c6a).
func (g *Game) opFadeToScenePalette(s *emc.State) int {
	g.Screen.FadePalette(g.scenePalette[:], fadeTicks, nil)
	g.paletteState = paletteNormal
	return 1
}

// opLoadPicture loads a CPS picture onto page 3 with its palette into the scene
// palette, and copies it to a page unless that is 2 (C: fn_0fbe_0c8a).
func (g *Game) opLoadPicture(s *emc.State) int {
	g.loadPicture(s.StringArg(0), pageScratch, g.scenePalette[:])
	if s.Arg(1) != pageBuild {
		g.Screen.HideMouse()
		g.Screen.CopyPage(pageScratch, s.Arg(1))
		g.Screen.ShowMouse()
	}
	return 1
}

// opStopAnimation stops a scene animation at once; slot 5 is also closed
// (C: fn_0fbe_0d17, fn_130a_0cf6).
func (g *Game) opStopAnimation(s *emc.State) int {
	g.stopAnimation(s.Arg(0))
	return 1
}

// stopAnimation stops scene animation n at once; slot 5 is also closed
// (C: fn_130a_0cf6).
func (g *Game) stopAnimation(n int) {
	a := &g.animations[n]
	a.playing = false
	a.stopAsked = false
	if n == speechAnimation {
		a.wsa = nil
	}
}

// speechAnimation is the slot speaking characters' animations use.
const speechAnimation = 5

// opAskAnimationStop lets a scene animation stop at the end of its sequence
// (C: fn_0fbe_0d34).
func (g *Game) opAskAnimationStop(s *emc.State) int {
	g.animations[s.Arg(0)].stopAsked = true
	return 1
}

// opPlayAnimationOnce plays a scene animation's sequence to its end (C: fn_0fbe_0d54).
func (g *Game) opPlayAnimationOnce(s *emc.State) int {
	a := &g.animations[s.Arg(0)]
	a.playing = true
	a.stopAsked = true
	return 1
}

// opToggleDoor opens a closed door block or closes an open one: block, way
// (C: fn_0fbe_0f94).
func (g *Game) opToggleDoor(s *emc.State) int {
	b := s.Arg(0)
	open := 1
	if g.lvl.wallFlags[g.lvl.blocks[b].Walls[0]]&wallIsDoor != 0 {
		open = 0
	}
	g.toggleDoor(b, open, s.Arg(1))
	return 1
}

// opPartyHasWeapon says whether a character holds an item of script 0 or 2 in a hand
// slot (C: fn_0fbe_0fe7).
func (g *Game) opPartyHasWeapon(s *emc.State) int {
	const handSlots = 4
	for ch := range partySize {
		c := &g.party[ch]
		if c.Flags&charPresent == 0 {
			continue
		}
		for _, it := range c.Items[:handSlots] {
			if it != 0 {
				if sc := g.itemTypes[g.items[it].Type].Script; sc == 0 || sc == 2 {
					return 1
				}
			}
		}
	}
	return 0
}

// opMarkAutomap shows a block on the automap as one or not (C: fn_0fbe_313b).
func (g *Game) opMarkAutomap(s *emc.State) int {
	b := &g.lvl.blocks[s.Arg(0)]
	if s.Arg(1) != 0 {
		b.Flags &^= blockSeen
		b.Flags |= blockAutomapMarker
	} else {
		b.Flags &^= blockAutomapMarker
	}
	return 1
}

// opAnyMonsterAwake says whether a monster of a type, or any for -1, is not waiting
// with the others; the first one found decides, and with none it is 1
// (C: fn_0fbe_254c).
func (g *Game) opAnyMonsterAwake(s *emc.State) int {
	for i := range g.monsters {
		m := &g.monsters[i]
		if m.TypeIndex == s.Arg(0) || s.Arg(0) == -1 {
			return boolInt(m.Mode != monsterModeAll)
		}
	}
	return 1
}

// opCopyRect copies a rectangle between pages: x, y, to x, to y, w, h, from, to
// (C: fn_0fbe_1126).
func (g *Game) opCopyRect(s *emc.State) int {
	g.Screen.HideMouse()
	g.Screen.CopyRect(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4), s.Arg(5), s.Arg(6), s.Arg(7))
	g.Screen.ShowMouse()
	return 1
}

// opFadeOutView (C: fn_0fbe_11cb).
func (g *Game) opFadeOutView(s *emc.State) int {
	g.fadeOutView(fadeTicks)
	return 1
}

// opFadeInLevel fades the level's colours back in over the scene palette
// (C: fn_0fbe_11db).
func (g *Game) opFadeInLevel(s *emc.State) int {
	at := levelColours * 3
	copy(g.scenePalette[at:], g.Palette[at:])
	g.keepInterfaceColours(g.scenePalette[:])
	g.Screen.FadePalette(g.scenePalette[:], fadeTicks, nil)
	g.paletteState = paletteNormal
	return 1
}

// opRedrawAfterScene (C: fn_0fbe_122c).
func (g *Game) opRedrawAfterScene(s *emc.State) int {
	g.redrawAfterScene()
	return 1
}

// opMonsterInFront is the monster ahead nearest to a party member (C: fn_0fbe_14fb).
func (g *Game) opMonsterInFront(s *emc.State) int { return g.monsterInFront(s.Arg(0)) }

// opNothingZero (C: fn_0fbe_1515).
func (g *Game) opNothingZero(s *emc.State) int { return 0 }

// opAttackHits rolls an attack: attacker, defender, skill (C: fn_0fbe_151c).
func (g *Game) opAttackHits(s *emc.State) int { return g.attackHits(s.Arg(0), s.Arg(1), s.Arg(2)) }

// opDamage deals damage to a fighter or, for -1, each character: who, damage,
// source, skill, kind (C: fn_0fbe_155a).
func (g *Game) opDamage(s *emc.State) int {
	if s.Arg(0) == -1 {
		for ch := range partySize {
			g.damageFighter(ch, s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4))
		}
	} else {
		g.damageFighter(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4))
	}
	return 1
}

// opDialogueChoices shows up to three answers by string id (C: fn_0fbe_170f).
func (g *Game) opDialogueChoices(s *emc.State) int {
	g.showDialogueChoices(s.Arg(0), []string{g.text(s.Arg(1)), g.text(s.Arg(2)), g.text(s.Arg(3))})
	return 1
}

// opCredits gives the party money, or takes it for a negative amount
// (C: fn_0fbe_1784).
func (g *Game) opCredits(s *emc.State) int {
	if d := s.Arg(0); d < 0 {
		g.takeCredits(-d, true)
	} else {
		g.giveCredits(d, true)
	}
	return 1
}

// opGiveHeldItem puts a new item on an empty cursor: type, flags, more flags
// (C: fn_0fbe_181b).
func (g *Game) opGiveHeldItem(s *emc.State) int {
	if g.heldItem != 0 {
		return 0
	}
	g.setHeldItem(g.newItem(s.Arg(0), s.Arg(1), s.Arg(2)))
	return 1
}

// itemSounds are the sounds of items by their skill's low bits (C: g_2798).
var itemSounds = [4]int{12, 62, 63, -1}

const itemSoundDefault = 12

// opItemSound plays the sound of an item type: the default for four runs of types,
// else by its skill (C: fn_0fbe_308b).
func (g *Game) opItemSound(s *emc.State) int {
	t := s.Arg(0)
	if (t >= 70 && t <= 74) || (t >= 81 && t <= 89) || (t >= 93 && t <= 97) || (t >= 102 && t <= 106) {
		g.playSound(itemSoundDefault, 1)
	} else {
		g.playSound(itemSounds[g.itemTypes[t].Skill&3], 1)
	}
	return 1
}

// opGiveInventoryItem makes an item and puts it in the inventory; with no room it is
// freed again and the result is 0 (C: fn_0fbe_1870).
func (g *Game) opGiveInventoryItem(s *emc.State) int {
	it := g.newItem(s.Arg(0), s.Arg(1), s.Arg(2))
	if !g.addToInventory(it) {
		g.freeItem(it)
		return 0
	}
	return 1
}

// addToInventory puts item in the first free inventory slot from the shown ones on,
// scrolling the bar until it shows; false when full (C: fn_0c59_03a8).
func (g *Game) addToInventory(item int) bool {
	s := 0
	i := 0
	for ; i < inventorySlots; i++ {
		s = (g.inventoryStart + i) % inventorySlots
		if g.inventory[s] == 0 {
			break
		}
	}
	if i == inventorySlots {
		return false
	}
	for g.inventoryStart-s > 0 || g.inventoryStart+inventoryShown <= s {
		g.inventoryStart = (g.inventoryStart + 1) % inventorySlots
		g.drawInventory()
	}
	g.inventory[s] = item
	g.drawInventory()
	return true
}

// opAddExperience: character, skill, points (C: fn_0fbe_18cb).
func (g *Game) opAddExperience(s *emc.State) int {
	g.addExperience(s.Arg(0), s.Arg(1), s.Arg(2))
	return 1
}

// opHeldItem (C: fn_0fbe_1a1d).
func (g *Game) opHeldItem(s *emc.State) int { return g.heldItem }

// opPlaySound (C: fn_0fbe_1a72).
func (g *Game) opPlaySound(s *emc.State) int {
	g.playSound(s.Arg(0), 0)
	return 1
}

// opDialogueAnswer waits for the answer to the shown choices (C: fn_0fbe_177a).
func (g *Game) opDialogueAnswer(s *emc.State) int { return g.dialogueAnswer() }

// Monster fields scripts read and set (C: fn_0fbe_1b20, fn_0fbe_1c17).
const (
	monsterFieldMode      = 0
	monsterFieldHitPoints = 1
	monsterFieldBlock     = 2
	monsterFieldFacing    = 3
	monsterFieldType      = 4
	monsterFieldMaxHP     = 5
	monsterFieldFlags     = 6
	monsterFieldTypeFlags = 7
	monsterFieldAnimType  = 8
	monsterMoveBlockers   = 7 // walls, monsters and the party
)

// opSetMonsterField: monster, field, value (C: fn_0fbe_1b20).
func (g *Game) opSetMonsterField(s *emc.State) int {
	if s.Arg(0) == -1 {
		return 1
	}
	m := &g.monsters[s.Arg(0)&monsterIndex]
	v := s.Arg(2)
	switch s.Arg(1) {
	case monsterFieldFlags:
		m.Flags |= v
	case monsterFieldBlock:
		x, y := blockPosition(v, m.X&lowByteMask, m.Y&lowByteMask)
		if g.stepBlocked(x, y, m, monsterMoveBlockers) == 0 {
			g.moveMonster(m, x, y)
		}
	case monsterFieldMode:
		g.setMonsterMode(m, v)
	case monsterFieldHitPoints:
		m.HitPoints = v
	case monsterFieldFacing:
		g.setMonsterDirection(m, v<<1)
	}
	return 1
}

// opGetMonsterField: monster, field (C: fn_0fbe_1c17).
func (g *Game) opGetMonsterField(s *emc.State) int {
	if s.Arg(0) == -1 {
		return 0
	}
	m := &g.monsters[s.Arg(0)&monsterIndex]
	switch s.Arg(1) {
	case monsterFieldAnimType:
		return g.lvl.monsterAnimType[m.Type.shapeIndex]
	case monsterFieldTypeFlags:
		return m.Type.flags
	case monsterFieldFlags:
		return m.Flags
	case monsterFieldHitPoints:
		return m.HitPoints
	case monsterFieldMaxHP:
		return m.Type.hitPoints
	case monsterFieldMode:
		return m.Mode
	case monsterFieldBlock:
		return m.Block
	case monsterFieldFacing:
		return m.Facing
	case monsterFieldType:
		return m.TypeIndex
	}
	return -1
}

// opCharacterSays has a character say a string: character, style, string id
// (C: fn_0fbe_1cf7).
func (g *Game) opCharacterSays(s *emc.State) int {
	g.stopSpeech(true)
	g.endTalk()
	return boolInt(g.characterText(s.Arg(0), s.Arg(1), true, g.text(s.Arg(2)), s.Arg(2), scriptArgs(s), 3))
}

// opIdle runs the game's idle work once (C: fn_0fbe_1cea).
func (g *Game) opIdle(s *emc.State) int {
	g.idleWork()
	return 1
}

// opPlaySoundAt plays a sound from a block, -1 the party's (C: fn_0fbe_1a93).
func (g *Game) opPlaySoundAt(s *emc.State) int {
	b := s.Arg(1)
	if b == -1 {
		b = g.currentBlock
	}
	g.playSoundAt(s.Arg(0), b)
	return 1
}

// opPlayMusic (C: fn_0fbe_1e0e).
func (g *Game) opPlayMusic(s *emc.State) int { return g.playMusic(s.Arg(0)) }

// opKillMonstersOn (C: fn_0fbe_334b).
func (g *Game) opKillMonstersOn(s *emc.State) int {
	g.killMonstersOn(s.Arg(0))
	return 1
}

// opCountItemsOn counts the items in a block (C: fn_0fbe_1e28).
func (g *Game) opCountItemsOn(s *emc.State) int {
	n := 0
	for id := g.lvl.blocks[s.Arg(0)].Objects; id != 0; id = *g.objectNext(id) {
		if id&monsterFlag == 0 {
			n++
		}
	}
	return n
}

// opSkillTest picks the character best at a skill and rolls whether they succeed:
// skill, bonus; it returns the character or -1 (C: fn_0fbe_1e6c).
func (g *Game) opSkillTest(s *emc.State) int {
	bestIdx, best := -1, 0
	k := s.Arg(0)
	for i := range g.activeCharacters() {
		c := &g.party[i]
		if v := (c.SkillModifiers[k]+c.SkillLevels[k])*10 + s.Arg(1) + 25; v > best {
			best, bestIdx = v, i
		}
	}
	if dice(1, 100) <= best {
		return bestIdx
	}
	return -1
}

// opHeal heals a character: character, points, redraw, with the healing animation
// (C: fn_0fbe_1d58).
func (g *Game) opHeal(s *emc.State) int {
	if s.Arg(3) != 0 {
		g.healAnimation(s.Arg(0), s.Arg(1))
	} else {
		g.heal(s.Arg(0), s.Arg(1), true)
		if s.Arg(2) != 0 {
			g.drawPortrait(s.Arg(0))
		}
	}
	return 1
}

// opCountLiveMonsters (C: fn_0fbe_1f05).
func (g *Game) opCountLiveMonsters(s *emc.State) int {
	n := 0
	for i := range g.monsters {
		if m := &g.monsters[i]; m.HitPoints > 0 && m.Mode != monsterDead {
			n++
		}
	}
	return n
}

// Finale modes by the first character's id (C: fn_0fbe_3282).
var finaleModes = map[int]int{-9: 1, -1: 2, -5: 3}

// opFinale ends the game and runs the finale for the hero who led the party
// (C: fn_0fbe_3282, which ran FINALE.EXE with the mode, the difficulty and the
// player's name).
func (g *Game) opFinale(s *emc.State) int {
	for g.updateSpeech() != 0 {
		g.waitTicks(1)
	}
	g.runFinale(finaleModes[g.party[0].ID], g.difficulty())
	return 0
}

func (g *Game) runFinale(mode, difficulty int) { notPorted("FINALE.EXE (the ending)") }

// opStopTalk (C: fn_0fbe_1f39).
func (g *Game) opStopTalk(s *emc.State) int {
	g.stopSpeech(true)
	g.endTalk()
	return 1
}

// opAttackDamage (C: fn_0fbe_2087).
func (g *Game) opAttackDamage(s *emc.State) int {
	return g.attackDamage(s.Arg(0), s.Arg(1), s.Arg(2))
}

// opScaledDamage (C: fn_0fbe_20c5).
func (g *Game) opScaledDamage(s *emc.State) int { return g.scaledDamage(s.Arg(0)) }

// opInParty says whether the character with an id is in the party, living or dead
// (C: fn_0fbe_1f4e).
func (g *Game) opInParty(s *emc.State) int {
	for i := range partySize {
		if c := &g.party[i]; c.Flags&(charPresent|charDead) != 0 && c.ID == s.Arg(0) {
			return 1
		}
	}
	return 0
}

// opMessage prints a message: kind, string id, sound, up to seven arguments
// (C: fn_0fbe_1fa0).
func (g *Game) opMessage(s *emc.State) int {
	args := make([]any, 7)
	for i := range args {
		args[i] = s.Arg(3 + i)
	}
	g.printMessage(s.Arg(0), g.text(s.Arg(1)), args...)
	if n := s.Arg(2); n > 0 {
		g.playSound(n, 0)
	}
	return 1
}

// opDestroyItem takes an item off its block and frees it (C: fn_0fbe_20df).
func (g *Game) opDestroyItem(s *emc.State) int {
	it := s.Arg(0)
	if b := g.items[it].Block; b != 0 {
		g.takeItemFromBlock(it, b)
	}
	g.freeItem(it)
	return 1
}

// opDamageOfKind (C: fn_0fbe_2126).
func (g *Game) opDamageOfKind(s *emc.State) int {
	return g.damageOfKind(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4))
}

// opLaunchObject sends an item flying: kind, item, block, x, y, power, direction,
// a, owner; with an owner not a monster from the block's centre. An item that cannot
// fly is freed (C: fn_0fbe_2188).
func (g *Game) opLaunchObject(s *emc.State) int {
	const launchFlags = 1<<6 - 1
	var x, y int
	if isMonster(s.Arg(8)) {
		x, y = blockPosition(s.Arg(2), s.Arg(3), s.Arg(4))
	} else {
		x, y = blockPosition(s.Arg(2), blockCentreSide, blockCentreSide)
	}
	if !g.launchObject(s.Arg(0), s.Arg(1), x, y, s.Arg(5), s.Arg(6)<<1, s.Arg(7), s.Arg(8), launchFlags) {
		g.freeItem(s.Arg(1))
		return 0
	}
	return 1
}

// opClearStatus ends a character's status effects from one to another
// (C: fn_0fbe_2290).
func (g *Game) opClearStatus(s *emc.State) int {
	g.clearStatusEffects(&g.party[s.Arg(0)], s.Arg(1), s.Arg(2))
	return 1
}

// opInventoryFull (C: fn_0fbe_22dc).
func (g *Game) opInventoryFull(s *emc.State) int {
	for _, it := range g.inventory {
		if it == 0 {
			return 0
		}
	}
	return 1
}

// opMoveBlockContents moves a block's floor items to another block, maybe on another
// level, and removes its monsters: block, to block, level, monsters, run the drop
// event, items (C: fn_0fbe_22fb).
func (g *Game) opMoveBlockContents(s *emc.State) int {
	found := false
	b, to, level := s.Arg(0), s.Arg(1), s.Arg(2)
	doMonsters, doItems := s.Arg(3) != 0, s.Arg(5) != 0
	for next := g.lvl.blocks[b].Objects; next != 0; {
		id := next
		next = *g.objectNext(id)
		if !isMonster(id) {
			it := &g.items[id]
			if it.Flags&itemOnFloor == 0 || !doItems {
				continue
			}
			g.placeItem(id, level, to, it.X&lowByteMask, it.Y&lowByteMask, it.Sub)
			if s.Arg(4) != 0 && g.level == level {
				g.runBlockScriptWith(to, eventItemDrop, -1, id, 0, 0)
			}
			found = true
		} else {
			if !doMonsters {
				continue
			}
			m := &g.monsters[id&monsterIndex]
			g.setMonsterMode(m, monsterModeRemove)
			g.noteBlockChanged(m.Block)
			g.moveMonster(m, 0, 0)
			found = true
		}
	}
	return boolInt(found)
}

// opLamp lights or puts out the lamp (C: fn_0fbe_248a).
func (g *Game) opLamp(s *emc.State) int {
	if s.Arg(1) != 0 {
		g.showLamp()
	} else {
		g.hideLamp(false)
	}
	return 1
}

// opDrawGameScreen (C: ovr_2d2c_00c0).
func (g *Game) opDrawGameScreen(s *emc.State) int {
	g.drawGameScreen()
	return 1
}

// opPrintString prints a string on the message line (C: fn_0fbe_24b4).
func (g *Game) opPrintString(s *emc.State) int {
	g.printText(textWindowMessage, g.text(s.Arg(0)), scriptArgs(s), 1)
	return 1
}

// opTrue (C: ovr_2d77_0020).
func (g *Game) opTrue(s *emc.State) int { return 1 }

// opQuitGame ends the program (C: fn_0fbe_25c4, fn_0845_0bf6).
func (g *Game) opQuitGame(s *emc.State) int {
	g.quit = true
	g.exitProgram = true
	return 1
}

// opRemoveMonster (C: fn_0fbe_25d1). The test for a monster is always true (sic).
func (g *Game) opRemoveMonster(s *emc.State) int {
	m := &g.monsters[s.Arg(0)&monsterIndex]
	g.setMonsterMode(m, monsterModeRemove)
	g.noteBlockChanged(m.Block)
	g.moveMonster(m, 0, 0)
	return 1
}

// opSetScriptValue (C: fn_0fbe_262d).
func (g *Game) opSetScriptValue(s *emc.State) int {
	g.scriptValue = s.Arg(0)
	return 1
}

// opKeyPressed says whether a key was pressed, and takes it or with keep puts it back
// for later (C: fn_0fbe_2647).
func (g *Game) opKeyPressed(s *emc.State) int {
	g.GUI.Changed = true
	g.updateSpeech()
	if input.Peek() != 0 {
		k := input.Get()
		if k != 0 && k&input.Released == 0 {
			if s.Arg(0) != 0 {
				input.Put(k)
				g.sceneRegionActive = true
			} else {
				input.Flush()
			}
			return 1
		}
	}
	return 0
}

// Print flags of opPrintInWindow.
const (
	printClear = 1 << 0
	printHome  = 1<<0 | 1<<1
)

// opPrintInWindow prints a string in a text window: window, flags, string id
// (C: fn_0fbe_269d).
func (g *Game) opPrintInWindow(s *emc.State) int {
	w, f := s.Arg(0), s.Arg(1)
	g.Text.SelectWindow(w)
	if f&printClear != 0 {
		g.Text.Clear()
	}
	if f&printHome != 0 {
		g.Screen.Windows[w].Column, g.Screen.Windows[w].Row = 0, 0
	}
	g.printText(w, g.text(s.Arg(2)), scriptArgs(s), 3)
	return 1
}

// opCountMonstersOfTypes counts the monsters there of the types listed up to a -1
// (C: fn_0fbe_2725).
func (g *Game) opCountMonstersOfTypes(s *emc.State) int {
	mask := 0
	for i := 0; s.Arg(i) != -1; i++ {
		mask |= 1 << s.Arg(i)
	}
	n := 0
	for i := range g.monsters {
		if m := &g.monsters[i]; mask&(1<<m.TypeIndex) != 0 && m.Mode < monsterModeRemove {
			n++
		}
	}
	return n
}

// opAnimateWall sets a wall to frame n of a list: block, side, n, count, frames
// (C: fn_0fbe_2798).
func (g *Game) opAnimateWall(s *emc.State) int {
	n, count := uint16(s.Arg(2)), uint16(s.Arg(3))
	g.setWalls(s.Arg(0), s.Arg(1), s.Arg(4+int(n%count)))
	return 0
}

// opHasItemType says whether an item of a type is in the inventory or held (unless
// the third argument), or carried by a character or anyone for -1 (C: fn_0fbe_27f6).
func (g *Game) opHasItemType(s *emc.State) int {
	who, t := s.Arg(0), s.Arg(1)
	if s.Arg(2) == 0 {
		for _, it := range g.inventory {
			if it != 0 && g.items[it].Type == t {
				return 1
			}
		}
		if g.items[g.heldItem].Type == t {
			return 1
		}
	}
	lo, hi := who, who
	if who == -1 {
		lo, hi = 0, partySize-1
	}
	for i := lo; i <= hi; i++ {
		if g.carries(i, t) {
			return 1
		}
	}
	return 0
}

// opSetDoorHandled tells a door's click that its script has dealt with it
// (C: fn_0fbe_28b5).
func (g *Game) opSetDoorHandled(s *emc.State) int {
	g.doorHandled = s.Arg(0)
	return g.doorHandled
}

// opFindItem finds the next item in a block, at a position inside it or anywhere for
// -1: block, x, y, start over (C: fn_0fbe_2922). Going on from the last item the
// original read the block numbered like that item instead of the item (sic).
func (g *Game) opFindItem(s *emc.State) int {
	x, y := s.Arg(1), s.Arg(2)
	if x != -1 {
		x &= lowByteMask
	}
	if y != -1 {
		y &= lowByteMask
	}
	if s.Arg(3) != 0 || g.lastFoundItem == -1 {
		g.lastFoundItem = g.lvl.blocks[s.Arg(0)].Objects
	} else {
		g.lastFoundItem = g.lvl.blocks[g.lastFoundItem&blockMask].Objects
	}
	for g.lastFoundItem != 0 {
		id := g.lastFoundItem
		if !isMonster(id) && (x == -1 || g.items[id].X&lowByteMask == x) && (y == -1 || g.items[id].Y&lowByteMask == y) {
			return id
		}
		g.lastFoundItem = g.lvl.blocks[id&monsterIndex&blockMask].Objects
	}
	return 0
}

// opTakeItemFromBlock: item, block (C: fn_0fbe_29f0).
func (g *Game) opTakeItemFromBlock(s *emc.State) int {
	g.takeItemFromBlock(s.Arg(0), s.Arg(1))
	return 1
}

// opNextMonsterOn: block, monster or -1; the monster on the block after the given
// one, or with -1 the first, else -1 (C: fn_0fbe_2a1f).
func (g *Game) opNextMonsterOn(s *emc.State) int {
	id := g.lvl.blocks[s.Arg(0)].Objects
	if m := s.Arg(1); m != -1 {
		id = *g.objectNext(m | monsterFlag)
	}
	for id != 0 {
		if isMonster(id) {
			return id & monsterIndex
		}
		id = *g.objectNext(id)
	}
	return -1
}

// opMonsterOn: block, monster; the monster if it stands on the block, -1 the first
// one, else -1 (C: fn_0fbe_2a97).
func (g *Game) opMonsterOn(s *emc.State) int {
	m := s.Arg(1) | monsterFlag
	for id := g.lvl.blocks[s.Arg(0)].Objects; isMonster(id); id = *g.objectNext(id) {
		if m == -1 || id == m { // sic: m has bit 15 set and is never -1
			return id & monsterIndex
		}
	}
	return -1
}

// opDissolve: x0, y0, x1, y1, w, h, from, to; copies a rectangle pixel by pixel in
// random order (C: fn_0fbe_2aff).
func (g *Game) opDissolve(s *emc.State) int {
	g.Screen.HideMouse()
	g.dissolve(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4), s.Arg(5), s.Arg(6), s.Arg(7))
	g.Screen.ShowMouse()
	return 1
}

// dissolve copies a w by h rectangle at x0, y0 of page 2 to x1, y1 of page to, its
// columns and rows shuffled (C: ovr_2d46_0039). The source is page 2 whatever from
// says (sic).
func (g *Game) dissolve(x0, y0, x1, y1, w, h, from, to int) {
	cols, rows := make([]int, w), make([]int, h)
	for i := range cols {
		cols[i] = i
	}
	for i := range rows {
		rows[i] = i
	}
	for i := range cols {
		r := random.Range(0, w-1)
		cols[r], cols[i] = cols[i], cols[r]
	}
	for i := range rows {
		r := random.Range(0, h-1)
		rows[r], rows[i] = rows[i], rows[r]
	}
	old := g.Screen.SetPage(from)
	src := g.Screen.Page(pageBuild)
	for k := range h {
		l := k
		for i := range w {
			x, y := x0+cols[i], y0+rows[l]
			x2, y2 := x1+cols[i], y1+rows[l]
			l++
			if l >= h {
				l = 0
			}
			g.Screen.SetPage(to)
			g.Screen.SetPixel(x2, y2, int(src[y*gfx.Width+x]))
		}
	}
	g.Screen.SetPage(old)
}

// opTurnInBlock: x, y, direction, which; the turned position's x (which non-zero) or
// y (C: fn_0fbe_2ba4).
func (g *Game) opTurnInBlock(s *emc.State) int {
	x, y := turnInBlock(s.Arg(0), s.Arg(1), s.Arg(2))
	if s.Arg(3) != 0 {
		return x
	}
	return y
}

// opNextCharacter: restart; the next present character's slot, or -1 after the last
// (C: fn_0fbe_2c12).
func (g *Game) opNextCharacter(s *emc.State) int {
	if s.Arg(0) != 0 {
		g.nextCharacter = 0
	} else {
		g.nextCharacter++
	}
	for ; g.nextCharacter < partySize; g.nextCharacter++ {
		if g.party[g.nextCharacter].Flags&charPresent != 0 {
			return g.nextCharacter
		}
	}
	return -1
}

// opAfflict: character, affliction, mask, chance, redraw (C: fn_0fbe_2c5c).
func (g *Game) opAfflict(s *emc.State) int {
	return boolInt(g.afflict(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4) != 0))
}

// opDrawPortraits: character or -1 for all (C: fn_0fbe_2cbe).
func (g *Game) opDrawPortraits(s *emc.State) int {
	if ch := s.Arg(0); ch != -1 {
		g.drawPortrait(ch)
	} else {
		g.drawPortraits()
	}
	return 1
}

// opRemoveItemOfType: type; takes the first item of the type from the inventory, else
// from a character, and says whether there was one (C: fn_0fbe_2cf4).
func (g *Game) opRemoveItemOfType(s *emc.State) int {
	t := s.Arg(0)
	for i, it := range g.inventory {
		if it != 0 && g.items[it].Type == t {
			g.inventory[i] = 0
			g.drawInventory()
			return 1
		}
	}
	for ch := range g.party {
		c := &g.party[ch]
		if c.Flags&charPresent == 0 {
			continue
		}
		for j, it := range c.Items {
			if it != 0 && g.items[it].Type == t {
				c.Items[j] = 0
				return 1
			}
		}
	}
	return 0
}

// opEnableClock: slot (C: fn_0fbe_2dc7).
func (g *Game) opEnableClock(s *emc.State) int {
	clock.Enable(s.Arg(0))
	return 1
}

// opDisableClock: slot (C: fn_0fbe_2de4).
func (g *Game) opDisableClock(s *emc.State) int {
	clock.Disable(s.Arg(0))
	return 1
}

// opLastSequence: animation; the sequence that last ended there, once
// (C: fn_0fbe_2e01).
func (g *Game) opLastSequence(s *emc.State) int {
	a := &g.animations[s.Arg(0)]
	r := a.lastSequence
	a.lastSequence = -1
	return r
}

// opWhoCarries: any, character or -1, type; the character with an item of the type,
// or -1. Without any an item of the type in the inventory makes it 0
// (C: fn_0fbe_2e3f).
func (g *Game) opWhoCarries(s *emc.State) int {
	return g.whoCarries(s.Arg(0) != 0, s.Arg(1), s.Arg(2))
}

// whoCarries (C: fn_130a_0d2b).
func (g *Game) whoCarries(any bool, ch, t int) int {
	if !any {
		for _, it := range g.inventory {
			if it != 0 && g.items[it].Type == t {
				return 0
			}
		}
	}
	from, end := ch, ch
	if ch == -1 {
		from, end = 0, partySize
	}
	for i := from; i < end; i++ {
		if g.carries(i, t) {
			return i
		}
	}
	return -1
}

// opRestoreLevelColours fades the upper half of the palette back to the level's
// (C: fn_0fbe_2e7d).
func (g *Game) opRestoreLevelColours(*emc.State) int {
	const upperHalf = 128 * 3
	copy(g.Palette[upperHalf:], g.darkPalette[upperHalf:])
	g.Screen.FadePalette(g.Palette[:], restoreColoursTicks, nil)
	g.paletteState = paletteNormal
	return 1
}

const restoreColoursTicks = 10

// opBlockInDirection: block, direction (C: fn_0fbe_2eca).
func (g *Game) opBlockInDirection(s *emc.State) int {
	return g.blockInDirection(s.Arg(0), s.Arg(1))
}

// opSelectedCharacter (C: fn_0fbe_2ec1).
func (g *Game) opSelectedCharacter(*emc.State) int { return g.selectedCharacter }

// opSetHeldItem: item (C: fn_0fbe_2ef6).
func (g *Game) opSetHeldItem(s *emc.State) int {
	g.setHeldItem(s.Arg(0))
	return 1
}

// opTakeFromInventory: type, redraw (C: fn_0fbe_2f13).
func (g *Game) opTakeFromInventory(s *emc.State) int {
	return g.takeFromInventory(s.Arg(0), s.Arg(1) != 0)
}

// takeFromInventory swaps the first inventory item of type t with the held one and
// returns what was held, or -1 without such an item (C: fn_0c59_0410).
func (g *Game) takeFromInventory(t int, redraw bool) int {
	slot := -1
	for i, it := range g.inventory {
		if it != 0 && g.items[it].Type == t {
			slot = i
			break
		}
	}
	if slot < 0 {
		return -1
	}
	g.inventoryStart = slot
	held := g.heldItem
	g.setHeldItem(g.inventory[slot])
	g.inventory[slot] = held
	if redraw {
		g.drawInventory()
	}
	return held
}

// opShakeScreen: ticks, dx, dy (C: fn_0fbe_30f8).
func (g *Game) opShakeScreen(s *emc.State) int {
	g.shakeScreen(s.Arg(0), s.Arg(1), s.Arg(2), shakeRedraw)
	return 1
}

// opDissolveScene draws the scene on the back page and dissolves it into the view
// (C: fn_0fbe_336f).
func (g *Game) opDissolveScene(*emc.State) int {
	g.drawScene(pageBuild)
	g.hideMouseInWindow(viewWindow)
	g.dissolve(viewLeftX, 0, viewLeftX, 0, viewCopyW, viewCopyH, pageBuild, pageScreen)
	g.copyViewFrame()
	g.Screen.ShowMouseIn()
	return 1
}

// opCopyViewFrame (C: fn_0fbe_33a8).
func (g *Game) opCopyViewFrame(*emc.State) int {
	g.copyViewFrame()
	return 1
}

// Special speech ids of opcode 187.
const (
	speechStop  = -1
	speechState = -2
)

// opSpeak: id, character, show; -1 stops the speech, -2 asks for its state
// (C: fn_0fbe_33b5).
func (g *Game) opSpeak(s *emc.State) int {
	switch s.Arg(0) {
	case speechStop:
		g.stopSpeech(true)
		return 1
	case speechState:
		return g.updateSpeech()
	}
	return boolInt(g.characterSays(s.Arg(0), s.Arg(1), s.Arg(2) != 0))
}

// opQueueSpeech: id, character; the speech to play once the current one ends
// (C: fn_0fbe_342b, fn_154c_010f).
func (g *Game) opQueueSpeech(s *emc.State) int {
	const queuedSpeechBase = 1000
	if id, ch := s.Arg(0), s.Arg(1); id != 0 && ch != 0 {
		g.queuedSpeech = id + queuedSpeechBase
		g.queuedSpeaker = ch
		g.queuedSpeechStarted = false
	}
	return 1
}

// priceSteps are the amounts a price is rounded up to (C: f_2c0c_0000).
var priceSteps = []int{
	1, 5, 8, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70,
	75, 80, 90, 100, 105, 120, 125, 130, 140, 150, 160, 175, 200, 225, 250, 275,
	300, 325, 350, 375, 400, 425, 450, 475, 500, 560, 1000, 1500, 3000,
}

// opRoundPrice: value; rounded up to a price step, or a negative one to hundreds,
// at least 50 (C: fn_0fbe_3457). 0 above the last step.
func (g *Game) opRoundPrice(s *emc.State) int {
	const (
		minimumHundreds = 50
		hundred         = 100
	)
	v := s.Arg(0)
	if v < 0 {
		v = -v
		if v < minimumHundreds {
			return minimumHundreds
		}
		return (v + hundred - 1) / hundred * hundred
	}
	for _, p := range priceSteps {
		if p >= v {
			return p
		}
	}
	return 0
}

// opLanguage (C: fn_0fbe_34b8, g_20db, which follows g_81fd).
func (g *Game) opLanguage(*emc.State) int { return g.language }
