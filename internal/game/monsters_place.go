package game

import "github.com/memmaker/lands-of-lore-1-go/internal/emc"

// Difficulty: option bits 4-5 pick the monsters' hit point and attack factors
// (C: fn_0845_1036, f_2803_002c, f_2803_0032).
const (
	difficultyShift = 4
	difficultyMask  = 1<<2 - 1
)

var (
	difficultyHitPoints = [3]int{102, 256, 384}
	difficultyAttacks   = [3]int{256, 256, 192}
)

func (g *Game) difficulty() int { return g.options >> difficultyShift & difficultyMask }

// What may stand in the way of a monster (C: fn_0d6c_0338's flag bits).
const (
	blockedByWalls    = 1 << 0
	blockedByMonsters = 1 << 1
	blockedByParty    = 1 << 2
	blockedByAll      = blockedByWalls | blockedByMonsters | blockedByParty

	wallBlocksAll = 1<<3 - 1 // every wall flag that stops something (C: ign 7)
	blockOccupied = 1 << 4   // (C: block flags & 0x10)
)

// Results of a placement test.
const (
	placeFree    = 0
	placeWall    = 1
	placeMonster = 2
	placeParty   = 4
)

// placementBlocked tests whether something of a size could stand at (x, y): in its
// block and, if it reaches over an edge, the neighbouring blocks (C: fn_0d6c_0338).
func (g *Game) placementBlocked(x, y, size, what, wallMask int) int {
	nx, ny, dx, dy := 0, 0, 0, 0
	g.blockingSide = 0
	b := positionBlock(x, y)
	if r := g.blockedIn(b, x, y, size, what, wallMask); r != placeFree {
		return r
	}
	if g.partyIn(x, y, what) {
		return placeParty
	}
	corner := false
	sub := x & (blockUnits - 1)
	if x&blockCentre != 0 {
		if (sub+size)&^(blockUnits-1) != 0 {
			nx, dx, g.blockingSide = x+size, 1, sideEast
			if r := g.blockedIn(positionBlock(nx, y), x, y, size, what, wallMask); r != placeFree {
				return r
			}
			if g.partyIn(x+dx, y, what) {
				return placeParty
			}
			corner = true
		}
	} else if (sub-size)&^(blockUnits-1) != 0 {
		nx, dx, g.blockingSide = x-size, -1, sideWest
		if r := g.blockedIn(positionBlock(nx, y), x, y, size, what, wallMask); r != placeFree {
			return r
		}
		if g.partyIn(x+dx, y, what) {
			return placeParty
		}
		corner = true
	}
	sub = y & (blockUnits - 1)
	switch {
	case y&blockCentre != 0 && (sub+size)&^(blockUnits-1) != 0:
		ny, dy, g.blockingSide = y+size, 1, sideSouth
	case y&blockCentre == 0 && (sub-size)&^(blockUnits-1) != 0:
		ny, dy, g.blockingSide = y-size, -1, sideNorth
	default:
		corner = false
	}
	if dy != 0 {
		if r := g.blockedIn(positionBlock(x, ny), x, y, size, what, wallMask); r != placeFree {
			return r
		}
		if g.partyIn(x, y+dy, what) {
			return placeParty
		}
	}
	if corner {
		if r := g.blockedIn(positionBlock(nx, ny), x, y, size, what, wallMask); r != placeFree {
			return r
		}
		if g.partyIn(x+dx, y+dy, what) {
			return placeParty
		}
	}
	return placeFree
}

// The side a placement test last reached over (C: g_4d60).
const (
	sideNorth = 0
	sideEast  = 2
	sideSouth = 4
	sideWest  = 6
)

// blockedIn tests block b for walls and for monsters too close to (x, y)
// (C: fn_0d6c_0278).
func (g *Game) blockedIn(b, x, y, size, what, wallMask int) int {
	if b == g.currentBlock {
		what &^= blockedByWalls
	}
	if what&blockedByWalls != 0 {
		g.blockingBlock = b
		if g.wallsBlock(b, anySide, wallMask) {
			return placeWall
		}
		g.blockingBlock = 0
	}
	if what&blockedByMonsters != 0 {
		for o := g.lvl.blocks[b].Objects; o&monsterFlag != 0; o = *g.objectNext(o) {
			m := &g.monsters[o&monsterIndex]
			if m.Mode < monsterDead && size+m.Type.maxWidth > distance(x, y, m.X, m.Y) {
				return placeMonster
			}
		}
	}
	return placeFree
}

// partyIn says whether (x, y) lies in the party's block, if the party counts
// (C: fn_0d6c_024d).
func (g *Game) partyIn(x, y, what int) bool {
	return what&blockedByParty != 0 && positionBlock(x, y) == g.currentBlock
}

const anySide = -1

// wallsBlock says whether block b is occupied or one of its walls (seen from dir, or
// any with anySide) has a flag of mask (C: fn_0845_0cf7).
func (g *Game) wallsBlock(b, dir, mask int) bool {
	blk := &g.lvl.blocks[b]
	if blk.Flags&blockOccupied != 0 {
		return true
	}
	if dir != anySide {
		return mask&g.lvl.wallFlags[blk.Walls[dir^2]] != 0
	}
	for d := range directionCount {
		if mask&g.lvl.wallFlags[blk.Walls[d]] != 0 {
			return true
		}
	}
	return false
}

// distance is the city block distance (C: fn_0845_04af).
func distance(x1, y1, x2, y2 int) int { return abs(x1-x2) + abs(y1-y2) }

