package game

import "github.com/memmaker/lands-of-lore-1-go/internal/gui"

// buttonDef is a button template (C: struct BtnDef, f_2568_0000; ScummVM: ButtonDef).
type buttonDef struct {
	flags      int
	key1, key2 int
	action     func(*Game, *gui.Button) bool
	x, y, w, h int
	arg        int
	window     int
}

// buttonDefCount is the number of templates.
const buttonDefCount = 95

// eventsBoth fires a button on either mouse button.
const eventsBoth = gui.EventLeftPress | gui.EventRightPress

// buttonDefs are the game's button templates. The list builders move some of them to the
// portraits or the inventory slots before use, so each game keeps its own copy.
var buttonDefs = [buttonDefCount]buttonDef{
	{eventsBoth, 96, 352, (*Game).clickMoveForward, 33, 131, 22, 19, 0, 0},               // 0
	{eventsBoth, 97, 353, (*Game).clickMoveBack, 33, 150, 22, 18, 0, 0},                  // 1
	{eventsBoth, 98, 354, (*Game).clickMoveBack, 33, 150, 22, 18, 0, 0},                  // 2
	{eventsBoth, 92, 348, (*Game).clickStrafeLeft, 11, 150, 22, 18, 0, 0},                // 3
	{eventsBoth, 102, 358, (*Game).clickStrafeRight, 53, 150, 22, 18, 0, 0},              // 4
	{eventsBoth, 91, 347, (*Game).clickTurnLeft, 11, 131, 22, 19, 0, 0},                  // 5
	{eventsBoth, 101, 357, (*Game).clickTurnRight, 53, 131, 22, 19, 0, 0},                // 6
	{gui.EventLeftPress, 112, 0, (*Game).clickAttack, 80, 143, 22, 17, 0, 0},             // 7
	{gui.EventLeftPress, 113, 0, (*Game).clickAttack, 80, 143, 22, 17, 1, 0},             // 8
	{gui.EventLeftPress, 114, 0, (*Game).clickAttack, 80, 143, 22, 17, 2, 0},             // 9
	{gui.EventLeftPress, 115, 0, (*Game).clickAttack, 80, 143, 22, 17, 3, 0},             // 10
	{eventsBoth, 0, 0, (*Game).clickMagic, 80, 160, 22, 17, 0, 0},                        // 11
	{eventsBoth, 0, 0, (*Game).clickMagic, 80, 160, 22, 17, 1, 0},                        // 12
	{eventsBoth, 0, 0, (*Game).clickMagic, 80, 160, 22, 17, 2, 0},                        // 13
	{eventsBoth, 0, 0, (*Game).clickMagic, 80, 160, 22, 17, 3, 0},                        // 14
	{eventsBoth, 0, 0, (*Game).clickMagicSubmenu, 80, 144, 22, 32, 0, 0},                 // 15
	{eventsBoth, 110, 0, (*Game).clickScreen, 0, 0, 320, 200, 3, 0},                      // 16
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 33, 34, 0, 0},         // 17
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 33, 34, 1, 0},         // 18
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 33, 34, 2, 0},         // 19
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 33, 34, 3, 0},         // 20
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 66, 34, 0, 0},         // 21
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 66, 34, 1, 0},         // 22
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 66, 34, 2, 0},         // 23
	{gui.EventLeftPress, 0, 0, (*Game).clickPortraitLeft, 80, 143, 66, 34, 3, 0},         // 24
	{gui.EventLeftPress, 0, 0, (*Game).clickLiveMagicBarsLeft, 80, 143, 12, 34, 0, 0},    // 25
	{gui.EventLeftPress, 0, 0, (*Game).clickLiveMagicBarsLeft, 80, 143, 12, 34, 1, 0},    // 26
	{gui.EventLeftPress, 0, 0, (*Game).clickLiveMagicBarsLeft, 80, 143, 12, 34, 2, 0},    // 27
	{gui.EventLeftPress, 0, 0, (*Game).clickLiveMagicBarsLeft, 80, 143, 12, 34, 3, 0},    // 28
	{gui.EventRightPress, 0, 0, (*Game).clickPortraitRight, 80, 143, 66, 34, 0, 0},       // 29
	{gui.EventRightPress, 0, 0, (*Game).clickPortraitRight, 80, 143, 66, 34, 1, 0},       // 30
	{gui.EventRightPress, 0, 0, (*Game).clickPortraitRight, 80, 143, 66, 34, 2, 0},       // 31
	{gui.EventRightPress, 0, 0, (*Game).clickPortraitRight, 80, 143, 66, 34, 3, 0},       // 32
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 0, 2},          // 33
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 1, 2},          // 34
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 2, 2},          // 35
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 3, 2},          // 36
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 4, 2},          // 37
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 5, 2},          // 38
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 6, 2},          // 39
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 7, 2},          // 40
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 20, 19, 8, 2},          // 41
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 12, 11, 9, 2},          // 42
	{eventsBoth, 0, 0, (*Game).clickCharacterInventorySlot, 0, 0, 12, 11, 10, 2},         // 43
	{eventsBoth, 110, 0, (*Game).clickExitCharacterInventory, 160, 93, 43, 19, 0, 2},     // 44
	{eventsBoth, 0, 0, (*Game).clickSceneDropItem, 0, 102, 88, 19, 0, 2},                 // 45
	{eventsBoth, 0, 0, (*Game).clickSceneDropItem, 88, 102, 88, 19, 1, 2},                // 46
	{eventsBoth, 0, 0, (*Game).clickSceneDropItem, 0, 85, 88, 17, 2, 2},                  // 47
	{eventsBoth, 0, 0, (*Game).clickSceneDropItem, 88, 85, 88, 17, 3, 2},                 // 48
	{eventsBoth, 0, 0, (*Game).clickScenePickUpItem, 0, 0, 176, 120, 0, 2},               // 49
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 106, 179, 21, 21, 0, 0},               // 50
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 127, 179, 21, 21, 1, 0},               // 51
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 148, 179, 21, 21, 2, 0},               // 52
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 169, 179, 21, 21, 3, 0},               // 53
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 190, 179, 21, 21, 4, 0},               // 54
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 211, 179, 21, 21, 5, 0},               // 55
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 232, 179, 21, 21, 6, 0},               // 56
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 253, 179, 21, 21, 7, 0},               // 57
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 274, 179, 21, 21, 8, 0},               // 58
	{eventsBoth, 0, 0, (*Game).clickInventorySlot, 274, 179, 21, 21, 9, 0},               // 59
	{eventsBoth, 0, 0, (*Game).clickInventoryScroll, 85, 179, 21, 21, -1, 0},             // 60
	{eventsBoth, 0, 0, (*Game).clickInventoryScroll, 295, 179, 22, 21, 1, 0},             // 61
	{eventsBoth, 0, 0, (*Game).clickWall, 24, 8, 128, 96, 1, 2},                          // 62
	{eventsBoth, 0, 0, (*Game).clickWall, 0, 0, 176, 120, 0, 2},                          // 63
	{eventsBoth, 0, 0, (*Game).clickDialogueArea, 0, 0, 0, 0, 0, 0},                      // 64
	{eventsBoth, 0, 0, (*Game).clickMoveForward, 112, 0, 176, 16, 1, 0},                  // 65
	{eventsBoth, 0, 0, (*Game).clickMoveBack, 112, 110, 176, 26, 1, 0},                   // 66
	{eventsBoth, 0, 0, (*Game).clickStrafeLeft, 96, 88, 32, 32, 1, 0},                    // 67
	{eventsBoth, 0, 0, (*Game).clickStrafeRight, 272, 88, 32, 32, 1, 0},                  // 68
	{eventsBoth, 0, 0, (*Game).clickTurnLeft, 96, 16, 32, 72, 1, 0},                      // 69
	{eventsBoth, 0, 0, (*Game).clickTurnRight, 272, 16, 32, 72, 1, 0},                    // 70
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 15, 61, 9, 0, 0},                 // 71
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 24, 61, 9, 0, 0},                 // 72
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 33, 61, 9, 0, 0},                 // 73
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 42, 61, 9, 0, 0},                 // 74
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 51, 61, 9, 0, 0},                 // 75
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 60, 61, 9, 0, 0},                 // 76
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 69, 61, 9, 0, 0},                 // 77
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 78, 61, 9, 0, 0},                 // 78
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 87, 61, 9, 0, 0},                 // 79
	{eventsBoth, 0, 0, (*Game).clickSpellListEntry, 24, 96, 61, 9, 0, 0},                 // 80
	{gui.EventLeftPress, 0, 0, (*Game).clickSpellTargetCharacter, 80, 143, 66, 34, 0, 0}, // 81
	{gui.EventLeftPress, 0, 0, (*Game).clickSpellTargetCharacter, 80, 143, 66, 34, 1, 0}, // 82
	{gui.EventLeftPress, 0, 0, (*Game).clickSpellTargetCharacter, 80, 143, 66, 34, 2, 0}, // 83
	{gui.EventLeftPress, 0, 0, (*Game).clickSpellTargetCharacter, 80, 143, 66, 34, 3, 0}, // 84
	{eventsBoth, 110, 0, (*Game).clickSpellTargetScene, 0, 0, 320, 200, 3, 0},            // 85
	{gui.EventLeftPress, 0, 0, (*Game).clickSceneThrow, 112, 0, 88, 80, 0, 0},            // 86
	{gui.EventLeftPress, 0, 0, (*Game).clickSceneThrow, 200, 0, 88, 80, 1, 0},            // 87
	{eventsBoth, 25, 110, (*Game).clickOptions, 11, 178, 21, 22, 0, 0},                   // 88
	{gui.EventLeftPress, 20, 0, (*Game).clickRestParty, 32, 178, 22, 22, 0, 0},           // 89
	{gui.EventLeftPress, 0, 0, (*Game).clickMoneyBox, 291, 96, 27, 24, 0, 0},             // 90
	{gui.EventLeftPress, 0, 0, (*Game).clickCompass, 289, 0, 31, 28, 0, 0},               // 91
	{gui.EventLeftPress, 0, 0, (*Game).clickAutomap, 289, 31, 31, 23, 0, 0},              // 92
	{gui.EventLeftPress, 0, 0, (*Game).clickLamp, 291, 56, 29, 24, 0, 0},                 // 93
	{eventsBoth, 0, 0, (*Game).clickStatsButtons, 220, 98, 43, 19, 0, 0},                 // 94
}

