// Package game is Lands of Lore itself: start-up, menus and play, on top of the library
// packages (data, gfx, text, input, clock).
package game

import (
	"errors"
	"fmt"
	"time"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/config"
	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
	"github.com/memmaker/lands-of-lore-1-go/internal/text"
)

// Game is the state the original kept in globals.
type Game struct {
	Files   *data.Files
	Screen  *gfx.Screen
	Text    *text.Printer
	Config  config.Config
	Saves   SaveStore   // where saved games go (C: the game folder)
	Palette gfx.Palette // the working palette (C: g_adea)

	SystemFont *gfx.Font // FONT9P.FNT (C: g_ae4f, then g_7fb8)
	SmallFont  *gfx.Font // FONT6P.FNT (C: g_7fbc)
	GUI        *gui.System

	buttons     *gui.Button // the current button list (C: g_7fae)
	buttonCount int         // (C: g_4b00)
	buttonDefs  [buttonDefCount]buttonDef

	strings      []byte // the language's LANDS.ENG, FRE or GER (C: g_4d6c)
	levelStrings []byte // the current level's strings (C: g_4d68)
	menu         menuState

	itemIcons []byte   // ITEMICN.SHP (C: in page 3)
	cursors   [][]byte // item icons, which are also the cursors (C: g_7d82)

	items     [maxItems]Item // (C: f_2803_1cfd)
	itemTypes []ItemType     // (C: g_7fb2)

	levelScript *emc.Script // (C: f_2803_3611)
	itemScript  *emc.Script // ITEM.INF (C: f_2803_35fd)
	runningOnce bool        // a one-off script is running (C: g_4bf6)

	itemShapes   [][]byte // (C: g_7c92)
	gameShapes   [][]byte // (C: f_2803_00a6)
	thrownShapes [][]byte // (C: g_4e54)
	iceShapes    [][]byte // (C: f_27b1_0000)
	fireShapes   [][]byte // (C: g_088e)
	shadowTable  []byte   // (C: f_2803_030a)
	tintTable    []byte   // (C: f_2803_020a)
	textReady    bool     // (C: g_20d9)

	party [partySize]Character // (C: f_2803_362d)
	level int                  // the current level (C: g_7fe5)
	faces [partySize][][]byte  // (C: g_4c06)

	portraitX         [partySize]int // (C: f_2803_3625)
	controlMode       int            // (C: f_2803_0000[69]; ScummVM: _currentControlMode)
	textEnabled       bool           // (C: g_81fe)
	selectedCharacter int            // (C: g_7fe3)
	updateFlags       int            // scene and screen update state (C: g_7fd6; ScummVM: _updateFlags)

	selectedSpell    int                 // (C: g_7fd9)
	compassDirection int                 // the direction the compass shows, -1 to redraw (C: g_7fe1)
	lastMouseRegion  int                 // (C: g_804a)
	options          int                 // option bits (C: g_7fda)
	inventory        [inventorySlots]int // (C: g_7fe6)
	availableSpells  [spellSlots]int     // (C: g_81f2)
	scriptVars       [scriptVarCount]int // (C: g_81e2; ScummVM: _globalScriptVars2)
	language         int                 // (C: g_81fd)
	gameFlags        [gameFlagWords]int  // (C: g_804c)

	musicOn, effectsOn                bool // (C: g_35e4, g_35e2)
	digitalEffectsOn, digitalSpeechOn bool // the VOC mixer's switches (C: g_20cd, g_20cf)
	musicVolume, effectsVolume        int  // (C: g_8201, g_8200)
	speechVolume                      int  // (C: g_81ff)

	credits   int            // the party's money (C: g_81e0)
	coinPiles [coinPiles]int // coins on each pile (C: g_09bc)

	lvl          level
	doors        [movingDoors]movingDoor     // (C: g_4ed0)
	stepSearches int                         // (C: g_247f)
	stepTurnsAlt bool                        // (C: g_247d)
	timers       [timerCount]timer           // (C: g_2298)
	monsters     [maxMonsters]Monster        // (C: f_2803_15fa)
	flying       [flyingObjects]flyingObject // (C: g_4dcc)

	brightness    int         // (C: g_81dc)
	lampLevel     int         // (C: g_81de)
	lampFlickerAt int         // (C: g_237e)
	darkPalette   gfx.Palette // the palette at the current brightness (C: g_540e)
	swampPalette  gfx.Palette // level 11 frozen or not: the palette swapped with Palette (C: g_4bcc)
	paletteState  int         // (C: g_20d5)
	musicTrack    int         // (C: g_1e94)

	currentBlock        int                          // the party's block (C: g_7fd0)
	direction           int                          // the way the party faces (C: g_7fe1's source g_7fdf)
	lastMoveDirection   int                          // (C: g_7fd8)
	levelFiles          map[int][]byte               // the level temp files by level (C: _LEVEL%d.TMP)
	visitedLevels       int                          // bit n-1: level n entered before (C: g_81f9)
	loadingLevelInit    bool                         // a level's .INI runs (C: g_206d)
	pendingBlockScript  int                          // a block whose script runs from the main loop (C: g_20d7)
	walkTransition      int                          // (C: g_7fdc)
	redTint             []byte                       // (C: g_aa03)
	sceneUpdateRequired bool                         // (C: g_7fb6)
	visibleBlocks       [visibleBlocks]int           // the blocks in the view (C: g_4eac)
	sceneRegion         [4]int                       // (C: g_4b74, g_4b72, g_4b70, g_4b6e)
	sceneRegionActive   bool                         // (C: g_20ec)
	lampPending         bool                         // (C: g_1216)
	textBoxBackground   []byte                       // what the dialogue box covers (C: g_4b66)
	messageID           int                          // the message shown last (C: g_20f4)
	messageFading       bool                         // the message fades out (C: g_20f6)
	scriptValue         int                          // the value a string's %a prints (C: g_4bea)
	printedLength       int                          // the length of the text printed last (C: g_2170)
	choiceSelected      int                          // the dialogue button lit, from 1 (C: g_ab2a)
	choiceLabels        [maxChoices]string           // (C: g_ab32)
	choiceX, choiceY    int                          // the first button (C: g_ab30, g_ab2e)
	choiceStep          int                          // from one button to the next (C: g_ab2c)
	talkWaitUntil       int                          // a buttonless answer waits until this tick (C: g_2138)
	messageFadeStart    int                          // (C: g_4bc8)
	iconReleaseAt       int                          // the pressed icon may come up from this tick (C: g_ab26)
	talkingCharacter    int                          // the character whose speech shows, or noTalk (C: g_4bf8)
	exitProgram         bool                         // a script quit the game (C: fn_0845_0bf6)
	doorHandled         int                          // a clicked door's script dealt with it (C: g_4be0)
	lastFoundItem       int                          // (C: g_2794)
	nextCharacter       int                          // (C: g_2796)
	queuedSpeech        int                          // (C: g_aba4)
	queuedSpeaker       int                          // (C: g_aba6)
	queuedSpeechStarted bool                         // (C: g_aba7)
	sparePageSaved      bool                         // (C: struct Scene c4a)
	pageBackup          [gfx.Width * gfx.Height]byte // (C: g_4d56)
	pageBackup2         [gfx.Width * gfx.Height]byte // (C: g_4d5a)
	statsShown          int                          // bit per character whose stats the screen showed (C: g_0c4c)
	shownStats          [characterStats]int          // the values on the screen as they count up (C: g_4b18)
	shownAilments       [ailmentIcons]int            // (C: g_4b12)
	characterPicture    int                          // the INVENTn.CPS on the back page, -1 for none (C: g_2136)
	healSparkles        [][]byte                     // (C: g_0afa)
	healGlows           [][]byte                     // (C: g_0aba)
	healTint            []byte                       // (C: f_25fd_0050)
	spell               spellCast                    // (C: g_4d78)
	swarmBlink          bool                         // the swarm shows the frozen view this frame (C: g_14ca)
	delaySounds         bool                         // effect sounds are queued for playDelayedSounds (C: g_2144)
	lightningFlashes    int                          // the lightning callback count, kept from cast to cast (C: g_0c92)
	sceneBoxOpen        bool                         // (C: struct Scene c46)
	controlsHidden      bool                         // (C: struct Scene c47)
	sceneArea           [4]int                       // x, y, w, h (C: struct Scene x, y, w, h)
	animationPoolRow    [animationPools]int          // (C: g_2124, g_2128, g_212c)
	talkStyle           int                          // where it talks (C: g_4bfa)
	talkSteps           int                          // mouth moves left (C: g_4bfc)
	talkRestore         bool                         // set the face back at the end (C: g_4c00)
	scriptErr           error                        // the first error a block script raised

	partyX, partyY int                // the party's position (C: g_7fd2, g_7fd4)
	levelVars      [levelVarCount]int // (C: g_8114)
	lampEffect     int                // (C: g_81dd; ScummVM: _lampEffect)
	compassBroken  int                // (C: g_7fdd; ScummVM: _compassBroken)
	drainMagic     int                // (C: g_7fde; ScummVM: _drainMagic)
	soundVar       int                // (C: g_abb8)
	sceneOn        bool               // a scene runs: play time stands still (C: struct Scene on, f_2803_0044)
	shownIcon      int                // the game screen icon drawn pressed (C: g_ab24)

	blockingSide  int // the side a placement test last reached over (C: g_4d60)
	blockingBlock int // the block whose walls stopped it (C: g_4d5e)

	inventoryStart int  // the first inventory slot the bar shows (C: g_8046)
	screenFrozen   bool // the view is not redrawn and portraits are dimmed (C: g_4d66)

	viewPage     int // the page the 3D view is drawn on (C: g_4d72)
	viewBackPage int // the other one (C: g_4d70)

	sceneReady bool // (C: g_20d3, 1 at start)

	view            viewState     // the 3D view renderer's state
	sceneOverlay    sceneOverlay  // (C: g_4bd0)
	overlayRows     []overlayLine // the overlay kept for the walk animations (C: g_4edc)
	compassNeedle   int           // the needle drawn, or noCompass (C: g_06d5)
	compassSwing    int           // (C: g_06d7)
	compassNextTick int           // (C: g_06d9)

	quit           bool                            // leaves the main loop (C: g_4bf4)
	deadPartyCheck int                             // flags for checkPartyDead, or noPartyCheck (C: g_20d1)
	controlsLocked int                             // the move buttons are off for a scene (C: f_2803_0048, struct Scene c48)
	spellTarget    int                             // the character picked as a spell's target, or noCharacter (C: g_23ba)
	talkEnds       int                             // when the character's speech ends (C: g_4c02)
	animations     [sceneAnimations]sceneAnimation // (C: g_4ee0)
	scenePalette   gfx.Palette                     // the palette scene animations bring (C: g_4edc)

	tims                [timSlots]*timScript // (C: g_20fc)
	tim                 *timScript           // the TIM running (C: g_4bae and the others)
	timFunction         int                  // its function running (C: g_4bb6)
	timResult           int                  // the dialogue's answer (C: g_4bb2)
	timDialogueFunction int                  // the function a dialogue moves on, or noTimFunction (C: g_4bb4)
	dialogueChoices     int                  // (C: g_4bde)
	speechDelay         int                  // ticks speech took, added to the next wait (C: g_213c)

	heldItem int // the item on the cursor, 0 for none (C: g_8048)
	busyIcon int // which icon it shows (C: g_ab1a)
}

