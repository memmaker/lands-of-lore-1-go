package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// The level scripts' opcodes: what a level's .INI loads and the engine variables.

const noPalette = -1 // opcode 9's palette argument for none

// opDice rolls arg 0 dice of arg 1 sides (C: fn_0fbe_0053).
func (g *Game) opDice(s *emc.State) int { return dice(s.Arg(0), s.Arg(1)) }

// opLoadGraphics loads the level's wall graphics (C: ovr_2d2c_0089).
func (g *Game) opLoadGraphics(s *emc.State) int {
	palette := ""
	if s.Arg(5) != noPalette {
		palette = s.StringArg(5)
	}
	if err := g.loadGraphics(s.StringArg(0), s.Arg(1), s.Arg(2), int(uint16(s.Arg(3))), int(uint16(s.Arg(4))), palette); err != nil {
		s.Err = err
	}
	return 1
}

// opLoadMaze loads the level's map (C: ovr_2d2c_008e).
func (g *Game) opLoadMaze(s *emc.State) int {
	if err := g.loadMaze(s.StringArg(0)); err != nil {
		s.Err = err
	}
	return 1
}

// opLoadMonsterShapes loads a monster's shapes into set arg 1 (C: ovr_2d2c_0093).
func (g *Game) opLoadMonsterShapes(s *emc.State) int {
	if err := g.loadMonsterShapes(s.StringArg(0), s.Arg(1), s.Arg(2)); err != nil {
		s.Err = err
	}
	return 1
}

// opOpenDecorations opens a decoration shape file and its definitions (C: ovr_2d2c_0098).
func (g *Game) opOpenDecorations(s *emc.State) int {
	if err := g.openDecorations(s.StringArg(0), s.StringArg(1), true); err != nil {
		s.Err = err
	}
	return 1
}

// opCloseDecorations (C: ovr_2d2c_009d).
func (g *Game) opCloseDecorations(*emc.State) int {
	g.closeDecorations()
	return 1
}

// opLoadDoorShapes (C: ovr_2d2c_00a2).
func (g *Game) opLoadDoorShapes(s *emc.State) int {
	if err := g.loadDoorShapes(s.StringArg(0), s.Arg(1), s.Arg(2), s.Arg(3) != 0, s.Arg(4) != 0); err != nil {
		s.Err = err
	}
	return 1
}

// Opcode 63's arguments.
const (
	monsterArgStats       = 2
	monsterArgMight       = 9
	monsterArgProtection  = 17
	monsterArgItemProtect = 25
	monsterArgHitPoints   = 26
	monsterArgFlags       = 27
	monsterArgField40     = 29 // the original wrote arg 28 here first and lost it
	monsterArgField42     = 30
	monsterArgSpells      = 31
	monsterArgAttackSkill = 35
	monsterArgDefense     = 37
	monsterArgSounds      = 39

	percentScale = 1 << 8 // stats are kept in 1/256
	percent      = 100
)

// toScale turns a percentage into 1/256.
func toScale(p int) int { return p * percentScale / percent }

// opDefineMonsterType sets monster type arg 0 (C: ovr_2d2c_00a7).
func (g *Game) opDefineMonsterType(s *emc.State) int {
	m := &g.lvl.monsterTypes[s.Arg(0)]
	m.shapeIndex = int(int8(s.Arg(1)))
	widest := 0
	for _, shape := range g.lvl.monsterShapes[m.shapeIndex] {
		if shape != nil {
			widest = max(widest, gfx.ShapeWidth(shape))
		}
	}
	m.maxWidth = int(int8(widest))
	m.fightingStats = [9]int{
		toScale(s.Arg(monsterArgStats)),
		percentScale,
		toScale(s.Arg(monsterArgStats + 1)),
		s.Arg(monsterArgStats + 2),
		toScale(s.Arg(monsterArgStats + 3)),
		toScale(s.Arg(monsterArgStats + 4)),
		toScale(s.Arg(monsterArgStats + 5)),
		toScale(s.Arg(monsterArgStats + 6)),
		0,
	}
	for i := range m.itemsMight {
		m.itemsMight[i] = s.Arg(monsterArgMight + i)
		m.protectionAgainstItems[i] = toScale(s.Arg(monsterArgProtection + i))
	}
	m.itemProtection = s.Arg(monsterArgItemProtect)
	m.hitPoints = s.Arg(monsterArgHitPoints)
	m.flags = s.Arg(monsterArgFlags)
	m.thinkDelay = 1
	m.field40 = s.Arg(monsterArgField40)
	m.field42 = s.Arg(monsterArgField42)
	m.spellCount = s.Arg(monsterArgSpells)
	for i := range m.spells {
		m.spells[i] = s.Arg(monsterArgSpells + 1 + i)
	}
	for i := range m.attackSkill {
		m.attackSkill[i] = int(int8(s.Arg(monsterArgAttackSkill + i)))
		m.defenseSkill[i] = int(int8(s.Arg(monsterArgDefense + i)))
	}
	m.moveSound = int(uint8(s.Arg(monsterArgSounds)))
	m.attackSound = int(uint8(s.Arg(monsterArgSounds + 1)))
	m.sound = int(uint8(s.Arg(monsterArgSounds + 2)))
	return 1
}

