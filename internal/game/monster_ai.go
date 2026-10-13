package game

import "github.com/memmaker/lands-of-lore-1-go/internal/random"

// Monster behaviour (C: s017): every few ticks a monster thinks in its mode: walks to a
// place, stands guard, strays, chases and fights the party, flees, dies.

// Monster modes (C: Monster.mode; the meanings follow ScummVM's LoLMonster.mode).
const (
	monsterModeStill     = 0  // stands where it is
	monsterModeAll       = 1  // waits with the others; one woken wakes them all
	monsterModeWalk      = 2  // walks to its destination, then turns its way
	monsterModeGuard     = 3  // looks out for the party
	monsterModeStray     = 4  // wanders about
	monsterModeRetreat   = 5  // steps back after an attack
	monsterModeWait      = 6  // waits a while
	monsterModeChase     = 7  // goes for the party
	monsterModeAttack    = 8  // its attack animation plays
	monsterModeFlee      = 9  // runs away for a while
	monsterModeHit       = 11 // shows being hit
	monsterModeVanish    = 12 // fades away, then its death script runs
	monsterDead          = 13
	monsterModeRemove    = 14 // gone
	monsterModeTesting   = 15 // set while a step is tried
	lastThinkingMode     = monsterModeRemove
	monsterDeathFunction = 1028 // the level script function run when a monster goes (C: 0x404)
)

// What a monster does in each mode, besides the mode's own work (C: g_2483).
const (
	modeWakesWhenHit = 1 << 0
	modeLooksOut     = 1 << 1
	modeEveryTick    = 1 << 2 // acts on every timer tick, not only after its think delay
)

var monsterModeFlags = [17]int{1, 0, 1, 3, 3, 0, 0, 3, 4, 1, 0, 0, 4}

// Monster type flags (C: MonType.w3e).
const (
	monsterLooksAhead     = 1 << 0  // sees only the way it walks
	monsterLooksBehind    = 1 << 1  // sees behind it too
	monsterAlwaysSees     = 1 << 3  // knows where the party is
	monsterRandomSpells   = 1 << 4  // casts its spells in random order
	monsterRegenerates    = 1 << 6  // heals a little every think
	monsterOpensDoors     = 1 << 7  // opens the doors it walks into
	monsterNeverMoves     = 1 << 10 // stays where it was put
	monsterPassesMonsters = 1 << 12 // only closed doors stop it, and it leaves no mark
)

// Monster flags (C: Monster.flags).
const (
	monsterFleeing   = 1 << 3
	monsterWasHit    = 1 << 4
	monsterDropItems = 1 << 5 // its death runs the level script instead of the usual end
)

// Walls monsters do not pass: the ones of type flag 2, or for those that pass monsters
// only closed doors (C: fn_0d6c_059d's 4 and 0x20).
const (
	wallStopsMonsters = 1 << 2
	stepCheckAll      = blockedByAll
	healDice          = 8
	attackReach       = 160 // how far ahead a monster can hit (C: 0xa0)
	attackSideReach   = 80  // and to the side (C: 0x50)
	retreatReach      = 256 // the party this near keeps a retreating monster backing off
	positionStep      = 32  // one step of a walking monster
	monsterTicks      = 15  // the animation tick wraps here
	strayTries        = 7
	strayStraight     = 3 // steps a strayer goes on before choosing anew
	flagsHitMarker    = 1 << 15
	attackPauseSteps  = 8
	attackFrameCount  = 2
	monsterLookRows   = 3
	monsterLookWindow = 13
	guardTurnChance   = 50
	deathFrames       = 13
	wakeAllRange      = 6
)

// The step to the next position in each of the 8 directions (C: g_243a, g_244a).
var (
	stepX = [8]int{0, 32, 32, 32, 0, -32, -32, -32}
	stepY = [8]int{-32, -32, 0, 32, 32, 32, 0, -32}
)

// Turns tried in order when the way is blocked; two tables, swapped every ten
// searches so monsters do not keep to one side (C: g_245d, g_246d).
var (
	tryTurns    = [8]int{7, -6, 5, -4, 3, -2, 1, 0}
	tryTurnsAlt = [8]int{-7, 6, -5, 4, -3, 2, -1, 0}
)