// Template indexes the list builders use.
const (
	buttonAttack            = 7
	buttonMagic             = 11
	buttonMagicSubmenu      = 15
	buttonPortraitLeft      = 17
	buttonPortraitInventory = 21
	buttonLiveMagicBars     = 25
	buttonPortraitRight     = 29
	buttonCharInventorySlot = 33
	buttonRegion            = 64
	buttonSpell             = 71
	buttonSpellTarget       = 81
	portraitActionsX        = 44 // the attack and magic buttons beside a face
	portraitBarsX           = 33
	defaultButtonColour     = 254
	defaultButtonColourDown = 1
	defaultButtonExtra      = 510
	regionWithInventory     = 1 << 0
	regionWithMovement      = 1 << 1
	noInventorySlot         = -1
)

// The game screens' button lists (C: f_25eb_0000 and the following segments).
var (
	mainButtons            = []int{89, 65, 66, 67, 68, 69, 70, 49, 62, 45, 46, 47, 48, 86, 87, 1, 2, 0, 3, 4, 5, 6, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 88, 90, 91, 92, 93}
	inventoryButtons       = []int{44, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 94}
	regionButtons          = []int{buttonRegion}
	regionInventoryButtons = []int{50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 90, 91}
	regionMovementButtons  = []int{1, 2, 0, 3, 4, 5, 6, 65, 66, 67, 68, 69, 70, 90, 91}
	characterViewButtons   = []int{44, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61}
	magicSubmenuButtons    = []int{buttonMagicSubmenu, 16}
	spellTargetButtons     = []int{85}
)

