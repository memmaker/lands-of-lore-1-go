package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/config"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// The game menu and its dialogs (C: o2d56). Each menu is a window with a title and a
// column of text buttons; the slot list scrolls through the saved games and the audio
// menu has three sliders.

// gameMenuItem is one button of a menu (C: struct MItem): its text id, or one of the
// negative ids menuText works out, and a label printed to its left.
type gameMenuItem struct{ text, x, y, w, h, label, key int }

// gameMenu is one menu (C: struct Menu, f_2612_0000).
type gameMenu struct {
	window, title, kind int
	buttons             int // how many of the items are buttons; sliders follow them
	items               []gameMenuItem
}

// Menu kinds.
const (
	menuPlain   = 0
	menuSlots   = 1 // with the scroll buttons when there are enough saves
	menuSliders = 2
)

// The menus by number.
const (
	menuMain = iota
	menuConfirm
	menuMessage
	menuSlotList
	menuNameEntry
	menuGameOver
	menuOptions
	menuAudio
)

// Special item texts (C: ovr_2d56_007f).
const (
	textSlotFirst      = -2 // -2 to -5: the save slot names shown
	textSlotLast       = -6
	textDifficulty     = -9
	textSmoothMoves    = -10
	textFloatingCursor = -11
	textLanguage       = -12
	textShowText       = -14
	textEndOfSliders   = -1

	sliderMusic   = 3 // the slider items' texts say which volume they set
	sliderEffects = 4
	sliderSpeech  = 5
)

const keyEscape = input.KeyEscape

var gameMenus = [...]gameMenu{
	menuMain: {9, 0x4000, menuPlain, 7, []gameMenuItem{
		{0x4001, 16, 23, 176, 15, 0, 0},
		{0x4002, 16, 40, 176, 15, 0, 0},
		{0x4003, 16, 57, 176, 15, 0, 0},
		{0x4004, 16, 74, 176, 15, 0, 0},
		{0x42d9, 16, 91, 176, 15, 0, 0},
		{0x4006, 16, 108, 176, 15, 0, 0},
		{0x4005, 88, 127, 104, 15, 0, keyEscape},
	}},
	menuConfirm: {11, 0, menuPlain, 2, []gameMenuItem{
		{0x4007, 8, 30, 72, 15, 0, 0},
		{0x4008, 208, 30, 72, 15, 0, 0},
	}},
	menuMessage: {11, 0, menuPlain, 1, []gameMenuItem{
		{0x425e, 208, 30, 72, 15, 0, 0},
	}},
	menuSlotList: {10, 0, menuSlots, 5, []gameMenuItem{
		{textSlotFirst, 8, 39, 256, 15, 0, 0},
		{textSlotFirst - 1, 8, 56, 256, 15, 0, 0},
		{textSlotFirst - 2, 8, 73, 256, 15, 0, 0},
		{textSlotFirst - 3, 8, 90, 256, 15, 0, 0},
		{0x4011, 168, 118, 96, 15, 0, keyEscape},
	}},
	menuNameEntry: {7, 0x4053, menuPlain, 2, []gameMenuItem{
		{0x4012, 8, 38, 72, 15, 0, keyReturn},
		{0x4011, 176, 38, 72, 15, 0, keyEscape},
	}},
	menuGameOver: {11, 0x4013, menuPlain, 2, []gameMenuItem{
		{0x4006, 8, 30, 104, 15, 0, 0},
		{0x4001, 176, 30, 104, 15, 0, 0},
	}},
	menuOptions: {17, 0x400c, menuPlain, 6, []gameMenuItem{
		{textDifficulty, 120, 22, 80, 15, 0x406e, 0},
		{textSmoothMoves, 120, 39, 80, 15, 0x406c, 0},
		{textFloatingCursor, 120, 56, 80, 15, 0x406d, 0},
		{textLanguage, 120, 73, 80, 15, 0x42d5, 0},
		{textShowText, 120, 90, 80, 15, 0x42d2, 0},
		{0x4072, 104, 110, 96, 15, 0, keyEscape},
	}},
	menuAudio: {18, 0x42d9, menuSliders, 1, []gameMenuItem{
		{0x4072, 152, 76, 96, 15, 0, keyEscape},
		{sliderMusic, 128, 22, 114, 14, 0x42db, 0},
		{sliderEffects, 128, 39, 114, 14, 0x42da, 0},
		{sliderSpeech, 128, 56, 114, 14, 0x42dc, 0},
		{textEndOfSliders, 0, 0, 0, 0, 0, 0},
	}},
}

// Key codes the menus react to.
const (
	keyReturn       = 43
	keyKeypadEnter  = 108
	keyBackspace    = 15
	keyScrollUp     = 96
	keyScrollDown   = 98
	buttonScrollUp  = 20
	buttonScrollDn  = 21
	buttonIDMask    = buttonFired - 1
	slotsShown      = 4
	maxMenuButtons  = 16
	saveNameLength  = 60
	nameEntryWindow = 8
	menuScratchWin  = 12 // the window a button is drawn through
)