// How a walking monster turns to face its target direction, by facing and target
// (C: g_241a).
var turnTowards = [16]int{0, 1, 3, 3, 0, 1, 2, 2, 1, 1, 2, 3, 0, 0, 2, 3}

// The slot of the monster's view the party is in, by its rows away and column; row 0
// read the byte before the table (C: g_245a).
var lookSlots = [monsterLookRows + 1]int{-1, 13, 9, 3}

// The turn a strayer makes, by a die of two (C: g_2480).
var strayTurns = [3]int{0, 2, -2}

// The pauses after an attack by difficulty (C: f_2803_0038).
var difficultyPauses = [3]int{320, 256, 128}

// nextStep is the position one step from (x, y) towards dir (C: fn_0d6c_0004).
func nextStep(x, y, dir int) (int, int) {
	return (x + stepX[dir]) & positionMask, (y + stepY[dir]) & positionMask
}

// dropMonsterItems drops what m carries where it stands (C: fn_0d6c_0034).
func (g *Game) dropMonsterItems(m *Monster) {
	for it := m.Items; it != 0; {
		t := it
		it = g.items[t].Next
		g.dropItem(t, m.X, m.Y, 0, true)
	}
}

// canSeeParty says whether m sees the party, through the walls of its own view
// (C: fn_0d6c_0109). The view's blocks are left set to the monster's, as the original
// left them.
func (g *Game) canSeeParty(m *Monster) bool {
	if m.Type.flags&monsterAlwaysSees != 0 {
		return true
	}
	mx := m.X&^(blockUnits-1) | blockCentre
	my := m.Y&^(blockUnits-1) | blockCentre
	var d int
	if m.Type.flags&monsterLooksAhead != 0 {
		d = m.Direction >> 1
	} else {
		d = directionTo(mx, my, g.partyX, g.partyY)
		if m.Type.flags&monsterLooksBehind == 0 && m.Direction^4 == d {
			return false
		}
		d >>= 1
	}
	px, py := relativePosition(mx, my, g.partyX, g.partyY, d)
	px, py = int(int8(px>>byteShift)), int(int8(py>>byteShift))
	if py < 0 || py > monsterLookRows || abs(px) > py {
		return false
	}
	if py == 0 {
		return true // fix: the original looked up the clipping of slot -1, before its cache
	}
	for i := range viewSlots {
		g.visibleBlocks[i] = (m.Block + viewBlockOffsets[d+i]) & blockMask // sic: d, not d * 18
	}
	a, b := g.clipColumns(lookSlots[py]+px, monsterLookWindow)
	return a < b
}

// stepBlocked tests whether m could stand at (x, y) (C: fn_0d6c_059d).
func (g *Game) stepBlocked(x, y int, m *Monster, wallMask int) int {
	old := m.Mode
	m.Mode = monsterModeTesting
	if m.Type.flags&monsterPassesMonsters != 0 {
		wallMask = wallDoorClosed
	}
	r := g.placementBlocked(x, y, m.Type.maxWidth, stepCheckAll, wallMask)
	m.Mode = old
	return r
}

// findStep picks the direction for m's next step towards (tx, ty), trying turns when
// the way is blocked; -1 for none. A monster that opens doors opens one in its way
// (C: fn_0d6c_05f4).
func (g *Game) findStep(m *Monster, tx, ty int) int {
	g.stepSearches++
	if g.stepSearches > 10 {
		g.stepSearches = 0
		g.stepTurnsAlt = !g.stepTurnsAlt
	}
	turns := &tryTurns
	if g.stepTurnsAlt {
		turns = &tryTurnsAlt
	}
	dir := m.Direction
	d := directionTo(m.X, m.Y, tx, ty)
	if m.Flags&monsterFleeing != 0 {
		d ^= 4
	}
	d = (d - dir) & 7
	if d != 0 {
		if d < 5 {
			dir++
		} else {
			dir--
		}
		dir &= 7
	}
	for k := 7; k >= 0; k-- {
		dir = (dir + turns[k]) & 7
		nx, ny := nextStep(m.X, m.Y, dir)
		r := g.stepBlocked(nx, ny, m, wallStopsMonsters)
		if r == placeFree {
			return dir
		}
		if r == placeWall && dir&1 == 0 && m.Type.flags&monsterOpensDoors != 0 {
			w := g.lvl.blocks[g.blockingBlock].Walls[(dir>>1)^2]
			if g.lvl.wallFlags[w]&wallDoorClosed != 0 && g.lvl.wallSpecialType[w] == specialWallDoor {
				g.startDoor(g.blockingBlock, doorOpening)
				return -1
			}
			if g.lvl.wallFlags[w]&wallIsDoor != 0 {
				return -1
			}
		}
	}
	return -1
}

