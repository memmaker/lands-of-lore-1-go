package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// Scenes (C: o2d5f): while a script plays a scene, play time stands still, the move
// buttons may be off and a dialogue box slides up over the bottom of the screen.

// Game screen icons and where they sit (C: g_2176, g_218c, both indexed by the icon's
// game shape number).
const (
	iconFirstSceneButton = 76 // the scene buttons: the two bottom ones, the automap and the six move buttons
	iconAutomap          = 78
	iconFirstMove        = 79
	iconLastSceneButton  = 84
)

var iconPositions = map[int][2]int{
	74: {86, 180}, 75: {296, 180},
	76: {12, 180}, 77: {33, 180}, 78: {290, 32},
	79: {12, 132}, 80: {33, 132}, 81: {54, 132},
	82: {12, 150}, 83: {33, 150}, 84: {54, 150},
}

// How setIcon draws an icon.
const (
	iconRelease  = iota // the pressed icon comes up again, when it was down long enough
	iconPress           // drawn pressed
	iconNormal          // drawn up
	iconDisabled        // drawn dithered over
)

const (
	iconPressTicks = 6
	pageIcon       = 6 // the page a disabled icon is put together on
	ditherColour   = 1
)

// setIcon draws game screen icon n pressed, up or disabled (C: fn_09aa_0007).
func (g *Game) setIcon(n, how int) {
	if n == iconAutomap && !g.flag(flagHasAutomap) {
		return
	}
	if g.controlMode != 0 && g.sceneOn {
		return
	}
	if how == iconRelease {
		n = g.shownIcon
	}
	at := iconPositions[n]
	x, y := at[0], at[1]
	page, flags := pageScreen, 0
	switch how {
	case iconPress:
		flags = gfx.ShapeFaded
		g.shownIcon = n
	case iconRelease:
		if g.shownIcon == 0 {
			return
		}
		for clock.Ticks() < g.iconReleaseAt {
			host.Yield()
		}
		g.shownIcon = 0
	case iconNormal:
		g.shownIcon = 0
	case iconDisabled:
		g.shownIcon = 0
		page = pageIcon
	}
	shape := g.gameShapes[n]
	s := g.Screen
	s.HideMouse()
	if page == pageIcon {
		s.DrawShape(page, shape, 0, 0, 0, flags, gfx.ShapeOptions{Fade: g.shadowTable, FadeSteps: 1})
		old := s.SetPage(page)
		w, h := gfx.ShapeWidth(shape), gfx.ShapeHeight(shape)
		s.DitherRect(0, 0, w, h, ditherColour)
		s.CopyRect(0, 0, x, y, w, h, page, pageScreen)
		s.SetPage(old)
	} else {
		s.DrawShape(page, shape, x, y, 0, flags, gfx.ShapeOptions{Fade: g.shadowTable, FadeSteps: 1})
	}
	s.ShowMouse()
	g.iconReleaseAt = clock.Ticks() + iconPressTicks
}

// The flag a character carries while the controls are locked (C: Char.flags 0x2000).
const charLocked = 1 << 13

// Locks of the controls (C: struct Scene c48).
const (
	controlsFree       = 0
	controlsLockedAll  = 1
	controlsLockedMove = 2 // the move buttons stay usable
	lockKeepMoves      = 1 << 1
	noSelection        = 99
)

// lockControls disables the scene buttons, all or all but the move buttons when how
// has lockKeepMoves (C: ovr_2d5f_002f).
func (g *Game) lockControls(how int) bool {
	if g.controlMode != 0 {
		return false
	}
	g.controlsLocked = controlsLockedAll
	if how&lockKeepMoves != 0 {
		g.controlsLocked = controlsLockedMove
	}
	g.markPartyLocked(true)
	for i := iconFirstSceneButton; i <= iconLastSceneButton; i++ {
		if how&lockKeepMoves == 0 || i < iconFirstMove {
			g.setIcon(i, iconDisabled)
		} else {
			g.setIcon(i, iconNormal)
		}
	}
	return true
}

