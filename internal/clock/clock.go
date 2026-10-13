// Package clock is the game's timer: the Westwood library's tick counter, which the
// original's timer interrupt advanced 60 times a second (C: g_384c, set up by
// ovr_2e13_0057 at 60 Hz).
package clock

import "github.com/memmaker/lands-of-lore-1-go/internal/host"

// Rate is how many ticks pass per second.
const Rate = 60

// startTicks is where the counter starts: on DOS start-up took a while, and waits that
// subtract from the counter must not see it below their delay.
const startTicks = Rate

// Ticks is the current tick count (C: g_384c).
func Ticks() int {
	return startTicks + int(host.Ticks)*Rate/host.TicksPerSecond
}

// WaitUntil lets time pass until the counter reaches t, running idle each pass.
func WaitUntil(t int, idle func()) {
	for Ticks() < t {
		if idle != nil {
			idle()
		}
		host.Yield()
	}
}

// Delay is the library's busy wait (C: fn_2120_0007): each of its n passes waits until
// the counter is past the next tick, so it takes about two ticks a pass.
func Delay(n int) {
	for ; n > 0; n-- {
		WaitUntil(Ticks()+2, nil)
	}
}