// walkMonster takes a step towards m's destination, unless it stays to cast at the
// party; boxed in, it backs out the way it was pushed (C: fn_0d6c_089e).
func (g *Game) walkMonster(m *Monster) {
	if m.Type.flags&monsterNeverMoves != 0 {
		return
	}
	d := g.findStep(m, m.DestX, m.DestY)
	if d != -1 {
		g.setMonsterDirection(m, d)
		if m.Casts == 0 || blockDistance(m.Block, g.currentBlock) < 2 ||
			g.lineToBlock(m.Block, m.Direction, 3, g.currentBlock) == noLine || m.CastDelay == 0 {
			nx, ny := nextStep(m.X, m.Y, d)
			g.moveMonster(m, nx, ny)
		}
	} else if g.stepBlocked(m.X, m.Y, m, wallStopsMonsters) == placeWall {
		g.blockingSide ^= 4
		g.setMonsterDirection(m, g.blockingSide)
		nx, ny := nextStep(m.X, m.Y, g.blockingSide)
		g.moveMonster(m, nx, ny)
	}
}

// setMonsterMode (C: fn_0d6c_0995). Chasing aims at the party; the first of a waiting
// group to chase sets the whole group on. A monster whose death runs a script goes
// through it instead of dying.
func (g *Game) setMonsterMode(m *Monster, mode int) {
	if m.Mode == monsterDead && mode != monsterModeRemove {
		return
	}
	if mode == monsterModeChase {
		m.DestX, m.DestY = g.partyX, g.partyY
	}
	if m.Mode == monsterModeAll && mode == monsterModeChase {
		for i := range g.monsters {
			o := &g.monsters[i]
			if o.Mode == monsterModeAll {
				o.Mode = mode
				o.Counter = 0
				o.DestX, o.DestY = g.partyX, g.partyY
				g.setMonsterDirection(o, directionTo(o.X, o.Y, o.DestX, o.DestY))
			}
		}
		return
	}
	m.Mode = mode
	m.Counter = 0
	if mode == monsterModeRemove {
		m.HitPoints = 0
	}
	if mode == monsterDead && m.Flags&monsterDropItems != 0 {
		m.Mode = monsterModeStill
		g.dropMonsterItems(m)
		if g.level != finalLevel {
			g.setMonsterMode(m, monsterModeRemove)
		}
		g.runBlockScriptWith(monsterDeathFunction, allEvents, m.ID, m.ID, 0, 0)
		g.noteBlockChanged(m.Block)
		if m.Mode == monsterModeRemove {
			g.moveMonster(m, 0, 0)
		}
	}
}

// Monster attack effects (C: MonType.hitKind).
const (
	hitDrainsParty  = 10
	hitHeals        = 9
	hitStealsItem   = 1
	hitBreaksItem   = 3
	hitKnocksItem   = 4
	hitDrainsMagic  = 6
	hitPoisons      = 2
	hitParalyses    = 8
	stealAnySlot    = 1<<11 - 1
	breakSlot       = 1 << 5
	knockHandSlots  = 1<<4 - 1
	messageWarning  = 6
	stringStolen    = 0x4019
	stringBroken    = 0x401b
	stringKnocked   = 0x401e
	stringDrained   = 0x4020
	stringHurtsBack = 0x401c
	stringDestroyed = 0x401d
	stringCallsHelp = 0x401a
)

