package input

import "github.com/memmaker/lands-of-lore-1-go/internal/host"

// ASCII is the character a key code types, or one of the library's codes above 127 for
// keys that type none; 0 for releases (C: fn_2169_0276, fn_2169_0482).
func ASCII(key int) int {
	if key&Released != 0 || key < 0 {
		return 0
	}
	n := key & lowByte
	c := 0
	switch {
	case n == KeyEscape:
		return escapeChar
	case n < keyFirstSpecial:
		i := n & typingKeyMask
		if key&Shift != 0 {
			c = shiftedASCII[i]
		} else {
			c = plainASCII[i]
		}
		if key&Ctrl != 0 && letterKeys[i/bitsPerByte]&(1<<(i%bitsPerByte)) != 0 {
			c &= controlMask
		}
	case n >= keyFirstNav && n < KeyEscape:
		// Num Lock counts as off: the navigation codes (C: cs:44bh)
		c = navASCII[n-keyFirstNav]
	case n >= keyF1 && n <= keyF10:
		base := functionPlain
		switch {
		case key&Alt != 0:
			base = functionAlt
		case key&Ctrl != 0:
			base = functionCtrl
		case key&Shift != 0:
			base = functionShift
		}
		c = base - (n - keyF1)
	case n >= keyFirstSystem && n < keyFirstNav:
		c = n + systemOffset
	default:
		c = n | systemBit
	}
	return key&^lowByte | c&lowByte
}

const (
	escapeChar    = 27
	typingKeyMask = 1<<6 - 1
	controlMask   = 1<<5 - 1
	systemBit     = 1 << 7
	systemOffset  = 133
	functionPlain = 197
	functionShift = 172
	functionCtrl  = 162
	functionAlt   = 152
)

// typesNothing says whether a queued key is one the character readers skip: the
// modifier and lock keys, releases, and keys from keyNoASCII on.
func typesNothing(key int) bool {
	n := key & lowByte
	return indexOf(silentKeys[:len(silentKeys)-1], n) >= 0 || n >= keyNoASCII
}

// entryWords is how many queue words the entry at i takes.
func entryWords(key int) int {
	if hasPosition(key) {
		return 3
	}
	return 1
}

// Check returns the character of the first key waiting that types one, leaving it
// queued; keys before it that type nothing are dropped; 0 if there is none
// (C: fn_2169_0516).
func Check() int {
	for in.head != in.tail {
		key := in.queue[in.head]
		if !typesNothing(key) && key&Released == 0 {
			return ASCII(key) & lowByte
		}
		in.head = (in.head + entryWords(key)) % queueWords
	}
	return 0
}

// CheckWithReleases is Check that also returns releases, as the character with the
// modifier and release bits (C: fn_2169_0593).
func CheckWithReleases() int {
	for in.head != in.tail {
		key := in.queue[in.head]
		if key&lowByte != 0 && !typesNothing(key) {
			return ASCII(key&^Released) | key&Released
		}
		in.head = next(in.head) // the original skips one word, even for a mouse event
	}
	return 0
}

// Peek returns the first queued key code without taking it, 0 if none (C: fn_2169_060c).
func Peek() int {
	if in.head == in.tail {
		return 0
	}
	return in.queue[in.head]
}

// Flush drops everything queued (C: fn_2169_062e).
func Flush() { in.head = in.tail }

// take removes the first queued entry; a mouse button event sets the click position
// (C: the routine at 2169:063c).
func take() int {
	key := in.queue[in.head]
	in.head = next(in.head)
	n := key & lowByte
	if hasPosition(key) {
		x, y := in.queue[in.head], in.queue[next(in.head)]
		in.head = next(next(in.head))
		if isMouseButton(n) {
			in.clickX, in.clickY = x, y
		}
	}
	return key
}

func waitQueued() {
	for in.head == in.tail {
		host.Yield()
	}
}

// Get waits for a key code and takes it (C: fn_2169_0790).
func Get() int {
	waitQueued()
	return take()
}

// WaitChar waits for a key that types a character and returns it (C: fn_2169_06e9).
func WaitChar() int {
	for {
		waitQueued()
		key := take()
		if !typesNothing(key) && key&Released == 0 {
			return ASCII(key) & lowByte
		}
	}
}

// WaitCharWithReleases is WaitChar that also returns releases, with the modifier and
// release bits (C: fn_2169_073a).
func WaitCharWithReleases() int {
	for {
		waitQueued()
		key := take()
		if !typesNothing(key) {
			return ASCII(key&^Released) | key&Released
		}
	}
}

// Down says whether a key is held (C: fn_2169_07ae).
func Down(key int) bool {
	n := remap(key) & Number
	return in.held(n)
}

// DownWith says whether a key is held with exactly the given modifiers; a release bit
// in key never matches (C: fn_2169_07df).
func DownWith(key int) bool {
	return Down(key) && in.heldModifiers() == key&(lowByte<<8)
}
