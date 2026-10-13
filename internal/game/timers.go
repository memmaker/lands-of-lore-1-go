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
	timerLamp
	timerFaces
	timerUpdates
	timerOrbs
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
		timerLamp:         {fn: (*Game).burnLamp, interval: 1200, clock: playClock, enabled: true},
		timerFaces:        {fn: (*Game).animateFaces, interval: 10, clock: playClock, enabled: true},
		timerUpdates:      {fn: (*Game).timerUpdates, interval: 360, clock: playClock, enabled: true},
		timerOrbs:         {fn: (*Game).timerOrbs, interval: 360, clock: clock.GameClock},
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

// Timer functions not ported yet.

func (g *Game) moveDoors(int)         { notPorted("fn_0bbc_004a (doors timer)") }
func (g *Game) moveMonsters(int)      { notPorted("fn_0bbc_000a (monsters timer)") }
func (g *Game) updateCharacters(int)  { notPorted("fn_0bbc_0143 (characters timer)") }
func (g *Game) moveFlyingObjects(int) { notPorted("fn_0bbc_040b (flying objects timer)") }
func (g *Game) levelTimer(int)        { notPorted("fn_0bbc_0472 (level script timer)") }
func (g *Game) burnLamp(int)          { notPorted("fn_0bbc_0488 (lamp timer)") }
func (g *Game) timerUpdates(int)      { notPorted("fn_0bbc_061e (update timer)") }
func (g *Game) timerOrbs(int)         { notPorted("fn_0bbc_094b (orbs timer)") }

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