// unlockControls (C: ovr_2d5f_0034).
func (g *Game) unlockControls() bool {
	g.controlsLocked = controlsFree
	if g.controlMode == 0 {
		for i := iconFirstSceneButton; i <= iconLastSceneButton; i++ {
			g.setIcon(i, iconNormal)
		}
	}
	g.markPartyLocked(false)
	return true
}

// markPartyLocked flags the party's characters and redraws their portraits; locked
// without the text area, none is drawn selected (C: o2d5f partyflags).
func (g *Game) markPartyLocked(on bool) {
	for i := range g.party {
		c := &g.party[i]
		if c.Flags&charPresent == 0 {
			continue
		}
		if on {
			c.Flags |= charLocked
		} else {
			c.Flags &^= charLocked
		}
		if on && !g.textEnabled {
			selected, flags := g.selectedCharacter, g.updateFlags
			g.selectedCharacter = noSelection
			g.updateFlags &^= updateNoPortraits
			g.drawPortrait(i)
			g.updateFlags = flags
			g.selectedCharacter = selected
		} else {
			g.drawPortrait(i)
		}
	}
}

// startSceneClock starts a scene: play time stands still (C: ovr_2d5f_0039).
func (g *Game) startSceneClock() {
	g.sceneOn = true
	if !g.textEnabled && g.controlMode&controlModeCharacter == 0 {
		g.animateFaces(1)
	}
	clock.Disable(playClockSlot)
}

// stopSceneClock (C: ovr_2d5f_003e).
func (g *Game) stopSceneClock() {
	g.sceneOn = false
	clock.Enable(playClockSlot)
}

// setSceneRegion makes the rectangle the one clickable region and shows the cursor;
// a full width region puts the lamp out of the way (C: ovr_2d5f_0020).
func (g *Game) setSceneRegion(x, y, w, h, with int) {
	g.buildRegionButtons(x, y, w, h, with)
	g.sceneRegion = [4]int{x, y, x + w, y + h}
	hot := 0
	if g.heldItem != 0 {
		hot = heldCursorHotSpot
	}
	g.Screen.SetMouseShape(hot, hot, g.itemIcon(g.heldItem))
	g.lastMouseRegion = noArrow
	if w == gfx.Width {
		g.hideLamp(false)
		g.lampPending = true
	}
}

// hideLamp puts the lamp out, maybe drawing its unlit icon (C: ovr_2d5f_0133).
func (g *Game) hideLamp(draw bool) {
	g.setFlag(flagLampLit, false)
	if g.flag(flagHasLamp) && draw {
		const unlitLampShape = 43
		s := g.Screen
		s.HideMouseIn(lampIconX, lampIconY, lampIconRight, lampIconBottom)
		s.DrawShape(pageScreen, shapeAt(g.gameShapes, unlitLampShape), lampIconX, lampIconY, 0, 0, gfx.ShapeOptions{})
		s.ShowMouseIn()
		g.lampLevel = lampOut
	}
}

// The scene region of a dialogue scene: the view (C: ovr_2d5f_00fc).
const (
	sceneViewX     = 112
	sceneViewY     = 0
	sceneViewW     = 176
	sceneViewH     = 120
	updateInScene  = 1 << 0 // a scene plays: doors move silently (C: g_7fd6 & 1)
	updateSceneBox = updateInScene | updateNoPortraits
)

// startScene opens a scene in the view (C: ovr_2d5f_00fc).
func (g *Game) startScene(how int) {
	g.startSceneClock()
	g.setSceneRegion(sceneViewX, sceneViewY, sceneViewW, sceneViewH, how)
	g.updateFlags |= updateSceneBox
	g.saveUnderTextBox()
	g.unrollTextBox()
	g.setUpTextWindows()
	g.lockControls(how)
}