// Menu text ids and colours.
const (
	stringVersionTitle  = 0x4009 // shown when there is no saved game to list
	stringQuitConfirm   = 0x400a
	stringDeleteConfirm = 0x400b
	stringSaveTitle     = 0x400d
	stringLoadTitle     = 0x400e
	stringDeleteTitle   = 0x400f
	stringEmptySlot     = 0x4010
	stringOn            = 0x4068
	stringOff           = 0x4069
	stringDifficulty    = 0x406f
	stringNoSpace       = 0x42cf
	stringLanguages     = 0x42d6
	stringNotSave       = 0x425c
	stringLoadFailed    = 0x425d
	stringAudioSample   = 0x42e0

	menuTextColour     = 204
	menuTextLitColour  = 254
	menuFrameColour    = 225
	menuTitleY         = 6
	menuNoSavesY       = 42
	menuButtonTextY    = 4
	menuLabelY         = 3
	versionBottom      = 12
	menuPage           = 2
	menuBackgroundPage = 7
	enoughDiskSpace    = 500000
	gameOverMusic      = 325
)

// Menu button flags (C: 0x4443 for the items, 0x4487 for the scroll buttons, 0x6606 and
// 0x4406 for the slider parts). Their bits 1 and 7 are set too, which the library never
// tests.
const (
	menuItemFlags     = gui.EventRightRelease | gui.EventLeftRelease | gui.FlagCapture | gui.FlagNeedsInside
	scrollButtonFlags = gui.EventRightRelease | gui.EventLeftRelease | gui.FlagSticky | gui.FlagNeedsInside
	sliderBarFlags    = gui.EventRightRelease | gui.EventRightHeld | gui.EventLeftRelease | gui.EventLeftHeld | gui.FlagSticky
	sliderEndFlags    = gui.EventRightRelease | gui.EventLeftRelease | gui.FlagSticky
)

// The slider's parts: the left end, the bar and the right end (C: g_11be, g_11c4).
var (
	sliderPartX = [3]int{0, 10, 124}
	sliderPartW = [3]int{10, 114, 10}
)

const (
	sliderLeft      = 0
	sliderBar       = 1
	sliderRight     = 2
	sliderStep      = 10
	sliderMax       = 100
	sliderKnobInset = 2
	sliderGrabX     = 7
	sliderBarWidth  = 114
)

// Game shapes the menus use (C: f_2803_00be[11-14], f_2803_01de[7-9]).
const (
	shapeScrollUp       = gameShapeHit + 11
	shapeScrollDown     = gameShapeHit + 12
	shapeScrollUpDown   = gameShapeHit + 13
	shapeScrollDownDown = gameShapeHit + 14
	shapeSlider         = gameShapeAutomap + 7
	shapeSliderKnob     = gameShapeAutomap + 8
	shapeSliderBlank    = gameShapeAutomap + 9
)

const iconOptions = iconFirstSceneButton

// menuState of the game menu (C: g_4b60, g_4b64, g_11ba, g_11bc, g_11cc, M.names).
type gameMenuState struct {
	list       *gui.Button
	buttons    [maxMenuButtons]gui.Button // (C: f_2803_0592)
	scrollUp   gui.Button                 // (C: g_1142)
	scrollDown gui.Button                 // (C: g_117e)
	background []byte
	saves      int // saved games found (C: g_4b64)
	newSlot    int // 1: the save list has a slot for a new save (C: g_11ba)
	top        int // the save shown first (C: g_11bc)
	shownTop   int // the top the scroll buttons were last set for (C: g_11cc)
	names      [slotsShown]string
	testSound  int // the effect the effects slider plays next (C: g_11ca)
	blinkAt    int // (C: g_14c4)
	blinkOn    bool
}

func newScrollButton(id, key, y int) gui.Button {
	return gui.Button{ID: id, Key1: key, Types: [3]int{gui.DrawShape, gui.DrawShape, gui.DrawShape},
		Flags: scrollButtonFlags, Window: gameMenus[menuSlotList].window, X: 128, Y: y, W: 25, H: 16,
		Colours: [3][2]int{{254, 1}, {254, 1}, {254, 1}}}
}

func (g *Game) initGameMenu() {
	m := &g.gameMenu
	m.newSlot = 1
	m.shownTop = -1
	m.testSound = firstTestSound
	m.scrollUp = newScrollButton(buttonScrollUp, keyScrollUp, 20)
	m.scrollDown = newScrollButton(buttonScrollDn, keyScrollDown, 118)
}

const firstTestSound = 11

// menuText is an item's text: a string id, a save name or one of the settings; false
// for an empty save slot (C: ovr_2d56_007f).
func (g *Game) menuText(id int) (string, bool) {
	switch {
	case id <= textSlotFirst && id >= textSlotLast:
		n := g.gameMenu.names[-id+textSlotFirst]
		return n, n != ""
	case id == textDifficulty:
		id = stringDifficulty + g.difficulty()
	case id == textSmoothMoves:
		id = onOff(g.options&optionEight != 0)
	case id == textFloatingCursor:
		id = onOff(g.options&optionFour != 0)
	case id == textLanguage:
		id = stringLanguages + g.language
	case id == textShowText:
		id = onOff(g.textEnabled)
	}
	return g.text(id), true
}