// monsterHitEffect is what else m's hit does to character ch (C: fn_0d6c_0acc).
func (g *Game) monsterHitEffect(m *Monster, ch, damage int) {
	if dice(1, 100) > m.Type.attackSkill[0] {
		return
	}
	c := &g.party[ch]
	switch m.Type.attackSkill[1] {
	case hitDrainsParty:
		g.drainParty(64, 72, 100)
	case hitHeals:
		m.HitPoints = min(m.HitPoints+damage, m.Type.hitPoints)
	case hitStealsItem:
		if it := g.takeCharacterItem(ch, stealAnySlot, 0); it != 0 {
			g.giveMonsterItem(m, it)
			if g.characterSays(stringStolen, c.ID, true) {
				g.printMessage(messageWarning, g.text(stringStolen))
			}
		}
	case hitBreaksItem:
		if it := g.takeCharacterItem(ch, breakSlot, 0); it != 0 {
			g.freeItem(it)
			if g.characterSays(stringBroken, c.ID, true) {
				g.printMessage(messageWarning, g.text(stringBroken))
			}
		}
	case hitKnocksItem:
		it := g.takeCharacterItem(ch, knockHandSlots, 0)
		if it == 0 {
			break
		}
		if g.characterSays(stringKnocked, c.ID, true) {
			g.printMessage(messageWarning, g.text(stringKnocked), c.Name)
		}
		g.dropItem(it, m.X, m.Y, 0, true)
	case hitDrainsMagic:
		if c.MP > 0 {
			m.HitPoints += c.MP
			c.MP = 0
			g.drawPortrait(ch)
			if g.characterSays(stringDrained, c.ID, true) {
				g.printMessage(messageWarning, g.text(stringDrained), c.Name)
			}
		}
	case hitPoisons:
		g.poisonCharacter(ch, 128, 136, 100, 1)
	case hitParalyses:
		g.paralyseCharacter(ch)
	}
}

// Monster defences against the party's hits (C: MonType.defKind).
const (
	defenceGetsFaster   = 6
	defenceHealsOnMagic = 4
	defenceHealsOnFire  = 5
	defenceTakesWeapon  = 1
	defenceBreaksWeapon = 2
	defenceFlees        = 3
	hitIsSpell          = 1 << 7
	hitByHand           = 1<<7 | 1<<2
	hitKindMask         = 1<<6 - 1
	hitKindMagic        = 3
	hitKindBlunt        = 2
	handSlots           = 3
	fleeTicks           = 30
)

// monsterDefence is what m does when character ch hits it with damage of kind kind
// (C: fn_0d6c_0d2a).
func (g *Game) monsterDefence(m *Monster, ch, kind, ranged, damage int) {
	if dice(1, 100) > m.Type.defenseSkill[0] {
		return
	}
	c := &g.party[ch]
	switch m.Type.defenseSkill[1] {
	case defenceGetsFaster:
		if kind&hitByHand == hitByHand {
			m.Casts++
		}
	case defenceHealsOnMagic:
		if kind == hitKindMagic {
			m.HitPoints = min(m.HitPoints+damage, m.Type.hitPoints)
		}
	case defenceHealsOnFire:
		if kind&hitIsSpell != 0 {
			m.HitPoints = min(m.HitPoints+damage, m.Type.hitPoints)
		}
	case defenceTakesWeapon, defenceBreaksWeapon:
		if kind&hitKindMask == hitKindBlunt || ranged != 0 {
			break
		}
		for k := range handSlots {
			it := c.Items[k]
			if it == 0 || g.itemTypes[g.items[it].Type].Flags&hitKindMask != kind {
				continue
			}
			g.takeCharacterItem(ch, stealAnySlot, 0)
			if m.Type.defenseSkill[1] == defenceTakesWeapon {
				g.giveMonsterItem(m, it)
				if g.characterSays(stringHurtsBack, c.ID, true) {
					g.printMessage(messageWarning, g.text(stringHurtsBack))
				}
			} else {
				g.freeItem(it)
				if g.characterSays(stringDestroyed, c.ID, true) {
					g.printMessage(messageWarning, g.text(stringDestroyed))
				}
			}
		}
	case defenceFlees:
		if kind&hitIsSpell != 0 {
			g.monsterFlee(m, fleeTicks)
		}
	}
}

