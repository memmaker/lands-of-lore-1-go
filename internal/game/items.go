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

// Item allocation (C: fn_07bc_012d and its helpers).
const (
	lastLevel       = 29
	itemNowhere     = 255 // the level of an item that is in no level
	itemFlagsMask   = 1<<13 - 1
	itemTypeNoClaim = 1 << 2 // items of such types are never taken back from a level
)

// reclaimable says whether an item lies on a floor and may be taken back for a new one
// (C: fn_07bc_000e).
func (g *Game) reclaimable(item int) bool {
	it := &g.items[item]
	return it.Flags&itemOnFloor != 0 && g.itemTypes[it.Type].Flags&itemTypeNoClaim == 0
}

// anyReclaimable says whether an item list holds a reclaimable item (C: fn_07bc_0050).
func (g *Game) anyReclaimable(item int) bool {
	for ; item != 0; item = g.items[item].Next {
		if g.reclaimable(item) {
			return true
		}
	}
	return false
}

// reclaimFrom takes the first reclaimable item out of a list and frees it
// (C: fn_07bc_0080).
func (g *Game) reclaimFrom(item int) int {
	if g.reclaimable(item) {
		if next := g.items[item].Next; next != 0 {
			g.items[next].Level = g.items[item].Level
		}
		g.freeItem(item)
		return item
	}
	for next := g.items[item].Next; next != 0; item, next = next, g.items[next].Next {
		if g.reclaimable(next) {
			g.items[item].Next = g.items[next].Next
			g.freeItem(next)
			return next
		}
	}
	return 0
}

// newItem makes an item of a type: a free entry, else one taken back from the floor of
// the level farthest from the current one (C: fn_07bc_012d).
func (g *Game) newItem(itemType, flags, moreFlags int) int {
	slot, farthest := 0, 0
	i := 1
	for ; i < maxItems; i++ {
		it := &g.items[i]
		if it.Flags&itemFree != 0 {
			farthest = 0
			break
		}
		if it.Level < 1 || it.Level > lastLevel || it.Level == g.level {
			continue
		}
		if d := abs(g.level - it.Level); d > farthest && g.anyReclaimable(i) {
			farthest, slot = d, i
		}
	}
	if farthest != 0 {
		i = g.reclaimFrom(slot)
	}
	if i >= maxItems { // fix: the original cleared the entry past the table's end
		return 0
	}
	g.items[i] = Item{Type: itemType, Flags: flags&itemFlagsMask | moreFlags, Level: itemNowhere}
	return i
}

// freeItem clears an item and marks it free (C: fn_07bc_044c).
func (g *Game) freeItem(item int) {
	g.items[item] = Item{Flags: itemFree}
}

// Item script events (the masks in front of ITEM.INF's functions).
const (
	itemEventEquip = 1 << 7
)

// noItemScript marks item types without a script; the hand has function 3.
const (
	noItemScript   = 255
	handItemScript = 3
)

// runItemScript runs an item's ITEM.INF function for an event if the function handles
// it (C: fn_07bc_047f).
func (g *Game) runItemScript(character, item, event, a, b int) error {
	function := handItemScript
	if item != 0 {
		function = int(g.itemTypes[g.items[item].Type].Script)
	}
	if function == noItemScript {
		return nil
	}
	var st emc.State
	st.Init(g.itemScript)
	st.Start(function)
	st.Regs[0], st.Regs[1], st.Regs[2], st.Regs[3], st.Regs[4] = event, character, item, a, b
	if event&st.EntryMask() == 0 {
		return nil
	}
	st.Run()
	return st.Err
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Item flags.
const (
	itemSubSnap   = 1<<16 - 1 - (1<<6 - 1) // a floor item's position snaps to quarter blocks (C: 0xffc0)
	itemSubCentre = 1 << 6
	eventItemDrop = 1 << 7
)

// dropItem puts item it at (x, y) on the level, at height sub; on the floor it runs the
// block's script (C: fn_07bc_0272).
func (g *Game) dropItem(it, x, y, sub int, floor bool) {
	if sub == 0 {
		x = x&itemSubSnap | itemSubCentre
		y = y&itemSubSnap | itemSubCentre
	}
	b := positionBlock(x, y)
	item := &g.items[it]
	item.X, item.Y, item.Block, item.Sub = x, y, b, sub
	if floor {
		item.Flags |= itemOnFloor
	} else {
		item.Flags &^= itemOnFloor
	}
	g.appendObject(&g.lvl.blocks[b].Objects, it)
	g.insertDrawObject(g.direction, it, &g.lvl.blocks[b], false)
	if floor {
		g.runBlockScriptWith(b, eventItemDrop, -1, it, 0, 0)
	}
	g.noteBlockChanged(b)
}