func onOff(on bool) int {
	if on {
		return stringOn
	}
	return stringOff
}

// windowOrigin is window n's top left corner in pixels.
func (g *Game) windowOrigin(n int) (int, int) {
	w := g.Screen.Windows[n]
	return w.Left(), w.Top()
}

// drawMenuButton draws a menu item as a box with its text, lit while the mouse is on
// it, sunk while pressed; it goes to the screen through page 2 (C: ovr_2d56_0048).
func (g *Game) drawMenuButton(b *gui.Button) {
	s := g.Screen
	page := s.SetPage(menuPage)
	wx, wy := g.windowOrigin(b.Window)
	x, y, w, h := b.X+wx, b.Y+wy, b.W, b.H
	s.SetWindow(menuScratchWin, x/gfx.ColumnWidth, y, w/gfx.ColumnWidth, h)
	kind, colour := 2, menuTextColour
	switch b.State & (gui.StateDown | gui.StateLatched) {
	case gui.StateLatched:
		colour = menuTextLitColour
	case gui.StateDown, gui.StateDown | gui.StateLatched:
		kind, colour = 0, menuTextLitColour
	}
	g.drawWindowBox(menuScratchWin, kind, true)
	text, _ := g.menuText(b.Arg)
	g.printAt(text, x+w>>1, y+menuButtonTextY, colour, 0, printCentred|printOutline)
	if page == pageScreen {
		s.HideMouseIn(x, y, x+w, y+h)
		s.CopyRectColumns(x/gfx.ColumnWidth, y, x/gfx.ColumnWidth, y, w/gfx.ColumnWidth, h, menuPage, pageScreen)
		s.ShowMouseIn()
	}
	s.SetPage(page)
}

// fillButton fills a button's box with a colour, hiding it (C: o2d56's static frame).
func (g *Game) fillButton(b *gui.Button, colour int) {
	s := g.Screen
	wx, wy := g.windowOrigin(b.Window)
	x, y := b.X+wx, b.Y+wy
	if s.Current == pageScreen {
		s.HideMouseIn(x, y, x+b.W, y+b.H)
	}
	s.FillRect(x, y, x+b.W-1, y+b.H-1, colour)
	if s.Current == pageScreen {
		s.ShowMouseIn()
	}
}

// saveMenuBackground keeps what lies under a menu's window (C: ovr_2d56_004d).
func (g *Game) saveMenuBackground(m *gameMenu) {
	g.Text.SelectWindow(m.window)
	w := g.Screen.Windows[m.window]
	g.Screen.HideMouse()
	g.gameMenu.background = g.Screen.SaveRect(w.Left(), w.Top(), w.W*gfx.ColumnWidth, w.H, pageScreen)
	g.Screen.ShowMouse()
}

// restoreMenuBackground puts it back (C: ovr_2d56_0052).
func (g *Game) restoreMenuBackground(m *gameMenu) {
	g.Text.SelectWindow(m.window)
	w := g.Screen.Windows[m.window]
	g.Screen.HideMouse()
	g.Screen.RestoreRect(w.Left(), w.Top(), w.W*gfx.ColumnWidth, w.H, g.gameMenu.background, pageScreen)
	g.Screen.ShowMouse()
}