// monsterAttack has m hit the character it faces (C: fn_0d6c_0f1e).
func (g *Game) monsterAttack(m *Monster) {
	ch := g.characterFacing(m.X, m.Y)
	who := m.ID | monsterFlag
	g.playSound(m.Type.attackSound, 0)
	if t := g.attackHits(who, ch, 0); t != 0 {
		damage := g.attackDamage(who, ch, t)
		damage = g.scaledDamage(damage)
		g.damageCharacter(ch, damage, m.ID|monsterFlag, 0, 0)
		g.monsterHitEffect(m, ch, damage)
	}
	g.setMonsterMode(m, monsterModeAttack)
	g.noteBlockChanged(m.Block)
}

// firstMonsterIn is the monster first on block b, -1 for none (C: fn_0d6c_0fcb).
func (g *Game) firstMonsterIn(b int) int {
	o := g.lvl.blocks[b].Objects
	if o&monsterFlag != 0 {
		return o & monsterIndex
	}
	return -1
}

// crowded says whether m shares its block or has a monster beside it
// (C: fn_0d6c_0fea).
func (g *Game) crowded(m *Monster) bool {
	b := m.Block
	d := m.Direction >> 1
	if m.Next&monsterFlag != 0 || g.firstMonsterIn(b) != m.ID {
		return true
	}
	for range directionCount - 1 {
		d = (d + 1) & (directionCount - 1)
		if g.firstMonsterIn(g.blockInDirection(b, d)) != -1 {
			return true
		}
	}
	return false
}

// shuffleMonster moves m across its block, to the centre line or off it, keeping the
// party in reach (C: fn_0d6c_1050).
func (g *Game) shuffleMonster(m *Monster) {
	x, y := m.X, m.Y
	p := &x
	if (m.Direction>>1)&1 != 0 {
		p = &y
	}
	centred := *p&(blockCentre-1) == 0
	away := m.Type.maxWidth >= blockCentre/2 || (centred && !g.crowded(m))
	if centred && away {
		return
	}
	toward := positionStep
	if *p&blockCentre != 0 {
		toward = -positionStep
	}
	if away {
		*p -= toward
	} else {
		*p += toward
	}
	if g.stepBlocked(x, y, m, wallStopsMonsters) != placeFree {
		return
	}
	px, py := relativePosition(x, y, g.partyX, g.partyY, m.Direction>>1)
	if py <= attackReach && abs(px) <= attackSideReach {
		g.moveMonster(m, x, y)
	}
}

// chaseParty turns to the party and hits it when in reach, or walks on; at its
// destination it turns its way and guards or strays (C: fn_0d6c_1183).
func (g *Game) chaseParty(m *Monster) {
	if m.Flags&monsterFleeing == 0 {
		d := directionTo(m.X&^(blockUnits-1), m.Y&^(blockUnits-1), g.partyX&^(blockUnits-1), g.partyY&^(blockUnits-1))
		px, py := relativePosition(m.X, m.Y, g.partyX, g.partyY, d>>1)
		if py <= attackReach && abs(px) <= attackSideReach {
			if m.Direction != d || m.Facing != d>>1 {
				g.setMonsterDirection(m, d)
				g.noteBlockChanged(m.Block)
			} else {
				g.monsterAttack(m)
			}
			return
		}
	}
	if m.X == m.DestX && m.Y == m.DestY {
		g.setMonsterDirection(m, m.TargetDir)
		if dice(1, 100) > guardTurnChance {
			g.setMonsterMode(m, monsterModeGuard)
		} else {
			g.setMonsterMode(m, monsterModeStray)
		}
	} else {
		g.walkMonster(m)
	}
}

// walkToDestination (C: fn_0d6c_129f).
func (g *Game) walkToDestination(m *Monster) {
	if m.X != m.DestX || m.Y != m.DestY {
		g.walkMonster(m)
	} else if m.Direction != m.TargetDir {
		g.setMonsterDirection(m, turnTowards[m.Facing*4+m.TargetDir>>1]<<1)
	}
}

