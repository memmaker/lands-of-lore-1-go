package game

import "github.com/memmaker/lands-of-lore-1-go/internal/clock"

// Combat (C: s004 fn_070c_*): hit chances, damage, experience and finding targets. A
// fighter is a character's index or a monster's number with monsterFlag.

// The fighting factors of a character's modifier set or a monster type's
// fightingStats (C: struct Attack).
const (
	statHit       = 0 // the attacker's skill
	statDamage    = 1 // scales the damage dealt
	statArmour    = 2 // scales the protection
	statDefence   = 3 // the defender's skill
	statRecovery  = 4 // scales the time between attacks
	statMagicGain = 6 // scales the magic points gained on a level up

	damageKinds     = 8
	resistanceKinds = 7
	damageAnyKind   = 1 << 7 // a damage kind of this bit uses the last resistance
	fullFactor      = 256
	maxProtection   = 217
	minDamage       = 2
	skillFighter    = 0
	skillRogue      = 1
	skillMage       = 2
)

// difficultyDefence scales the characters' defence by difficulty (C: f_2803_003e).
var difficultyDefence = [3]int{128, 256, 256}

// isMonster says whether fighter who is a monster.
func isMonster(who int) bool { return who&monsterFlag != 0 }

// fighterStats returns a fighter's fighting factors, the might of each damage kind and
// the protection against each (C: fn_070c_01b6).
func (g *Game) fighterStats(who int) (fighting []int, might, protection *[damageKinds]int) {
	if isMonster(who) {
		t := g.monsters[who&monsterIndex].Type
		return t.fightingStats[:], &t.itemsMight, &t.protectionAgainstItems
	}
	c := &g.party[who]
	return characterModifiers[c.Modifiers][:], &c.ItemsMight, &c.ProtectionAgainstItems
}

// protectionOf is a fighter's protection (C: fn_070c_000a).
func (g *Game) protectionOf(who int) int {
	if isMonster(who) {
		t := g.monsters[who&monsterIndex].Type
		return scale(t.itemProtection, t.fightingStats[statArmour])
	}
	c := &g.party[who]
	v := c.ItemProtection + c.Protection
	v = scale(v, characterModifiers[c.Modifiers][statArmour])
	return scale(v, c.TotalProtectionModifier)
}

// selectCharacter makes character n the selected one and redraws both portraits
// (C: fn_070c_009f).
func (g *Game) selectCharacter(n int) {
	if g.selectedCharacter != n {
		old := g.selectedCharacter
		g.selectedCharacter = n
		g.drawPortrait(old)
	}
	g.drawPortrait(n)
}

// experienceNeeded is the experience each skill level starts at (C: f_2803_0000).
var experienceNeeded = [...]int{0, 500, 1500, 5000, 25000, 50000, 80000, 120000, 180000, 270000, 1<<31 - 1}

// addExperience gives character who experience in a skill and raises its levels
// (C: fn_070c_00cf).
func (g *Game) addExperience(who, skill, exp int) {
	if isMonster(who) || who < 0 || g.party[who].Flags&charDead != 0 {
		return
	}
	g.party[who].Experience[skill] += exp
	for g.raiseSkill(who, skill) {
	}
}

// Level ups.
const (
	charSkillRaised    = 1 << 9 // shifted by the skill
	stringFighterLevel = 0x4023
	stringRogueLevel   = 0x4025
	stringMageLevel    = 0x4024
	soundLevelUp       = 118
	messageLevelUp     = messageSilent | 3
	mageMagicBase      = 17
)

// raiseSkill raises character ch's skill a level if its experience is enough, with more
// hit points and for a mage magic points (C: ovr_2cf4_0025).
func (g *Game) raiseSkill(ch, skill int) bool {
	c := &g.party[ch]
	if experienceNeeded[c.SkillLevels[skill]] > c.Experience[skill] {
		return false
	}
	c.SkillLevels[skill]++
	c.Flags |= charSkillRaised << skill
	var n int
	switch skill {
	case skillFighter:
		g.printMessage(messageLevelUp, g.text(stringFighterLevel), c.Name)
		n = dice(4, 6)
	case skillMage:
		g.printMessage(messageLevelUp, g.text(stringMageLevel), c.Name)
		m := scale(dice(1, 8)+mageMagicBase, characterModifiers[c.Modifiers][statMagicGain])
		c.MP += m
		c.MaxMP += m
		n = dice(1, 6)
	case skillRogue:
		g.printMessage(messageLevelUp, g.text(stringRogueLevel), c.Name)
		n = dice(2, 6)
	}
	c.HP += n
	c.MaxHP += n
	g.playSound(soundLevelUp, 0)
	g.drawPortrait(ch)
	return true
}

// attackRecovery is the time before character who may attack again after base: shorter
// with rogue skill (C: fn_070c_012a).
func (g *Game) attackRecovery(who, base int) int {
	fighting, _, _ := g.fighterStats(who)
	v := ratio(base, fighting[statRecovery])
	if isMonster(who) {
		return v // fix: the original went on to read a character entry for the monster
	}
	v = scale(v, difficultyAttacks[g.difficulty()])
	c := &g.party[who]
	switch {
	case c.SkillLevels[skillRogue] > 7:
		v -= v >> 1
	case c.SkillLevels[skillRogue] > 3:
		v -= v >> 2
	}
	return v
}