// opLoadLevelStrings (C: ovr_2d2c_00ac).
func (g *Game) opLoadLevelStrings(s *emc.State) int {
	if err := g.loadLevelStrings(s.StringArg(0)); err != nil {
		s.Err = err
	}
	return 1
}

// opLoadSoundFile (C: ovr_2d2c_00b6).
func (g *Game) opLoadSoundFile(s *emc.State) int {
	g.loadSoundFile(s.StringArg(0), s.Arg(1))
	return 0
}

// opSetBrightness sets the brightness, now if arg 1 is 1, and returns the old one
// (C: ovr_2d2c_00c5).
func (g *Game) opSetBrightness(s *emc.State) int {
	old := g.brightness
	g.brightness = s.Arg(0)
	if s.Arg(1) == 1 {
		g.setBrightness(g.Palette[:], s.Arg(0), g.lampLevel)
	}
	return old
}

// The two door rows of wall types whose decoration opcode 54 sets (C: g_536e + 3, + 13).
var doorDecorationRows = [2]int{firstDoorType, firstDoorType + doorTypesShown}

// opSetDoorDecoration gives the five steps of both doors a decoration (C: ovr_2d2c_00cf).
func (g *Game) opSetDoorDecoration(s *emc.State) int {
	for _, row := range doorDecorationRows {
		for i := range doorSteps {
			g.lvl.wallShape[row+i] = int(int8(s.Arg(0)))
		}
	}
	return 1
}

// opAssignDecoration (C: ovr_2d2c_00d4).
func (g *Game) opAssignDecoration(s *emc.State) int { return g.assignDecoration(s.Arg(0)) }

// opSetLevelVar sets one of the level variables (C: fn_0fbe_0d8c).
func (g *Game) opSetLevelVar(s *emc.State) int {
	if n := s.Arg(0); n >= 0 && n < levelVarCount { // fix: the original wrote anywhere
		g.levelVars[n] = s.Arg(1)
	}
	return 1
}

const levelVarCount = 100 // (C: g_8114)

// Engine variables of opcodes 47 and 48 (C: fn_0fbe_0dd3, fn_0fbe_0e81).
const (
	varBlock = iota
	varDirection
	varLevel
	varHeldItem
	varBrightness
	varCredits
	varScriptVar
	_
	varUpdateFlags
	varLampEffect
	varWalkTransition
	varCompassBroken
	varDrainMagic
	varSpeechVolume
	varSound
)

// opGetEngineVar (C: fn_0fbe_0dd3).
func (g *Game) opGetEngineVar(s *emc.State) int {
	switch s.Arg(0) {
	case varDrainMagic:
		return g.drainMagic
	case varCompassBroken:
		return g.compassBroken
	case varBlock:
		return g.currentBlock
	case varDirection:
		return g.direction
	case varLevel:
		return g.level
	case varHeldItem:
		return g.heldItem
	case varBrightness:
		return g.brightness
	case varCredits:
		return g.credits
	case varScriptVar:
		return g.scriptVars[s.Arg(1)]
	case varUpdateFlags:
		return g.updateFlags
	case varLampEffect:
		return g.lampEffect
	case varWalkTransition:
		return g.walkTransition
	case varSpeechVolume:
		return g.speechVolume
	case varSound:
		return g.soundVar
	}
	return 0
}

