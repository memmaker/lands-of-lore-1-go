package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// Full screen scenes (C: o2d5f): a script can save the spare page, open a box over
// the lower screen for the text, hide the controls and take a clickable area.
const (
	pageSpare = 5

	sceneBoxTop      = 128
	sceneBoxBottom   = 199
	sceneBoxFill     = 1
	sceneBoxDark     = 136
	sceneBoxLight    = 251
	sceneBoxInner    = 252
	sceneTextTimer   = 11
	sceneTalkerShape = 88 // the frame drawn over each portrait without the text area
	sceneTalkerDX    = 8
	sceneTalkerY     = 142
	sceneNoTalkers   = 1 << 1 // a box mode that draws no talker frames

	updateHideControls = updateInScene | updateGameMenu
	sceneFullHeight    = 130
)

// saveSparePage keeps the spare page while a scene uses it (C: ovr_2d5f_0048).
func (g *Game) saveSparePage() {
	if !g.sparePageSaved {
		copy(g.pageBackup[:], g.Screen.Page(pageSpare))
		g.sparePageSaved = true
	}
}

// restoreSparePage puts it back, closing the animations kept in it
// (C: ovr_2d5f_004d).
func (g *Game) restoreSparePage() {
	if g.sparePageSaved {
		for i := range g.animations {
			if g.animations[i].inSparePage {
				g.closeSceneAnimation(i)
			}
		}
		copy(g.Screen.Page(pageSpare), g.pageBackup[:])
	}
	g.sparePageSaved = false
}

// openSceneBox opens the full width text box over the lower screen (mode non-zero) or
// unrolls the dialogue box (C: ovr_2d5f_0052).
func (g *Game) openSceneBox(mode, page int) {
	s := g.Screen
	if mode != 0 {
		g.enableTimer(sceneTextTimer, false)
		g.messageFading = false
		old := s.SetPage(page)
		s.HideMouse()
		s.FillRect(0, sceneBoxTop, gfx.Width-1, sceneBoxBottom, sceneBoxFill)
		g.drawFrame(0, sceneBoxTop+1, gfx.Width, sceneBoxBottom-sceneBoxTop, sceneBoxDark, sceneBoxLight, noFill)
		g.drawFrame(1, sceneBoxTop+2, gfx.Width-2, sceneBoxBottom-sceneBoxTop-2, sceneBoxDark, sceneBoxLight, sceneBoxInner)
		s.ShowMouse()
		s.SetWindow(textWindowMain, 8, 131, 304, 66)
		s.SetWindow(textWindowLabel, 1, 133, 38, 60)
		g.Text.SelectAndClear(textWindowLabel)
		g.updateFlags |= updateNoPortraits
		g.controlMode = mode
		g.layOutPortraits()
		if !g.textEnabled && mode&sceneNoTalkers == 0 {
			for i := 0; i < g.partyCount(); i++ {
				g.talkStyle = talkCharacter
				g.talkingCharacter = i
				s.HideMouse()
				s.DrawShape(pageScreen, shapeAt(g.gameShapes, sceneTalkerShape), g.portraitX[i]+sceneTalkerDX, sceneTalkerY, 0, 0, gfx.ShapeOptions{})
				s.ShowMouse()
				g.endTalk()
			}
		}
		s.SetPage(old)
	} else {
		g.saveUnderTextBox()
		g.unrollTextBox()
		g.setUpTextWindows()
		g.Text.SelectAndClear(textWindowLabel)
		s.HideMouse()
		g.Text.SelectAndClear(textWindowLabel)
		s.ShowMouse()
	}
	g.controlMode = mode
	g.sceneBoxOpen = true
}

// closeSceneBox closes it again, into control mode mode (C: ovr_2d5f_0057).
func (g *Game) closeSceneBox(mode int) {
	if g.sceneBoxOpen {
		g.endTalk()
		g.controlMode = mode
		g.layOutPortraits()
		s := g.Screen
		if mode != 0 {
			s.SetWindow(textWindowLabel, 11, 124, 28, 45)
			s.SetWindow(textWindowMain, 85, 123, 233, 54)
			s.Windows[textWindowLabel].Background = sceneBoxInner
			s.Windows[textWindowMain].Background = sceneBoxInner
			s.Windows[textWindowLabel].Column, s.Windows[textWindowLabel].Row = 0, 0
			g.updateFlags &^= updateNoPortraits
		} else {
			g.clearDialogueText()
			g.closeTextBox()
		}
	}
	g.sceneBoxOpen = false
}

