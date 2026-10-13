package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// Level files (C: g_0eb3, g_0ebf, g_0e54).
const (
	levelInitFile   = "LEVEL%d.INI"
	levelScriptFile = "LEVEL%d.INF"
	playfieldFile   = "PLAYFLD.CPS"
)

// Block script events: the script function of block n runs for the events in its entry
// mask (C: fn_0845_1343's mask).
const (
	levelEntryFunction = blockCount // the level script's function run on entering
	allEvents          = -1
)

// The red tint table the original builds on every level load (C: ovr_2ce1_0043 into
// g_aa03: half brightness plus 16 red over the level's 128 colours).
const (
	redTintScale = 32
	redTintAdd   = 16
)

// loadLevel enters level n: frees the last one, loads the walls, runs the init script
// (with function 1 on the first visit), loads the level script and items and draws the
// game screen (C: ovr_2d2c_006b).
func (g *Game) loadLevel(n int) error {
	g.setFlag(flagLevelChanged, true)
	g.setBusyCursor(busyCursor)
	g.pendingBlockScript = 0
	g.stopMusic()
	g.endTalk()
	g.freeLevel()
	g.clearLevelObjects(true)
	g.resetMonsters()
	g.updateAutomapFlags()
	g.freeMonsterShapes(0)
	g.freeMonsterShapes(1)
	for t := timerLevel5; t <= timerLevel7; t++ {
		g.enableTimer(t, false)
	}
	g.level = n
	g.updateFlags = 0
	g.resetSceneButtons()
	g.loadSpeechIndex(n)
	if err := g.loadWalls(g.level, true); err != nil {
		return err
	}
	g.loadingLevelInit = true
	visited := g.visitedLevels&(1<<(g.level-1)) != 0
	function := 1
	if visited {
		function = 0
	}
	if err := g.runScriptOnce(fmt.Sprintf(levelInitFile, n), function); err != nil {
		return err
	}
	g.loadingLevelInit = false
	if visited {
		g.restoreLevelState(g.level)
	}
	if err := g.loadLevelScript(fmt.Sprintf(levelScriptFile, n)); err != nil {
		return err
	}
	g.placeLevelItems()
	g.pushMonstersOff(g.currentBlock)
	g.redTint = tintedTable(g.Palette[:], redTintScale, redTintAdd, 0, 0, levelColours, true)
	g.walkTransition = 0
	if g.paletteState == paletteToFadeIn {
		g.fadeToBlack(levelFadeTicks)
	}
	g.drawGameScreen()
	g.setBrightness(g.Palette[:], g.brightness, g.lampLevel)
	g.restoreCursor()
	g.playMusic(g.lvl.music)
	g.flushInput()
	return g.scriptErr
}

const levelFadeTicks = 10

// loadLevelScript loads the level's script and runs its entry function
// (C: ovr_2d2c_007a).
func (g *Game) loadLevelScript(name string) error {
	s, err := g.loadScript(name)
	if err != nil {
		return err
	}
	g.levelScript = s
	g.runBlockScript(levelEntryFunction, allEvents)
	return g.scriptErr
}

// runBlockScript runs block b's level script function for an event in mask, its
// arguments in registers 1-4 (C: fn_0845_13d6).
func (g *Game) runBlockScript(b, mask int) { g.runBlockScriptWith(b, mask, -1, 0, 0, 0) }

// runBlockScriptWith is runBlockScript with all arguments (C: fn_0845_1343).
func (g *Game) runBlockScriptWith(b, mask, a1, a2, a3, a4 int) {
	if !g.runningOnce && g.levelScript != nil {
		var st emc.State
		st.Init(g.levelScript)
		st.Start(b)
		regs := []int{mask, a1, a2, a3, a4, b, g.lastMoveDirection}
		for i, v := range regs {
			st.Regs[i] = v
		}
		if st.Active() && mask&st.EntryMask() != 0 {
			st.Run()
			if st.Err != nil && g.scriptErr == nil {
				g.scriptErr = fmt.Errorf("level %d block %d: %w", g.level, b, st.Err)
			}
		}
	}
	g.noteBlockChanged(b)
}

// noteBlockChanged asks for the view to be redrawn if block b is in sight
// (C: fn_0845_046c).
func (g *Game) noteBlockChanged(b int) bool {
	if g.sceneUpdateRequired {
		return true
	}
	for _, v := range g.visibleBlocks[:visibleBlocksChecked] {
		if v == b {
			g.sceneUpdateRequired = true
			return true
		}
	}
	if b == g.currentBlock {
		g.sceneUpdateRequired = true
		return true
	}
	return false
}

const (
	visibleBlocks        = 18
	visibleBlocksChecked = 15
)

// updateAutomapFlags moves the automap marks on as a level is left (C: ovr_2d1b_0061).
func (g *Game) updateAutomapFlags() {
	for i := range g.lvl.blocks {
		b := &g.lvl.blocks[i]
		switch {
		case b.Flags&blockSeen != 0:
			b.Flags &^= blockSeen
			if g.wallSeeThrough(i, 0) && g.wallSeeThrough(i, 1) {
				b.Flags |= blockAutomapLeft
			}
		case b.Flags&blockAutomapLeft != 0:
			b.Flags &^= blockAutomapLeft
		case b.Flags&blockAutomapRight != 0:
			b.Flags &^= blockAutomapRight
		}
	}
}

const (
	blockAutomapLeft  = 1 << 6
	blockAutomapRight = 1 << 7
)

// wallSeeThrough says whether block b's wall on side dir shows neither wall nor
// decoration and the block has no right mark (C: ovr_2d1b_0057).
func (g *Game) wallSeeThrough(b, dir int) bool {
	w := g.lvl.blocks[b].Walls[dir]
	return g.lvl.wallVMP[w] == 0 && g.lvl.wallShape[w] == 0 && g.lvl.blocks[b].Flags&blockAutomapRight == 0
}

// resetSceneButtons rebuilds the buttons and forgets the clickable scene region
// (C: ovr_2d5f_0025).
func (g *Game) resetSceneButtons() {
	g.buildButtons()
	g.sceneRegion = [4]int{}
	g.sceneRegionActive = false
	if g.lampPending {
		g.showLamp()
		g.lampPending = false
	}
}

// showLamp starts the lamp's flicker (C: ovr_2d5f_012e).
func (g *Game) showLamp() {
	g.setFlag(flagLampLit, true)
	g.lampLevel = noLampLevel
	g.updateLamp()
}

// fadeToBlack fades the palette to black (C: fn_0845_08d9).
func (g *Game) fadeToBlack(ticks int) {
	var black [len(g.darkPalette)]byte
	g.Screen.FadePalette(black[:], ticks, nil)
	g.paletteState = paletteBlack
}

// flushInput drops queued keys and clicks (C: fn_2169_062e).
func (g *Game) flushInput() { input.Flush() }

// Not ported yet.

func (g *Game) endTalk() {
	if g.talkingCharacter != noTalk {
		notPorted("fn_130a_0784 (end a character's speech)")
	}
}
func (g *Game) loadSpeechIndex(level int)   {} // sound: the speech index (C: ovr_2da3_0020)
func (g *Game) restoreLevelState(level int) { notPorted("ovr_2d3c_0025 (restore a visited level)") }
func (g *Game) pushMonstersOff(b int) {
	for o := g.lvl.blocks[b].Objects; o != 0; o = *g.objectNext(o) {
		if o&monsterFlag != 0 {
			notPorted("fn_0845_155d (monsters on the party's block)")
		}
	}
}

const noTalk = -1
