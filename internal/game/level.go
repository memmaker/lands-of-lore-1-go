package game

import (
	"encoding/binary"
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// Block is one square of the level map (C: struct Block, g_8203; ScummVM:
// LevelBlockProperty).
type Block struct {
	Walls       [directionCount]int // wall type seen from each side: north, east, south, west
	Objects     int                 // first item or monster here; monsters have bit 15 (C: mon)
	DrawObjects int                 // the draw list built for the 3D view (C: draw)
	Direction   int                 // (C: drawDir)
	Flags       int
}

// Block flags.
const (
	blockSeen          = 1 << 4 // shown on the automap (C: 0x10)
	blockAutomapMarker = 1 << 5
	blockFlagsMask     = 1<<8 - 1
	noDirection        = 5 // Direction of a block nothing is to be drawn for
)

// Wall types index tables of 256 entries the level's WLL file fills.
const wallTypes = 256

// wallFlagDoor and friends are the bits of a wall type's flags (C: g_531e).
const (
	wallPassable    = 1 << 1
	wallDoorMask    = 1<<3 - 1
	wallStopsShift  = 1<<4 | 1<<5 // a moving wall stops at a type with either bit
	automapDoorType = 17          // automap data of a door frame (C: 0x11)
)

// Door wall types: frames 3 to 22, five steps of four doors (C: ovr_2d2c_00a2).
const (
	firstDoorType  = 3
	doorTypes      = 20
	doorSteps      = 5
	doorTypesShown = 10
)

// decoration is a wall decoration, drawn at up to ten distances (C: struct Deco, 72 bytes
// of g_5a52; ScummVM: LevelDecorationProperty).
type decoration struct {
	shapes    [decorationViews]int // index into decorationShapes, -1 for none
	viewFlags [decorationViews]int // bit 0: drawn as its base view, moved by x and y (C: fl)
	x, y      [decorationViews]int
	next      int // another decoration drawn with this one, 0 for none
	flags     int
}

const (
	decorationViews     = 10
	decorationSize      = 72
	maxDecorationShapes = 400
	noShape             = -1
)

// monsterType is a kind of monster (C: struct MonType, 83 bytes of f_2803_1b5e;
// ScummVM: MonsterProperty).
type monsterType struct {
	shapeIndex             int    // which of the three loaded monster shape sets (C: c00)
	maxWidth               int    // the widest shape's width, a signed byte (C: size)
	fightingStats          [9]int // 1/256 factors; [4] scales the attack count (C: a, atk)
	itemsMight             [8]int // (C: dmg)
	protectionAgainstItems [8]int // (C: res)
	itemProtection         int    // (C: w38)
	hitPoints              int    // (C: w3a, maxHp)
	thinkDelay             int    // (C: c3c)
	evade                  int    // never set by opcode 63 (C: c3d)
	flags                  int    // (C: w3e)
	field40                int    // (C: w40)
	field42                int    // copied into each monster (C: w42)
	spellCount             int    // (C: nSpells)
	spells                 [3]int // (C: spells)
	attackSkill            [2]int // chance and kind (C: hitChance, hitKind)
	defenseSkill           [2]int // chance and kind (C: defChance, defKind)
	moveSound              int    // (C: moveSnd)
	attackSound            int    // (C: atkSnd)
	sound                  int    // (C: sound)
}

const (
	monsterTypes       = 5 // (C: f_2803_1b5e is sized for the level's few kinds)
	monsterShapeSets   = 3
	monsterShapeCount  = 16
	monsterAnimations  = 4
	monsterPalettes    = 8
	monsterOverlayBase = 17 // shape index of the first attack/death frame
	monsterPaletteKey  = 16 // the shape whose first columns map the palettes
	paletteKeyRows     = 64
)

// monsterFrame is a monster's animation frame with its offsets (C: struct Ovl of g_7672).
type monsterFrame struct {
	shape []byte
	x, y  int
}

// level is what a level's files and init script set up (the original's many globals of
// o2d2c).
type level struct {
	blocks [blockCount]Block // (C: g_8203)

	wallShape       [wallTypes]int // decoration drawn on the wall, or a negative door (C: g_536e)
	wallVMP         [wallTypes]int // which wall set in the VMP (C: g_53be; ScummVM: _wllVmpMap)
	wallSpecialType [wallTypes]int // (C: f_2803_0056; ScummVM: _specialWallTypes)
	wallFlags       [wallTypes]int // (C: g_531e; ScummVM: _wllWallFlags)
	automapData     [wallTypes]int // (C: f_2803_3835; ScummVM: _wllAutomapData)

	decorationFile   []byte       // the open .SHP of decorations (C: g_4b44)
	decorationDefs   []byte       // the .DAT: 72-byte definitions (C: g_4b4a)
	decorations      []decoration // (C: g_5a52; slot 0 unused)
	decorationShapes [][]byte     // (C: g_5412; slot 0 unused)
	shapeSlot        map[int]int  // file shape -> decorationShapes slot (C: g_4edc)
	decorationSlot   map[int]int  // definition -> decorations slot (C: g_4edc + 4000)

	vcnShift   []byte // per block of the VCN: its palette shift (C: g_7fc0)
	vcnColours []byte // the renderer's colour map for VCN pixels (C: fn_1aff_001c's table)
	vcnData    []byte // the 8x8 4-bit wall and floor blocks (C: g_7fc4)
	vmpData    []int  // wall sets as lists of VCN blocks (C: g_7fc8)
	fadeTables [monsterPalettes][]byte
	tlcTable   []byte // (C: g_4bec)
	tlcLevels  []byte // (C: g_4bf0)

	// saved for reloading the graphics (C: g_4b22, g_4b40, g_4b42, g_4b31)
	graphicsName                 string
	fadeTarget, fadeStep         int
	paletteName                  string
	doorShapes                   [2][]byte                                   // (C: f_2803_058a)
	monsterShapes                [monsterShapeSets][monsterShapeCount][]byte // (C: f_2803_04ca)
	monsterPalettes              [monsterShapeSets][monsterShapeCount][]byte // (C: f_2803_040a)
	monsterFrames                [monsterShapeSets][monsterAnimations][monsterShapeCount]monsterFrame
	monsterAnimType              [monsterShapeSets]int // (C: f_2803_0053)
	monsterTypes                 [monsterTypes]monsterType
	music                        int  // the level's music track (C: g_0cf4)
	firstDecoration, firstShapes bool // (C: g_4b46, g_4b48 restart at 1)
}

// openDecorations opens a decoration set: the shape file stays open and the definitions
// are read; keep adds to the slots loaded so far (C: ovr_2d2c_0034).
func (g *Game) openDecorations(shapeFile, defFile string, keep bool) error {
	lv := &g.lvl
	lv.shapeSlot = map[int]int{}
	lv.decorationSlot = map[int]int{}
	shapes, err := g.Files.Read(shapeFile)
	if err != nil {
		return err
	}
	defs, err := g.Files.Read(defFile)
	if err != nil {
		return err
	}
	lv.decorationFile = shapes
	n := 0
	if len(defs) >= 2 {
		n = int(binary.LittleEndian.Uint16(defs))
	}
	lv.decorationDefs = defs[2:min(2+n*decorationSize, len(defs))]
	if !keep {
		lv.decorations = make([]decoration, 1)
		lv.decorationShapes = make([][]byte, 1)
	}
	return nil
}

// closeDecorations closes the decoration shape file (C: ovr_2d2c_003e).
func (g *Game) closeDecorations() { g.lvl.decorationFile = nil }

// assignDecoration loads definition idx and the shapes it uses, returning its slot
// (C: ovr_2d2c_0039; ScummVM: assignLevelDecorationShapes).
func (g *Game) assignDecoration(idx int) int {
	lv := &g.lvl
	if slot, ok := lv.decorationSlot[idx]; ok && slot != 0 {
		return slot
	}
	n := len(lv.decorations)
	d := readDecoration(lv.decorationDefs, idx)
	lv.decorations = append(lv.decorations, d)
	for j := range decorationViews {
		v := lv.decorations[n].shapes[j]
		if v == noShape {
			continue
		}
		if slot, ok := lv.shapeSlot[v]; ok && slot != 0 {
			lv.decorations[n].shapes[j] = slot
			continue
		}
		slot := len(lv.decorationShapes)
		lv.decorationShapes = append(lv.decorationShapes, gfx.CopyShape(lv.decorationFile, v))
		lv.shapeSlot[v] = slot
		lv.decorations[n].shapes[j] = slot
	}
	lv.decorationSlot[idx] = n
	if next := lv.decorations[n].next; next != 0 {
		lv.decorations[n].next = g.assignDecoration(next)
	}
	return n
}

// readDecoration decodes definition idx of a .DAT.
func readDecoration(defs []byte, idx int) decoration {
	var d decoration
	at := idx * decorationSize
	if at < 0 || at+decorationSize > len(defs) {
		for i := range d.shapes {
			d.shapes[i] = noShape
		}
		return d
	}
	b := defs[at : at+decorationSize]
	word := func(o int) int { return int(int16(binary.LittleEndian.Uint16(b[o:]))) }
	for i := range decorationViews {
		d.shapes[i] = word(2 * i)
		d.viewFlags[i] = int(b[2*decorationViews+i])
		d.x[i] = word(3*decorationViews + 2*i)
		d.y[i] = word(5*decorationViews + 2*i)
	}
	d.next = int(b[7*decorationViews])
	d.flags = int(b[7*decorationViews+1])
	return d
}

// wllRecordSize is one wall type's record in a level's .WLL.
const wllRecordSize = 12

// loadWalls reads level n's wall types; with assign it also loads their decorations
// (C: ovr_2d2c_0075; ScummVM: loadLevelWallData).
func (g *Game) loadWalls(n int, assign bool) error {
	file, err := g.Files.Read(fmt.Sprintf("LEVEL%d.WLL", n))
	if err != nil {
		return err
	}
	if len(file) < 2 {
		return fmt.Errorf("level %d: no wall data", n)
	}
	set := int(binary.LittleEndian.Uint16(file))
	if set < 0 || set >= len(decorationSets) {
		return fmt.Errorf("level %d: decoration set %d", n, set)
	}
	if err := g.openDecorations(decorationSets[set].shapes, decorationSets[set].defs, false); err != nil {
		return err
	}
	lv := &g.lvl
	for at := 2; at+wllRecordSize <= len(file); at += wllRecordSize {
		r := file[at : at+wllRecordSize]
		word := func(o int) int { return int(int16(binary.LittleEndian.Uint16(r[o:]))) }
		idx := word(0) & (wallTypes - 1)
		lv.wallVMP[idx] = int(int8(word(2)))
		if assign {
			if sh := word(4); sh > 0 {
				lv.wallShape[idx] = int(int8(g.assignDecoration(sh)))
			} else {
				lv.wallShape[idx] = int(int8(sh))
			}
		}
		lv.wallSpecialType[idx] = int(int8(word(6)))
		lv.wallFlags[idx] = int(uint8(word(8)))
		lv.automapData[idx] = int(int8(word(10)))
	}
	g.closeDecorations()
	return nil
}

// decorationSets name each level's decoration shapes and definitions
// (C: g_0cf6, g_0d26).
var decorationSets = []struct{ shapes, defs string }{
	{"KEEP.SHP", "KEEP.DAT"},
	{"FOREST1.SHP", "FOREST1.DAT"},
	{"MANOR.SHP", "MANOR.DAT"},
	{"CAVE1.SHP", "CAVE1.DAT"},
	{"SWAMP.SHP", "SWAMP.DAT"},
	{"URBISH.SHP", "URBISH.DAT"},
	{"MINE1.SHP", "MINE1.DAT"},
	{"TOWER1.SHP", "TOWER1.DAT"},
	{"YVEL1.SHP", "YVEL1.DAT"},
	{"CATWALK.SHP", "CATWALK.DAT"},
	{"RUIN.SHP", "RUIN.DAT"},
	{"CIMMERIA.SHP", "CIMMERIA.DAT"},
}

// The maze file: a CPS holding a header of three words, the third the bytes per block,
// then the blocks from byte 6, their first four bytes the walls.
const (
	mazeHeaderSize      = 6
	mazeBytesPerBlockAt = 4
)

// loadMaze reads a level's map (C: ovr_2d2c_0048; ScummVM: loadBlockProperties).
func (g *Game) loadMaze(name string) error {
	file, err := g.Files.Read(name)
	if err != nil {
		return err
	}
	img, err := data.DecodeCPS(file)
	if err != nil {
		return err
	}
	maze := img.Pixels
	if len(maze) < mazeHeaderSize {
		return fmt.Errorf("%s: too short", name)
	}
	stride := int(binary.LittleEndian.Uint16(maze[mazeBytesPerBlockAt:]))
	lv := &g.lvl
	lv.blocks = [blockCount]Block{}
	for i := range blockCount {
		b := &lv.blocks[i]
		at := mazeHeaderSize + i*stride
		if at+directionCount <= len(maze) {
			for d := range directionCount {
				b.Walls[d] = int(maze[at+d])
			}
		}
		b.Direction = noDirection
		if lv.automapData[b.Walls[0]] == automapDoorType {
			b.Flags &^= blockSeen
			b.Flags |= blockAutomapMarker
		}
	}
	return nil
}

// VCN and VMP layout: a block count, a shift per block, 128 bytes of colour map, 128
// colours of palette, then the blocks (32 bytes each); the VMP is a word count, then the
// words.
const (
	vcnColourMapSize = 128
	levelColours     = 128
	levelPaletteSize = levelColours * 3
	vcnBlockSize     = 32
	sizeFromFile     = 1<<16 - 1 // a size argument meaning "as big as the file says"
	fadeTableSize    = 256
	tlcTableSize     = 256
	tlcLevelsSize    = 5120
	fullPercent      = 100
	fullAmount       = 255
)

// loadGraphics reads a level's VCN and VMP, its palette and colour tables (C:
// ovr_2d2c_004d; ScummVM: loadLevelGraphics). An empty name reloads the last set.
func (g *Game) loadGraphics(name string, fadeTarget, fadeStep, vcnSize, vmpSize int, palette string) error {
	lv := &g.lvl
	if name != "" {
		lv.graphicsName, lv.fadeTarget, lv.fadeStep, lv.paletteName = name, fadeTarget, fadeStep, palette
	} else {
		name, fadeTarget, fadeStep, palette = lv.graphicsName, lv.fadeTarget, lv.fadeStep, lv.paletteName
	}
	vcn, err := g.readCPS(name + ".VCN")
	if err != nil {
		return err
	}
	if len(vcn) < 2 {
		return fmt.Errorf("%s.VCN: too short", name)
	}
	n := int(binary.LittleEndian.Uint16(vcn))
	p := vcn[2:]
	if vcnSize == sizeFromFile {
		vcnSize = n * vcnBlockSize
	}
	lv.vcnShift = append([]byte(nil), p[:n]...)
	p = p[n:]
	lv.vcnColours = append([]byte(nil), p[:vcnColourMapSize]...)
	p = p[vcnColourMapSize:]
	pal := append([]byte(nil), p[:levelPaletteSize]...)
	if palette != "" {
		raw, err := g.Files.Read(palette)
		if err != nil {
			return err
		}
		copy(pal, raw)
	}
	copy(g.Palette[:levelPaletteSize], pal)
	g.swapLevelPalette()
	p = p[levelPaletteSize:]
	lv.vcnData = append([]byte(nil), p[:min(vcnSize, len(p))]...)

	vmp, err := g.readCPS(name + ".VMP")
	if err != nil {
		return err
	}
	if vmpSize == sizeFromFile {
		vmpSize = len(vmp) - 2
	}
	words := vmp[2:min(2+vmpSize, len(vmp))]
	lv.vmpData = make([]int, len(words)/2)
	for i := range lv.vmpData {
		lv.vmpData[i] = int(binary.LittleEndian.Uint16(words[2*i:]))
	}

	last := len(lv.fadeTables) - 1
	for i := range lv.fadeTables {
		if i == last {
			lv.fadeTables[i] = identityTable()
			continue
		}
		k := fullPercent - i*fadeStep
		if k <= 0 {
			k = 0
		} else {
			k = k * fullAmount / fullPercent
		}
		t := gfx.MixTableLow(g.Palette[:], fadeTarget, k)
		for j := range levelColours {
			if t[j] == fullAmount {
				t[j] = 0
			}
		}
		for j := levelColours; j < fadeTableSize; j++ {
			t[j] = byte(j)
		}
		lv.fadeTables[i] = t
	}
	g.textReady = false
	g.darkenPalette(g.Palette[:], g.brightness, g.lampLevel)
	tlc, err := g.Files.Read(fmt.Sprintf("LEVEL%02d.TLC", g.level))
	if err != nil {
		return err
	}
	lv.tlcTable = append([]byte(nil), tlc[:min(tlcTableSize, len(tlc))]...)
	lv.tlcLevels = append([]byte(nil), tlc[min(tlcTableSize, len(tlc)):min(tlcTableSize+tlcLevelsSize, len(tlc))]...)
	g.textReady = true
	return nil
}

// readCPS reads and unpacks a CPS file (C: ovr_2ded_002f).
func (g *Game) readCPS(name string) ([]byte, error) {
	file, err := g.Files.Read(name)
	if err != nil {
		return nil, err
	}
	img, err := data.DecodeCPS(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return img.Pixels, nil
}

func identityTable() []byte {
	t := make([]byte, fadeTableSize)
	for i := range t {
		t[i] = byte(i)
	}
	return t
}

// Level 11 has a second palette for the first 128 colours, swapped in by a game flag
// (C: ovr_2d1b_002a, g_0b6a, g_8080 bit 2).
const (
	swapPaletteLevel = 11
	swapPaletteFile  = "SWAMPICE.COL"
)

// swapLevelPalette loads level 11's other palette and swaps it in if its flag is set
// (C: ovr_2d1b_002a).
func (g *Game) swapLevelPalette() {
	if g.level != swapPaletteLevel {
		return
	}
	notPorted("ovr_2d1b_002a (level 11 palette swap)")
}

// Brightness: 8 levels; the lamp, if the party has one, lifts dark levels
// (C: fn_0845_02c6).
const (
	brightnessLevels = 8
	brightnessShift  = 5
	lampSteps        = 8
	uiColoursStart   = 192 * 3 // colours 192-195 always come from the screen
	uiColoursSize    = 4 * 3
)

// darkenPalette makes the game palette darker by the brightness setting into the fade
// target (C: fn_0845_02c6 into g_540e).
func (g *Game) darkenPalette(src []byte, brightness, lamp int) {
	dst := g.darkPalette[:]
	copy(dst, src)
	copy(dst[uiColoursStart:uiColoursStart+uiColoursSize], g.Screen.Palette[uiColoursStart:])
	s := (brightnessLevels - brightness) << brightnessShift
	if lamp >= 0 && lamp < lampSteps && g.flag(flagHasLamp) {
		s = fixedOne - scale(fixedOne-s, (lamp&^1)<<brightnessShift)
		s = max(s, 0)
	}
	for i := range levelPaletteSize {
		dst[i] = byte(scale(int(dst[i]), s))
	}
}

// setBrightness darkens the palette and fades to it (C: fn_0845_035b).
func (g *Game) setBrightness(src []byte, brightness, lamp int) {
	g.darkenPalette(src, brightness, lamp)
	g.Screen.FadePalette(g.darkPalette[:], brightnessFadeTicks, nil)
	g.paletteState = paletteNormal
}

const brightnessFadeTicks = 5

// Palette states (C: g_20d5).
const (
	paletteNormal = iota
	paletteFadedOut
	paletteBlack
	paletteToFadeIn
)

// loadMonsterShapes loads a monster's shapes into set (C: ovr_2d2c_0057).
func (g *Game) loadMonsterShapes(name string, set, colour int) error {
	g.freeMonsterShapes(set)
	file, err := g.readCPS(name)
	if err != nil {
		return err
	}
	lv := &g.lvl
	for j := range monsterShapeCount {
		frame := gfx.ShapeFrame(file, j)
		shared := false
		for i := range j {
			if frame != nil && &frame[0] == shapeStart(gfx.ShapeFrame(file, i)) {
				lv.monsterShapes[set][j] = lv.monsterShapes[set][i]
				shared = true
				break
			}
		}
		if !shared {
			lv.monsterShapes[set][j] = gfx.CopyShape(file, j)
		}
		shape := lv.monsterShapes[set][j]
		n := 0
		if len(shape) > colourCountOffset {
			n = int(shape[colourCountOffset])
		}
		lv.monsterPalettes[set][j] = make([]byte, n*monsterPalettes)
	}
	for j := range monsterAnimations {
		for i := range monsterShapeCount {
			id := j*monsterShapeCount + i + monsterOverlayBase
			lv.monsterFrames[set][j][i] = monsterFrame{
				shape: gfx.CopyShape(file, id),
				x:     gfx.ShapeWordBefore(file, id, 2),
				y:     gfx.ShapeWordBefore(file, id, 1),
			}
		}
	}
	lv.monsterAnimType[set] = colour
	g.buildMonsterPalettes(set, gfx.CopyShape(file, monsterPaletteKey))
	return nil
}

// colourCountOffset is where a shape keeps its colour count.
const colourCountOffset = 10

func shapeStart(b []byte) *byte {
	if len(b) == 0 {
		return nil
	}
	return &b[0]
}

// buildMonsterPalettes makes the eight colour variants of a monster's shapes from a key
// shape: its first column lists colours, the next eight columns their replacements
// (C: ovr_2d2c_0052).
func (g *Game) buildMonsterPalettes(set int, key []byte) {
	s := g.Screen
	s.ClearPage(keyPage)
	s.DrawShape(keyPage, key, 0, 0, 0, 0, gfx.ShapeOptions{})
	page := s.Page(keyPage)
	var column [paletteKeyRows]byte
	for j := range column {
		column[j] = page[j*gfx.Width]
	}
	lv := &g.lvl
	for i := range monsterShapeCount {
		colours := gfx.ShapeColours(lv.monsterShapes[set][i])
		rowOf := make([]int, len(colours))
		for j, c := range colours {
			rowOf[j] = -1
			for r, k := range column {
				if k == c {
					rowOf[j] = r
					break
				}
			}
		}
		out := lv.monsterPalettes[set][i]
		for k := range monsterPalettes {
			for j, c := range colours {
				v := c
				if rowOf[j] != -1 {
					if p := page[rowOf[j]*gfx.Width+k+1]; p != 0 {
						v = p
					}
				}
				if k*len(colours)+j < len(out) {
					out[k*len(colours)+j] = v
				}
			}
		}
	}
}

// keyPage is where the palette key is drawn (C: page 2, after clearing page 3, which is
// the same buffer).
const keyPage = 2

// freeMonsterShapes drops a monster set (C: ovr_2d2c_00f2).
func (g *Game) freeMonsterShapes(set int) {
	lv := &g.lvl
	lv.monsterShapes[set] = [monsterShapeCount][]byte{}
	lv.monsterPalettes[set] = [monsterShapeCount][]byte{}
}

// loadDoorShapes reads the two door shapes and sets the door frames' flags: all
// passable sides, the fourth step not passable and the fifth closed; the two arguments
// make the first or second ten frames impassable (C: ovr_2d2c_00a2).
func (g *Game) loadDoorShapes(name string, first, second int, firstClosed, secondClosed bool) error {
	file, err := g.readCPS(name)
	if err != nil {
		return err
	}
	lv := &g.lvl
	lv.doorShapes[0] = gfx.CopyShape(file, first)
	lv.doorShapes[1] = gfx.CopyShape(file, second)
	for i := range doorTypes {
		f := &lv.wallFlags[i+firstDoorType]
		*f |= wallDoorMask
		if i%doorSteps == doorSteps-1 {
			*f &^= wallDoorMask
		}
		if i%doorSteps == doorSteps-2 {
			*f &^= wallPassable
		}
	}
	if firstClosed {
		for i := range doorTypesShown {
			lv.wallFlags[i+firstDoorType] &^= wallPassable
		}
	}
	if secondClosed {
		for i := range doorTypesShown {
			lv.wallFlags[i+firstDoorType+doorTypesShown] &^= wallPassable
		}
	}
	return nil
}

// freeLevel drops everything a level loaded (C: ovr_2d2c_005c).
func (g *Game) freeLevel() {
	lv := &g.lvl
	lv.doorShapes = [2][]byte{}
	for set := range monsterShapeSets {
		g.freeMonsterShapes(set)
		lv.monsterFrames[set] = [monsterAnimations][monsterShapeCount]monsterFrame{}
	}
	lv.decorationShapes = nil
	g.levelScript = nil
	lv.vmpData, lv.vcnData, lv.vcnShift = nil, nil, nil
	lv.fadeTables = [monsterPalettes][]byte{}
	g.levelStrings = nil
	g.stopSounds()
	lv.wallShape[0] = 0
	lv.wallShape[1] = 0
}

// clearLevelObjects takes the items off the map, remembering where they lie, and with
// clearFlying also the flying objects (C: ovr_2d2c_0061).
func (g *Game) clearLevelObjects(clearFlying bool) {
	lv := &g.lvl
	for i := range lv.blocks {
		b := &lv.blocks[i]
		b.Direction = noDirection
		p := &b.Objects
		for *p&monsterFlag != 0 {
			p = g.objectNext(*p)
		}
		if *p != 0 {
			g.items[*p].Level = g.level
			g.items[*p].Block = i
			*p = 0
		}
	}
	if clearFlying {
		g.flying = [flyingObjects]flyingObject{}
	}
}

// placeLevelItems puts the items of this level back on the map (C: ovr_2d2c_0066).
func (g *Game) placeLevelItems() {
	for i := range g.items {
		it := &g.items[i]
		if it.Level == g.level {
			b := it.Block
			g.addObject(&g.lvl.blocks[b].Objects, i)
			g.lvl.blocks[b].Direction = noDirection
		}
		it.DrawNext = 0
	}
}

// loadLevelStrings reads the level's strings in the game's language (C: ovr_2d2c_002a).
func (g *Game) loadLevelStrings(name string) error {
	g.levelStrings = nil
	if name == "" {
		return nil
	}
	raw, err := g.Files.Read(languageFile(name, g.language))
	if err != nil {
		return err
	}
	g.levelStrings = raw
	return nil
}