// buildMenu draws a menu on page 2, makes its buttons and copies it to the screen
// (C: ovr_2d56_0057).
func (g *Game) buildMenu(m *gameMenu) {
	s := g.Screen
	st := &g.gameMenu
	st.list = nil
	s.SetPage(menuPage)
	g.Text.SelectWindow(m.window)
	g.drawWindowBox(g.Text.Window, 2, true)
	win := s.Windows[m.window]
	centre := win.Left() + win.W*gfx.ColumnWidth/2
	if title, ok := g.menuText(m.title); ok && title != "" {
		g.printAt(title, centre, win.Top()+menuTitleY, menuTextLitColour, 0, printCentred|printOutline)
	}
	if _, ok := g.menuText(m.items[0].text); !ok {
		g.printAt(g.text(stringVersionTitle), centre, win.Top()+menuNoSavesY, menuTextColour, 0, printCentred|printOutline)
	}
	i := 0
	for ; i < m.buttons; i++ {
		it := m.items[i]
		if _, ok := g.menuText(it.text); !ok {
			continue
		}
		b := &st.buttons[i]
		*b = gui.Button{ID: i + 1, Key1: it.key, X: it.x, Y: it.y, W: it.w, H: it.h,
			Types: [3]int{gui.DrawFunc, gui.DrawFunc, gui.DrawFunc}, Flags: menuItemFlags,
			Window: m.window, Arg: it.text}
		draw := func(b *gui.Button) { g.drawMenuButton(b) }
		b.Data = [3]any{draw, draw, draw}
		st.list = g.GUI.Append(st.list, b)
		g.GUI.Enable(b)
		g.GUI.Release(b, false)
		g.GUI.Draw(b)
		if it.label != 0 {
			label := g.text(it.label)
			g.printAt(label, b.X+win.Left()-1-s.TextWidth(label), b.Y+win.Top()+menuLabelY, menuTextColour, 0, printOutline)
		}
	}
	if st.saves >= slotsShown && m.kind == menuSlots {
		for _, sb := range []struct {
			b                 *gui.Button
			up, down, latched int
		}{{&st.scrollUp, shapeScrollUp, shapeScrollUpDown, shapeScrollUpDown},
			{&st.scrollDown, shapeScrollDown, shapeScrollDownDown, shapeScrollDownDown}} {
			b := sb.b
			b.Data = [3]any{shapeAt(g.gameShapes, sb.up), shapeAt(g.gameShapes, sb.down), shapeAt(g.gameShapes, sb.latched)}
			b.Next = nil
			b.Window = m.window
			g.GUI.Release(b, false)
			g.GUI.Disable(b)
			g.fillButton(b, menuFrameColour)
			st.list = g.GUI.Append(st.list, b)
		}
	}
	if m.kind == menuSliders {
		k := i
		for ; m.items[i].text != textEndOfSliders; i++ {
			it := m.items[i]
			for j := range sliderPartX {
				b := &st.buttons[k]
				k++
				flags := sliderEndFlags
				if j == sliderBar {
					flags = sliderBarFlags
				}
				*b = gui.Button{ID: k, X: it.x + sliderPartX[j], Y: it.y, W: sliderPartW[j], H: it.h,
					Flags: flags, Window: m.window, Arg: it.text, Index: j}
				b.Func = g.clickSlider
				st.list = g.GUI.Append(st.list, b)
				g.GUI.Enable(b)
				g.GUI.Release(b, false)
			}
			s.DrawShape(s.Current, shapeAt(g.gameShapes, shapeSlider), it.x, it.y, m.window, gfx.ShapeWindowRelative, gfx.ShapeOptions{})
			v := 0
			switch it.text {
			case sliderMusic:
				v = g.musicVolume
			case sliderEffects:
				v = g.effectsVolume
			case sliderSpeech:
				v = g.speechVolume
			}
			g.drawSliderKnob(s.Current, it.x+sliderPartX[sliderBar], it.y, v, 0, m.window)
			if it.label != 0 {
				label := g.text(it.label)
				g.printAt(label, it.x+win.Left()-5-s.TextWidth(label), it.y+win.Top()+menuLabelY, menuTextColour, 0, printOutline)
			}
		}
	}
	s.HideMouse()
	g.Text.SelectWindow(m.window)
	s.CopyRectColumns(win.X, win.Y, win.X, win.Y, win.W, win.H, menuPage, pageScreen)
	s.ShowMouse()
	s.SetPage(pageScreen)
}

// drawSliderKnob moves a slider's knob from old to value (C: ovr_2d56_005c).
func (g *Game) drawSliderKnob(page, x, y, value, old, window int) {
	s := g.Screen
	x += sliderKnobInset
	pg := s.SetPage(page)
	s.HideMouse()
	s.DrawShape(s.Current, shapeAt(g.gameShapes, shapeSliderBlank), x+old, y, window, gfx.ShapeWindowRelative, gfx.ShapeOptions{})
	s.DrawShape(s.Current, shapeAt(g.gameShapes, shapeSliderKnob), x+value, y, window, gfx.ShapeWindowRelative, gfx.ShapeOptions{})
	s.ShowMouse()
	s.SetPage(pg)
}