// errSetup is main's message when LANDS.CFG is missing or damaged (C: g_18fc, whose
// French and German lines follow the English one).
var errSetup = errors.New("The setup program must be run first.")

// New opens the game folder and reads the configuration (C: main up to the video set-up;
// the memory, disk space and EMS checks have nothing to check here).
func New(dir string) (*Game, error) {
	files, err := data.Open(dir)
	if err != nil {
		return nil, err
	}
	g := &Game{Files: files, Saves: dirStore{dir}, Screen: gfx.NewScreen(), menu: menuState{mouseHovers: true}}
	raw, err := files.Read(config.FileName)
	if err != nil {
		return nil, errSetup
	}
	cfg, ok := config.Decode(raw)
	if !ok {
		return nil, errSetup
	}
	if !cfg.VideoSupported() {
		return nil, fmt.Errorf("video mode %d is not supported", cfg.Video)
	}
	g.Config = cfg
	files.SetLanguage(cfg.Language)
	g.Text = text.New(g.Screen)
	g.GUI = gui.New(g.Screen)
	g.buttonDefs = buttonDefs
	g.timers = newTimers()
	g.talkingCharacter = noTalk
	g.lastFoundItem = -1
	g.characterPicture = -1
	g.sceneReady = true
	g.compassNeedle = noCompass
	g.deadPartyCheck = noPartyCheck
	g.spellTarget = noCharacter
	return g, nil
}

