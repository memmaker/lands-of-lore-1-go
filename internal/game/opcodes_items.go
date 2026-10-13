package game

import "github.com/memmaker/lands-of-lore-1-go/internal/emc"

// Game flags are a bit array scripts set and test (C: g_804c, 16 words up to g_806a).
// The engine keeps its own state in the last word, flags 240 to 255 (C: g_806a).
const (
	gameFlagWords = 16
	wordBits      = 16
	gameFlagCount = gameFlagWords * wordBits

	engineFlags    = (gameFlagWords - 1) * wordBits
	flagBusy       = engineFlags + 9  // the busy cursor is up
	flagLampLit    = engineFlags + 10 // (C: 0x400)
	flagHasLamp    = engineFlags + 11 // (C: 0x800)
	flagHasAutomap = engineFlags + 12 // (C: 0x1000)
	flagHasMagic   = engineFlags + 13 // the spell list is shown (C: 0x2000)
	flagHasCompass = engineFlags + 14 // (C: 0x4000)
)

// flag says whether game flag n is set.
func (g *Game) flag(n int) bool {
	if n < 0 || n >= gameFlagCount {
		return false
	}
	return g.gameFlags[n/wordBits]&(1<<(n%wordBits)) != 0
}

// setFlag sets or clears game flag n.
func (g *Game) setFlag(n int, on bool) {
	if n < 0 || n >= gameFlagCount { // fix: the original wrote past the flags
		return
	}
	if on {
		g.gameFlags[n/wordBits] |= 1 << (n % wordBits)
	} else {
		g.gameFlags[n/wordBits] &^= 1 << (n % wordBits)
	}
}

// opSetGameFlag sets or clears game flag arg 0 (C: fn_0fbe_0290).
func (g *Game) opSetGameFlag(s *emc.State) int {
	g.setFlag(int(uint16(s.Arg(0))), s.Arg(1) != 0)
	return 1
}

// opTestGameFlag returns game flag arg 0 (C: fn_0fbe_02e8).
func (g *Game) opTestGameFlag(s *emc.State) int {
	if g.flag(s.Arg(0)) {
		return 1
	}
	return 0
}

// opMakeItem makes an item of type arg 0 with flags arg 1 and arg 2 (C: fn_0fbe_0341).
func (g *Game) opMakeItem(s *emc.State) int {
	return g.newItem(s.Arg(0), s.Arg(1), s.Arg(2))
}

// Item fields scripts read and write (C: fn_0fbe_04a8, fn_0fbe_05fa).
const (
	itemFieldBlock = iota
	itemFieldX
	itemFieldY
	itemFieldLevel
	itemFieldType
	itemFieldFlags
	itemFieldName
	itemFieldUnused
	itemFieldShape
	itemFieldSlot
	itemFieldScript
	itemFieldMight
	itemFieldSkill
	itemFieldProtection
	itemFieldUnknown
	itemFieldCount // the item's own number: its flags' low bits
	itemFieldTypeFlags
	itemFieldSkillAndMight
)

// opGetItemField returns a field of item arg 0 or its type, -1 for an unknown field
// (C: fn_0fbe_04a8).
func (g *Game) opGetItemField(s *emc.State) int {
	item := s.Arg(0)
	if item < 1 || item >= maxItems {
		return 0
	}
	it := &g.items[item]
	if it.Type < 0 || it.Type >= len(g.itemTypes) { // fix: the original read past the types
		return -1
	}
	t := &g.itemTypes[it.Type]
	switch s.Arg(1) {
	case itemFieldBlock:
		return it.Block
	case itemFieldX:
		return it.X
	case itemFieldY:
		return it.Y
	case itemFieldLevel:
		return it.Level
	case itemFieldType:
		return it.Type
	case itemFieldFlags:
		return it.Flags
	case itemFieldName:
		return t.Name
	case itemFieldTypeFlags:
		return t.Flags
	case itemFieldShape:
		return int(t.Shape)
	case itemFieldSlot:
		return t.Slot
	case itemFieldScript:
		return int(t.Script)
	case itemFieldMight:
		return int(t.Might)
	case itemFieldSkill:
		return int(t.Skill)
	case itemFieldProtection:
		return int(t.Protection)
	case itemFieldSkillAndMight:
		return int(t.Skill)<<8 | int(t.Might)
	case itemFieldUnknown:
		return t.Unknown
	case itemFieldCount:
		return it.Flags & itemFlagsMask
	}
	return -1
}

