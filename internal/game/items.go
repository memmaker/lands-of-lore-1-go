package game

import "github.com/memmaker/lands-of-lore-1-go/internal/emc"

// maxItems is the size of the item table (C: f_2803_1cfd[400]).
const maxItems = 400

// Item flags.
const (
	itemOnFloor = 1 << 14
	itemFree    = 1 << 15
)

// Item is one item in the world or carried (C: struct Item).
type Item struct {
	Next     int // next item in the same list
	DrawNext int // next item to draw in the same block
	Sub      int // position inside the block
	Block    int
	X, Y     int
	Level    int
	Type     int // index into the item types
	Flags    int
}

// ItemType describes a kind of item (C: struct ItemType, filled by ONETIME.INF; the names
// follow ScummVM's ItemProperty, whose fields are set in the same order).
type ItemType struct {
	Name       int  // string id
	Shape      byte // index into the item shapes and icons
	Flags      int
	Slot       int  // where it is worn or held (ScummVM: type)
	Script     byte // the ITEM.INF function run when it is used
	Might      byte
	Skill      byte
	Protection byte
	Unknown    int // ScummVM: unkB
}

// freeAllItems marks every item free (C: ovr_2d50_0025).
func (g *Game) freeAllItems() {
	for i := range g.items {
		g.items[i].Flags |= itemFree
	}
}

// opAllocItemTypes makes room for arg 0 item types (C: ovr_2d50_003e).
func (g *Game) opAllocItemTypes(s *emc.State) int {
	g.itemTypes = make([]ItemType, max(s.Arg(0), 0))
	return 1
}

// opDefineItemType sets item type arg 0 (C: ovr_2d50_0043).
func (g *Game) opDefineItemType(s *emc.State) int {
	n := s.Arg(0)
	if n < 0 || n >= len(g.itemTypes) { // fix: the original wrote outside the table
		return 1
	}
	g.itemTypes[n] = ItemType{
		Name:       s.Arg(1),
		Shape:      byte(s.Arg(2)),
		Slot:       s.Arg(3),
		Script:     byte(s.Arg(4)),
		Might:      byte(s.Arg(5)),
		Skill:      byte(s.Arg(6)),
		Protection: byte(s.Arg(7)),
		Flags:      int(uint16(s.Arg(8))),
		Unknown:    s.Arg(9),
	}
	return 1
}
