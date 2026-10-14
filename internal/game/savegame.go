package game

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
)

// Saved games keep the original's layout byte for byte, so DOS saves load here and
// the other way round (C: o2d3c). A save is a 60-byte description, "CD04", the version
// string, the party, the block of globals from g_7fd0, the items, then for each level
// the party has been to its state as the level temp file held it.
const (
	saveDescriptionSize = 60
	saveMagic           = "CD04"
	saveMagicSize       = 5
	saveVersion         = "V CD1.02 D" // (C: g_0f6f, also printed on the menu as g_11ce)
	saveVersionSize     = 11
	characterSize       = 130
	partyRecordSize     = partySize * characterSize
	globalsSize         = 563 // g_7fd0 up to the blocks at g_8203
	itemSize            = 16
	itemTableSize       = maxItems * itemSize
	monsterSize         = 46
	monsterTableSize    = maxMonsters * monsterSize
	flyingSize          = 17
	flyingTableSize     = flyingObjects * flyingSize
	savedLevels         = 30
	levelRecordSize     = 2496 // what of a level temp file a save keeps
	levelLengthSize     = 4
	levelStateBytes     = blockCount*directionCount + blockCount // walls, then block flags
	savedFile           = "_SAVE%03d.DAT"
	levelTempFile       = "_LEVEL%d.TMP"
	levelMazeFile       = "level%d.cmz"
	loadFailed          = 0
	loadDone            = 1
	loadBadSave         = 2 // the load result for a file without the magic
	saveSize            = saveDescriptionSize + saveMagicSize + saveVersionSize + partyRecordSize +
		globalsSize + itemTableSize + savedLevels*(levelLengthSize+levelRecordSize)
	loadFadeTicks = 10
)

// applySoundSettings turns off what a loaded game has on but no driver plays, and
// without speech the text is on (C: ovr_2d3c_002a).
func (g *Game) applySoundSettings() {
	if g.Config.Music == 0 {
		g.musicVolume = 0
	}
	if g.musicVolume != 0 {
		g.musicOn = true
	} else {
		g.stopSong()
		g.musicOn = false
	}
	if g.Config.Effects == 0 {
		g.effectsVolume = 0
	}
	g.effectsOn = g.effectsVolume != 0
	g.digitalEffectsOn = g.effectsOn
	if !g.effectsOn {
		g.stopSoundEffects()
	}
	if g.Config.Speech == 0 {
		g.speechVolume = 0
		g.textEnabled = true
		g.digitalEffectsOn = false
	}
	g.digitalSpeechOn = g.speechVolume != 0
}

// record writes the little-endian fields of the original's structures.
type record struct{ b []byte }

func (r *record) u8(v int)  { r.b = append(r.b, byte(v)) }
func (r *record) u16(v int) { r.b = binary.LittleEndian.AppendUint16(r.b, uint16(v)) }
func (r *record) u32(v int) { r.b = binary.LittleEndian.AppendUint32(r.b, uint32(v)) }
func (r *record) zero(n int) {
	r.b = append(r.b, make([]byte, n)...)
}
func (r *record) str(s string, n int) {
	b := make([]byte, n)
	copy(b[:n-1], s)
	r.b = append(r.b, b...)
}

// reader reads them back; past the end it reads zeros.
type reader struct {
	b  []byte
	at int
}

