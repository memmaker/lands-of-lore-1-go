package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// Clicks on the wall ahead: a wall type with a special kind reacts when its decoration
// is hit (C: s021 g_282c).
const (
	wallClickScript = 1 // runs the block's script
	wallClickNext   = 2 // a switch: the wall type after it
	wallClickPrev   = 3 // and back
	wallClickAlways = 4 // the script, wherever the wall is hit
	wallClickDoor   = 5 // a door button
	wallClickPut    = 6 // a niche to put the held item in

	eventWallClick    = 1 << 6 // (C: 0x40)
	decorationNear    = 1      // the view of a decoration on the wall ahead
	decorationViewX   = 136
	decorationViewY   = 8
	decorationMargin  = 4
	decorationMirror  = 1 << 0
	soundSwitchOn     = 30
	soundSwitchOff    = 29
	soundDoorButton   = 78
	doorButtonDelay   = 15
	nicheX            = 128
	nicheY            = 255
	nicheHeight       = 8
	inventoryWrap     = inventorySlots
	inventoryPage     = 1 << 12 // the scroll button moves nine slots
	inventoryPageStep = inventoryShown
	iconScrollLeft    = 74
	iconScrollRight   = 75

	itemTypeTruthHalfA = 281 // the two halves of the Ruby of Truth
	itemTypeTruthHalfB = 220
	itemTypeTruth      = 280
	truthFrames        = 25
	truthFrameTicks    = 7
	truthSound         = 99
	truthW             = 25
	truthH             = 27
	truthDY            = 3
	truthWSAMode       = 1 << 14
)

// decorationHit says whether the mouse is on one of the decorations of chain d, as
// drawn on the wall ahead (C: fn_13e8_0005).
func (g *Game) decorationHit(d int) bool {
	mx, my := g.Screen.Mouse()
	for d != 0 {
		dec := &g.lvl.decorations[d]
		if v := dec.shapes[decorationNear]; v != noShape {
			if g.anyDecoration { // changed: Space hits any decoration of the wall
				return true
			}
			shape := g.lvl.decorationShapes[v]
			x := dec.x[decorationNear] + decorationViewX
			y := dec.y[decorationNear] + decorationViewY
			w, h := gfx.ShapeWidth(shape), gfx.ShapeHeight(shape)
			if dec.flags&decorationMirror != 0 {
				w <<= 1
			}
			if (rect{x - decorationMargin, y - decorationMargin, x + w + 2*decorationMargin, y + h + 2*decorationMargin}).has(mx, my) {
				return true
			}
		}
		d = dec.next
	}
	return false
}

// clickWall runs the wall ahead's reaction to a click (C: fn_13e8_02fa).
func (g *Game) clickWall(*gui.Button) bool {
	b := g.blockInDirection(g.currentBlock, g.direction)
	side := g.direction ^ 2
	hit := false
	switch g.lvl.wallSpecialType[g.lvl.blocks[b].Walls[side]] {
	case wallClickScript:
		hit = g.wallClickScript(b, side)
	case wallClickNext:
		hit = g.wallClickSwitch(b, side, 1, soundSwitchOn)
	case wallClickPrev:
		hit = g.wallClickSwitch(b, side, -1, soundSwitchOff)
	case wallClickAlways:
		g.runBlockScript(b, eventWallClick)
		hit = true
	case wallClickDoor:
		hit = g.wallClickDoor(b, side)
	case wallClickPut:
		hit = g.wallClickPut(b, side)
	}
	g.flushInput()
	return hit
}

// wallDecoration is the decoration chain on a block's side.
func (g *Game) wallDecoration(b, side int) int {
	return g.lvl.wallShape[g.lvl.blocks[b].Walls[side]]
}

// (C: fn_13e8_00d4)
func (g *Game) wallClickScript(b, side int) bool {
	if !g.decorationHit(g.wallDecoration(b, side)) {
		return false
	}
	g.stopSpeech(true)
	g.runBlockScript(b, eventWallClick)
	return true
}

// (C: fn_13e8_011c, fn_13e8_017c)
func (g *Game) wallClickSwitch(b, side, step, sound int) bool {
	if !g.decorationHit(g.wallDecoration(b, side)) {
		return false
	}
	g.lvl.blocks[b].Walls[side] += step
	g.sceneUpdateRequired = true
	g.playSound(sound, 0)
	g.runBlockScript(b, eventWallClick)
	return true
}