// opPlaceMonster puts a monster of type arg 4 at offset (arg 1, arg 2) of block arg 0,
// facing arg 3, with flags arg 5, mode arg 6 and up to four equipment numbers; it
// returns the monster's slot or -1 (C: fn_0fbe_1239).
func (g *Game) opPlaceMonster(s *emc.State) int {
	x, y := blockPosition(s.Arg(0), s.Arg(1), s.Arg(2))
	typ := s.Arg(4)
	if g.placementBlocked(x, y, g.lvl.monsterTypes[typ].maxWidth, blockedByAll, wallBlocksAll) != placeFree {
		return -1
	}
	for i := range g.monsters {
		m := &g.monsters[i]
		if m.HitPoints != 0 || m.Mode == monsterDead {
			continue
		}
		*m = Monster{ID: i, X: x, Y: y, Facing: s.Arg(3), TypeIndex: int(int8(typ))}
		m.Type = &g.lvl.monsterTypes[typ]
		m.Direction = m.Facing << 1
		m.HitPoints = scale(m.Type.hitPoints, difficultyHitPoints[g.difficulty()])
		if g.level != hauntedLevel || m.TypeIndex != hauntedType {
			m.HitPoints = scale(m.HitPoints, dice(1, hitPointSpread)+hitPointBase)
		}
		if int16(m.HitPoints) <= 0 {
			m.HitPoints = 1
		}
		m.Casts = int(int8(m.Type.field42))
		m.CastDelay = int(int8(dice(1, g.attackCount(i|monsterFlag, monsterAttackBase)) - 1))
		m.Sub = monsterStartDest
		m.Flags = int(uint16(s.Arg(5)))
		g.setMonsterMode(m, s.Arg(6))
		g.moveMonster(m, m.X, m.Y)
		m.DestX, m.DestY, m.TargetDir = m.X, m.Y, m.Direction
		for j := range m.Equipment {
			v := s.Arg(monsterArgEquipment + j)
			if v == 0 {
				break
			}
			m.Equipment[j] = int(int8(v))
		}
		g.noteBlockChanged(m.Block)
		return i
	}
	return -1
}

const (
	hauntedLevel        = 12 // its type 2 keeps the full hit points (C: g_7fe5 != 12)
	hauntedType         = 2
	hitPointSpread      = 128 // monsters get 192/256 to 320/256 of their hit points
	hitPointBase        = 192
	monsterAttackBase   = 8
	monsterStartDest    = 2 // (C: c04)
	monsterArgEquipment = 7
)

// attackCount is how many attacks base makes for character or monster who
// (C: fn_070c_012a).
func (g *Game) attackCount(who, base int) int {
	var v int
	if who&monsterFlag != 0 {
		v = ratio(base, g.monsters[who&monsterIndex].Type.fightingStats[attackFactor])
	} else {
		notPorted("fn_070c_012a (a character's attacks)")
	}
	// The original read the rogue skill of party slot who, wrapping a monster's number
	// past the end of the party.
	if c := who & monsterIndex; c < partySize { // fix: past the party it read other data
		switch skill := g.party[c].SkillLevels[skillRogue]; {
		case skill > 7:
			v -= v >> 1
		case skill > 3:
			v -= v >> 2
		}
	}
	return v
}

const (
	attackFactor = 4 // (C: atk->w8)
)

const finalLevel = 29

// moveMonster moves m to (x, y), relinking it from block to block (C: fn_0d6c_0742).
func (g *Game) moveMonster(m *Monster, x, y int) {
	had := m.Block != 0
	old := m.Block
	if had {
		g.removeObject(&g.lvl.blocks[m.Block].Objects, m.ID|monsterFlag)
		g.lvl.blocks[m.Block].Direction = noDirection
		g.noteBlockChanged(m.Block)
	}
	m.Block = positionBlock(x, y)
	if m.X != x || m.Y != y {
		m.X, m.Y = x, y
		m.AnimFrame = (m.AnimFrame + 1) & (directionCount - 1)
	}
	if m.Block == 0 {
		return
	}
	g.pushObject(&g.lvl.blocks[m.Block].Objects, m.ID|monsterFlag)
	g.lvl.blocks[m.Block].Direction = noDirection
	g.noteBlockChanged(m.Block)
	stepped := m.Type.flags&monsterStepSound != 0 && m.AnimFrame&1 != 0
	if m.Type.moveSound != 0 && had && (stepped || m.Block != old) {
		if m.Block != old {
			g.runBlockScriptWith(m.Block, eventMonsterEnters, -1, m.ID, 0, 0)
		}
		if g.updateFlags&updateMenu == 0 {
			g.playSoundAt(m.Type.moveSound, m.Block)
		}
	}
}

const (
	monsterStepSound   = 1 << 8 // (C: type flags & 0x100)
	eventMonsterEnters = 1 << 11
	updateMenu         = 1 << 0
)

// pushObject puts id first in a list (C: fn_080e_0034).
func (g *Game) pushObject(list *int, id int) {
	*g.objectNext(id) = *list
	*list = id
}

// removeObject takes id out of a list (C: fn_080e_013c).
func (g *Game) removeObject(list *int, id int) {
	for *list != 0 {
		next := g.objectNext(*list)
		if *list == id {
			*list = *next
			*next = 0
			return
		}
		list = next
	}
	// The original reported "L_R_O: Object %d didn't exist." in its debug build only.
}

// giveMonsterItem appends an item to a monster's items (C: ovr_2d2c_00b1).
func (g *Game) giveMonsterItem(m *Monster, item int) {
	p := &m.Items
	for *p != 0 {
		p = &g.items[*p].Next
	}
	*p = item
	g.items[item].Next = 0
}

// opGiveMonsterItem (C: fn_0fbe_1a27).
func (g *Game) opGiveMonsterItem(s *emc.State) int {
	if m := s.Arg(0); m != -1 {
		g.giveMonsterItem(&g.monsters[m], s.Arg(1))
		return 1
	}
	return 0
}
