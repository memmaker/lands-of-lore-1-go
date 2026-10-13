// Package random holds the game's two random number generators: the Westwood library's
// shift register (C: fn_2313_0000) and the C library's rand (Borland's linear
// congruential generator) with the library's range helper (C: fn_2164_000d).
package random

// The shift register's three bytes (C: g_3842; the fourth byte is never read).
var register = [3]byte{initialLow, initialMid, initialHigh}

const (
	initialLow  = 118
	initialMid  = 152
	initialHigh = 52
)

// Seeded values the original stores before recording or playing back input
// (C: o2df4, g_3842 = 0x12344321).
const (
	replayLow  = 33
	replayMid  = 67
	replayHigh = 52
	replaySeed = 4660
)

// Byte steps the shift register and returns a byte of it (C: fn_2313_0000).
func Byte() int {
	low, mid, high := register[0], register[1], register[2]
	in := low >> 1 & 1 // bit 1 of the low byte, shifted in at the top byte's bottom
	carry := high >> 7
	high = high<<1 | in
	nextCarry := mid >> 7
	mid = mid<<1 | carry
	// sbb: (low>>2) - low - !carry, of which bit 0 enters the low byte from above
	diff := low>>2 - low - (1 - nextCarry)
	low = low>>1 | (diff&1)<<7
	register = [3]byte{low, mid, high}
	return int(low ^ mid)
}

// randState is the C library's seed (Borland: starts at 1).
var randState uint32 = 1

const (
	randMultiplier = 22695477
	randIncrement  = 1
	// RandMax is one more than rand's largest result.
	RandMax = 1 << 15
)

// Seed seeds rand (C: srand).
func Seed(n int) { randState = uint32(n) }

// Rand is the C library's rand: 0..RandMax-1.
func Rand() int {
	randState = randState*randMultiplier + randIncrement
	return int(randState>>16) & (RandMax - 1)
}

// SeedForReplay sets both generators as the original does before recording or
// playing back input (C: o2df4).
func SeedForReplay() {
	Seed(replaySeed)
	register = [3]byte{replayLow, replayMid, replayHigh}
}

// Range returns a number from lo to hi, either way round (C: fn_2164_000d).
func Range(lo, hi int) int {
	lo, hi = lo&wordMask, hi&wordMask
	if lo > hi {
		lo, hi = hi, lo
	}
	for {
		span := (hi - (lo - 1)) & wordMask
		r := int(int16(Rand()*span/RandMax)) + lo
		if r&wordMask <= hi {
			return r & wordMask
		}
	}
}

const wordMask = 1<<16 - 1
