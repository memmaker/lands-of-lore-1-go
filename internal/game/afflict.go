package game

// Afflictions a monster's hit or a script lays on a character (C: o2d00).
const (
	afflictMute      = charMute
	afflictPoison    = charFaceTinted
	afflictOverlay   = charFaceOverlay
	muteTicks        = 120
	poisonTicks      = 10
	overlayTicks     = 3600
	poisonAntidote   = 225 // an item type that keeps poison off whoever carries it
	stringPoisonedBy = 0x4021
	stringParalysed  = 0x4026
	messagePoison    = 6
	paralysisTicks   = 20
)

// afflict gives character ch affliction kind with chance percent, unless one of mask's
// flags is already set. With redraw it redraws the portrait itself and returns false
// (C: ovr_2d00_0025).
func (g *Game) afflict(ch, kind, mask, chance int, redraw bool) bool {
	c := &g.party[ch]
	if c.Flags&charPresent == 0 || c.Flags&mask != 0 {
		return false
	}
	if dice(1, 100) > chance {
		return false
	}
	done := false
	switch kind {
	case afflictMute:
		c.Flags |= charMute
		done = true
		g.addStatusEffect(ch, effectMute, muteTicks, true)
	case afflictPoison:
		if g.carries(ch, poisonAntidote) {
			break
		}
		c.Flags |= charFaceTinted
		g.addStatusEffect(ch, effectPoison, poisonTicks, true)
		if g.characterSays(stringPoisonedBy, c.ID, true) {
			g.printMessage(messagePoison, g.text(stringPoisonedBy), c.Name)
		}
		done = true
	case afflictOverlay:
		g.addStatusEffect(ch, effectFaceOverlay, overlayTicks, true)
		c.Flags |= charFaceOverlay
		done = true
	}
	if redraw && done {
		g.drawPortrait(ch)
		return false
	}
	return done
}

// afflictParty tries the affliction on every character (C: ovr_2d00_002a).
func (g *Game) afflictParty(kind, mask, chance int) {
	any := false
	for ch := range partySize {
		if g.afflict(ch, kind, mask, chance, false) {
			any = true
		}
	}
	if any {
		g.drawPortraits()
	}
}

// paralyseCharacter stops character ch for a while unless it is dead or already
// paralysed (C: ovr_2d00_0020).
func (g *Game) paralyseCharacter(ch int) {
	c := &g.party[ch]
	if c.Flags&charPresent == 0 || c.Flags&(charParalysed|charDead) != 0 {
		return
	}
	c.Flags |= charParalysed
	g.addStatusEffect(ch, effectParalysis, paralysisTicks, true)
	g.drawPortrait(ch)
	g.printMessage(0, g.text(stringParalysed), c.Name)
}