// (C: fn_13e8_01f1)
func (g *Game) wallClickDoor(b, side int) bool {
	if !g.decorationHit(g.wallDecoration(b, side)) {
		return false
	}
	g.doorHandled = 0
	g.playSound(soundDoorButton, 0)
	g.runBlockScript(b, eventWallClick)
	if g.doorHandled == 0 {
		clock.Delay(doorButtonDelay)
		g.toggleDoor(b, side, 0)
	}
	return true
}

// (C: fn_13e8_0261)
func (g *Game) wallClickPut(b, side int) bool {
	if !g.decorationHit(g.wallDecoration(b, side)) || g.heldItem == 0 {
		return false
	}
	x, y := turnInBlock(nicheX, nicheY, g.direction)
	x, y = blockPosition(b, x, y)
	g.dropItem(g.heldItem, x, y, nicheHeight, true)
	g.setHeldItem(0)
	return true
}

// clickDialogueArea runs the script of the block ahead; a click outside the scene's
// region drops the queued input (C: ovr_2d5f_002a).
func (g *Game) clickDialogueArea(*gui.Button) bool {
	b := g.blockInDirection(g.currentBlock, g.direction)
	g.runBlockScript(b, eventWallClick)
	x, y := g.Screen.Mouse()
	r := g.sceneRegion
	if !g.sceneRegionActive || !(rect{r[0], r[1], r[2], r[3]}).has(x, y) {
		g.flushInput()
		g.sceneRegionActive = false
	}
	return true
}

// clickInventorySlot swaps the held item with a slot's; the Ruby of Truth's halves
// join (C: fn_0c59_015f).
func (g *Game) clickInventorySlot(b *gui.Button) bool {
	slot := g.inventoryStart + b.Arg
	if slot >= inventorySlots {
		slot -= inventorySlots
	}
	held := g.heldItem
	other := g.inventory[slot]
	ht, ot := g.items[held].Type, g.items[other].Type
	if (ht == itemTypeTruthHalfA || ot == itemTypeTruthHalfA) && (ht == itemTypeTruthHalfB || ot == itemTypeTruthHalfB) {
		g.joinTruth(b, slot, held, other)
	} else {
		g.setHeldItem(g.inventory[slot])
		g.inventory[slot] = held
	}
	g.drawInventorySlot(b.Arg)
	return true
}

// joinTruth plays the halves joining in the slot and makes the whole ruby.
func (g *Game) joinTruth(b *gui.Button, slot, held, other int) {
	s := g.Screen
	w := g.openWSA("truth.wsa", 0, nil)
	s.HideMouse()
	g.inventory[slot] = 0
	g.drawInventorySlot(b.Arg)
	y := b.Y - truthDY
	s.CopyRect(b.X, y, b.X, y, truthW, truthH, pageScreen, pageBuild)
	g.playSoundEffect(truthSound)
	for i := range truthFrames {
		clock.Set(clock.Countdown, truthFrameTicks)
		s.CopyRect(b.X, y, 0, 0, truthW, truthH, pageBuild, pageBuild)
		if w != nil {
			g.drawWSA(w, i, 0, 0, pageBuild, truthWSAMode, nil, nil)
		}
		s.CopyRect(0, 0, b.X, y, truthW, truthH, pageBuild, pageScreen)
		for clock.Get(clock.Countdown) != 0 {
			host.Yield()
		}
	}
	g.freeItem(held)
	g.freeItem(other)
	g.setHeldItem(0)
	g.inventory[slot] = g.newItem(itemTypeTruth, 0, 0)
	s.ShowMouse()
}

// clickInventoryScroll scrolls the inventory bar by one slot or a page
// (C: fn_0c59_00ef).
func (g *Game) clickInventoryScroll(b *gui.Button) bool {
	step := b.Arg
	if b.Flags&inventoryPage != 0 {
		step *= inventoryPageStep
	}
	g.inventoryStart += step
	icon := iconScrollRight
	if b.Arg < 0 {
		icon = iconScrollLeft
	}
	g.setIcon(icon, iconPress)
	if g.inventoryStart < 0 {
		g.inventoryStart += inventoryWrap
	}
	if g.inventoryStart >= inventoryWrap {
		g.inventoryStart -= inventoryWrap
	}
	g.drawInventory()
	g.setIcon(icon, iconRelease)
	return true
}