func (r *reader) byte() byte {
	if r.at >= len(r.b) {
		r.at++
		return 0
	}
	r.at++
	return r.b[r.at-1]
}
func (r *reader) u8() int    { return int(r.byte()) }
func (r *reader) s8() int    { return int(int8(r.byte())) }
func (r *reader) u16() int   { lo := r.u8(); return lo | r.u8()<<8 }
func (r *reader) s16() int   { return int(int16(r.u16())) }
func (r *reader) s32() int   { lo := r.u16(); return int(int32(uint32(lo) | uint32(r.u16())<<16)) }
func (r *reader) skip(n int) { r.at += n }
func (r *reader) str(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = r.byte()
	}
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// writeCharacter writes a struct Char; the modifier set's far pointer is left 0, for
// loading takes it from the race (C: ovr_2d3c_002a).
func writeCharacter(r *record, c *Character) {
	r.u16(c.Flags)
	r.str(c.Name, nameBytes)
	r.u8(c.RaceClassSex)
	r.u16(c.ID)
	r.u8(c.Face)
	r.u8(c.FaceDefault)
	r.u8(c.Scream)
	r.u32(0)
	for _, v := range c.ItemsMight {
		r.u16(v)
	}
	for _, v := range c.ProtectionAgainstItems {
		r.u16(v)
	}
	for _, v := range []int{c.ItemProtection, c.HP, c.MaxHP, c.MP, c.MaxMP} {
		r.u16(v)
	}
	r.u8(c.Field41)
	for _, v := range []int{c.DamageSuffered, c.WeaponHit, c.TotalMightModifier,
		c.TotalProtectionModifier, c.Might, c.Protection, c.AnimCountdown} {
		r.u16(v)
	}
	for _, v := range c.Items {
		r.u16(v)
	}
	for _, v := range c.SkillLevels {
		r.u8(v)
	}
	for _, v := range c.SkillModifiers {
		r.u8(v)
	}
	for _, v := range c.Experience {
		r.u32(v)
	}
	for _, v := range c.StatusTimers {
		r.u8(v)
	}
	for _, v := range c.StatusEffects {
		r.u8(v)
	}
}

const nameBytes = 11

func readCharacter(r *reader) Character {
	var c Character
	c.Flags = r.u16()
	c.Name = r.str(nameBytes)
	c.RaceClassSex = r.s8()
	c.ID = r.s16()
	c.Face = r.s8()
	c.FaceDefault = r.s8()
	c.Scream = r.s8()
	r.skip(4)
	c.Modifiers = c.RaceClassSex
	for i := range c.ItemsMight {
		c.ItemsMight[i] = r.s16()
	}
	for i := range c.ProtectionAgainstItems {
		c.ProtectionAgainstItems[i] = r.s16()
	}
	for _, p := range []*int{&c.ItemProtection, &c.HP, &c.MaxHP, &c.MP, &c.MaxMP} {
		*p = r.s16()
	}
	c.Field41 = r.s8()
	for _, p := range []*int{&c.DamageSuffered, &c.WeaponHit, &c.TotalMightModifier,
		&c.TotalProtectionModifier, &c.Might, &c.Protection, &c.AnimCountdown} {
		*p = r.s16()
	}
	for i := range c.Items {
		c.Items[i] = r.s16()
	}
	for i := range c.SkillLevels {
		c.SkillLevels[i] = r.s8()
	}
	for i := range c.SkillModifiers {
		c.SkillModifiers[i] = r.s8()
	}
	for i := range c.Experience {
		c.Experience[i] = r.s32()
	}
	for i := range c.StatusTimers {
		c.StatusTimers[i] = r.s8()
	}
	for i := range c.StatusEffects {
		c.StatusEffects[i] = r.s8()
	}
	return c
}

func writeItem(r *record, it *Item) {
	r.u16(it.Next)
	r.u16(it.DrawNext)
	r.u8(it.Sub)
	r.u16(it.Block)
	r.u16(it.X)
	r.u16(it.Y)
	r.u8(it.Level)
	r.u16(it.Type)
	r.u16(it.Flags)
}

func readItem(r *reader) Item {
	return Item{Next: r.u16(), DrawNext: r.u16(), Sub: r.u8(), Block: r.u16(), X: r.u16(),
		Y: r.u16(), Level: r.u8(), Type: r.s16(), Flags: r.u16()}
}

// writeMonster writes a struct Monster; its type's far pointer is left 0, for a
// restored level takes it from the type index (C: ovr_2d3c_0025).
func writeMonster(r *record, m *Monster) {
	r.u16(m.Next)
	r.u16(m.DrawNext)
	r.u8(m.Sub)
	r.u16(m.Block)
	r.u16(m.X)
	r.u16(m.Y)
	r.u8(m.Shift)
	r.u16(m.DestX)
	r.u16(m.DestY)
	for _, v := range []int{m.TargetDir, m.DX, m.DY, m.AnimFrame, m.Mode, m.Counter, m.ID,
		m.Direction, m.Facing} {
		r.u8(v)
	}
	r.u16(m.Flags)
	r.u16(m.Damage)
	r.u16(m.HitPoints)
	r.u8(m.Thinking)
	r.u8(m.TypeIndex)
	r.u32(0)
	r.u8(m.Casts)
	r.u8(m.NextSpell)
	r.u8(m.CastDelay)
	r.u16(m.Items)
	for _, v := range m.Equipment {
		r.u8(v)
	}
}