// clearDialogueText clears the dialogue window and the label above it; the dialogue
// window, unlike the others, is set in pixels (C: fn_057f_0a54's body).
func (g *Game) clearDialogueText() {
	t := g.Text
	t.SelectWindow(textWindowMain)
	g.Screen.HideMouse()
	g.Screen.FillRect(t.X, t.Y, t.X+t.W-2, t.Y+t.H-2, t.Background)
	t.SelectAndClear(textWindowLabel)
	g.Screen.ShowMouse()
	w := &g.Screen.Windows[textWindowLabel]
	w.Column, w.Row = 0, 0
}

// openSceneArea makes the rectangle the clickable area of a scene (C: ovr_2d5f_005c).
func (g *Game) openSceneArea(x, y, w, h, with int) {
	g.buildRegionButtons(x, y, w, h, with)
	g.sceneArea = [4]int{x, y, w, h}
}

// closeSceneArea gives the game screen its buttons back (C: ovr_2d5f_0061).
func (g *Game) closeSceneArea() {
	if g.sceneArea[2]+g.sceneArea[3] != 0 {
		g.buildButtons()
		g.sceneArea = [4]int{}
		g.sceneRegionActive = false
	}
}

// hideControls stops the game screen's updates; with full also the lamp and the
// automap and compass (C: ovr_2d5f_0066).
func (g *Game) hideControls(full int) {
	if full != 0 {
		g.updateFlags |= updateHideControls
		g.hideLamp(false)
	} else {
		g.updateFlags |= updateInScene
	}
	g.controlsHidden = true
	g.controlMode = full
	g.layOutPortraits()
	g.updateArrowCursor()
}

// showControls undoes hideControls (C: ovr_2d5f_006b).
func (g *Game) showControls(full int) {
	if full != 0 {
		g.updateFlags &^= updateHideControls
		g.showLamp()
	} else {
		g.updateFlags &^= updateInScene
	}
	g.controlsHidden = false
	g.updateArrowCursor()
}

// beginFullScene sets a scene up: full screen or in the view, with the text box,
// hidden controls and a faded in palette as asked (C: ovr_2d5f_0070).
func (g *Game) beginFullScene(full, box, hide, keepWalk, with, fade int) {
	g.startSceneClock()
	if full != 0 {
		if hide != 0 {
			g.hideControls(1)
		}
		if keepWalk == 0 {
			g.walkTransition = 0
		}
		if box != 0 {
			g.openSceneBox(full, pageScreen)
		}
		if fade != 0 {
			g.Screen.FadePalette(g.scenePalette[:], fadeTicks, nil)
			g.paletteState = paletteNormal
		}
		g.openSceneArea(0, 0, gfx.Width, sceneFullHeight, with)
		return
	}
	if hide != 0 {
		g.hideControls(full)
	}
	if keepWalk == 0 {
		g.walkTransition = 0
	}
	g.lockControls(with)
	if fade != 0 {
		copy(g.scenePalette[levelColours*3:], g.Palette[levelColours*3:])
		g.keepInterfaceColours(g.scenePalette[:])
		g.Screen.FadePalette(g.scenePalette[:], fadeTicks, nil)
		g.paletteState = paletteNormal
	}
	if box != 0 {
		g.openSceneBox(full, pageScreen)
	}
	g.openSceneArea(sceneViewX, sceneViewY, sceneViewW, sceneViewH, with)
}

