package game

// Status effects: each character has five slots of an effect and the thinks of the
// characters timer (timer 3, every 15 ticks) until it ends (C: fn_0bbc_0143).
const (
	effectAttackRecovery = 1 // the hand shows used until the character may hit again
	effectDamageShown    = 2 // the damage taken shows on the portrait
	effectFaceOverlay    = 3 // a shape lies over the face
	effectPoison         = 4
	effectParalysis      = 5
	effectTemporaryFace  = 6
	effectMute           = 7
	effectLevelPalette   = 8
	charParalysed        = 1 << 8
	poisonDice           = 2
	poisonRepeat         = 10
	poisonDamageKind     = 1 << 7
	poisonSource         = monsterFlag
	stringPoisoned       = 0x4022
	stringCanMove        = 0x4027
	messageEffect        = 2
	characterTicks       = 15
	recoveryBase         = 6
)

func (g *Game) updateCharacters(int) {
	longest := 0
	for ch := range g.party {
		c := &g.party[ch]
		if c.Flags&charPresent == 0 {
			continue
		}
		for k := range updateSlots {
			if c.StatusEffects[k] == 0 {
				continue
			}
			c.StatusTimers[k] = int(int8(c.StatusTimers[k] - 1))
			if c.StatusTimers[k] > 0 {
				longest = max(longest, c.StatusTimers[k])
				continue
			}
			switch c.StatusEffects[k] {
			case effectAttackRecovery:
				if c.WeaponHit != 0 {
					c.WeaponHit = 0
					c.StatusTimers[k] = int(int8(g.attackRecovery(ch, recoveryBase)))
					longest = max(longest, c.StatusTimers[k])
				} else {
					c.Flags &^= charHitOnly
				}
				g.drawPortrait(ch)
			case effectDamageShown:
				c.DamageSuffered = 0
				g.drawPortrait(ch)
			case effectFaceOverlay:
				c.Flags &^= charFaceOverlay
				g.drawPortrait(ch)
			case effectPoison:
				longest = dice(1, poisonDice) // sic: the original kept its maximum in the same variable
				if g.damageFighter(ch, longest, poisonSource, 0, poisonDamageKind) != 0 {
					g.printMessage(messageEffect, g.text(stringPoisoned), c.Name)
					c.StatusTimers[k] = poisonRepeat
					longest = max(longest, c.StatusTimers[k])
				}
			case effectParalysis:
				c.Flags &^= charParalysed
				g.printMessage(0, g.text(stringCanMove), c.Name)
				g.drawPortrait(ch)
			case effectTemporaryFace:
				g.setTemporaryFace(ch, 0, 0, true)
			case effectMute:
				c.Flags &^= charMute
				g.drawPortrait(ch)
			case effectLevelPalette:
				g.swapLevelPaletteBack()
			}
			if c.StatusTimers[k] <= 0 {
				c.StatusEffects[k] = 0
			}
		}
	}
	if longest != 0 {
		g.setTimer(timerCharacters, longest*characterTicks, true)
	} else {
		g.enableTimer(timerCharacters, false)
	}
}