func readMonster(r *reader) Monster {
	var m Monster
	m.Next = r.u16()
	m.DrawNext = r.u16()
	m.Sub = r.u8()
	m.Block = r.u16()
	m.X = r.u16()
	m.Y = r.u16()
	m.Shift = r.u8()
	m.DestX = r.u16()
	m.DestY = r.u16()
	for _, p := range []*int{&m.TargetDir, &m.DX, &m.DY, &m.AnimFrame, &m.Mode, &m.Counter, &m.ID,
		&m.Direction, &m.Facing} {
		*p = r.s8()
	}
	m.Flags = r.u16()
	m.Damage = r.s16()
	m.HitPoints = r.s16()
	m.Thinking = r.s8()
	m.TypeIndex = r.s8()
	r.skip(4)
	m.Casts = r.s8()
	m.NextSpell = r.s8()
	m.CastDelay = r.s8()
	m.Items = r.u16()
	for i := range m.Equipment {
		m.Equipment[i] = r.s8()
	}
	return m
}

func writeFlying(r *record, f *flyingObject) {
	active := 0
	if f.Active {
		active = 1
	}
	r.u8(active)
	r.u8(f.Type)
	r.u16(f.Owner)
	r.u16(f.Item)
	r.u16(f.X)
	r.u16(f.Y)
	for _, v := range []int{f.Power, f.Dir, f.Range, 0, f.Unknown, f.Flags, f.Field16} {
		r.u8(v)
	}
}

func readFlying(r *reader) flyingObject {
	var f flyingObject
	f.Active = r.u8() != 0
	f.Type = r.s8()
	f.Owner = r.u16()
	f.Item = r.u16()
	f.X = r.u16()
	f.Y = r.u16()
	f.Power = r.s8()
	f.Dir = r.s8()
	f.Range = r.s8()
	r.skip(1)
	f.Unknown = r.s8()
	f.Flags = r.s8()
	f.Field16 = r.s8()
	return f
}

// globalField is one variable of the globals block: its size in bytes and whether
// it reads back signed. A nil field is a byte nothing uses.
type globalField struct {
	v      *int
	size   int
	signed bool
}

// globals lists the block from g_7fd0 in memory order.
func (g *Game) globals() []globalField {
	word := func(v *int) globalField { return globalField{v, 2, true} }
	uword := func(v *int) globalField { return globalField{v, 2, false} }
	char := func(v *int) globalField { return globalField{v, 1, true} }
	f := []globalField{
		word(&g.currentBlock), word(&g.partyX), word(&g.partyY), uword(&g.updateFlags),
		char(&g.lastMoveDirection), char(&g.selectedSpell), uword(&g.options),
		char(&g.walkTransition), char(&g.compassBroken), char(&g.drainMagic),
		word(&g.direction), word(&g.compassDirection), char(&g.selectedCharacter),
		{nil, 1, false}, char(&g.level),
	}
	for i := range g.inventory {
		f = append(f, word(&g.inventory[i]))
	}
	f = append(f, word(&g.inventoryStart), word(&g.heldItem), word(&g.lastMouseRegion))
	for i := range g.gameFlags {
		f = append(f, uword(&g.gameFlags[i]))
	}
	for i := range g.levelVars {
		f = append(f, word(&g.levelVars[i]))
	}
	f = append(f, char(&g.brightness), globalField{&g.lampEffect, 1, false}, char(&g.lampLevel),
		globalField{nil, 1, false}, word(&g.credits))
	for i := range g.scriptVars {
		f = append(f, word(&g.scriptVars[i]))
	}
	for i := range g.availableSpells {
		f = append(f, char(&g.availableSpells[i]))
	}
	return append(f, globalField{&g.visitedLevels, 4, false}, char(&g.language),
		globalField{nil, 1, false}, char(&g.speechVolume), char(&g.effectsVolume),
		char(&g.musicVolume), globalField{nil, 1, false})
}

const textEnabledAt = 558 // g_81fe within the block