// endFullScene ends it: fades out and redraws the game screen as asked, and with
// closeAll frees its animations and TIMs (C: ovr_2d5f_0075). False without a scene.
func (g *Game) endFullScene(fade, redraw, closeAll, walk int) bool {
	if !g.sceneOn {
		return false
	}
	g.stopSceneClock()
	if g.sceneBoxOpen {
		g.closeSceneBox(g.controlMode)
	}
	if g.controlsHidden {
		g.showControls(g.controlMode)
	}
	save := g.controlMode
	g.controlMode = 0
	g.layOutPortraits()
	g.closeSceneArea()
	g.controlMode = save
	if closeAll != 0 {
		for i := range g.animations {
			g.closeSceneAnimation(i)
		}
		for i := range g.tims {
			g.tims[i] = nil
		}
	}
	g.unlockControls()
	switch {
	case fade != 0:
		if (g.paletteState != paletteBlack && g.paletteState != paletteFadedOut) || (g.paletteState == paletteFadedOut && g.controlMode != 0) {
			if g.controlMode != 0 {
				g.fadeToBlack(fadeTicks)
			} else {
				g.fadeOutView(fadeTicks)
			}
		}
		g.controlMode = 0
		g.layOutPortraits()
		if redraw != 0 {
			g.drawGameScreen()
		}
		g.setBrightness(g.Palette[:], g.brightness, g.lampLevel)
	case redraw != 0:
		g.controlMode = 0
		g.layOutPortraits()
		g.drawGameScreen()
	default:
		g.controlMode = 0
		g.layOutPortraits()
	}
	g.walkTransition = walk
	return true
}

func (g *Game) opSaveSparePage(*emc.State) int    { g.saveSparePage(); return 1 }    // (C: ovr_2d5f_0093)
func (g *Game) opRestoreSparePage(*emc.State) int { g.restoreSparePage(); return 1 } // (C: ovr_2d5f_0098)

// opOpenSceneBox: mode, page (C: ovr_2d5f_009d).
func (g *Game) opOpenSceneBox(s *emc.State) int { g.openSceneBox(s.Arg(0), s.Arg(1)); return 1 }

// opCloseSceneBox: mode (C: ovr_2d5f_00a2).
func (g *Game) opCloseSceneBox(s *emc.State) int { g.closeSceneBox(s.Arg(0)); return 1 }

// opOpenSceneArea: x, y, w, h, with (C: ovr_2d5f_00a7).
func (g *Game) opOpenSceneArea(s *emc.State) int {
	g.openSceneArea(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4))
	return 1
}

func (g *Game) opCloseSceneArea(*emc.State) int { g.closeSceneArea(); return 1 } // (C: ovr_2d5f_00ac)

// opHideControls: full (C: ovr_2d5f_00b1).
func (g *Game) opHideControls(s *emc.State) int { g.hideControls(s.Arg(0)); return 1 }

// opShowControls: full (C: ovr_2d5f_00b6).
func (g *Game) opShowControls(s *emc.State) int { g.showControls(s.Arg(0)); return 1 }

// opBeginFullScene: full, box, hide, keep walk, with, fade (C: ovr_2d5f_00bb).
func (g *Game) opBeginFullScene(s *emc.State) int {
	g.beginFullScene(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3), s.Arg(4), s.Arg(5))
	return 1
}

// opEndFullScene: fade, redraw, close all, walk (C: ovr_2d5f_00c0).
func (g *Game) opEndFullScene(s *emc.State) int {
	return boolInt(g.endFullScene(s.Arg(0), s.Arg(1), s.Arg(2), s.Arg(3)))
}

// opSceneIsOn (C: ovr_2d5f_00cf).
func (g *Game) opSceneIsOn(*emc.State) int { return boolInt(g.sceneOn) }

// opRaiseLevel: character, skill; gives the experience to the next level and returns
// the levels gained (C: ovr_2d5f_00d4).
func (g *Game) opRaiseLevel(s *emc.State) int {
	ch, skill := s.Arg(0), s.Arg(1)
	c := &g.party[ch]
	level := c.SkillLevels[skill]
	g.addExperience(ch, skill, experienceNeeded[level]-c.Experience[skill])
	return c.SkillLevels[skill] - level
}

// opLockControls: how (C: ovr_2d5f_00e3).
func (g *Game) opLockControls(s *emc.State) int { return boolInt(g.lockControls(s.Arg(0))) }

// opUnlockControls (C: ovr_2d5f_00e8).
func (g *Game) opUnlockControls(*emc.State) int { return boolInt(g.unlockControls()) }

// opSetPoolStart: pool page, row; where animations kept in a spare page start
// (C: ovr_2d5f_00ed). The port keeps animations apart from the pages, so this only
// records the row.
func (g *Game) opSetPoolStart(s *emc.State) int {
	g.animationPoolRow[s.Arg(0)] = s.Arg(1)
	return 1
}