// endScene closes it, its animations and, with how 1, redraws the game screen
// (C: ovr_2d5f_0101).
func (g *Game) endScene(how int) {
	g.unlockControls()
	g.closeTextBox()
	g.resetSceneButtons()
	for i := range g.animations {
		g.closeSceneAnimation(i)
	}
	g.updateFlags = 0
	if how == 1 {
		g.redrawAfterScene()
	}
	g.flushInput()
	g.stopSceneClock()
}

// redrawAfterScene (C: ovr_2d5f_00f2).
func (g *Game) redrawAfterScene() {
	if g.paletteState != paletteBlack {
		g.fadeOutView(fadeTicks)
	}
	g.drawGameScreen()
	g.setBrightness(g.Palette[:], g.brightness, g.lampLevel)
	g.paletteState = paletteNormal
}

// The dialogue box: the part of the screen it covers and the strip its edge is drawn
// from as it slides (C: ovr_2d5f_00f7, ovr_2d5f_013d).
const (
	textBoxX          = 80
	textBoxY          = 142
	textBoxW          = 240
	textBoxH          = 37
	slideX            = 83
	slideW            = 235
	slideH            = 3
	slideTop          = 140
	slideBottom       = 177
	textWindowMessage = 3
	scratchRows       = 40 // the rows of the build page the original used as scratch memory
	stripRows         = 10
	frameOn           = 212
	frameOff          = 1
	frameTop          = 143
	frameBottom       = 176
	frameWidth        = 65
)

// saveUnderTextBox keeps what the dialogue box will cover (C: ovr_2d5f_00f7 mode 1).
func (g *Game) saveUnderTextBox() {
	s := g.Screen
	if !g.textEnabled {
		g.drawTalkerFrame(true)
		return
	}
	s.HideMouse()
	g.textBoxBackground = s.SaveRect(textBoxX, textBoxY, textBoxW, textBoxH, pageScreen)
	s.ShowMouse()
}

// closeTextBox rolls the dialogue box up again, from the bottom
// (C: ovr_2d5f_00f7 mode 0).
func (g *Game) closeTextBox() {
	s := g.Screen
	if !g.textEnabled {
		s.HideMouse()
		g.Text.SelectWindow(g.Text.SelectAndClear(textWindowLabel))
		s.ShowMouse()
		g.drawTalkerFrame(true)
		return
	}
	// The original kept the build page's top rows in memory and put them back afterwards.
	scratch := s.SaveRect(0, 0, gfx.Width, scratchRows, pageBuild)
	s.HideMouse()
	g.Text.SelectWindow(g.Text.SelectAndClear(textWindowLabel))
	s.ShowMouse()
	old := s.SetPage(pageBuild)
	s.RestoreRect(textBoxX, textBoxY, textBoxW, textBoxH, g.textBoxBackground, pageBuild)
	for y := slideBottom; y > textBoxY-1; y-- {
		next := clock.Ticks() + 1
		s.HideMouse()
		s.CopyRect(slideX, y, slideX, y-1, slideW, slideH, pageScreen, pageScreen)
		s.CopyRect(slideX, y+1, slideX, y+1, slideW, 1, pageBuild, pageScreen)
		s.ShowMouse()
		for clock.Ticks() < next {
			host.Yield()
		}
	}
	s.RestoreRect(0, 0, gfx.Width, scratchRows, scratch, pageBuild)
	s.SetPage(old)
	g.textBoxBackground = nil
	g.updateFlags &^= updateNoPortraits
}