// thinkStill: a hit monster wakes the waiting group (C: fn_0d6c_12fc).
func (g *Game) thinkStill(m *Monster) {
	if m.Flags&monsterWasHit != 0 {
		for i := range g.monsters {
			if g.monsters[i].Mode == monsterModeAll {
				g.setMonsterMode(&g.monsters[i], monsterModeChase)
			}
		}
	} else if m.Mode == monsterModeAll {
		g.walkToDestination(m)
	}
}

// thinkDying plays the death frames, then removes the monster (C: fn_0d6c_1354).
func (g *Game) thinkDying(m *Monster) {
	m.Counter++
	if m.Counter > attackFrameCount {
		g.killMonster(m)
	}
	g.noteBlockChanged(m.Block)
}

// monsterFlee sends m running from the party for ticks thinks (C: fn_0d6c_1383).
func (g *Game) monsterFlee(m *Monster, ticks int) {
	m.Flags |= monsterFleeing
	m.Direction = directionTo(m.X, m.Y, g.partyX, g.partyY) ^ 4
	g.setMonsterMode(m, monsterModeFlee)
	m.Counter = int(int8(ticks))
}

// thinkGuard chases the party once it sees it or stands beside it (C: fn_0d6c_13c5).
func (g *Game) thinkGuard(m *Monster) {
	if g.canSeeParty(m) {
		g.setMonsterMode(m, monsterModeChase)
	}
	for d := range directionCount {
		if g.blockInDirection(m.Block, d) == g.currentBlock {
			g.setMonsterMode(m, monsterModeChase)
		}
	}
}

// thinkStray walks straight on; blocked, it turns and tries the other ways, guarding
// when there is none (C: fn_0d6c_1419).
func (g *Game) thinkStray(m *Monster) {
	if m.Counter == 0 {
		m.Direction &= 6
		nx, ny := nextStep(m.X, m.Y, m.Direction)
		if g.stepBlocked(nx, ny, m, wallStopsMonsters) == placeFree {
			g.moveMonster(m, nx, ny)
			return
		}
		m.Counter = strayTurns[dice(1, 2)]
		m.Direction = (m.Direction + m.Counter) & 6
		return
	}
	d := (m.Direction - m.Counter) & 6
	steps := m.ID // the original counted in a copy of the id byte and dropped it
	for i := range strayTries {
		nx, ny := nextStep(m.X, m.Y, d)
		if g.stepBlocked(nx, ny, m, wallStopsMonsters) == placeFree {
			g.moveMonster(m, nx, ny)
			g.setMonsterDirection(m, d)
			if i == 0 {
				steps++
				if steps > strayStraight {
					m.Counter = 0
				}
			}
			return
		}
		d = (d + m.Counter) & 6
	}
	g.setMonsterMode(m, monsterModeGuard)
}

// noLine is lineToBlock's answer when target is not in a straight free line.
const noLine = 5

// lineToBlock is dir when target lies within n blocks of b straight along dir, with no
// wall or monster between; noLine otherwise (C: fn_0d6c_156b).
func (g *Game) lineToBlock(b, dir, n, target int) int {
	if blockDistance(b, target) > n {
		return noLine
	}
	k := directionTo(b&columnMask, b>>5, target&columnMask, target>>5)
	if k&1 != 0 || dir<<1 != k {
		return noLine
	}
	if b&columnMask != target&columnMask && b&^columnMask != target&^columnMask {
		return noLine
	}
	for range n {
		b = g.blockInDirection(b, dir)
		if b == target {
			return dir
		}
		if g.lvl.wallFlags[g.lvl.blocks[b].Walls[dir^2]]&wallPassable != 0 {
			break
		}
		if g.lvl.blocks[b].Objects&monsterFlag != 0 {
			break
		}
	}
	return noLine
}