// Where each party member stands in the block, by party size (C: g_1f16).
var memberPositions = [3][4]int{{128}, {64, 192}, {64, 128, 192}}

const blockCentreY = 128

// memberPosition is where in the party's block member stands; the party is never
// more than three (C: fn_070c_0201).
func (g *Game) memberPosition(member int) (x, y int) {
	x, y = turnInBlock(memberPositions[g.activeCharacters()-1][member], blockCentreY, g.direction)
	x |= g.partyX &^ lowByteMask
	y |= g.partyY &^ lowByteMask
	return x, y
}

// monsterInFront is the monster in the block ahead nearest to member
// (C: fn_070c_0253).
func (g *Game) monsterInFront(member int) int {
	return g.nearestMonsterIn(g.blockInDirection(g.currentBlock, g.direction), member)
}

// noTarget is no fighter.
const noTarget = -1

// nearestMonsterIn is the monster in block nearest to member, or noTarget
// (C: fn_070c_027a).
func (g *Game) nearestMonsterIn(block, member int) int {
	x, y := g.memberPosition(member)
	best, res := 1<<15-1, noTarget
	for cur := g.lvl.blocks[block].Objects; cur&monsterFlag != 0; {
		m := &g.monsters[cur&monsterIndex]
		if m.Mode < monsterDead {
			if d := abs(x-m.X) + abs(y-m.Y); d < best {
				best, res = d, cur
			}
		}
		cur = m.Next
	}
	return res
}

// nearestMonster is the monster nearest to (x, y) anywhere (C: fn_070c_0322).
func (g *Game) nearestMonster(x, y int) int {
	best, res := 1<<15-1, noTarget
	for i := range g.monsters {
		m := &g.monsters[i]
		if m.Mode >= monsterDead {
			continue
		}
		if d := abs(x-m.X) + abs(y-m.Y); d < best {
			best, res = d, i|monsterFlag
		}
	}
	return res
}

// characterFacing is the living character nearest to (x, y), whom a monster there
// attacks (C: fn_070c_03b1).
func (g *Game) characterFacing(x, y int) int {
	best, res := 1<<15-1, noTarget
	for i := range partySize {
		c := &g.party[i]
		if c.Flags&charPresent != 0 && c.HP > 0 {
			px, py := g.memberPosition(i)
			if d := abs(px-x) + abs(py-y); d < best {
				best, res = d, i
			}
		}
	}
	return res
}

// Results of attackHits.
const (
	attackMissed  = 0
	attackHit     = 1
	attackSkilled = 2 // beat the attacker's skill roll
)

// attackHits rolls whether att hits def with skill skill (C: fn_070c_044a).
func (g *Game) attackHits(att, def, skill int) int {
	if def == noTarget {
		return attackMissed
	}
	if att == noTarget {
		return attackHit
	}
	if isMonster(def) && g.monsters[def&monsterIndex].Mode >= monsterDead {
		return attackMissed
	}
	var chance int
	if isMonster(att) {
		chance = 100 - g.monsters[att&monsterIndex].Type.evade
	} else {
		c := &g.party[att]
		bonus := c.SkillModifiers[skill]
		if skill == skillRogue {
			bonus *= 3
		}
		chance = 100 - (c.SkillLevels[skill] + bonus)
	}
	if isMonster(def) {
		g.monsters[def&monsterIndex].Flags |= monsterWasHit
	}
	attacker, _, _ := g.fighterStats(att)
	defender, _, _ := g.fighterStats(def)
	r := dice(1, 100)
	if r >= chance {
		return attackSkilled
	}
	e := scale(defender[statDefence], difficultyDefence[g.difficulty()])
	if r < ratio(e, attacker[statHit]) {
		return attackMissed
	}
	return attackHit
}

// characterDies (C: fn_070c_0664).
func (g *Game) characterDies(who, how int) {
	c := &g.party[who]
	c.HP = 0
	c.Flags |= charDead
	g.clearStatusEffects(c, effectAttackRecovery, effectParalysis)
	g.deadPartyCheck = how
}

// damageOfKind is the damage att deals def with base might of a kind, after def's
// resistance and, unless mode is 2, its protection (C: fn_070c_06a3).
func (g *Game) damageOfKind(att, def, base, kind, mode int) int {
	v := fullFactor
	if att != noTarget {
		fighting, _, _ := g.fighterStats(att)
		v = fighting[statDamage]
	}
	_, _, protection := g.fighterStats(def)
	if v = scale(base, v); v == 0 {
		return 0
	}
	if !isMonster(att) {
		if v = scale(v, g.party[att].TotalMightModifier); v == 0 {
			return 0
		}
	}
	if kind&damageAnyKind != 0 {
		v = scaleSigned(v, protection[resistanceKinds])
	} else {
		v = scaleSigned(v, protection[kind])
	}
	if mode != 2 && v != 0 {
		t := g.protectionOf(def) << 7 / v
		t = min(t, maxProtection)
		v = max(scaleSigned(v, fullFactor-t), minDamage)
	}
	if v == 1 {
		v = minDamage
	}
	return v
}

