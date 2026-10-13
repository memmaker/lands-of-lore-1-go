package game

// Item types whose icon depends on the item: the icon is the type's plus the item's
// flags' low bits, less one (C: fn_0845_0c08).
const (
	itemTypeIconVaries = 1 << 9
	itemIconOffsetMask = 1<<13 - 1
)

// heldCursorHotSpot is where the cursor points while it carries an item.
const heldCursorHotSpot = 10

// itemIcon returns an item's icon, the hand for 0 (C: fn_0845_0c08).
func (g *Game) itemIcon(item int) []byte {
	icon := 0
	if item != 0 {
		it := &g.items[item]
		t := g.itemTypes[it.Type]
		icon = int(t.Shape)
		if t.Flags&itemTypeIconVaries != 0 {
			icon += it.Flags&itemIconOffsetMask - 1
		}
	}
	return g.cursors[icon]
}

// heldItemShape is the held item's type's shape, -1 before the item types are loaded
// (fix: the original read through a null pointer then).
func (g *Game) heldItemShape() int {
	t := g.items[g.heldItem].Type
	if t < 0 || t >= len(g.itemTypes) {
		return -1
	}
	return int(g.itemTypes[t].Shape)
}

// setBusyCursor shows an icon as the cursor while the game works (C: fn_0845_09f1).
func (g *Game) setBusyCursor(icon int) {
	g.setFlag(flagBusy, true)
	if g.heldItemShape() != icon {
		g.Screen.SetMouseShape(0, 0, g.cursors[icon])
		g.busyIcon = icon
	}
}

// restoreCursor puts the hand or the held item back (C: fn_0845_0a5b).
func (g *Game) restoreCursor() {
	g.setFlag(flagBusy, false)
	if g.heldItemShape() != g.busyIcon {
		hot := 0
		if g.heldItem != 0 {
			hot = heldCursorHotSpot
		}
		g.Screen.SetMouseShape(hot, hot, g.itemIcon(g.heldItem))
	}
}
