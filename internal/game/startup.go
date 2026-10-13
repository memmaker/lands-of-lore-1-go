package game

import (
	"errors"
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// Files the start-up reads (C: g_10a1, g_10ab, g_1066, g_10d6).
const (
	titlePicture = "title.cps"
	smallFont    = "FONT6P.FNT"
	itemIcons    = "ITEMICN.SHP"
	firstSave    = "_save000.dat"
)

// The version text in the title's corner (C: g_10c5).
const (
	versionText   = "V CD1.02 D"
	versionX      = 260
	versionY      = 193
	versionColour = 103
)

// titleFadeTicks is how long the title fades in.
const titleFadeTicks = 30

// Pages the start-up uses.
const (
	pageScreen  = 0
	pageTitle   = 2
	pageScratch = 3
)

// The main menu (C: g_1052): its text window, and the choices as string ids in the
// language file, without and with a saved game to load.
const (
	menuWindow      = 6
	mainMenu        = 0
	menuLineHeight  = 8
	menuWindowExtra = 11
)

const (
	choiceNewGame  = stringLanguageFile | 584 // "Start a new game", which runs character generation
	choiceIntro    = stringLanguageFile | 585
	choiceHistory  = stringLanguageFile | 733
	choiceLoadGame = stringLanguageFile | 1
	choiceExit     = stringLanguageFile | 586
)

var mainMenuChoices = [2][]int{
	{choiceNewGame, choiceIntro, choiceHistory, choiceExit},
	{choiceNewGame, choiceIntro, choiceHistory, choiceLoadGame, choiceExit},
}

// errNotPorted ends the run where the port does not go further yet.
var errNotPorted = errors.New("not ported yet")

// start is the game proper (C: ovr_2d40_0020 with ovr_2d50_0039 for "LANDS.EXE PLAIN",
// the way the launcher starts it).
func (g *Game) start() error {
	if err := g.showTitle(); err != nil {
		return err
	}
	choice := g.runMainMenu()
	switch choice {
	case choiceNewGame:
		// Character generation is the language folder's LANDS.EXE, which is not ported;
		// it would start this program again with the chosen hero's number, which shows
		// the title once more and goes on below. Until then the default hero starts.
		if err := g.loadGameData(); err != nil {
			return err
		}
		if err := g.startNewGame(defaultHero); err != nil {
			return err
		}
		return g.mainLoop()
	case choiceIntro:
		return fmt.Errorf("intro (LANDS.EXE INTRO): %w", errNotPorted)
	case choiceHistory:
		return fmt.Errorf("history (LANDS.EXE HISTORY): %w", errNotPorted)
	case choiceLoadGame:
		return fmt.Errorf("loading a game: %w", errNotPorted)
	}
	return nil // C: fn_0845_0bf6, exit
}

// showTitle fades in the title picture, loads the second font, the strings and the
// cursor, and prints the version (C: ovr_2d50_0039 up to the menu).
func (g *Game) showTitle() error {
	s := g.Screen
	g.Palette = gfx.Palette{}
	s.SetPalette(g.Palette[:])
	file, err := g.Files.Read(titlePicture)
	if err != nil {
		return err
	}
	title, err := data.DecodeCPS(file)
	if err != nil {
		return err
	}
	copy(s.Page(pageTitle), title.Pixels)
	copy(g.Palette[:], title.Palette)
	s.CopyRectColumns(0, 0, 0, 0, gfx.Width/gfx.ColumnWidth, gfx.Height, pageTitle, pageScreen)
	s.FadePalette(g.Palette[:], titleFadeTicks, nil)

	if g.SmallFont, err = g.loadFont(smallFont); err != nil {
		return err
	}
	g.Text.SetFont(g.SystemFont)
	// The sound index (ovr_2da3_0020) and the MT-32 set-up (LOLSYSEX) belong to the sound
	// port.
	if err := g.loadLanguageStrings(); err != nil {
		return err
	}
	if err := g.loadCursors(); err != nil {
		return err
	}
	g.Text.SetFont(g.SmallFont)
	g.printAt(versionText, versionX, versionY, versionColour, 0, printShadow)
	g.Text.SetFont(g.SystemFont)
	return nil
}

// loadCursors loads the item icons and makes the first, the hand, the cursor
// (C: ovr_2d50_004d and fn_0845_0acf(0)). The original made every icon the cursor in
// turn first, which only left the last one's image in the cursor buffer.
func (g *Game) loadCursors() error {
	file, err := g.Files.Read(itemIcons)
	if err != nil {
		return err
	}
	icons, err := data.DecodeCPS(file)
	if err != nil {
		return err
	}
	g.itemIcons = icons.Pixels
	g.cursors = [][]byte{gfx.ShapeFrame(g.itemIcons, 0)}
	g.Screen.SetMouseShape(0, 0, g.cursors[0])
	return nil
}

// runMainMenu shows the main menu over the title and returns the choice's string id
// (C: the PLAIN branch of ovr_2d50_0039).
func (g *Game) runMainMenu() int {
	s := g.Screen
	s.ShowMouse()
	s.HideMouse()
	hasSave := 0
	if g.Files.Exists(firstSave) {
		hasSave = 1
	}
	choices := mainMenuChoices[hasSave]
	items := make([]string, len(choices))
	for i, id := range choices {
		items[i] = g.text(id)
	}
	textMenus[mainMenu].n = len(choices)
	s.Windows[menuWindow].H = len(choices)*menuLineHeight + menuWindowExtra
	g.Text.SelectWindow(menuWindow)
	win := g.textWindow()
	x, y, w, h := win.Left(), win.Y, win.W*gfx.ColumnWidth, win.H
	under := s.SaveRect(x, y, w, h, pageScreen)
	g.drawBox(x, y, w, h, boxMenu, true)
	g.drawBox(x+1, y+1, w-2, h-2, boxMenu, false)
	g.drawMenu(mainMenu, items, showAll, 0, 0)
	s.ShowMouse()
	input.Flush()
	choice := g.pollMenu(mainMenu, items, menuMaskFirstEight, 0)
	for choice == -1 {
		host.Yield()
		choice = g.pollMenu(mainMenu, items, menuMaskFirstEight, 0)
	}
	s.HideMouse()
	s.RestoreRect(x, y, w, h, under, pageScreen)
	return choices[choice]
}

// menuMaskFirstEight is the mask the main menu polls with: entries 0-7 shown.
const menuMaskFirstEight = 1<<8 - 1
