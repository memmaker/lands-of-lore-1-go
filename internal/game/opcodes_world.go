package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
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

// Item flags of placeItem.
const itemOnOtherLevel = 1 << 14 // (C: 0x4000 on an item left on a level not loaded)

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
		item.Flags |= itemOnOtherLevel
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
	n := s.Arg(0)
	a := &g.animations[n]
	a.playing = false
	a.stopAsked = false
	if n == speechAnimation {
		a.wsa = nil
	}
	return 1
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
	g.speechKey(1)
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

func (g *Game) healAnimation(who, kind int) { notPorted("ovr_2d18_002a (healing animation)") }
