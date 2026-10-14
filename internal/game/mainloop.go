package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// Events a block script can be run for (C: fn_0845_1343's mask bits).
const eventScriptCall = 1 << 1 // run on a script's request from the main loop

// noPartyCheck is deadPartyCheck's idle value (C: g_20d1 = -1).
const noPartyCheck = -1

// mainLoop runs the game until it is quit: timers, the cursor, input, idle work and
// the view (C: fn_0ca0_0c33).
func (g *Game) mainLoop() error {
	clock.Enable(playClockSlot)
	g.quit = false
	g.setFlag(flagLevelChanged, true)
	for !g.quit {
		if g.pendingBlockScript != 0 {
			g.runBlockScript(g.pendingBlockScript, eventScriptCall)
			g.pendingBlockScript = 0
		}
		g.runTimers()
		g.updateArrowCursor()
		g.handleInput()
		dice(1, 2) // the original rolled and dropped a die each round, which moves the generator on
		g.idleWork()
		if g.sceneUpdateRequired {
			g.drawScene(pageScreen)
		} else {
			g.playSoundAt(0, g.currentBlock)
		}
		if g.deadPartyCheck != noPartyCheck {
			g.checkPartyDead(g.deadPartyCheck)
			g.deadPartyCheck = noPartyCheck
		}
		if g.scriptErr != nil {
			return g.scriptErr
		}
		host.Yield()
	}
	return nil
}

// runTimers runs the timers that are due; those on the play clock only while it runs
// (C: fn_0bbc_0645).
func (g *Game) runTimers() {
	for i := range g.timers {
		t := &g.timers[i]
		g.updateMessageFade()
		if t.enabled && t.next < clock.Get(t.clock) {
			if clock.Enabled(playClockSlot) || t.clock != playClock {
				t.fn(g, t.arg)
				if t.enabled {
					t.next = clock.Get(t.clock) + t.interval
				}
			}
		}
	}
}

// Arrow cursors: with the option on, the cursor over the edges of the view turns into
// an arrow for the way the party would move there, or a stop sign where it cannot
// (C: fn_0ca0_097b).
const (
	optionArrowCursors = 1 << 2
	gameShapeArrows    = 10 // (C: f_2803_00a6[10 + s])
	noArrow            = -1
	blockedArrow       = 6
	arrowTurnLeft      = 4
)

var (
	arrowHotX = [...]int{7, 13, 7, 0, 0, 15, 7} // (C: g_23ee)
	arrowHotY = [...]int{0, 7, 12, 7, 6, 6, 7}  // (C: g_23f5)
)

type rect struct{ x1, y1, x2, y2 int }

func (r rect) has(x, y int) bool { return x >= r.x1 && x <= r.x2 && y >= r.y1 && y <= r.y2 }

var (
	arrowArea   = rect{96, 0, 303, 136}
	arrowCentre = rect{128, 16, 271, 119}
	arrowAreas  = [...]rect{ // forward, right, back, left, turn left, turn right
		{112, 0, 287, 15},
		{272, 88, 303, 319},
		{112, 110, 287, 135},
		{96, 88, 127, 119},
		{96, 16, 127, 87},
		{272, 16, 303, 87},
	}
	moveOffsets = [directionCount]int{-mapSize, 1, mapSize, -1} // (C: g_2236)
)

func (g *Game) updateArrowCursor() {
	if g.options&optionArrowCursors == 0 {
		return
	}
	s := noArrow
	mx, my := g.Screen.Mouse()
	if g.updateFlags&updateGameMenu == 0 && g.controlsLocked == 0 && arrowArea.has(mx, my) && !arrowCentre.has(mx, my) {
		for i, r := range arrowAreas {
			if r.has(mx, my) {
				s = i
			}
		}
		if s < arrowTurnLeft {
			dir := (g.direction + s) & (directionCount - 1)
			if !g.canEnter(g.blockInDirection(g.currentBlock, dir), dir) {
				s = blockedArrow
			}
		}
	}
	if s != g.lastMouseRegion {
		if s == noArrow {
			hot := 0
			if g.heldItem != 0 {
				hot = heldCursorHotSpot
			}
			g.Screen.SetMouseShape(hot, hot, g.itemIcon(g.heldItem))
		} else {
			g.Screen.SetMouseShape(arrowHotX[s], arrowHotY[s], g.gameShapes[gameShapeArrows+s])
		}
		g.lastMouseRegion = s
	}
}

// blockInDirection is the block next to b towards dir (C: fn_09aa_0183).
func (g *Game) blockInDirection(b, dir int) int { return (b + moveOffsets[dir]) & blockMask }

// canEnter says whether the party may step into block b moving towards dir: no wall
// stops it and no monster stands there (C: fn_09aa_0404).
func (g *Game) canEnter(b, dir int) bool {
	if g.wallsBlock(b, dir, blockedByWalls) {
		return false
	}
	for o := g.lvl.blocks[b].Objects; o != 0; o = *g.objectNext(o) {
		if o&monsterFlag != 0 {
			return false
		}
	}
	return true
}

// Keys the main loop handles itself (C: fn_0ca0_0b2f).
const (
	keyNextSpell  = 0x37
	keyDebugPrint = 0x71a
	keyVolumeDown = 0x2b
	keyVolumeUp   = 0x3d
	buttonFired   = 1 << 15
	keyReleased   = 1<<15 | 1<<11
	noCharacter   = -1
	charSelected  = 1 << 4
)