// clickSlider sets a volume from a click on a slider: the ends step by 10, the bar
// takes the place clicked. Without the device the slider does nothing
// (C: ovr_2d56_0061).
func (g *Game) clickSlider(b *gui.Button) bool {
	var old int
	switch b.Arg {
	case sliderMusic:
		if g.Config.Music == 0 {
			return false
		}
		old = g.musicVolume
	case sliderEffects:
		if g.Config.Effects == 0 {
			return false
		}
		old = g.effectsVolume
	case sliderSpeech:
		if g.Config.Speech == 0 {
			return false
		}
		old = g.speechVolume
	}
	v, x := old, b.X
	switch b.Index {
	case sliderBar:
		mx, _ := g.Screen.Mouse()
		wx, _ := g.windowOrigin(b.Window)
		v = mx - (b.X + wx + sliderGrabX)
	case sliderRight:
		x -= sliderBarWidth
		v += sliderStep
	case sliderLeft:
		v -= sliderStep
		x += sliderStep
	}
	v = max(0, min(v, sliderMax))
	if v == old {
		return false
	}
	g.drawSliderKnob(pageScreen, x, b.Y, v, old, b.Window)
	g.stopSpeech(false)
	switch b.Arg {
	case sliderMusic:
		g.musicVolume = v
		if v != 0 {
			g.setMusicVolume(musicDriverVolume(v))
			g.options |= optionMusic
			if old == 0 {
				g.musicOn = true
				g.playMusic(g.playMusic(-1))
			}
		} else {
			g.options &^= optionMusic
			g.setMusicVolume(0)
			g.stopSong()
			g.musicOn = false
		}
	case sliderEffects:
		g.effectsVolume = v
		if v != 0 {
			g.setEffectVolume(effectDriverVolume(v))
			g.options |= optionSoundEffects
		} else {
			g.options &^= optionSoundEffects
		}
		if g.options&optionSoundEffects != 0 {
			g.effectsOn, g.digitalEffectsOn = true, true
		} else {
			g.setEffectVolume(1) // sic: not 0, which would switch the driver off
			g.effectsOn, g.digitalEffectsOn = false, false
		}
		g.playSound(g.gameMenu.testSound, 0)
		if g.gameMenu.testSound = g.nextSoundEffect(g.gameMenu.testSound); g.gameMenu.testSound == skippedTestEffect {
			g.gameMenu.testSound++
		}
	case sliderSpeech:
		g.speechVolume = v
		g.digitalSpeechOn = v != 0 && g.speechSample != nil
		if g.digitalSpeechOn {
			g.audio.PlayDigital(g.speechSample, speechPriority, effectDriverVolume(v))
		}
	}
	return true
}

// waitMenuButton runs the idle work until a menu button fires and returns its number.
func (g *Game) waitMenuButton() int {
	for {
		id := g.GUI.ProcessKeys(g.gameMenu.list)
		if id&buttonFired != 0 {
			return id & buttonIDMask
		}
		g.idleWork()
		host.Yield()
		if g.quit {
			return 0
		}
	}
}

// clickOptions opens the game menu (C: ovr_2d56_0020).
func (g *Game) clickOptions(b *gui.Button) bool {
	g.setIcon(iconOptions, iconPress)
	g.updateFlags |= updateGameMenu
	if g.screenFrozen {
		g.clickExitCharacterInventory(&scriptButton)
	}
	g.fadeMessage(0, true)
	g.endTalk()
	g.hideLamp(true)
	g.setBusyCursor(0)
	clock.Disable(playClockSlot)
	g.gameMenu.newSlot = 1
	m := &gameMenus[menuMain]
	g.saveMenuBackground(m)
	g.setIcon(iconOptions, iconRelease)
	g.showMainMenu(m)
	for run := true; run && !g.quit; {
		id := g.GUI.ProcessKeys(g.gameMenu.list)
		if id == 0 {
			host.Yield() // sic: the original spun here without its idle work
			continue
		}
		if id&buttonFired != 0 {
			b = gui.Find(g.gameMenu.list, id) // sic: with bit 15 still set it finds nothing
			g.restoreMenuBackground(m)
			switch id & buttonIDMask {
			case 1:
				if g.slotMenu(slotLoad) != 0 {
					run = false
				}
			case 2:
				if g.slotMenu(slotSave) != 0 {
					run = false
				}
			case 3:
				g.slotMenu(slotDelete)
			case 4:
				g.optionsMenu()
			case 5:
				g.audioMenu()
			case 7:
				run = false
			case 6:
				if g.confirm(stringQuitConfirm) {
					g.quitGame()
				}
			}
			if run && !g.quit {
				g.saveMenuBackground(m)
				g.showMainMenu(m)
			}
		}
		g.idleWork()
	}
	g.updateFlags &^= updateGameMenu
	g.GUI.Press(b, false)
	g.restoreCursor()
	g.showLamp()
	g.buildButtons()
	clock.Enable(playClockSlot)
	g.copyViewFrame()
	g.reloadLanguage()
	if g.speechVolume == 0 {
		g.textEnabled = true
	}
	return true
}

// showMainMenu builds the main menu and prints the version at its bottom.
func (g *Game) showMainMenu(m *gameMenu) {
	g.buildMenu(m)
	font := g.Text.SetFont(g.SmallFont)
	t := g.Text
	g.Screen.HideMouse()
	g.printAt(saveVersion, (t.X+1)*gfx.ColumnWidth, t.Y+t.H-versionBottom, menuTextColour, 0, printOutline)
	g.Screen.ShowMouse()
	g.Text.SetFont(font)
}

// quitGame ends the program (C: fn_0845_0bf6).
func (g *Game) quitGame() {
	g.quit = true
	g.exitProgram = true
}

// confirm asks a yes or no question (C: ovr_2d56_0025).
func (g *Game) confirm(msg int) bool {
	m := gameMenus[menuConfirm]
	m.title = msg
	g.saveMenuBackground(&m)
	g.buildMenu(&m)
	yes := g.waitMenuButton() == 1
	g.restoreMenuBackground(&m)
	return yes
}

// showMenuMessage shows a message until its button is clicked (C: ovr_2d56_002a).
func (g *Game) showMenuMessage(msg int) {
	m := gameMenus[menuMessage]
	m.title = msg
	g.saveMenuBackground(&m)
	g.buildMenu(&m)
	g.waitMenuButton()
	g.restoreMenuBackground(&m)
}