// attackDamage sums the damage of all kinds who deals def (C: fn_070c_0793).
func (g *Game) attackDamage(who, def, mode int) int {
	_, might, _ := g.fighterStats(who)
	sum := 0
	for i := range damageKinds {
		sum += g.damageOfKind(who, def, might[i], i, mode)
	}
	return sum
}

// scaledDamage rolls the damage dealt for strength n (C: fn_070c_07e3).
func (g *Game) scaledDamage(n int) int { return dice(2, n) }

// Damage effects.
const (
	damageShownFlag   = 1 << 15 // a monster's Damage shows above it
	hitShake          = 24
	hurtFace          = 6
	hurtFaceTicks     = 4
	damageFire        = 4
	itemTypeFireGuard = 229 // carried, it quarters fire damage
	damageShownTicks  = 4
)

// damageFighter deals dmg to who from src (noTarget or a monster for none to credit)
// with skill skill; it returns the damage done (C: fn_070c_07f5).
func (g *Game) damageFighter(who, dmg, src, skill, how int) int {
	var m *Monster
	var c *Character
	if isMonster(who) {
		m = &g.monsters[who&monsterIndex]
		if m.Mode >= monsterDead {
			return 0
		}
	} else {
		c = &g.party[who]
		if c.Flags&charPresent == 0 || c.Flags&charDead != 0 {
			return 0
		}
		if c.Flags&charMute == 0 {
			g.playSound(c.Scream, 0)
		}
		g.setTemporaryFace(who, hurtFace, hurtFaceTicks, false)
		// the original printed the damage to its debug console here
	}
	if m != nil {
		if dmg > 0 {
			m.HitPoints -= dmg
			m.Damage = damageShownFlag | dmg
			m.Flags |= monsterWasHit
			m.DX = dice(1, hitShake) - hitShake/2
			m.DY = dice(1, hitShake) - hitShake/2
			m.HitPoints = max(min(m.HitPoints, m.Type.hitPoints), 0)
			if !isMonster(src) {
				g.monsterDefence(m, src, how, skill, dmg)
			}
			g.playSoundAt(m.Type.sound, m.Block)
			g.noteBlockChanged(m.Block)
			if m.HitPoints <= 0 {
				m.HitPoints = 0
				if !isMonster(src) {
					g.addExperience(src, skill, m.Type.hitPoints)
				}
				g.setMonsterMode(m, monsterDead)
			}
		} else {
			m.HitPoints -= dmg
			dmg = 0
			m.HitPoints = max(min(m.HitPoints, m.Type.hitPoints), 1)
		}
	} else {
		if how == damageFire && g.carries(who, itemTypeFireGuard) {
			dmg >>= 2
		}
		g.setPoints(who, false, -dmg, pointsAdd)
		if c.HP <= 0 {
			g.characterDies(who, how)
		} else {
			c.DamageSuffered = dmg
			g.addStatusEffect(who, effectDamageShown, damageShownTicks, true)
		}
		g.drawPortrait(who)
	}
	if !isMonster(src) {
		if skill == 0 {
			g.party[src].WeaponHit += dmg
		}
		g.addExperience(src, skill, dmg)
	}
	return dmg
}

// addStatusEffect starts effect kind on character ch for ticks of the characters
// timer, in a free slot or with replace in the effect's own (C: fn_0bbc_0744).
func (g *Game) addStatusEffect(ch, kind, ticks int, replace bool) {
	c := &g.party[ch]
	for i := range c.StatusEffects {
		if c.StatusEffects[i] == 0 || (replace && c.StatusEffects[i] == kind) {
			c.StatusEffects[i] = kind
			c.StatusTimers[i] = int(int8(ticks))
			t := &g.timers[timerCharacters]
			t.next = clock.Get(t.clock)
			t.enabled = true
			break
		}
	}
}

// clearStatusEffects ends effects from to to on character c (C: fn_0bbc_0365).
func (g *Game) clearStatusEffects(c *Character, from, to int) {
	for kind := from; kind <= to; kind++ {
		switch kind {
		case effectAttackRecovery:
			c.Flags &^= charHitOnly
			c.WeaponHit = 0
		case effectDamageShown:
			c.DamageSuffered = 0
		case effectFaceOverlay:
			c.Flags &^= charFaceOverlay
		case effectPoison:
			c.Flags &^= charFaceTinted
		case effectParalysis:
			c.Flags &^= charParalysed
		case effectMute:
			c.Flags &^= charMute
		}
		for k := range c.StatusEffects {
			if c.StatusEffects[k] == kind {
				c.StatusEffects[k] = 0
				c.StatusTimers[k] = 0
			}
		}
	}
	g.setTimer(timerCharacters, 1, true)
}
