package game

import "github.com/memmaker/lands-of-lore-1-go/internal/emc"

// Monster is a monster on the level (C: struct Monster, 46 bytes of f_2803_15fa;
// ScummVM: MonsterInPlay).
type Monster struct {
	Next         int // next object on the same block
	DrawNext     int // next object to draw in the same block (C: c02)
	Sub          int // height in the block, like an item's sub position (C: c04)
	Block        int
	X, Y         int
	Shift        int // animation tick (C: c0b)
	DestX, DestY int // where it is going (C: tx, ty)
	TargetDir    int // (C: tdir)
	DX, DY       int // (C: dx, dy)
	AnimFrame    int // (C: anim)
	Mode         int // 13 and up: dying or dead
	Counter      int // the mode's counter: animation step, wait or turn (C: c15)
	ID           int
	Direction    int // 0-7
	Facing       int // 0-3
	Flags        int
	Damage       int // (C: hitDmg)
	HitPoints    int
	Thinking     int // countdown to the next decision (C: c1f)
	TypeIndex    int // (C: c20)
	Type         *monsterType
	Casts        int    // spells left to cast, -1 for no limit (C: c25)
	NextSpell    int    // (C: c26)
	CastDelay    int    // ticks before the next spell (C: c27)
	Items        int    // first carried item
	Equipment    [4]int // (C: c2a)
}

const (
	maxMonsters     = 30
	monsterFlag     = 1 << 15 // object numbers with this bit are monsters
	monsterIndex    = monsterFlag - 1
	monsterModeIdle = 16 // (C: ovr_2d2c_002f's reset mode 0x10)
)

// resetMonsters empties the monster slots (C: ovr_2d2c_002f).
func (g *Game) resetMonsters() {
	for i := range g.monsters {
		g.monsters[i] = Monster{Mode: monsterModeIdle}
	}
}

// objectNext is the link to the object after o on its block (C: fn_080e_000b's next).
func (g *Game) objectNext(o int) *int {
	if o&monsterFlag != 0 {
		return &g.monsters[o&monsterIndex].Next
	}
	return &g.items[o].Next
}

// addObject puts item or monster id first after a list's monsters (C: fn_080e_00b4).
func (g *Game) addObject(list *int, id int) {
	for *list&monsterFlag != 0 {
		list = g.objectNext(*list)
	}
	if id&monsterFlag == 0 {
		g.items[id].Level = noLevel
	} else {
		g.monsters[id&monsterIndex].Shift = noLevel // the item's level byte is the monster's tick
	}
	old := *list
	if *list == id {
		return
	}
	*list = id
	list = g.objectNext(id)
	for *list != 0 {
		list = g.objectNext(*list)
	}
	*list = old
}

// noLevel marks an item that is in a list rather than lying on a level (C: 0xff).
const noLevel = 255

// flyingObject is a thrown item or missile (C: struct Fly, 17 bytes of g_4dcc).
type flyingObject struct {
	Active  bool
	Type    int
	Owner   int
	Item    int
	X, Y    int
	Power   int
	Dir     int
	Range   int
	Unknown int // (C: e)
	Flags   int
	Field16 int // (C: f10)
}

const flyingObjects = 8

// setMonsterDirection turns m to dir (0-7); it faces the new way unless the turn is a
// diagonal just clockwise of where it faces (C: fn_0d6c_0073).
func (g *Game) setMonsterDirection(m *Monster, dir int) {
	m.Direction = dir
	if dir&1 == 0 || m.Direction-m.Facing*2 >= 2 {
		m.Facing = m.Direction >> 1
	}
	g.noteBlockChanged(m.Block)
}

// opMoveMonster sends a walking monster to a place in a block and turns it there
// (C: fn_0fbe_162c): monster, block, x and y in the block, direction 0-3.
func (g *Game) opMoveMonster(s *emc.State) int {
	m := &g.monsters[s.Arg(0)]
	if m.Mode == monsterModeAll || m.Mode == monsterModeWalk {
		m.DestX, m.DestY = blockPosition(s.Arg(1), s.Arg(2), s.Arg(3))
		m.TargetDir = s.Arg(4) << 1
		if m.X != m.DestX || m.Y != m.DestY {
			g.setMonsterDirection(m, directionTo(m.X, m.Y, m.DestX, m.DestY))
		}
	}
	return 1
}

// appendObject puts item id first after a list's monsters, marking it as in a list
// (C: fn_080e_005f).
func (g *Game) appendObject(list *int, id int) {
	for *list&monsterFlag != 0 {
		list = g.objectNext(*list)
	}
	g.items[id].Next = *list
	g.items[id].Level = noLevel
	*list = id
}
