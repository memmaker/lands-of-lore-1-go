// Package clock is the game's timer: the Westwood library's sixteen tick counters, which
// the original's timer interrupt stepped 60 times a second (C: fn_2318_0002, installed by
// ovr_2db6_0020 at 60 Hz). Counter 0 is the game clock (C: g_384c), counter 1 a countdown
// the game sets and waits on (C: g_3850); the others are slots the game enables by number.
package clock

import "github.com/memmaker/lands-of-lore-1-go/internal/host"

// Rate is how many ticks pass per second.
const Rate = 60

// Counters is how many tick counters there are.
const Counters = 16

// The counters the game uses by name.
const (
	GameClock = 0
	Countdown = 1
)

// The slot numbers SetEnabled and friends accept (the original refused others); slot n
// is counter n+1.
const (
	firstSlot = 1
	lastSlot  = 14
)

// startTicks is where the game clock starts: on DOS start-up took a while, and waits that
// subtract from the clock must not see it below their delay.
const startTicks = Rate

type state struct {
	counters [Counters]int // (C: g_384c[16])
	enabled  uint16        // counters that run (C: g_384a)
	down     uint16        // counters that count down to 0 instead of up (C: g_3848)
	passes   int           // interrupts run (C: g_3846)
	stepped  int           // 60 Hz ticks already applied
	hostAt   uint64        // host.Ticks when the clock was (re)started
}

var clk = newState()

func newState() state {
	s := state{enabled: 1<<GameClock | 1<<Countdown, down: 1 << Countdown}
	s.counters[GameClock] = startTicks
	return s
}

// Reset puts the counters back as at program start, counting from the host's now.
func Reset() {
	clk = newState()
	clk.hostAt = host.Ticks
}

// catchUp runs the interrupt for every 60 Hz tick the host clock has passed.
func catchUp() {
	due := int((host.Ticks - clk.hostAt) * Rate / host.TicksPerSecond)
	for ; clk.stepped < due; clk.stepped++ {
		tick()
	}
}

// tick is one timer interrupt (C: fn_2318_0002).
func tick() {
	for i := range Counters {
		if clk.enabled>>i == 0 {
			break
		}
		if clk.enabled&(1<<i) == 0 {
			continue
		}
		if clk.down&(1<<i) == 0 {
			clk.counters[i]++
		} else if clk.counters[i] != 0 {
			clk.counters[i]--
		}
	}
	clk.passes++
}

// Ticks is the game clock (C: g_384c).
func Ticks() int { return Get(GameClock) }

// Get reads counter i.
func Get(i int) int {
	catchUp()
	return clk.counters[i]
}

// Set stores n into counter i.
func Set(i, n int) {
	catchUp()
	clk.counters[i] = n
}

func slotBit(slot int) (uint16, bool) {
	if slot < firstSlot || slot > lastSlot {
		return 0, false
	}
	return 1 << (slot + 1), true
}

// Enable starts slot's counter and says whether it ran already (C: fn_231c_0008).
func Enable(slot int) bool {
	bit, ok := slotBit(slot)
	if !ok {
		return false
	}
	catchUp()
	old := clk.enabled&bit != 0
	clk.enabled |= bit
	return old
}

// Disable stops slot's counter and says whether it ran (C: fn_231c_0048).
func Disable(slot int) bool {
	bit, ok := slotBit(slot)
	if !ok {
		return false
	}
	catchUp()
	old := clk.enabled&bit != 0
	clk.enabled &^= bit
	return old
}

// Enabled says whether slot's counter runs (C: fn_231c_008a).
func Enabled(slot int) bool {
	bit, ok := slotBit(slot)
	return ok && clk.enabled&bit != 0
}

// SetCountingDown makes slot's counter count down (or up) and says whether it counted
// down before (C: fn_231c_00b6).
func SetCountingDown(slot int, down bool) bool {
	bit, ok := slotBit(slot)
	if !ok {
		return false
	}
	catchUp()
	old := clk.down&bit != 0
	if down {
		clk.down |= bit
	} else {
		clk.down &^= bit
	}
	return old
}

// SlotCounter is the counter a slot number drives.
func SlotCounter(slot int) int { return slot + 1 }

// WaitUntil lets time pass until the game clock reaches t, running idle each pass.
func WaitUntil(t int, idle func()) {
	for Ticks() < t {
		if idle != nil {
			idle()
		}
		host.Yield()
	}
}

// WaitCountdown sets the countdown to n ticks and waits for it to run out, as the
// original's busy loops on g_3850 do.
func WaitCountdown(n int) {
	Set(Countdown, n)
	for Get(Countdown) != 0 {
		host.Yield()
	}
}

// Delay is the library's busy wait (C: fn_2120_0007): each of its n passes waits until
// the clock is past the next tick, so it takes about two ticks a pass.
func Delay(n int) {
	for ; n > 0; n-- {
		WaitUntil(Ticks()+2, nil)
	}
}