func (g *Game) writeGlobals(r *record) {
	start := len(r.b)
	for _, f := range g.globals() {
		v := 0
		if f.v != nil {
			v = *f.v
		}
		switch f.size {
		case 1:
			r.u8(v)
		case 2:
			r.u16(v)
		default:
			r.u32(v)
		}
	}
	if g.textEnabled {
		r.b[start+textEnabledAt] = 1
	}
}

func (g *Game) readGlobals(r *reader) {
	start := r.at
	for _, f := range g.globals() {
		var v int
		switch {
		case f.size == 1 && f.signed:
			v = r.s8()
		case f.size == 1:
			v = r.u8()
		case f.size == 2 && f.signed:
			v = r.s16()
		case f.size == 2:
			v = r.u16()
		default:
			v = int(uint32(r.s32()))
		}
		if f.v != nil {
			*f.v = v
		}
	}
	g.textEnabled = start+textEnabledAt < len(r.b) && r.b[start+textEnabledAt] != 0
}

// saveLevelState keeps the current level's changes, its monsters and flying objects
// as the level temp file did: the walls XORed with the map's own, the block flags,
// both packed, then the difficulty they were made at (C: ovr_2d3c_0020).
func (g *Game) saveLevelState() error {
	orig, err := g.mazeWalls(g.level)
	if err != nil {
		return err
	}
	state := make([]byte, levelStateBytes)
	for i := range blockCount {
		for d := range directionCount {
			state[i*directionCount+d] = orig[i*directionCount+d] ^ byte(g.lvl.blocks[i].Walls[d])
		}
		state[blockCount*directionCount+i] = byte(g.lvl.blocks[i].Flags)
	}
	packed := data.EncodeLCW(state)
	var r record
	r.u16(len(packed))
	r.b = append(r.b, packed...)
	r.u16(g.difficulty())
	for i := range g.monsters {
		writeMonster(&r, &g.monsters[i])
	}
	for i := range g.flying {
		writeFlying(&r, &g.flying[i])
	}
	if g.levelFiles == nil {
		g.levelFiles = map[int][]byte{}
	}
	g.levelFiles[g.level] = r.b
	g.visitedLevels |= 1 << (g.level - 1)
	return nil
}

// mazeWalls reads the four walls of each block from a level's map.
func (g *Game) mazeWalls(level int) ([]byte, error) {
	file, err := g.Files.Read(fmt.Sprintf(levelMazeFile, level))
	if err != nil {
		return nil, err
	}
	img, err := data.DecodeCPS(file)
	if err != nil {
		return nil, err
	}
	maze := img.Pixels
	walls := make([]byte, blockCount*directionCount)
	if len(maze) < mazeHeaderSize {
		return walls, nil
	}
	stride := int(binary.LittleEndian.Uint16(maze[mazeBytesPerBlockAt:]))
	for i := range blockCount {
		at := mazeHeaderSize + i*stride
		if at+directionCount <= len(maze) {
			copy(walls[i*directionCount:], maze[at:at+directionCount])
		}
	}
	return walls, nil
}

// restoreLevelState sets a visited level back the way the party left it: walls, block
// flags, monsters (relinked to their blocks) and flying objects, the monsters' hit
// points rescaled if the difficulty changed since (C: ovr_2d3c_0025).
func (g *Game) restoreLevelState(level int) {
	file := g.levelFiles[level]
	r := reader{b: file}
	n := r.u16()
	packed := file[min(r.at, len(file)):min(r.at+n, len(file))]
	r.skip(n)
	difficulty := r.u16()
	for i := range g.monsters {
		g.monsters[i] = readMonster(&r)
	}
	for i := range g.flying {
		g.flying[i] = readFlying(&r)
	}
	state := make([]byte, levelStateBytes)
	data.DecodeLCW(packed, state)
	orig, err := g.mazeWalls(level)
	if err != nil {
		g.scriptErr = err
		return
	}
	g.lvl.blocks = [blockCount]Block{}
	for i := range blockCount {
		b := &g.lvl.blocks[i]
		for d := range directionCount {
			b.Walls[d] = int(orig[i*directionCount+d] ^ state[i*directionCount+d])
		}
		b.Flags = int(state[blockCount*directionCount+i])
	}
	for i := range g.monsters {
		m := &g.monsters[i]
		if m.Block != 0 {
			m.Block = 0
			m.Type = &g.lvl.monsterTypes[m.TypeIndex]
			g.moveMonster(m, m.X, m.Y)
		}
	}
	g.rescaleMonsterHitPoints(difficulty, g.difficulty())
}