// Monster spells (C: MonType.spells): an item to throw, or a magic effect.
const (
	specialWallDoor   = 5 // a door a monster can open
	spellThrowsItem   = 1<<15 | 1<<14
	spellThrowsMagic  = 1 << 14
	spellItemMask     = 1<<14 - 1
	spellNotNear      = 1 << 13
	spellFreezes      = 4
	spellCallsHelp    = 3
	spellQuake        = 1
	throwPower        = 12
	throwFlags        = 1<<6 - 1
	quakeSound        = 147
	quakeDamage       = 20
	quakeKnockSlots   = 1<<4 - 1
	flyingItem        = 1
	flyingMagic       = 0
	damageFromNowhere = -1
)

// castMonsterSpell has m cast at the party in a straight line ahead, if it can now; it
// says whether it did (C: fn_0d6c_1630).
func (g *Game) castMonsterSpell(m *Monster) bool {
	if m.Casts == 0 {
		return false
	}
	if m.CastDelay > 0 {
		m.CastDelay--
		return false
	}
	dir := g.lineToBlock(m.Block, m.Facing, 4, g.currentBlock)
	if dir == noLine {
		return false
	}
	var s int
	if m.Type.flags&monsterRandomSpells != 0 {
		s = dice(1, m.Type.spellCount) - 1
	} else {
		s = m.NextSpell
		m.NextSpell++
		if m.NextSpell >= m.Type.spellCount {
			m.NextSpell = 0
		}
	}
	s = m.Type.spells[s]
	if s&spellThrowsItem != 0 {
		if blockDistance(m.Block, g.currentBlock) > 1 {
			kind := flyingItem
			if s&spellThrowsMagic != 0 {
				kind = flyingMagic
			}
			if it := g.newItem(s&spellItemMask, 0, 0); it != 0 {
				if !g.launchObject(kind, it, m.X, m.Y, throwPower, dir<<1, -1, m.ID|monsterFlag, throwFlags) {
					g.freeItem(it)
				}
			}
		}
	} else if s&spellNotNear == 0 {
		if blockDistance(m.Block, g.currentBlock) >= 2 {
			return false
		}
		switch s {
		case spellFreezes:
			g.freezeParty()
		case spellCallsHelp:
			for i := range g.monsters {
				if blockDistance(g.monsters[i].Block, m.Block) <= wakeAllRange {
					g.setMonsterMode(m, monsterModeChase) // sic: the caster, not the others
				}
			}
			g.printMessage(2, g.text(stringCallsHelp))
		case spellQuake:
			g.playSoundEffect(quakeSound)
			g.shakeScreen(10, 2, 2, 1)
			for i := range g.party {
				if g.party[i].Flags&charPresent != 0 {
					if it := g.takeCharacterItem(i, quakeKnockSlots, 0); it != 0 {
						g.dropItem(it, g.partyX, g.partyY, 0, true)
					}
					g.damageCharacter(i, quakeDamage, damageFromNowhere, 0, 2)
				}
			}
		default:
			return false
		}
	}
	if m.Casts != -1 {
		m.Casts--
	}
	m.CastDelay = int(int8(scale(attackPauseSteps, m.Type.fightingStats[attackFactor])))
	return true
}

