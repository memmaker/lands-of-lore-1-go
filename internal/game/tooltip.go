package game

import (
	"fmt"
	"image"
	"strings"

	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
)

// Tooltips are an addition of the port: with the pointer resting on a spell in the
// list, on a magic button or on a level of the spell menu, a box beside it names the
// spell and what its levels cost; on an item in the inventory bar or a character's
// slots it names the item, says what it is for and gives a weapon's or armour's
// numbers (itemtips.go). The box is an overlay of the frame (gfx.SetOverlay),
// so the game's pages stay as the original drew them.

const (
	tooltipDelay            = 20 // ticks the pointer rests before the box shows
	tooltipPadding          = 3
	tooltipGap              = 10 // between the pointer and the box
	buttonInventoryFirst    = 50
	buttonInventoryLast     = 58
	buttonCharInventoryLast = 43
	buttonSpellListFirst    = 71
	buttonSpellListLast     = 80
)

// The tooltip colours, the nearest of the palette to these.
var (
	tooltipBackRGB   = []byte{6, 4, 2}
	tooltipBorderRGB = []byte{44, 34, 18}
	tooltipTextRGB   = []byte{63, 60, 50}
)

// tooltipState is the box shown or waited for.
type tooltipState struct {
	text      string // the shown or waited-for text, "" for none
	shown     bool
	restSince int // the tick the pointer came to the text's button
}

// updateTooltip shows, moves or takes away the tooltip for the button under the pointer.
func (g *Game) updateTooltip(now int) {
	t := &g.tooltip
	text := ""
	if g.updateFlags&updateGameMenu == 0 && g.heldItem == 0 {
		text = g.tooltipText()
	}
	if text != t.text {
		t.text, t.shown, t.restSince = text, false, now
		g.Screen.ClearOverlay()
	}
	if text == "" || t.shown || now-t.restSince < tooltipDelay {
		return
	}
	t.shown = true
	g.drawTooltip(text)
}

// tooltipText is the text for the button under the pointer, "" if it has none.
func (g *Game) tooltipText() string {
	mx, my := g.Screen.Mouse()
	for b := g.buttons; b != nil; b = b.Next {
		if !g.GUI.Contains(b, mx, my) {
			continue
		}
		switch {
		case b.Index >= buttonInventoryFirst && b.Index <= buttonInventoryLast:
			return g.itemTooltip(g.inventory[(g.inventoryStart+b.Arg)%inventorySlots])
		case b.Index >= buttonCharInventorySlot && b.Index <= buttonCharInventoryLast:
			return g.itemTooltip(g.party[g.selectedCharacter].Items[b.Arg])
		case g.screenFrozen:
		case b.Index >= buttonSpellListFirst && b.Index <= buttonSpellListLast:
			return g.spellTooltip(g.availableSpells[b.Arg])
		case b.Index >= buttonMagic && b.Index < buttonMagicSubmenu:
			return g.spellTooltip(g.availableSpells[g.selectedSpell])
		case b.Index == buttonMagicSubmenu:
			return g.spellLevelTooltip(b, my)
		}
	}
	return ""
}

// spellTooltip names spell sp and the points its levels cost.
func (g *Game) spellTooltip(sp int) string {
	if sp < 0 || sp >= len(spells) || spells[sp].Name == 0 {
		return ""
	}
	costs := make([]string, spellLevels)
	for i, c := range spells[sp].MPCost {
		costs[i] = fmt.Sprint(c)
	}
	return g.text(spells[sp].Name) + "\rMP " + strings.Join(costs, " / ")
}

// spellLevelTooltip is the selected spell's level the pointer is on in the spell menu.
func (g *Game) spellLevelTooltip(b *gui.Button, my int) string {
	sp := g.availableSpells[g.selectedSpell]
	level := (my - spellLevelRow) >> spellLevelRowShift
	if sp < 0 || sp >= len(spells) || level < 0 || level >= g.castableLevels(&g.party[b.Arg], sp) {
		return ""
	}
	return fmt.Sprintf("%s %d\r%d MP", g.text(spells[sp].Name), level+1, spells[sp].MPCost[level])
}

// drawTooltip puts the box with text beside the pointer, inside the screen.
func (g *Game) drawTooltip(text string) {
	s := g.Screen
	font := s.SetFont(g.SmallFont)
	defer s.SetFont(font)
	lines := strings.Split(text, "\r")
	w := 0
	for _, l := range lines {
		w = max(w, s.TextWidth(l))
	}
	lineH := g.SmallFont.Height() + 1
	w += 2 * tooltipPadding
	h := len(lines)*lineH + 2*tooltipPadding - 1
	mx, my := s.Mouse()
	x, y := mx+tooltipGap, my+tooltipGap
	if x+w > gfx.Width {
		x = mx - tooltipGap - w
	}
	if y+h > gfx.Height {
		y = my - tooltipGap - h
	}
	x, y = max(x, 0), max(y, 0)
	pal := g.Screen.Palette[:]
	back := gfx.NearestColour(tooltipBackRGB, pal, fadeTableSize, false)
	border := gfx.NearestColour(tooltipBorderRGB, pal, fadeTableSize, false)
	fore := gfx.NearestColour(tooltipTextRGB, pal, fadeTableSize, false)
	s.SetOverlay(image.Rect(x, y, x+w, y+h), func() {
		s.FillRect(x, y, x+w-1, y+h-1, border)
		s.FillRect(x+1, y+1, x+w-2, y+h-2, back)
		for i, l := range lines {
			s.DrawText(l, x+tooltipPadding, y+tooltipPadding+i*lineH, fore, 0)
		}
	})
}