// Run is the program from the video set-up on.
func (g *Game) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(notPortedError)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	host.ShowFrame = g.Screen.Frame
	clock.Reset()
	input.Init(g.Screen)
	if err := g.setUpVideo(); err != nil {
		return err
	}
	input.SetNoMouse(false) // C: main's g_2f43 = 0
	return g.start()
}

// systemFont is the font the video set-up loads (C: main's ovr_2df7_0020 argument).
const systemFont = "FONT9P.FNT"

// setUpVideo clears the screen, loads the system font and sets an all-black palette
// (C: ovr_2df7_0020 in MCGA mode). The original also sampled the retrace bit's polarity
// (ovr_2de6_0020) for its retrace waits, which here are host ticks.
func (g *Game) setUpVideo() error {
	g.Screen.ClearPage(g.Screen.Current)
	font, err := g.loadFont(systemFont)
	if err != nil {
		return err
	}
	g.SystemFont = font
	g.Text.SetFont(font)
	g.Palette = gfx.Palette{}
	g.Screen.SetPalette(g.Palette[:])
	random.Seed(int(time.Now().Unix())) // C: srand(time(0))
	g.Text.SelectWindow(0)
	return nil
}

// A font file starts with its length and then this version word (00 05).
const (
	fontVersionAt = 2
	fontVersion   = 5
)

// loadFont reads a font file and checks its version (C: ovr_2df1_0020).
func (g *Game) loadFont(name string) (*gfx.Font, error) {
	b, err := g.Files.Read(name)
	if err != nil || len(b) <= fontVersionAt+1 || b[fontVersionAt] != 0 || b[fontVersionAt+1] != fontVersion {
		return nil, fmt.Errorf("unable to load font %s", name)
	}
	return gfx.NewFont(b), nil
}