// updateMonster lets m think, when its think delay is up (C: fn_0d6c_189f).
func (g *Game) updateMonster(m *Monster) {
	if m.Mode > lastThinkingMode {
		return
	}
	f := monsterModeFlags[m.Mode]
	m.Thinking++
	if m.Thinking-1 < m.Type.thinkDelay && f&modeEveryTick == 0 {
		return
	}
	m.Thinking = 0
	if m.Type.flags&monsterRegenerates != 0 {
		m.HitPoints = min(m.HitPoints+dice(1, healDice), m.Type.hitPoints)
	}
	if m.Flags&monsterFleeing != 0 {
		m.DestX, m.DestY = g.partyX, g.partyY
	}
	if (f&modeLooksOut != 0 && g.canSeeParty(m)) || (f&modeWakesWhenHit != 0 && m.Flags&monsterWasHit != 0) {
		g.setMonsterMode(m, monsterModeChase)
	}
	if m.Mode != monsterModeHit && m.Mode != monsterModeRemove && random.Byte()&3 == 0 {
		m.Shift = (m.Shift + 1) & monsterTicks
		g.noteBlockChanged(m.Block)
	}
	switch m.Mode {
	case monsterModeVanish:
		m.Counter++
		g.noteBlockChanged(m.Block)
		if m.Counter > deathFrames {
			g.runBlockScriptWith(monsterDeathFunction, allEvents, m.ID, m.ID, 0, 0)
		}
	case monsterModeStill, monsterModeAll:
		g.thinkStill(m)
	case monsterModeWalk:
		g.walkToDestination(m)
	case monsterModeGuard:
		g.thinkGuard(m)
	case monsterModeStray:
		g.thinkStray(m)
	case monsterModeRetreat:
		g.sceneReady = true
		m.Counter = int(int8(m.Counter - 1))
		if m.Counter > 0 && distance(g.partyX, g.partyY, m.X, m.Y) <= retreatReach && m.Flags&monsterFleeing == 0 {
			g.shuffleMonster(m)
			break
		}
		g.setMonsterMode(m, monsterModeChase)
	case monsterModeWait:
		m.Counter = int(int8(m.Counter - 1))
		if m.Counter <= 0 {
			g.setMonsterMode(m, monsterModeChase)
		}
	case monsterModeChase:
		if !g.castMonsterSpell(m) {
			g.chaseParty(m)
		}
		g.noteBlockChanged(m.Block)
	case monsterModeAttack:
		m.Counter++
		if m.Counter > attackFrameCount {
			g.setMonsterMode(m, monsterModeRetreat)
			pause := ratio(attackPauseSteps, m.Type.fightingStats[attackFactor])
			m.Counter = int(int8(scale(pause, difficultyPauses[g.difficulty()])))
		}
		g.noteBlockChanged(m.Block)
	case monsterModeFlee:
		m.Counter = int(int8(m.Counter - 1))
		if m.Counter == 0 {
			g.setMonsterMode(m, monsterModeChase)
			m.Flags &^= monsterFleeing
		} else {
			g.chaseParty(m)
		}
	case monsterDead:
		g.thinkDying(m)
	case monsterModeRemove:
		m.Damage = 0
		return
	}
	if m.Damage != 0 {
		if m.Damage&flagsHitMarker != 0 {
			m.Damage &^= flagsHitMarker
		} else {
			m.Damage = 0
		}
		g.noteBlockChanged(m.Block)
	}
	m.Flags &^= monsterWasHit
}

// moveMonsters runs every other monster, from the first or the second (C: fn_0bbc_000a;
// its switch g_206b was always on).
func (g *Game) moveMonsters(first int) {
	for i := first; i < maxMonsters; i += 2 {
		g.updateMonster(&g.monsters[i])
	}
}

// killMonster removes m, leaving its items and a mark on the automap (C: fn_070c_05fa).
func (g *Game) killMonster(m *Monster) {
	g.setMonsterMode(m, monsterModeRemove)
	g.dropMonsterItems(m)
	g.noteBlockChanged(m.Block)
	if g.blockEmptyNorth(m.Block) && m.Type.flags&monsterPassesMonsters == 0 {
		g.lvl.blocks[m.Block].Flags |= blockAutomapRight
	}
	g.moveMonster(m, 0, 0)
}

// blockEmptyNorth says whether block b's north wall shows nothing and it has no left
// automap mark (C: ovr_2d1b_005c).
func (g *Game) blockEmptyNorth(b int) bool {
	w := g.lvl.blocks[b].Walls[0]
	return g.lvl.wallVMP[w] == 0 && g.lvl.wallShape[w] == 0 && g.lvl.blocks[b].Flags&blockAutomapLeft == 0
}

// killMonstersOn removes the monsters on block b, as when a wall appears there, and
// says how many there were (C: fn_0845_155d).
func (g *Game) killMonstersOn(b int) int {
	n := 0
	for o := g.lvl.blocks[b].Objects; o != 0; {
		next := *g.objectNext(o)
		if o&monsterFlag != 0 {
			m := &g.monsters[o&monsterIndex]
			n++
			g.setMonsterMode(m, monsterModeRemove)
			g.noteBlockChanged(m.Block)
			g.moveMonster(m, 0, 0)
		}
		o = next
	}
	return n
}