// What the slot list is for.
const (
	slotLoad = iota
	slotSave
	slotDelete
)

// lastSlot is the highest save number the list may show: one more when saving
// (C: o2d56's LAST).
func (g *Game) lastSlot(saving bool) int {
	st := &g.gameMenu
	n := st.saves - 1
	if saving {
		n += st.newSlot
	}
	return n
}

// slotMenu lists the saved games to load, save over or delete; non-zero when one was
// loaded or saved, or a delete question answered no (C: ovr_2d56_002f).
func (g *Game) slotMenu(mode int) int {
	st := &g.gameMenu
	saving := mode == slotSave
	st.saves = g.countSaves()
	st.top = max(g.lastSlot(saving), 0)
	g.readSlotNames(saving)
	m := gameMenus[menuSlotList]
	switch mode {
	case slotLoad:
		m.title = stringLoadTitle
	case slotSave:
		m.title = stringSaveTitle
	case slotDelete:
		m.title = stringDeleteTitle
	}
	g.saveMenuBackground(&m)
	g.buildMenu(&m)
	g.updateScrollButtons(saving, true)
	for !g.quit {
		g.updateScrollButtons(saving, false)
		id := g.GUI.ProcessKeys(st.list)
		if id&buttonFired != 0 {
			id &= buttonIDMask
			b := gui.Find(st.list, id)
			switch id {
			case buttonScrollUp:
				st.top = min(st.top+1, g.lastSlot(saving))
				g.readSlotNames(saving)
				g.GUI.DrawList(st.list)
			case buttonScrollDn:
				st.top = max(st.top-1, 0)
				g.readSlotNames(saving)
				g.GUI.DrawList(st.list)
			case len(st.names) + 1:
				g.restoreMenuBackground(&m)
				return 0
			default:
				g.restoreMenuBackground(&m)
				slot := id - 1
				name := saveFileName(st.top - slot)
				switch mode {
				case slotLoad:
					switch g.loadGame(name) {
					case loadDone:
						return 1
					case loadBadSave:
						g.showMenuMessage(stringNotSave)
					default:
						g.showMenuMessage(stringLoadFailed)
					}
				case slotSave:
					if r := g.enterSaveName(slot); r != 0 {
						return r
					}
				case slotDelete:
					if !g.confirm(stringDeleteConfirm) {
						return 1
					}
					n := st.top - slot
					g.Saves.Remove(name)
					g.removeNotes(name)
					for {
						n++
						next := saveFileName(n)
						if !g.Saves.Exists(next) {
							break
						}
						g.Saves.Rename(next, name)
						g.moveNotes(next, name)
						name = next
					}
					return 1
				}
				g.saveMenuBackground(&m)
				g.buildMenu(&m)
				g.updateScrollButtons(saving, true)
			}
			g.GUI.Release(b, false)
		}
		g.idleWork()
		host.Yield()
	}
	return 0
}

func saveFileName(n int) string { return fmt.Sprintf(savedFile, n) }

// countSaves counts the saves numbered from 0 on (C: ovr_2d56_006b).
func (g *Game) countSaves() int {
	n := 0
	for g.Saves.Exists(saveFileName(n)) {
		n++
	}
	return n
}

// updateScrollButtons enables the scroll buttons that can scroll and hides the others
// (C: ovr_2d56_0075).
func (g *Game) updateScrollButtons(saving, force bool) {
	st := &g.gameMenu
	if !force && st.top == st.shownTop {
		return
	}
	st.shownTop = st.top
	if st.top >= slotsShown {
		g.GUI.Enable(&st.scrollDown)
	} else {
		g.GUI.Disable(&st.scrollDown)
		g.fillButton(&st.scrollDown, menuFrameColour)
	}
	if g.lastSlot(saving) > st.top {
		g.GUI.Enable(&st.scrollUp)
	} else {
		g.GUI.Disable(&st.scrollUp)
		g.fillButton(&st.scrollUp, menuFrameColour)
	}
}

// readSlotNames reads the descriptions of the four saves shown; saving, the slot past
// the last save offers a new one (C: ovr_2d56_007a).
func (g *Game) readSlotNames(saving bool) {
	st := &g.gameMenu
	for i := range st.names {
		st.names[i] = ""
		n := st.top - i
		if n < 0 {
			continue
		}
		if n == st.saves {
			if saving {
				st.names[i] = g.text(stringEmptySlot)
			}
			continue
		}
		if b, err := g.Saves.Read(saveFileName(n)); err == nil {
			r := reader{b: b}
			st.names[i] = r.str(saveDescriptionSize)
		}
	}
}