// unrollTextBox draws the dialogue box downwards, its top edge sliding to the bottom
// (C: ovr_2d5f_013d).
func (g *Game) unrollTextBox() {
	s := g.Screen
	if !g.textEnabled {
		g.drawTalkerFrame(false)
		g.Text.SelectAndClear(textWindowMessage)
		return
	}
	s.HideMouse()
	g.messageFading = false
	g.messageID = 0
	g.enableTimer(timerMessage, false)
	g.Text.SelectWindow(g.Text.SelectAndClear(textWindowMessage))
	s.ShowMouse()
	old := s.SetPage(pageBuild)
	strip := s.SaveRect(0, 0, gfx.Width, stripRows, pageBuild)
	s.HideMouse()
	s.CopyRect(slideX, slideTop, 0, 0, slideW, slideH, pageScreen, pageBuild)
	s.ShowMouse()
	for y := slideTop; y < slideBottom; y++ {
		next := clock.Ticks() + 1
		s.HideMouse()
		s.CopyRect(0, 0, slideX, y, slideW, slideH, pageBuild, pageScreen)
		s.ShowMouse()
		for clock.Ticks() < next {
			host.Yield()
		}
	}
	s.RestoreRect(0, 0, gfx.Width, stripRows, strip, pageBuild)
	s.SetPage(old)
	g.updateFlags |= updateNoPortraits
}

// drawTalkerFrame frames, or unframes, the selected character's portrait for a talk
// without the text area (C: ovr_2d5f_0142).
func (g *Game) drawTalkerFrame(on bool) {
	if g.partyCount() == 1 {
		return
	}
	c := frameOff
	if on {
		c = frameOn
	}
	s := g.Screen
	old := s.SetPage(pageScreen)
	s.HideMouse()
	x := g.portraitX[g.selectedCharacter]
	s.DrawRectOutline(x, frameTop, x+frameWidth, frameBottom, c)
	s.ShowMouse()
	s.SetPage(old)
}

// fadeMessage lets message id, or any for 0, fade out, or with now clears it at once
// (C: fn_0845_114d).
func (g *Game) fadeMessage(id int, now bool) {
	if g.messageID == id || id == 0 {
		g.messageFading = true
		g.messageFadeStart = clock.Ticks()
	}
	if now {
		g.endTalk()
		if !g.sceneOn {
			g.Screen.HideMouse()
			old := g.Text.SelectAndClear(textWindowMessage)
			g.Screen.ShowMouse()
			g.Text.SelectWindow(old)
		}
		g.messageFading = false
		g.enableTimer(timerMessage, false)
	}
}

// Scene opcodes.

func (g *Game) opSceneOn(*emc.State) int  { g.startSceneClock(); return 1 } // (C: ovr_2d5f_00c5)
func (g *Game) opSceneOff(*emc.State) int { g.stopSceneClock(); return 1 }  // (C: ovr_2d5f_00ca)

// opStartScene (C: ovr_2d5f_0110).
func (g *Game) opStartScene(s *emc.State) int { g.startScene(s.Arg(0)); return 1 }

// opEndScene (C: ovr_2d5f_0115).
func (g *Game) opEndScene(s *emc.State) int { g.endScene(s.Arg(0)); return 1 }

// The game flag that says the dialogue box is up (C: g_804c bit 7).
const flagTextBox = 7

// opTextBox saves what the box will cover or rolls it up again (C: ovr_2d5f_011a).
func (g *Game) opTextBox(s *emc.State) int {
	g.setFlag(flagTextBox, s.Arg(0) != 0)
	if s.Arg(0) != 0 {
		g.saveUnderTextBox()
	} else {
		g.closeTextBox()
	}
	return 1
}

// opUnrollTextBox (C: ovr_2d5f_011f).
func (g *Game) opUnrollTextBox(*emc.State) int { g.unrollTextBox(); return 1 }

// opSceneRegion (C: ovr_2d5f_0124).
func (g *Game) opSceneRegion(s *emc.State) int {
	g.setSceneRegion(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4))
	return 1
}

// opResetSceneRegion (C: ovr_2d5f_0129).
func (g *Game) opResetSceneRegion(*emc.State) int { g.resetSceneButtons(); return 1 }

// opClearDialogue clears the dialogue window and the label above it, unless the
// controls are in a mode without the text area (C: fn_0fbe_0aa2). The dialogue window,
// unlike the others, is set in pixels.
func (g *Game) opClearDialogue(*emc.State) int {
	if g.controlMode == 0 || g.textEnabled {
		g.clearDialogueText()
	}
	return 1
}
