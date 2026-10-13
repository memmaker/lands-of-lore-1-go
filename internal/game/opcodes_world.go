package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
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