// enterSaveName lets the player name a save and saves; non-zero when saved
// (C: ovr_2d56_0066).
func (g *Game) enterSaveName(slot int) int {
	st := &g.gameMenu
	name := &st.names[slot]
	empty := len(*name) > 0 && (*name)[0] == '['
	if empty && !g.diskHasSpace(enoughDiskSpace) {
		g.showMenuMessage(stringNoSpace)
		return 0
	}
	if empty {
		*name = ""
	}
	m := &gameMenus[menuNameEntry]
	g.saveMenuBackground(m)
	g.buildMenu(m)
	res := 0
	first := textInputClear | textInputFirst
	g.Screen.SetPage(pageScreen)
	g.Text.SelectWindow(nameEntryWindow)
	t := g.Text
	g.Screen.HideMouse()
	g.drawBox(t.X*gfx.ColumnWidth-1, t.Y-1, t.W*gfx.ColumnWidth+2, t.H+2, 4, false)
	g.Screen.ShowMouse()
	for run := true; run && !g.quit; {
		r := g.textInput(name, saveNameLength, nameEntryWindow, st.list, first)
		first = textInputKeep
		if r&buttonFired == 0 {
			continue
		}
		g.GUI.Release(gui.Find(st.list, r&buttonIDMask), false)
		switch r & buttonIDMask {
		case 1:
			if *name != "" {
				if err := g.saveGame(saveFileName(st.top-slot), *name); err != nil && g.scriptErr == nil {
					g.scriptErr = err
				}
				run = false
				res = 1
			}
		case 2:
			run = false
			g.readSlotNames(true)
		}
	}
	g.restoreMenuBackground(m)
	return res
}

// diskHasSpace says whether the disk has room for a new save (C: ovr_2d56_0039);
// the stores here always have.
func (g *Game) diskHasSpace(int) bool { return true }

// Text input flags (C: ovr_2d96_0020's last argument).
const (
	textInputFirst = 1 << 0 // the first call: clear the field
	textInputKeep  = 1 << 1
	textInputClear = 1 << 2
	cursorBlink    = 20
	maxTyped       = '~'
)

// textInput edits text in a window until a button of list fires, Return or Esc; Esc
// empties it. It returns the button with bit 15, or the key that ended it
// (C: ovr_2d96_0020).
func (g *Game) textInput(text *string, maxLen, window int, list *gui.Button, flags int) int {
	s := g.Screen
	input.SetFlags(input.FlagNoRemap)
	input.ClearFlags(input.FlagKeypadButtons)
	page := s.SetPage(pageScreen)
	oldWindow := g.Text.SelectWindow(window)
	t := g.Text
	x0 := t.X * gfx.ColumnWidth
	maxW := t.W*gfx.ColumnWidth - (s.CharWidth('W') + 1)
	buf := []byte(*text)
	w, n := 0, 0
	for n < len(buf) && w < maxW {
		w += s.CharWidth(buf[n])
		n++
	}
	buf = buf[:n]
	if flags&textInputFirst != 0 {
		flags |= textInputClear
	}
	s.HideMouse()
	if flags&textInputClear != 0 {
		t.Clear()
	}
	g.printAt(string(buf), x0, t.Y+2, t.Foreground, t.Background, 0)
	g.blinkCursor(x0+w, true)
	s.ShowMouse()
	result := 0
	for done := false; !done && !g.quit; {
		key := g.GUI.ProcessKeys(list)
		g.blinkCursor(x0+w, true)
		if key&buttonFired != 0 {
			done, result = true, key
			continue
		}
		switch key {
		case 0:
			host.Yield()
		case keyEscape:
			done, result = true, keyEscape
			buf = buf[:0]
		case keyReturn, keyKeypadEnter:
			done, result = true, keyReturn
		case keyBackspace:
			if n == 0 {
				break
			}
			c := buf[n-1]
			g.blinkCursor(x0+w, false)
			w -= s.CharWidth(c)
			g.blinkCursor(x0+w, true)
			buf = buf[:n-1]
			n--
		default:
			c := input.ASCII(key) & lowByteMask
			if (c == ' ' && n == 0) || c < ' ' || c > maxTyped {
				break
			}
			cw := s.CharWidth(byte(c))
			if w+cw < maxW && n < maxLen {
				buf = append(buf, byte(c))
				n++
				s.HideMouse()
				g.blinkCursor(x0+w, false)
				g.printAt(string(rune(c)), x0+w, t.Y+2, t.Foreground, t.Background, 0)
				s.ShowMouse()
				w += cw
				g.blinkCursor(x0+w, true)
			}
		}
		*text = string(buf)
	}
	*text = string(buf)
	input.ClearFlags(input.FlagNoRemap)
	input.SetFlags(input.FlagKeypadButtons)
	g.Text.SelectWindow(oldWindow)
	s.SetPage(page)
	return result
}

// blinkCursor blinks the text cursor at x every 20 ticks; off, it is wiped at once
// (C: ovr_2d96_0025).
func (g *Game) blinkCursor(x int, on bool) {
	st := &g.gameMenu
	if !on {
		st.blinkAt = 0
		st.blinkOn = true
	}
	if st.blinkAt > clock.Ticks() {
		return
	}
	st.blinkOn = !st.blinkOn
	if on {
		st.blinkAt = clock.Ticks() + cursorBlink
	}
	t, s := g.Text, g.Screen
	colour := t.Background
	if st.blinkOn {
		colour = t.Foreground
	}
	s.HideMouse()
	s.FillRect(x, t.Y, x+s.CharWidth('W'), t.Y+t.H-1, colour)
	s.ShowMouse()
}

