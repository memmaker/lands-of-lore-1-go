package game

import "github.com/memmaker/lands-of-lore-1-go/internal/gfx"

// The game screen: PLAYFLD.CPS on page 3, built up on page 2 and shown
// (C: ovr_2d2c_0043).
const (
	pageBuild = 2

	compassSourceY   = 32
	compassX         = 36 // in 8-pixel columns
	compassW         = 4
	compassH         = 32
	automapIconX     = 290
	automapIconY     = 32
	gameShapeAutomap = 78 // (C: f_2803_01de lies 78 shapes into f_2803_00a6)
	viewWindow       = 2
	viewFrameColumn  = 14
	viewFrameColumns = 22
	viewFrameH       = 120
	noCompassReading = -1
)

// compassColumn is where the language's compass rose lies on page 2 (C: 14, 19, 28).
var compassColumn = [...]int{14, 19, 28}

// drawGameScreen (C: ovr_2d2c_0043).
func (g *Game) drawGameScreen() {
	s := g.Screen
	if img, err := g.readCPS(playfieldFile); err == nil {
		copy(s.Page(pageScratch), img)
	} else if g.scriptErr == nil {
		g.scriptErr = err
	}
	if g.flag(flagHasCompass) {
		s.CopyRectColumns(compassColumn[min(g.language, len(compassColumn)-1)], compassSourceY, compassX, 0, compassW, compassH, pageBuild, pageBuild)
		g.compassDirection = noCompassReading
	}
	if g.flag(flagHasAutomap) {
		s.DrawShape(pageBuild, g.gameShapes[gameShapeAutomap], automapIconX, automapIconY, 0, 0, gfx.ShapeOptions{})
	}
	page := s.SetPage(pageBuild)
	if g.flag(flagHasMagic) {
		g.drawSpellList()
	} else {
		g.selectedSpell = 0
	}
	if g.flag(flagHasLamp) {
		g.showLamp()
	}
	g.copyViewFrame()
	g.drawScene(pageBuild)
	g.drawPortraits()
	g.drawInventory()
	g.drawCredits(s.Current)
	s.SetPage(page)
	s.HideMouse()
	s.CopyPage(pageBuild, pageScreen)
	g.copyViewFrame()
	s.ShowMouse()
}

// copyViewFrame copies the view's area of the screen to the back page
// (C: fn_0845_08aa).
func (g *Game) copyViewFrame() {
	g.hideMouseInWindow(viewWindow)
	g.Screen.CopyRectColumns(viewFrameColumn, 0, viewFrameColumn, 0, viewFrameColumns, viewFrameH, pageScreen, g.viewBackPage)
	g.Screen.ShowMouseIn()
}

// hideMouseInWindow hides the cursor over window n (C: fn_234e_0003).
func (g *Game) hideMouseInWindow(n int) {
	w := g.Screen.Windows[n]
	g.Screen.HideMouseIn(w.Left(), w.Top(), w.Right()-1, w.Bottom()-1)
}

// drawScene draws the 3D view unless the screen is busy (C: fn_0845_15d9).
func (g *Game) drawScene(page int) {
	if g.updateFlags&updateMenu == 0 && !g.screenFrozen && g.sceneReady && g.lvl.vcnData != nil {
		g.renderScene(page)
	}
}

// The spell list in the top left (C: ovr_2d2c_0020).
const (
	spellListTopX     = 112
	spellListX        = 12
	spellListTopW     = 87
	spellListTopH     = 15
	spellLineH        = 9
	spellListSideY    = 1
	spellListSideW    = 6
	spellLeftSideX    = 201
	spellLeftX        = 17
	spellRightSideX   = 208
	spellRightX       = 89
	spellListFillX    = 21
	spellListFill     = 206
	spellListBottomY  = 16
	spellListBottomH  = 14
	spellTextX        = 24
	spellColour       = 1
	spellChosenColour = 132
	twoSpellLines     = 2 * spellLineH
	threeSpellLines   = 3 * spellLineH
)

func (g *Game) drawSpellList() {
	s := g.Screen
	s.CopyRect(spellListTopX, 0, spellListX, 0, spellListTopW, spellListTopH, pageBuild, pageBuild)
	h := 0
	for _, sp := range g.availableSpells {
		if sp != noSpell {
			h += spellLineH
		}
	}
	if h == twoSpellLines {
		h = threeSpellLines
	}
	if h != 0 {
		s.CopyRect(spellLeftSideX, spellListSideY, spellLeftX, spellListTopH, spellListSideW, h, pageBuild, pageBuild)
		s.CopyRect(spellRightSideX, spellListSideY, spellRightX, spellListTopH, spellListSideW, h, pageBuild, pageBuild)
		s.FillRect(spellListFillX, spellListTopH, spellRightX, h+spellListTopH, spellListFill)
	}
	s.CopyRect(spellListTopX, spellListBottomY, spellListX, h+spellListTopH, spellListTopW, spellListBottomH, pageBuild, pageBuild)
	y := spellListTopH
	for i, sp := range g.availableSpells {
		if sp == noSpell {
			continue
		}
		colour := spellColour
		if i == g.selectedSpell {
			colour = spellChosenColour
		}
		g.printAt(g.text(spells[sp].Name), spellTextX, y, colour, 0, 0)
		y += spellLineH
	}
}

func (g *Game) renderScene(page int) { notPorted("ovr_2cda_0061 (draw the 3D view)") }
