package game

import "github.com/memmaker/lands-of-lore-1-go/internal/clock"

// The game's timed events: each runs its function every interval ticks of a clock while
// enabled (C: struct Timer, g_2298 of s013; the main loop runs the due ones).
type timer struct {
	fn       func(g *Game, arg int)
	interval int
	next     int
	arg      int
	clock    int // which tick counter it counts
	enabled  bool
}

// playClockSlot is the counter slot of the clock that stops while the game waits on a
// menu (C: g_3858, counter 3, slot 2 of fn_231c_0008 and fn_231c_0048).
const playClockSlot = 2

// playClock is the counter the play-time timers count.
var playClock = clock.SlotCounter(playClockSlot)

// Timer numbers.
const (
	timerDoors = iota
	timerMonstersEven
	timerMonstersOdd
	timerCharacters
	timerFlying
	timerLevel5
	timerLevel6
	timerLevel7
	timerRegenerate
	timerFaces
	timerLampOil
	timerMessage
	timerCount
)

// newTimers is the timer table as the program starts (C: g_2298).
func newTimers() [timerCount]timer {
	return [timerCount]timer{
		timerDoors:        {fn: (*Game).moveDoors, interval: 15, clock: playClock, enabled: true},
		timerMonstersEven: {fn: (*Game).moveMonsters, interval: 6, clock: playClock, enabled: true},
		timerMonstersOdd:  {fn: (*Game).moveMonsters, interval: 6, next: 3, arg: 1, clock: playClock, enabled: true},
		timerCharacters:   {fn: (*Game).updateCharacters, interval: 15, clock: playClock, enabled: true},
		timerFlying:       {fn: (*Game).moveFlyingObjects, interval: 1, clock: playClock, enabled: true},
		timerLevel5:       {fn: (*Game).levelTimer, clock: playClock},
		timerLevel6:       {fn: (*Game).levelTimer, arg: 1, clock: playClock},
		timerLevel7:       {fn: (*Game).levelTimer, arg: 2, clock: playClock},
		timerRegenerate:   {fn: (*Game).regenerate, interval: 1200, clock: playClock, enabled: true},
		timerFaces:        {fn: (*Game).animateFaces, interval: 10, clock: playClock, enabled: true},
		timerLampOil:      {fn: (*Game).burnLampOil, interval: 360, clock: playClock, enabled: true},
		timerMessage:      {fn: (*Game).messageTimeout, interval: 360, clock: clock.GameClock},
	}
}

// enableTimer switches timer n on or off and restarts its interval (C: fn_0bbc_070e).
func (g *Game) enableTimer(n int, on bool) {
	t := &g.timers[n]
	t.enabled = on
	t.next = clock.Get(t.clock) + t.interval
}

// setTimer switches timer n and gives it an argument (C: fn_0bbc_06d1).
func (g *Game) setTimer(n, arg int, on bool) {
	t := &g.timers[n]
	t.enabled = on
	t.next = clock.Get(t.clock) + t.interval
	t.arg = arg
}

// moveFlyingObjects moves the thrown items and missiles (C: fn_0bbc_040b; it also
// printed each to a debug console).
func (g *Game) moveFlyingObjects(int) {
	for i := range g.flying {
		if g.flying[i].Active {
			g.moveFlyingObject(&g.flying[i])
		}
	}
}

// levelTimer runs the level script's timer function n (C: fn_0bbc_0472).
func (g *Game) levelTimer(n int) { g.runBlockScript(levelTimerFunction+n, allEvents) }

const levelTimerFunction = 1025 // (C: 0x401)

// Rings that speed up healing (C: item types 0xe3 and 0xe4).
const (
	itemTypeMagicRing   = 227
	itemTypeHealingRing = 228
	ringHealing         = 4
	magicRingShare      = 10
	drainShift          = 5
	charNoHealing       = 1 << 7
)

// regenerate gives every living character a point of magic and health; a drain takes
// magic instead (C: fn_0bbc_0488).
func (g *Game) regenerate(int) {
	for ch := range g.party {
		c := &g.party[ch]
		if c.Flags&charPresent == 0 {
			continue
		}
		hp, mp := 1, 1
		if c.Flags&charDead != 0 {
			hp, mp = 0, 0
		}
		if mp != 0 && g.carries(ch, itemTypeMagicRing) {
			mp = c.MaxMP / magicRingShare
		}
		if hp != 0 && g.carries(ch, itemTypeHealingRing) {
			hp = ringHealing
		}
		if g.drainMagic != 0 {
			c.MP -= c.MaxMP >> drainShift
		} else {
			c.MP += mp
		}
		c.MP = min(max(c.MP, 0), c.MaxMP)
		if c.Flags&charNoHealing == 0 {
			g.heal(ch, hp, false)
		}
		g.drawPortrait(ch)
	}
}

// carries says whether character ch has an item of type t (C: fn_0a21_0997).
func (g *Game) carries(ch, t int) bool {
	if ch < 0 || ch >= partySize || g.party[ch].Flags&charPresent == 0 {
		return false
	}
	for _, it := range g.party[ch].Items {
		if it != 0 && g.items[it].Type == t {
			return true
		}
	}
	return false
}

// heal gives character ch amount, at least 1, hit points; the dead only with force
// (C: fn_070c_0a8f).
func (g *Game) heal(ch, amount int, force bool) {
	c := &g.party[ch]
	if c.HP <= 0 && !force {
		return
	}
	c.HP = max(min(c.HP+max(amount, 1), c.MaxHP), 1)
	if c.HP != 0 && c.Flags&charDead != 0 {
		c.Flags &^= charDead
	}
}

// burnLampOil burns a unit of the lit lamp's oil (C: fn_0bbc_061e).
func (g *Game) burnLampOil(int) {
	if g.flag(flagHasLamp) && g.flag(flagLampLit) && g.brightness != 0 && g.lampEffect != 0 {
		g.lampEffect--
	}
}

// messageTimeout lets the message fade (C: fn_0bbc_094b).
func (g *Game) messageTimeout(int) {
	g.enableTimer(timerMessage, false)
	g.fadeMessage(0, false)
}

// Face animation: a face blinks (frame 1) now and then and opens its eyes again
// (C: fn_0bbc_058a). noBlink stops new blinks.
const (
	faceOpen     = 0
	faceBlinking = 1
	blinkDiceMin = 6
	blinkSides   = 12
)

func (g *Game) animateFaces(noBlink int) {
	for n := range g.party {
		c := &g.party[n]
		if c.Flags&charPresent == 0 || c.Flags&charDead != 0 || c.Face > faceBlinking {
			continue
		}
		if c.Face == faceBlinking {
			c.Face = faceOpen
			g.drawPortrait(n)
			c.AnimCountdown = dice(1, blinkSides) + blinkDiceMin
			continue
		}
		c.AnimCountdown--
		if c.AnimCountdown <= 0 && noBlink == 0 {
			c.Face = faceBlinking
			g.drawPortrait(n)
			g.setTimer(timerFaces, 0, true)
		}
	}
}

// startTimer switches timer n on with a new interval (C: fn_0bbc_07b1).
func (g *Game) startTimer(n, interval int) {
	t := &g.timers[n]
	t.enabled = true
	t.interval = interval
	t.next = clock.Get(t.clock) + t.interval
}