// rescaleMonsterHitPoints scales the living monsters' hit points from one difficulty
// to another (C: ovr_2d56_003e).
func (g *Game) rescaleMonsterHitPoints(from, to int) {
	if from == to {
		return
	}
	f := difficultyHitPoints[to] << byteShift / difficultyHitPoints[from]
	for i := range g.monsters {
		m := &g.monsters[i]
		if m.Mode < monsterModeRemove && m.Block != 0 && m.HitPoints > 0 {
			m.HitPoints = scale(m.HitPoints, f)
			if to < from {
				m.HitPoints++
			}
			if m.HitPoints == 0 {
				m.HitPoints = 1
			}
		}
	}
}

// saveGame writes the game to a file with a description (C: ovr_2d3c_002f).
func (g *Game) saveGame(name, description string) error {
	g.finishDoors()
	if err := g.saveLevelState(); err != nil {
		return err
	}
	var r record
	r.str(description, saveDescriptionSize)
	r.str(saveMagic, saveMagicSize)
	r.str(saveVersion, saveVersionSize)
	for i := range g.party {
		writeCharacter(&r, &g.party[i])
	}
	g.writeGlobals(&r)
	g.clearLevelObjects(false)
	for i := range g.items {
		writeItem(&r, &g.items[i])
	}
	g.placeLevelItems()
	for i := range savedLevels {
		if g.visitedLevels&(1<<i) == 0 {
			r.zero(levelLengthSize + levelRecordSize)
			continue
		}
		file := g.levelFiles[i+1]
		r.u32(len(file))
		rec := make([]byte, levelRecordSize) // fix: the original wrote what its buffer held past the file
		copy(rec, file)                      // sic: a longer file is cut short
		r.b = append(r.b, rec...)
	}
	return g.Saves.Write(name, r.b)
}

// loadGame reads a saved game; it returns loadBadSave for a file that is not one,
// loadFailed for one it cannot read, else loadDone (C: ovr_2d3c_002a). Errors past the
// header stop the game like a script's.
func (g *Game) loadGame(name string) int {
	r, err := g.readSave(name)
	if err != nil && r == loadDone && g.scriptErr == nil {
		g.scriptErr = err
	}
	return r
}

func (g *Game) readSave(name string) (int, error) {
	g.fadeOutView(loadFadeTicks)
	g.finishDoors()
	g.Screen.FillRect(viewLeft, 0, viewLeft+viewWidth-1, viewHeight-1, 0)
	file, err := g.Saves.Read(name)
	if err != nil {
		return loadFailed, nil // fix: the original stopped on a file it could not open
	}
	r := reader{b: file, at: saveDescriptionSize}
	if r.str(saveMagicSize) != saveMagic {
		return loadBadSave, nil
	}
	if len(file) < saveSize {
		return loadFailed, nil // fix: the original read on past the end
	}
	r.skip(saveVersionSize)
	for i := range g.party {
		g.party[i] = readCharacter(&r)
		if g.party[i].Flags&charPresent != 0 {
			if err := g.loadFaces(i, g.party[i].ID); err != nil {
				return loadDone, err
			}
		}
	}
	g.readGlobals(&r)
	g.applySoundSettings()
	for i := range g.items {
		g.items[i] = readItem(&r)
	}
	for i := range g.lvl.blocks {
		b := &g.lvl.blocks[i]
		b.Objects, b.DrawObjects, b.Direction = 0, 0, noDirection
	}
	g.levelFiles = map[int][]byte{}
	for i := range savedLevels {
		if g.visitedLevels&(1<<i) == 0 {
			r.skip(levelLengthSize + levelRecordSize)
			continue
		}
		n := r.s32()
		rec := make([]byte, levelRecordSize)
		for k := range rec {
			rec[k] = r.byte()
		}
		g.levelFiles[i+1] = rec[:min(max(n, 0), levelRecordSize)]
	}
	g.layOutPortraits()
	g.recountCredits()
	g.setHeldItem(g.heldItem)
	if err := g.loadLevel(g.level); err != nil {
		return loadDone, err
	}
	g.drawGameScreen()
	g.updateCharacters(0)
	g.setFlag(flagLevelChanged, true)
	return loadDone, nil
}
