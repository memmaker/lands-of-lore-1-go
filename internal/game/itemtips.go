package game

import (
	"fmt"
	"strings"
)

// The item tooltips' texts (an addition of the port, see tooltip.go). The item types
// say how an item is used (its ITEM.INF function) and where it is worn; a melee
// weapon's might is its damage and the low bits of its skill the kind of blow, a
// launcher's might the ammunition it takes and its skill the damage; a herb heals its
// might in dice of its skill's sides.

// ITEM.INF functions the texts tell apart.
const (
	itemScriptMelee    = 0
	itemScriptWorn     = 1
	itemScriptThrown   = 2
	itemScriptHeal     = 4
	itemScriptOil      = 8
	itemScriptPurse    = 9
	itemScriptLantern  = 10
	itemScriptLauncher = 11
	itemScriptRead     = 12
	itemScriptScroll   = 14
	itemScriptWand     = 20
	itemScriptNoUse    = 255
	itemAmmoArrow      = 2
	itemAmmoGoodArrow  = 3
	itemAmmoQuarrel    = 4
	itemAmmoStar       = 5
	itemBlowMask       = 3
	itemFlagKey        = 1 << 8
	slotHand           = 1<<0 | 1<<2 // either hand
	slotShieldHand     = 1<<1 | 1<<3
	slotHead           = 1 << 4
	slotBody           = 1 << 5
	slotNeck           = 1 << 6
	slotWrists         = 1 << 7
	slotFeet           = 1 << 8
	slotFinger         = 1<<9 | 1<<10
)

var blowKinds = [...]string{"Edged", "Piercing", "Blunt", "Magic"}

// itemTooltip is the text for item, "" for no item.
func (g *Game) itemTooltip(item int) string {
	if item <= 0 || item >= len(g.items) || g.items[item].Flags&itemFree != 0 {
		return ""
	}
	t := g.items[item].Type
	if t <= 0 || t >= len(g.itemTypes) {
		return ""
	}
	it := &g.itemTypes[t]
	lines := []string{g.text(it.Name)}
	if what := itemPurpose(it); what != "" {
		lines = append(lines, what)
	}
	switch it.Script {
	case itemScriptMelee:
		lines = append(lines, fmt.Sprintf("Damage %d", it.Might))
	case itemScriptLauncher:
		lines = append(lines, fmt.Sprintf("Damage %d", it.Skill))
	case itemScriptThrown:
		if it.Slot&slotHand != 0 {
			lines = append(lines, fmt.Sprintf("Damage %d", it.Might))
		}
	}
	if it.Protection != 0 {
		label := "Protection"
		if it.Script == itemScriptMelee {
			label = "Defence"
		}
		lines = append(lines, fmt.Sprintf("%s %d", label, it.Protection))
	}
	return strings.Join(lines, "\r")
}

// itemPurpose is a few words on what an item type is for, "" when there is nothing to say.
func itemPurpose(it *ItemType) string {
	switch it.Script {
	case itemScriptMelee:
		if it.Slot&slotHand == 0 {
			return ""
		}
		return blowKinds[int(it.Skill)&itemBlowMask] + " weapon"
	case itemScriptLauncher:
		switch it.Might {
		case itemAmmoArrow, itemAmmoGoodArrow:
			return "Bow: shoots arrows"
		case itemAmmoQuarrel:
			return "Crossbow: shoots quarrels"
		case itemAmmoStar:
			return "Throwing star"
		}
		return "Ranged weapon"
	case itemScriptThrown:
		if it.Slot&slotHand != 0 {
			return "Thrown weapon"
		}
		return "Ammunition"
	case itemScriptWorn:
		switch {
		case it.Slot&slotHead != 0:
			return "Worn on the head"
		case it.Slot&slotBody != 0:
			return "Body armour"
		case it.Slot&slotShieldHand != 0:
			return "Shield"
		case it.Slot&slotFeet != 0:
			return "Footwear"
		case it.Slot&slotWrists != 0:
			return "Worn on the wrists"
		case it.Slot&slotFinger != 0:
			return "Ring"
		}
	case itemScriptHeal: // heals might dice of skill sides (Aloe 1d6, Ginseng 2d6, Salve 4d16)
		return fmt.Sprintf("Heals %d-%d HP", it.Might, int(it.Might)*int(it.Skill))
	case itemScriptOil:
		return "Fills the lantern"
	case itemScriptPurse:
		return "Use to take the coins"
	case itemScriptLantern:
		return "Gives light"
	case itemScriptRead:
		return "Use to read"
	case itemScriptScroll:
		return "Casts its spell once"
	case itemScriptWand:
		return "Casts a spell"
	case itemScriptNoUse:
		switch {
		case it.Flags&itemFlagKey != 0:
			return "Opens a lock"
		case it.Slot&slotFinger != 0:
			return "Ring"
		}
	}
	return ""
}