// inventoryLayouts place a character's eleven equipment slots, by body shape; x -1 means
// the slot is missing (C: f_27b1_016a, picked through f_27b1_0160).
var inventoryLayouts = [4][equipmentSlots][2]int{
	{{5, 52}, {80, 52}, {-1, -1}, {-1, -1}, {5, 4}, {5, 28}, {80, 4}, {80, 28}, {5, 93}, {15, 78}, {80, 78}},
	{{5, 25}, {80, 25}, {5, 73}, {80, 73}, {5, 1}, {5, 49}, {80, 1}, {80, 49}, {5, 97}, {-1, -1}, {-1, -1}},
	{{5, 52}, {80, 52}, {-1, -1}, {-1, -1}, {5, 4}, {5, 28}, {80, 4}, {80, 28}, {5, 93}, {15, 78}, {80, 78}},
	{{5, 52}, {80, 52}, {-1, -1}, {-1, -1}, {5, 4}, {5, 28}, {80, 4}, {80, 28}, {-1, -1}, {15, 78}, {80, 78}},
}

// inventoryLayoutOf picks the layout by race (C: f_27b1_0160).
var inventoryLayoutOf = [...]int{0, 0, 3, 1, 2}

// makeButton makes a button from template id (C: o2ceb mkbtn).
func (g *Game) makeButton(id int) *gui.Button {
	d := &g.buttonDefs[id]
	g.buttonCount++
	b := &gui.Button{
		ID:     g.buttonCount,
		Key1:   d.key1,
		Key2:   d.key2,
		Flags:  d.flags,
		X:      d.x,
		Y:      d.y,
		W:      d.w - 1,
		H:      d.h - 1,
		Arg:    d.arg,
		Window: d.window,
		Index:  id,
		Extra:  defaultButtonExtra,
	}
	for i := range b.Colours {
		b.Colours[i] = [2]int{defaultButtonColour, defaultButtonColourDown}
	}
	if action := d.action; action != nil {
		b.Func = func(b *gui.Button) bool { return action(g, b) }
	}
	return b
}

