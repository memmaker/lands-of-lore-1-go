package game

// Option bits (C: g_7fda): sound effects and music on, two more on/off options the game
// menu shows as entries -11 and -10, and a three-way setting in bits 4-5.
const (
	optionSoundEffects = 1 << 0
	optionMusic        = 1 << 1
	optionFour         = 1 << 2
	optionEight        = 1 << 3
	newGameOptions     = optionSoundEffects | optionMusic | optionEight | 1<<4
)

// What a new game starts with (C: ovr_2d2c_0070; ScummVM: startupNew).
const (
	startCredits   = 41
	startLevel     = 1
	noCompass      = -1
	noMouseRegion  = -1
	spellSlots     = 7
	noSpell        = -1
	scriptVarCount = 8
	scriptVarStart = 256
	defaultVolume  = 75
)

// The three items the party starts with in its inventory.
// inventorySlots is the size of the party inventory (C: g_7fe6[48]).
const inventorySlots = 48

var startInventory = []int{216, 217, 218}

// heroes are the four characters one can start with, by MAIN.EXE's argument 0-3; without
// an argument the game starts with Conrad, and with another number with nobody.
var heroes = []int{-9, -1, -8, -5}

const defaultHero = 3

// startNewGame sets up a new game with a hero and loads the first level
// (C: ovr_2d2c_0070).
func (g *Game) startNewGame(hero int) error {
	g.selectedSpell = 0
	g.compassDirection = noCompass
	g.lastMouseRegion = noMouseRegion
	g.options |= newGameOptions
	g.musicOn, g.effectsOn, g.digitalEffectsOn, g.digitalSpeechOn = true, true, true, true
	g.level = startLevel
	g.giveCredits(startCredits, false)
	for i, t := range startInventory {
		g.inventory[i] = g.newItem(t, 0, 0)
	}
	for i := range g.availableSpells {
		g.availableSpells[i] = noSpell
	}
	g.availableSpells[0] = 0
	// The original asks the sound start-up which drivers it installed; the configured ones
	// stand for them, as every driver is the port's.
	g.language = g.Config.Language
	if g.Config.Speech == 0 {
		g.speechVolume, g.textEnabled = 0, true
	} else {
		g.speechVolume, g.textEnabled = defaultVolume, g.language != 0
	}
	g.musicVolume = volumeIf(g.Config.Music != 0)
	g.effectsVolume = volumeIf(g.Config.Effects != 0)
	g.setUpTextWindows()
	for i := range g.scriptVars {
		g.scriptVars[i] = scriptVarStart
	}
	if hero >= 0 && hero < len(heroes) {
		if _, err := g.addCharacter(heroes[hero]); err != nil {
			return err
		}
	}
	g.buildButtons()
	if err := g.loadLevel(startLevel); err != nil {
		return err
	}
	g.Screen.ShowMouse()
	return nil
}

// volumeIf is the default volume for a sound that is on, else 0.
func volumeIf(on bool) int {
	if on {
		return defaultVolume
	}
	return 0
}

// The two text windows under the view, taller when text is shown (C: fn_0845_1605;
// ScummVM: setupScreenDims).
const (
	textWindowLabel = 4
	textWindowMain  = 5
)

type windowRect struct{ x, y, w, h int }

var textWindowSizes = map[bool][2]windowRect{
	true:  {{11, 124, 28, 45}, {85, 123, 233, 54}},
	false: {{11, 124, 28, 9}, {85, 123, 233, 18}},
}

// setUpTextWindows sizes the text windows (C: fn_0845_1605).
func (g *Game) setUpTextWindows() {
	sizes := textWindowSizes[g.textEnabled]
	for i, n := range []int{textWindowLabel, textWindowMain} {
		r := sizes[i]
		g.Screen.SetWindow(n, r.x, r.y, r.w, r.h)
	}
}
