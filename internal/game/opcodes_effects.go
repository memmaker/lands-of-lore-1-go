package game

import (
	"errors"
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
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

// The two places opcode 98 labels: the right end x/2, the bottom and the width taken
// by the button (C: struct Pos3 g_1218).
var labelPlaces = [...]struct{ x, y, w int }{{144, 120, 12}, {159, 128, 30}}

const (
	labelString     = 0x4033
	labelHeight     = 9
	labelFrameLeft  = 251
	labelFrameRight = 252
	labelFill       = 136
	labelColour     = 144
	labelDither     = 1
)

// opDrawLabel draws string 0x4033 in a box left of place a0; with a1 1 the box is
// greyed out (C: ovr_2d5f_0106).
func (g *Game) opDrawLabel(s *emc.State) int {
	scr := g.Screen
	old := scr.SetPage(pageScreen)
	font := scr.SetFont(g.SmallFont)
	p := labelPlaces[s.Arg(0)]
	x, y, w := p.x*2, p.y, p.w
	str := g.text(labelString)
	tw := scr.TextWidth(str)
	scr.HideMouse()
	g.drawFrame(x-w-tw, y-labelHeight, tw+w, labelHeight, labelFill, labelFrameLeft, labelFrameRight)
	scr.DrawText(str, x-w>>1-tw+1, y-7, labelColour, 0)
	scr.ShowMouse()
	if s.Arg(1) == 1 {
		scr.HideMouse()
		scr.DitherRect(x-w-tw+1, y-8, tw+w-2, labelHeight-2, labelDither)
		scr.ShowMouse()
	}
	scr.SetPage(old)
	scr.SetFont(font)
	return 1
}

// The bezel animation's frame ranges by kind (C: struct AnimRange f_25fb_0000).
var bezelFrames = [...]struct{ first, last int }{{0, 26}, {27, 61}, {62, 92}, {93, 131}}

const (
	bezelFile       = "BEZEL.WSA"
	bezelSound      = 73
	bezelY          = 124
	bezelDX         = 11
	bezelFrameTicks = 4
	bezelKinds      = 3 // opcode 170's a0 counts down from it
)

// opBezelHeal heals character a1 fully while the bezel animation of kind 3-a0 plays
// over its portrait (C: ovr_2cef_0039).
func (g *Game) opBezelHeal(s *emc.State) int {
	g.bezelHeal(bezelKinds-s.Arg(0), s.Arg(1))
	return 1
}

// bezelHeal plays the bezel over character c's portrait, the hit points rising with
// the frames (C: ovr_2cef_0034). The page 5 backup around the original's load is not
// needed: the animation has a buffer of its own.
func (g *Game) bezelHeal(kind, c int) {
	scr := g.Screen
	old := scr.SetPage(pageBuild)
	g.playSound(bezelSound, 0)
	w := g.openWSA(bezelFile, 0, nil)
	if w == nil {
		g.fail(errors.New("Bezel: Can't load bezel.wsa")) // C: ovr_2ce1_007f, the fatal exit
		scr.SetPage(old)
		return
	}
	x, y := g.portraitX[c]-bezelDX, bezelY
	width, height := w.width(), w.height()
	scr.HideMouse()
	scr.CopyRect(x, y, 0, 0, width, height, pageScreen, pageBuild)
	scr.ShowMouse()
	r := bezelFrames[kind]
	ch := &g.party[c]
	delta := ch.MaxHP - ch.HP
	acc := 0
	for frame := r.first; frame <= r.last && !g.quit; frame++ {
		acc &= 1<<8 - 1
		acc += (delta << 8) / r.last
		g.heal(c, acc/256, true)
		g.drawPortrait(c)
		clock.Set(clock.Countdown, bezelFrameTicks)
		scr.CopyRect(0, 0, x, y, width, height, pageBuild, pageBuild)
		g.drawWSA(w, frame, x, y, pageBuild, wsaOverScreen, g.lvl.tlcTable, g.lvl.tlcLevels)
		scr.HideMouseIn(x, y, x+width, y+height)
		scr.CopyRect(x, y, x, y, width, height, pageBuild, pageScreen)
		scr.ShowMouseIn()
		for clock.Get(clock.Countdown) != 0 && !g.quit {
			host.Yield()
		}
	}
	ch.HP = ch.MaxHP
	scr.CopyRect(0, 0, x, y, width, height, pageBuild, pageBuild)
	g.clearStatusEffects(ch, effectPoison, effectPoison)
	g.drawPortrait(c)
	scr.CopyRect(x, y, x, y, width, height, pageBuild, pageScreen)
	scr.SetPage(old)
}

// The end of the game: the view wiped away (C: o2d7d).
const (
	wipeDeathSteps   = 9
	wipeEndSteps     = 12
	deathShakeTicks  = 30
	deathShakeDX     = 4
	firstHeroScream  = 54 // the first character's sound, if a starting hero (C: 0x36)
	noSound          = -1
	wipeFractionBits = 8
)

// opGameEnd ends the game: with a0 1 a shake after the wipe, else the view cleared
// (C: ovr_2d7d_0025).
func (g *Game) opGameEnd(s *emc.State) int {
	if s.Arg(0) == 1 {
		g.drawScene(pageBuild)
		g.wipeView(wipeDeathSteps)
		g.playSound(noSound, 0)
		g.shakeScreen(deathShakeTicks, deathShakeDX, 0, shakeRedraw)
		return 1
	}
	sound := noSound
	for i := range partySize {
		if c := &g.party[i]; c.Flags&charPresent != 0 && c.ID < 0 {
			if i == 0 {
				sound = firstHeroScream
			}
			// sic: the sounds of slots 5, 8 and 9 can never be chosen
			break
		}
	}
	old := g.Screen.SetPage(pageBuild)
	g.Screen.FillRect(viewLeft, 0, viewLeft+viewWidth, viewHeight, 0)
	g.Screen.SetPage(old)
	g.playSound(sound, 0)
	g.wipeView(wipeEndSteps)
	return 1
}

// wipeView slides the view on the screen down in n steps, the build page's view
// coming up from under it; page 6 holds the old view unstrided (C: ovr_2d7d_0020).
func (g *Game) wipeView(n int) {
	scr := g.Screen
	screen, saved, build := scr.Page(pageScreen), scr.Page(pageIcon), scr.Page(pageBuild)
	scr.HideMouseIn(viewLeft, 0, viewLeft+viewWidth, viewHeight)
	viewToBuffer(screen, saved)
	scr.ShowMouseIn()
	for i := range n {
		t := clock.Ticks()
		off := (viewHeight << wipeFractionBits / n * i) >> wipeFractionBits
		scr.HideMouseIn(viewLeft, 0, viewLeft+viewWidth, viewHeight)
		for y := range viewHeight - off {
			copy(screen[y*gfx.Width+viewLeft:][:viewWidth], saved[(y+off)*viewWidth:])
		}
		for y := range off {
			at := y*gfx.Width + viewLeft
			copy(screen[(viewHeight-off)*gfx.Width+at:][:viewWidth], build[at:])
		}
		scr.ShowMouseIn()
		clock.WaitUntil(t+1, nil)
	}
	scr.HideMouseIn(viewLeft, 0, viewLeft+viewWidth, viewHeight)
	t := clock.Ticks()
	copyView(build, screen)
	clock.WaitUntil(t+1, nil)
	scr.ShowMouseIn()
	g.copyViewFrame()
}

// The dark light animation: its palettes from the screen's to LITEPAL1 over the
// first frames, LITEPAL2 faded to, then on to LITEPAL3 (C: o2cf7).
const (
	darkLiteFile       = "DARKLITE.WSA"
	darkLiteFirstSteps = 21
	darkLiteLastSteps  = 4
	darkLiteFirstTicks = 20
	darkLiteLastTicks  = 12
	darkLiteFadeTicks  = 300
	darkLiteFirstEnd   = 21
	darkLiteLastStart  = 22
	darkLiteLastEnd    = 38
	darkLiteTick       = 95  // the sound of the first frames' marked steps (C: 0x5f)
	darkLiteFade       = 91  // (C: 0x5b)
	darkLiteFlash      = 131 // (C: 0x83)
)

var (
	darkLiteTickFrames  = map[int]bool{2: true, 5: true, 8: true, 11: true, 13: true, 15: true, 17: true, 19: true}
	darkLiteFlashFrames = map[int]bool{22: true, 24: true, 28: true, 32: true}
	darkLitePalettes    = [...]string{"LITEPAL1.COL", "LITEPAL2.COL", "LITEPAL3.COL"}
)

// opDarkLite plays the dark light animation over the screen; 0 if it is missing
// (C: fn_0fbe_3081, ovr_2cf7_0020).
func (g *Game) opDarkLite(s *emc.State) int {
	scr := g.Screen
	scr.HideMouse()
	g.saveSparePage()
	w := g.openWSA(darkLiteFile, wsaDirect, nil)
	if w == nil {
		// sic: the mouse stays hidden and the spare page saved
		return 0
	}
	pals := g.darkLitePalettes()
	k := 0
	for i := range darkLiteFirstEnd {
		t := clock.Ticks() + darkLiteFirstTicks
		g.drawWSA(w, i, 0, 0, pageScreen, 0, nil, nil)
		scr.SetPalette(pals[k])
		k++
		if darkLiteTickFrames[i] {
			g.playSound(darkLiteTick, 0)
		}
		clock.WaitUntil(t, nil)
	}
	g.playSound(darkLiteFade, 0)
	scr.FadePalette(pals[k], darkLiteFadeTicks, nil)
	k++
	for i := darkLiteLastStart; i < darkLiteLastEnd; i++ {
		t := clock.Ticks() + darkLiteLastTicks
		g.drawWSA(w, i, 0, 0, pageScreen, 0, nil, nil)
		if darkLiteFlashFrames[i] {
			g.playSound(darkLiteFlash, 0)
			scr.SetPalette(pals[k])
			k++
		}
		clock.WaitUntil(t, nil)
	}
	g.restoreSparePage()
	scr.ShowMouse()
	return 1
}

// darkLitePalettes builds the animation's palettes (C: o2cf7's mkpal). A missing file
// leaves its colours black, as the original's unchecked read left the buffer.
func (g *Game) darkLitePalettes() [][]byte {
	var col [len(darkLitePalettes)][]byte
	for i, name := range darkLitePalettes {
		col[i] = make([]byte, len(g.Screen.Palette))
		if b, err := g.Files.Read(name); err == nil {
			copy(col[i], b)
		}
	}
	pals := blendPalettes(g.Screen.Palette[:], col[0], darkLiteFirstSteps)
	pals = append(pals, col[1])
	return append(pals, blendPalettes(col[1], col[2], darkLiteLastSteps)...)
}

// blendPalettes makes n palettes from a to b: a, the steps between, b. The difference
// is kept in a byte, and so wraps (sic) (C: ovr_2cf7_0025).
func blendPalettes(a, b []byte, n int) [][]byte {
	diff := make([]int, len(a))
	for i := range a {
		diff[i] = int(int8(b[i] - a[i]))
	}
	out := [][]byte{append([]byte(nil), a...)}
	step := (1 << 8) / n
	acc := 0
	for range n - 2 {
		acc += step
		p := make([]byte, len(a))
		for i := range p {
			p[i] = byte(int(a[i]) + diff[i]*acc>>8)
		}
		out = append(out, p)
	}
	return append(out, append([]byte(nil), b...))
}

// The gas cloud: an explosion where the gas stops ahead, or the screen flushing red
// when it stops in the party's block (C: o2d0a).
const (
	gasFile        = "gasexp%0d.wsa"
	gasReach       = 3
	gasEndFrame    = 6
	gasFrameTicks  = 1
	gasRedInTicks  = 10
	gasRedOutTicks = 50
	gasFirstRed    = 1
	gasLastRed     = 128
	gasBackupRows  = 120 // the screen rows kept around it (C: 0x9600 bytes)
	gasBackupSize  = gasBackupRows * gfx.Width
)

// The gas's sounds by kind (C: g_0964).
var gasSounds = [...]int{98, 167, 167, 168}

// opGas lets off gas of kind a0 (C: ovr_2d0a_002f).
func (g *Game) opGas(s *emc.State) int {
	g.gas(s.Arg(0))
	return 1
}

// gas plays the gas cloud of kind n (C: ovr_2d0a_002a).
func (g *Game) gas(n int) {
	scr := g.Screen
	old := scr.SetPage(pageBuild)
	scr.HideMouse()
	copy(g.pageBackup2[:gasBackupSize], scr.Page(pageScreen))
	scr.ShowMouse()
	g.playSound(gasSounds[n], 0)
	if r, _ := g.firstStopAhead(gasReach); r != 0 {
		w := g.openWSA(fmt.Sprintf(gasFile, r), wsaDirect, nil)
		if w == nil {
			g.fail(errors.New("Gas: Unable to load gasexp.wsa")) // C: ovr_2ce1_007f, the fatal exit
			scr.SetPage(old)
			return
		}
		g.playWSARange(w, 0, gasEndFrame, gasFrameTicks, (viewWidth-w.width())>>1+viewLeft, (viewHeight-w.height())>>1, nil, nil, false)
	} else {
		red := g.darkPalette
		for i := gasFirstRed; i < gasLastRed; i++ {
			red[i*3] = channelMax
		}
		g.fadeWhileMoving(g.Palette[:], red[:], gasRedInTicks)
		g.fadeWhileMoving(red[:], g.Palette[:], gasRedOutTicks)
	}
	copy(scr.Page(pageBuild), g.pageBackup2[:gasBackupSize])
	scr.SetPage(old)
	g.copyViewFrame()
	g.sceneUpdateRequired = true
	g.drawScene(pageScreen)
}

// fadeWhileMoving runs a palette fade until no colour moves any more.
func (g *Game) fadeWhileMoving(from, to []byte, ticks int) {
	f := &paletteFade{from: from, to: to, ticks: ticks}
	t := clock.Ticks()
	for g.fadeStep(f, clock.Ticks()-t) && !g.quit {
		host.Yield()
	}
}