func (g *Game) addButton(id int) *gui.Button {
	b := g.makeButton(id)
	g.buttons = g.GUI.Append(g.buttons, b)
	return b
}

func (g *Game) addButtons(ids []int) {
	for _, id := range ids {
		g.addButton(id)
	}
}

// resetButtons starts a new list (C: o2ceb reset).
func (g *Game) resetButtons() {
	g.GUI.MarkChanged()
	g.buttonCount = 0
	g.buttons = nil
}

// addSpellButtons adds one button per spell the party knows (C: o2ceb addspells).
func (g *Game) addSpellButtons() {
	n := 0
	for i, spell := range g.availableSpells {
		if spell == noSpell {
			continue
		}
		g.buttonDefs[buttonSpell+n].arg = i
		g.addButton(buttonSpell + n)
		n++
	}
}

// addPortraitButtons adds one button per party member from template first on, moved to
// the portrait plus dx (C: o2ceb addrange).
func (g *Game) addPortraitButtons(first, dx int) {
	for i := range g.activeCharacters() {
		g.buttonDefs[first+i].x = g.portraitX[i] + dx
		g.addButton(first + i)
	}
}

// addInventorySlotButtons adds character c's equipment slots (C: o2ceb addinv).
func (g *Game) addInventorySlotButtons(c int) {
	layout := &inventoryLayouts[inventoryLayoutOf[g.party[c].RaceClassSex]]
	for i, at := range layout {
		if at[0] == noInventorySlot {
			continue
		}
		d := &g.buttonDefs[buttonCharInventorySlot+i]
		d.x, d.y = at[0], at[1]
		g.addButton(buttonCharInventorySlot + i).Arg = i
	}
}

// buildButtons makes the main screen's buttons (C: ovr_2ceb_0020).
func (g *Game) buildButtons() {
	g.resetButtons()
	g.addButtons(mainButtons)
	g.addPortraitButtons(buttonAttack, portraitActionsX)
	g.addPortraitButtons(buttonMagic, portraitActionsX)
	g.addPortraitButtons(buttonPortraitLeft, 0)
	g.addPortraitButtons(buttonPortraitRight, 0)
	g.addPortraitButtons(buttonLiveMagicBars, portraitBarsX)
	if g.flag(flagHasMagic) {
		g.addSpellButtons()
	}
}

// buildInventoryButtons makes the character view's buttons (C: ovr_2ceb_0025).
func (g *Game) buildInventoryButtons(c int) {
	g.resetButtons()
	g.addButtons(inventoryButtons)
	g.addInventorySlotButtons(c)
	g.addPortraitButtons(buttonPortraitInventory, 0)
}

// buildRegionButtons makes a list of one clickable region, with the inventory or the
// movement buttons if asked (C: ovr_2ceb_002a).
func (g *Game) buildRegionButtons(x, y, w, h, with int) {
	g.resetButtons()
	d := &g.buttonDefs[buttonRegion]
	d.x, d.y, d.w, d.h = x, y, w, h
	g.addButtons(regionButtons)
	if with&regionWithInventory != 0 {
		g.addButtons(regionInventoryButtons)
	}
	if with&regionWithMovement != 0 {
		g.addButtons(regionMovementButtons)
	}
}

// buildCharacterViewButtons makes the buttons of a character's equipment without the
// portraits (C: ovr_2ceb_002f).
func (g *Game) buildCharacterViewButtons(c int) {
	g.resetButtons()
	g.addButtons(characterViewButtons)
	g.addInventorySlotButtons(c)
}