// handleInput runs the buttons and the keys the loop handles (C: fn_0ca0_0b2f).
func (g *Game) handleInput() {
	k := g.GUI.ProcessKeys(g.buttons)
	if k != 0 && g.spellTarget != noCharacter && k&keyReleased == 0 {
		g.buildButtons()
		g.party[g.spellTarget].Flags &^= charSelected
		g.drawPortrait(g.spellTarget)
		g.putKeyBack(k)
		g.spellTarget = noCharacter
		k = 0
	}
	switch k {
	case keyNextSpell:
		if g.screenFrozen {
			break
		}
		if g.availableSpells[1] != noSpell {
			g.highlightSpell(false)
			g.selectedSpell++
			if g.availableSpells[g.selectedSpell] == noSpell {
				g.selectedSpell = 0
			}
			g.highlightSpell(true)
			g.drawPortraits()
		}
	case keyDebugPrint:
		// the original printed the level, block and direction to a debug console
	case keyVolumeDown, keyVolumeUp:
		g.stopSpeech(true)
	}
}

// idleWork animates, burns the lamp, swings the compass and keeps the speech going
// (C: fn_0845_093a).
func (g *Game) idleWork() {
	g.updateAnimations()
	if g.talkingCharacter != noTalk && clock.Ticks() > g.talkEnds {
		g.animateTalk()
	}
	if g.flag(flagHasLamp) || g.updateFlags&updateGameMenu == 0 {
		g.updateLamp()
	}
	if g.flag(flagHasCompass) && g.updateFlags&updateGameMenu == 0 &&
		(g.compassDirection == noCompass || g.direction*compassUnitsPerQuarter != g.compassDirection || g.compassSwing != 0) {
		g.swingCompass()
	}
	g.updateSpeech()
	g.updateMessageFade()
}

// updateAnimations runs the level's looping WSA animations (C: fn_130a_028a).
func (g *Game) updateAnimations() {
	const updateNoAnimations = 1 << 3
	if g.updateFlags&updateNoAnimations != 0 {
		return
	}
	for i := range g.animations {
		a := &g.animations[i]
		if a.playing && a.next < clock.Ticks() && g.stepAnimation(a) {
			g.Screen.HideMouseIn(a.x, a.y, a.x+a.wsa.width(), a.y+a.wsa.height())
			g.drawWSA(a.wsa, a.frame-1, a.x, a.y, pageScreen, 0, nil, nil)
			g.Screen.ShowMouseIn()
			a.next += clock.Ticks()
		}
	}
}

// The words of an animation sequence (C: struct Anim's frames).
const (
	seqFirst = iota
	seqLast
	seqLoops
	seqNext // the sequence that follows, -1 for none
	seqNextDelay
	seqStopsOnAsk // with a stop asked it does not go on to the next
	seqSound      // -1 for none
	seqSoundFrame
)

// stepAnimation moves an animation a frame on: at its sequence's end it loops, goes
// on to the next sequence or stops; it says whether there is a frame to show. next is
// left as the delay to the frame (C: fn_130a_012a).
func (g *Game) stepAnimation(a *sceneAnimation) bool {
	f := &a.sequences[a.sequence]
	a.next = 0
	step := 1
	if f[seqLast] < f[seqFirst] {
		step = -1
	}
	a.frame = int(int8(a.frame + step))
	if a.frame == f[seqLast]+step {
		a.loops = int(int8(a.loops + 1))
		if a.loops <= f[seqLoops] && !a.stopAsked {
			a.frame = f[seqFirst]
		} else {
			a.lastSequence = a.sequence
			if f[seqNext] != -1 && !(a.stopAsked && f[seqStopsOnAsk] != 0) {
				a.next += f[seqNextDelay]
				a.sequence = int(int8(f[seqNext]))
				f = &a.sequences[a.sequence]
				a.frame = f[seqFirst]
				a.loops = 0
			} else {
				a.playing = false
				a.stopAsked = false
				return false
			}
		}
	}
	if f[seqSound] != -1 && a.frame == f[seqSoundFrame] {
		g.playSound(f[seqSound], 0)
	}
	a.next += a.delay
	return true
}

// checkPartyDead ends the game when nobody is left standing, or with flag 64 raises
// the dead (C: ovr_2cf4_0020).
func (g *Game) checkPartyDead(flags int) bool {
	for i := range g.party {
		c := &g.party[i]
		if c.Flags&charPresent != 0 && c.HP > 0 {
			return false
		}
	}
	if g.screenFrozen {
		g.clickExitCharacterInventory(nil)
	}
	g.drawPortraits()
	if flags&raiseDead != 0 {
		g.fadeToBlack(raiseDeadFade)
		for i := range g.party {
			if g.party[i].Flags&charPresent != 0 {
				g.heal(i, 1, true)
			}
		}
		g.drawPortraits()
		g.fadeInDark(raiseDeadFade)
		return false
	}
	g.fadeOutView(levelFadeTicks)
	g.endFullScene(0, 1, 1, 0)
	g.gameOverMenu()
	return true
}

const (
	raiseDead     = 1 << 6
	raiseDeadFade = 40
)

// putKeyBack queues key k again (C: fn_2169_0945).
func (g *Game) putKeyBack(k int) { input.Put(k) }

// highlightSpell prints the selected spell's name in the list lit or plain
// (C: fn_0ca0_05bb).
func (g *Game) highlightSpell(on bool) {
	const (
		listX, listY, lineHeight = 24, 15, 9
		plainColour, litColour   = 1, 132
	)
	colour := plainColour
	if on {
		colour = litColour
	}
	y := listY
	for i, sp := range g.availableSpells {
		if sp == noSpell {
			break
		}
		if g.selectedSpell == i {
			g.Screen.HideMouse()
			g.printAt(g.text(spells[sp].Name), listX, y, colour, 0, 0)
			g.Screen.ShowMouse()
			break
		}
		y += lineHeight
	}
}
