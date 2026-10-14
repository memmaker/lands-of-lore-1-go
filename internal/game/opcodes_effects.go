package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
)

// opFreeMonsterShapes drops every monster set's shapes (C: fn_0fbe_333f,
// ovr_2d2c_00e8).
func (g *Game) opFreeMonsterShapes(s *emc.State) int {
	for set := range monsterShapeSets {
		g.freeMonsterShapes(set)
	}
	return 0
}

// opSoundLoaded says whether effect a0 is loaded (C: ovr_2d2c_00bb).
func (g *Game) opSoundLoaded(s *emc.State) int {
	if g.soundLoaded(s.Arg(0)) {
		return 1
	}
	return 0
}

// opSetSceneOverlay sets the decoration drawn over the whole view: view a1 of wall
// type a0's decoration at (a2, a3), mirrored with bit 0 of a4; 0 removes it
// (C: ovr_2d46_004d).
func (g *Game) opSetSceneOverlay(s *emc.State) int {
	if s.Arg(0) == 0 {
		g.sceneOverlay = sceneOverlay{}
		return 1
	}
	o := sceneOverlay{x: s.Arg(2), y: s.Arg(3), mirrored: s.Arg(4)&1 != 0}
	lv := &g.lvl
	if d := lv.wallShape[s.Arg(0)]; d >= 0 && d < len(lv.decorations) {
		if i := lv.decorations[d].shapes[s.Arg(1)]; i >= 0 && i < len(lv.decorationShapes) {
			o.shape = lv.decorationShapes[i]
		}
	}
	g.sceneOverlay = o
	return 1
}

// opCastSpell casts spell a1 at level a2 for character a0 (C: fn_0fbe_2f3f).
func (g *Game) opCastSpell(s *emc.State) int {
	if g.castSpell(s.Arg(0), s.Arg(1), s.Arg(2)) {
		return 1
	}
	return 0
}

// opBusyCursor shows the busy icon with a0 1, else puts the cursor back
// (C: ovr_2d5f_0138).
func (g *Game) opBusyCursor(s *emc.State) int {
	const busyIcon = 133
	if s.Arg(0) == 1 {
		g.setBusyCursor(busyIcon)
	} else {
		g.restoreCursor()
	}
	return 1
}

// The flash of opcode 176: the level colours halfway to white in the channels a0's
// bits name, halfway to black in the others, and the view's colour 1 as colour 6.
const (
	flashFirstColour  = 2
	flashLastColour   = 128
	paletteFlashTicks = 2
	flashShadowFrom   = 1
	flashShadowTo     = 6
	channelMax        = 63
)

// opFlashPalette flashes the palette for a moment (C: fn_0fbe_2f7d, fn_0845_0f86). The
// mouse driver call between is a vendor extension that does nothing here.
func (g *Game) opFlashPalette(s *emc.State) int {
	pal := g.darkPalette
	mask := s.Arg(0)
	for i := flashFirstColour * 3; i < flashLastColour*3; i++ {
		v := int(pal[i]) & channelMax
		if mask&(1<<(i%3)) != 0 {
			v += (channelMax - v) >> 1
		} else {
			v -= v >> 1
		}
		pal[i] = byte(v)
	}
	g.keepInterfaceColours(pal[:])
	g.keepInterfaceColours(g.darkPalette[:])
	scr := g.Screen
	if fastMachine {
		table := identityTable()
		table[flashShadowFrom] = flashShadowTo
		g.hideMouseInWindow(viewWindow)
		scr.CopyRectColumns(viewFrameColumn, 0, viewFrameColumn, 0, viewFrameColumns, viewFrameH, pageScreen, pageBuild)
		scr.RemapRect(viewLeft, 0, viewWidth, viewHeight, pageScreen, table)
		scr.ShowMouseIn()
	}
	scr.SetPalette(pal[:])
	clock.Delay(paletteFlashTicks)
	scr.SetPalette(g.darkPalette[:])
	if fastMachine {
		g.hideMouseInWindow(viewWindow)
		scr.CopyRectColumns(viewFrameColumn, 0, viewFrameColumn, 0, viewFrameColumns, viewFrameH, pageBuild, pageScreen)
		scr.ShowMouseIn()
	}
	return 0
}