// buildMagicSubmenuButtons makes the buttons of character c's open spell menu
// (C: ovr_2ceb_0034).
func (g *Game) buildMagicSubmenuButtons(c int) {
	g.resetButtons()
	d := &g.buttonDefs[buttonMagicSubmenu]
	d.x, d.arg = g.portraitX[c]+portraitActionsX, c
	g.addButtons(magicSubmenuButtons)
}

// buildSpellTargetButtons makes the buttons for picking a spell's target
// (C: ovr_2ceb_0039).
func (g *Game) buildSpellTargetButtons() {
	g.resetButtons()
	g.addPortraitButtons(buttonSpellTarget, 0)
	g.addButtons(spellTargetButtons)
}

// Button callbacks not ported yet.

func (g *Game) clickAttack(*gui.Button) bool {
	notPorted("fn_0ca0_0012 (button)")
	return false
}

func (g *Game) clickMagic(*gui.Button) bool {
	notPorted("fn_0ca0_0662 (button)")
	return false
}

func (g *Game) clickMagicSubmenu(*gui.Button) bool {
	notPorted("fn_0ca0_06cb (button)")
	return false
}

func (g *Game) clickScreen(*gui.Button) bool {
	notPorted("fn_0ca0_079a (button)")
	return false
}

func (g *Game) clickPortraitLeft(*gui.Button) bool {
	notPorted("ovr_2d22_0039 (button)")
	return false
}

func (g *Game) clickLiveMagicBarsLeft(*gui.Button) bool {
	notPorted("fn_0ca0_030a (button)")
	return false
}

func (g *Game) clickPortraitRight(*gui.Button) bool {
	notPorted("fn_0ca0_04dc (button)")
	return false
}

func (g *Game) clickCharacterInventorySlot(*gui.Button) bool {
	notPorted("ovr_2d22_004d (button)")
	return false
}

func (g *Game) clickExitCharacterInventory(*gui.Button) bool {
	notPorted("ovr_2d22_0034 (button)")
	return false
}

func (g *Game) clickSceneDropItem(*gui.Button) bool {
	notPorted("fn_0ca0_08af (button)")
	return false
}

func (g *Game) clickScenePickUpItem(*gui.Button) bool {
	notPorted("fn_0ca0_07e9 (button)")
	return false
}

func (g *Game) clickInventorySlot(*gui.Button) bool {
	notPorted("fn_0c59_015f (button)")
	return false
}

func (g *Game) clickInventoryScroll(*gui.Button) bool {
	notPorted("fn_0c59_00ef (button)")
	return false
}

func (g *Game) clickWall(*gui.Button) bool {
	notPorted("fn_13e8_02fa (button)")
	return false
}

func (g *Game) clickDialogueArea(*gui.Button) bool {
	notPorted("ovr_2d5f_002a (button)")
	return false
}

func (g *Game) clickSpellListEntry(*gui.Button) bool {
	notPorted("fn_0ca0_0630 (button)")
	return false
}

func (g *Game) clickSpellTargetCharacter(*gui.Button) bool {
	notPorted("ovr_2d8a_0034 (button)")
	return false
}

func (g *Game) clickSpellTargetScene(*gui.Button) bool {
	notPorted("ovr_2d8a_002f (button)")
	return false
}

func (g *Game) clickSceneThrow(*gui.Button) bool {
	notPorted("fn_0ca0_01cf (button)")
	return false
}

func (g *Game) clickOptions(*gui.Button) bool {
	notPorted("ovr_2d56_0020 (button)")
	return false
}

func (g *Game) clickRestParty(*gui.Button) bool {
	notPorted("ovr_2d7a_0020 (button)")
	return false
}

func (g *Game) clickMoneyBox(*gui.Button) bool {
	notPorted("fn_0ca0_0281 (button)")
	return false
}

func (g *Game) clickCompass(*gui.Button) bool {
	notPorted("fn_0ca0_02b2 (button)")
	return false
}

func (g *Game) clickAutomap(*gui.Button) bool {
	notPorted("fn_0ca0_039a (button)")
	return false
}

func (g *Game) clickLamp(*gui.Button) bool {
	notPorted("fn_0ca0_03cf (button)")
	return false
}

func (g *Game) clickStatsButtons(*gui.Button) bool {
	notPorted("ovr_2d22_0057 (button)")
	return false
}