// opSetEngineVar (C: fn_0fbe_0e81).
func (g *Game) opSetEngineVar(s *emc.State) int {
	i, v := s.Arg(1), s.Arg(2)
	switch s.Arg(0) {
	case varDrainMagic:
		g.drainMagic = i
	case varCompassBroken:
		g.compassBroken = i
	case varBlock:
		g.currentBlock = v
		g.partyX, g.partyY = blockPosition(g.currentBlock, blockCentre, blockCentre)
		g.revealAround(g.currentBlock)
	case varDirection:
		g.direction = v
	case varLevel:
		g.level = v
	case varHeldItem:
		g.setHeldItem(v)
	case varBrightness:
		g.brightness = v
	case varCredits:
		g.credits = v
	case varScriptVar:
		g.scriptVars[i] = v
	case varUpdateFlags:
		g.updateFlags = v
		if v == 1 {
			if !g.textEnabled && g.controlMode&controlModeCharacter == 0 {
				g.animateFaces(1)
			}
			clock.Disable(playClockSlot)
		} else {
			clock.Enable(playClockSlot)
		}
	case varLampEffect:
		g.lampEffect = v
	case varWalkTransition:
		g.walkTransition = v
		g.hideIcon()
	}
	return 1
}

const controlModeCharacter = 1 << 1

// opQueueTrack (C: fn_0fbe_1df1).
func (g *Game) opQueueTrack(s *emc.State) int {
	g.queueTrack(s.Arg(0))
	return 1
}

// opSetPendingBlockScript (C: fn_0fbe_25aa).
func (g *Game) opSetPendingBlockScript(s *emc.State) int {
	g.pendingBlockScript = s.Arg(0)
	return 1
}

// Automap reveal.
const (
	blockRevealed = 1<<3 - 1    // (C: flags |= 7)
	automapWall   = 1<<6 | 1<<7 // the wall hides what lies beyond (C: & 0xc0)
	noWall        = -1
)

// The wall to look through and the step to the next block for dx (or dy) -1, 0, 1
// (C: g_223e, g_224a, g_2244, g_2250).
var (
	revealWallX = [3]int{1, noWall, 3}
	revealStepX = [3]int{-1, 0, 1}
	revealWallY = [3]int{2, noWall, 0}
	revealStepY = [3]int{-mapSize, 0, mapSize}
)

// revealAround marks the party's block on the automap and the neighbours it can see,
// once the party has the map (C: fn_09aa_05b3, fn_09aa_06d7).
func (g *Game) revealAround(b int) {
	if !g.flag(flagHasAutomap) {
		return
	}
	g.lvl.blocks[b].Flags |= blockRevealed
	x, y := b%mapSize, b/mapSize
	for _, d := range [8][2]int{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
		g.revealNeighbour(b, x, y, d[0], d[1])
	}
}

// revealNeighbour marks the block at (dx, dy) from b if no wall hides it
// (C: fn_09aa_05d4).
func (g *Game) revealNeighbour(b, x, y, dx, dy int) bool {
	x += dx
	y += dy
	if x < 0 || x >= mapSize || y < 0 || y >= mapSize {
		return false
	}
	hides := func(block, side int) bool {
		return g.lvl.automapData[g.lvl.blocks[block].Walls[side]]&automapWall != 0
	}
	wallX, wallY := revealWallX[dx+1], revealWallY[dy+1]
	if wallX != noWall && hides(b+revealStepX[dx+1], wallX) {
		return false
	}
	if wallY != noWall && hides(b+revealStepY[dy+1], wallY) {
		return false
	}
	n := b + revealStepX[dx+1] + revealStepY[dy+1]
	if wallX != noWall && wallY != noWall && hides(n, wallX) && hides(n, wallY) {
		return false
	}
	g.lvl.blocks[n].Flags |= blockRevealed
	return true
}

func (g *Game) hideIcon() { g.setIcon(0, iconRelease) }

// opSetLevelMusic (C: ovr_2d2c_00de).
func (g *Game) opSetLevelMusic(s *emc.State) int {
	g.lvl.music = s.Arg(0)
	return 1
}