// gameOverMenu offers to load a game or quit once the party is dead
// (C: ovr_2d56_0034).
func (g *Game) gameOverMenu() {
	g.playMusic(gameOverMusic)
	g.endTalk()
	g.fadeMessage(0, true)
	g.setBusyCursor(0)
	g.updateFlags |= updateGameMenu
	g.hideLamp(true)
	clock.Disable(playClockSlot)
	m := &gameMenus[menuGameOver]
	g.saveMenuBackground(m)
	g.buildMenu(m)
	var b *gui.Button
	for run := true; run && !g.quit; {
		id := g.GUI.ProcessKeys(g.gameMenu.list)
		if id&buttonFired != 0 {
			b = gui.Find(g.gameMenu.list, id) // sic: with bit 15 still set it finds nothing
			g.restoreMenuBackground(m)
			switch id & buttonIDMask {
			case 1:
				g.quitGame()
			case 2:
				if g.slotMenu(slotLoad) != 0 {
					run = false
				}
			}
			if run && !g.quit {
				g.saveMenuBackground(m)
				g.buildMenu(m)
			}
		}
		g.idleWork()
		host.Yield()
	}
	g.restoreCursor()
	g.updateFlags &^= updateGameMenu
	g.showLamp()
	g.GUI.Press(b, false)
	g.buildButtons()
	clock.Enable(playClockSlot)
	g.copyViewFrame()
}

// optionsMenu cycles the difficulty, the two move options, the language and the text
// (C: ovr_2d56_0084).
func (g *Game) optionsMenu() {
	if g.speechVolume == 0 {
		g.textEnabled = true
	}
	m := &gameMenus[menuOptions]
	g.saveMenuBackground(m)
	g.buildMenu(m)
	for run := true; run && !g.quit; {
		id := g.GUI.ProcessKeys(g.gameMenu.list)
		if id&buttonFired != 0 {
			b := gui.Find(g.gameMenu.list, id&buttonIDMask)
			switch id & buttonIDMask {
			case 1:
				old := g.difficulty()
				d := old + 1
				if d > difficultyMask-1 {
					d = 0
				}
				g.options &= 1<<difficultyShift - 1
				g.options |= d << difficultyShift
				g.rescaleMonsterHitPoints(old, d)
			case 2:
				g.options ^= optionEight
			case 3:
				g.options ^= optionFour
			case 4:
				if g.language != 0 {
					g.language--
				} else {
					g.language = len(languageExtensions) - 1 // German
				}
			case 5:
				if g.speechVolume == 0 || g.Config.Speech == 0 {
					g.textEnabled = true
				} else {
					g.textEnabled = !g.textEnabled
				}
				g.setUpTextWindows()
			case 6:
				run = false
			}
			g.GUI.Release(b, false)
			g.GUI.Draw(b)
		}
		g.idleWork()
		host.Yield()
	}
	g.restoreMenuBackground(m)
}

// audioMenu shows the volume sliders and plays a speech sample (C: ovr_2d56_0089).
func (g *Game) audioMenu() {
	m := &gameMenus[menuAudio]
	g.saveMenuBackground(m)
	g.buildMenu(m)
	g.endTalk()
	g.playSpeechSample(stringAudioSample)
	for run := true; run && !g.quit; {
		id := g.GUI.ProcessKeys(g.gameMenu.list)
		if id&buttonFired != 0 {
			b := gui.Find(g.gameMenu.list, id&buttonIDMask)
			if id&buttonIDMask == 1 {
				run = false
			}
			g.GUI.Release(b, false)
			g.GUI.Draw(b)
		}
		g.idleWork()
		host.Yield()
	}
	g.restoreMenuBackground(m)
}

// reloadLanguage loads the strings anew when the menu changed the language, and
// writes the language into LANDS.CFG (C: ovr_2d56_0043).
func (g *Game) reloadLanguage() {
	if g.loadedLanguage == g.language {
		return
	}
	g.loadedLanguage = g.language
	g.Files.SetLanguage(g.language)
	g.Config.Language = g.language
	if err := g.loadLanguageStrings(); err != nil && g.scriptErr == nil {
		g.scriptErr = err
	}
	if err := g.loadLevelStrings(fmt.Sprintf(levelStringsFile, g.level)); err != nil && g.scriptErr == nil {
		g.scriptErr = err
	}
	g.drawGameScreen()
	if raw, err := g.Saves.Read(config.FileName); err == nil {
		if c, ok := config.Decode(raw); ok {
			c.Language = g.language
			g.Saves.Write(config.FileName, c.Encode())
		}
	}
}

const levelStringsFile = "LEVEL%02d"