// opSetItemField sets an item's number or type (C: fn_0fbe_05fa).
func (g *Game) opSetItemField(s *emc.State) int {
	item := s.Arg(0)
	if item < 1 || item >= maxItems {
		return 0
	}
	it := &g.items[item]
	v := s.Arg(2)
	switch s.Arg(1) {
	case itemFieldCount:
		v &= itemFlagsMask
		it.Flags = it.Flags&^itemFlagsMask | v
	case itemFieldType:
		it.Type = v
	default:
		v = -1
	}
	return v
}

// Character fields scripts read and write (C: fn_0fbe_0677, fn_0fbe_079e).
const (
	charFieldFlags          = 0
	charFieldRaceClassSex   = 1
	charFieldHP             = 5
	charFieldMaxHP          = 6
	charFieldMP             = 7
	charFieldMaxMP          = 8
	charFieldItemProtection = 9
	charFieldItem           = 10
	charFieldSkill          = 11
	charFieldProtection     = 12
	charFieldMight          = 13
	charFieldSkillModifier  = 14
	charFieldID             = 15

	mightSlotFlag = 1 << 7 // any index with this bit reads the last might slot
)

// opGetCharacterField returns field arg 1 (index arg 2) of character arg 0
// (C: fn_0fbe_0677).
func (g *Game) opGetCharacterField(s *emc.State) int {
	c := &g.party[s.Arg(0)&(partySize-1)]
	i := s.Arg(2)
	switch s.Arg(1) {
	case charFieldFlags:
		return c.Flags
	case charFieldRaceClassSex:
		return c.RaceClassSex
	case charFieldHP:
		return c.HP
	case charFieldMaxHP:
		return c.MaxHP
	case charFieldMP:
		return c.MP
	case charFieldMaxMP:
		return c.MaxMP
	case charFieldItemProtection:
		return c.ItemProtection
	case charFieldItem:
		return c.Items[i]
	case charFieldSkill:
		return c.SkillLevels[i] + c.SkillModifiers[i]
	case charFieldSkillModifier:
		return c.SkillModifiers[i]
	case charFieldProtection:
		return c.ProtectionAgainstItems[i]
	case charFieldMight:
		if i&mightSlotFlag != 0 {
			return c.ItemsMight[itemStatSlots-1]
		}
		return c.ItemsMight[i]
	case charFieldID:
		return c.ID
	}
	return 0
}

// opSetCharacterField sets field arg 1 (index arg 2) of character arg 0 to arg 3; hit
// and magic points go through setPoints, which animates the bar (C: fn_0fbe_079e).
func (g *Game) opSetCharacterField(s *emc.State) int {
	n := s.Arg(0) & (partySize - 1)
	c := &g.party[n]
	i, v := s.Arg(2), s.Arg(3)
	switch s.Arg(1) {
	case charFieldFlags:
		c.Flags = int(uint16(v))
	case charFieldRaceClassSex:
		c.RaceClassSex = int(int8(v))
	case charFieldHP:
		g.setPoints(n, false, v, pointsSet)
	case charFieldMaxHP:
		c.MaxHP = v
	case charFieldMP:
		g.setPoints(n, true, v, pointsSet)
	case charFieldMaxMP:
		c.MaxMP = v
	case charFieldItemProtection:
		c.ItemProtection = v
	case charFieldItem:
		c.Items[i] = v
	case charFieldSkill:
		c.SkillLevels[i] = int(int8(v))
	case charFieldSkillModifier:
		c.SkillModifiers[i] = int(int8(v))
	case charFieldProtection:
		c.ProtectionAgainstItems[i] = v
	case charFieldMight:
		if i&mightSlotFlag != 0 {
			c.ItemsMight[itemStatSlots-1] = v
		} else {
			c.ItemsMight[i] = v
		}
	}
	return 0
}
