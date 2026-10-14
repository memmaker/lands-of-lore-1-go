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
	itemScriptGlass    = 13
	itemScriptScroll   = 14
	itemScriptGoblet   = 15
	itemScriptRiddle   = 17
	itemScriptBezel    = 18
	itemScriptReadOnce = 19
	itemScriptWand     = 20
	itemScriptCube     = 21
	itemScriptCard     = 24
	itemScriptGauntlet = 28
	itemScriptSkull    = 29
	itemScriptMagic    = 30
	itemScriptNecklace = 31
	itemScriptVial     = 32
	itemTypeFullCube   = 263
	itemTypeJade       = 219
	itemTypeGinseng    = 218
	itemTypePoisonStar = 6
	spellHandOfFate    = 4
	spellReveal        = 12
	spellSwarm         = 9
	spellGuardian      = 16
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
	if what := g.itemPurpose(item, t, it); what != "" {
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
	if fx, ok := itemSpecials[t]; ok {
		lines = append(lines, fx)
	}
	return strings.Join(lines, "\r")
}

// itemSpecials are the extra effects ITEM.INF gives single item types: function 0 for
// the weapons, 2 and 11 for the missiles, 4 for the herbs, 31 for the necklaces.
var itemSpecials = map[int]string{
	itemTypePoisonStar: "Poisons",
	31:                 "+4 damage for a Dracoid", // C: RaceClassSex 3
	32:                 "+4 damage for a Dracoid",
	103:                "+4 damage for a Dracoid",
	104:                "+4 damage for a Dracoid",
	36:                 "Shoots two arrows",
	53:                 "Rogue skill +1",
	54:                 "Rogue skill -1",
	64:                 "1 in 10: triple damage",
	82:                 "1 in 7: also hurts its wielder",
	88:                 "Fighter skill +1",
	94:                 "Slays a foe below 1/4 health",
	114:                "1 in 20: triple damage",
	123:                "1 in 4: slays, then breaks",
	127:                "Fighter skill -1",
	274:                "Triple damage to illusions", // C: monsterIsIllusion
	itemTypeGinseng:    "Also cures poison",
	itemTypeJade:       "Worn: rogue skill +1",
	itemTypeTruthHalfB: "Worn: protection +5",
	itemTypeTruthHalfA: "Worn: protection +5",
}

// itemPurpose is a few words on what an item type is for, "" when there is nothing to say.
func (g *Game) itemPurpose(item, t int, it *ItemType) string {
	spell := func(sp int) string {
		if sp < 0 || sp >= len(spells) || spells[sp].Name == 0 {
			return "a spell"
		}
		return g.text(spells[sp].Name)
	}
	// an item's cast runs the handler of that number (spellcast.go's castSpell)
	cast := func(h int) string {
		for i := range spells {
			if g.handlerOf(i) == h && spells[i].Name != 0 {
				return g.text(spells[i].Name)
			}
		}
		switch h {
		case spellSwarm:
			return "a swarm"
		case spellGuardian:
			return "a guardian"
		case spellReveal:
			return "a spell that unmasks illusions"
		}
		return "a spell"
	}
	count := g.items[item].Flags & itemFlagsMask
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
	case itemScriptRead, itemScriptRiddle, itemScriptReadOnce:
		return "Use to read"
	case itemScriptGlass:
		return "Useless"
	case itemScriptScroll: // C: opLearnSpell
		return "Teaches " + spell(int(it.Might))
	case itemScriptGoblet: // C: heals Might HP for 1 MP
		return "Heals fully for 1 MP"
	case itemScriptBezel:
		return fmt.Sprintf("Heals fully, %d uses left", count)
	case itemScriptWand:
		return fmt.Sprintf("Casts %s, %d charges", cast(int(it.Might)), count)
	case itemScriptCube:
		if t == itemTypeFullCube {
			return "Restores 25 MP"
		}
		return "Unmasks illusions, then holds 25 MP"
	case itemScriptCard:
		return "Casts " + cast(int(it.Might)) + " once"
	case itemScriptGauntlet:
		return "Casts " + cast(spellHandOfFate)
	case itemScriptSkull:
		return "Casts " + spell(int(it.Might)) + " with your MP"
	case itemScriptMagic:
		return "Restores all MP, once"
	case itemScriptNecklace:
		if t == itemTypeTruth {
			return "Use on a creature to show its true self"
		}
		return ""
	case itemScriptVial:
		return "Can be emptied"
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
