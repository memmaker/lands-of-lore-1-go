// Package input is the Westwood library's keyboard and mouse-button input (C: s127, and
// the mouse handler's button part in s046). Keys travel as Westwood key codes: an IBM key
// position number (1-127) with modifier bits above it. The keyboard interrupt turned scan
// codes into such codes and queued them; mouse button changes queued the two mouse "keys"
// followed by the cursor position. Poll does both once per host tick, standing in for the
// interrupts.
//
// Left out: input recording and playback (the game only ever switches it off), the
// keypad driving the cursor (only without a mouse driver), and Caps Lock and Num Lock,
// which the host cannot read; they count as off.
package input

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// Modifier bits of a key code.
const (
	Shift    = 1 << 8
	Ctrl     = 1 << 9
	Alt      = 1 << 10
	Released = 1 << 11
	// Number takes the key number out of a key code.
	Number    = 1<<7 - 1
	modifiers = Shift | Ctrl | Alt
	lowByte   = 1<<8 - 1
)

// Key numbers the library itself treats specially.
const (
	KeyMouseMoved   = 45
	KeyMouseLeft    = 65
	KeyMouseRight   = 66
	keyMouseThird   = 67 // like the buttons, followed by a position (no device sends it)
	keyMouseFourth  = 68
	KeyEscape       = 110
	keyNone         = 127
	keyF1           = 112
	keyF10          = 121
	keyFirstNav     = 75  // Insert: first of the navigation and keypad keys
	keyLastNav      = 109 // last of them
	keyFirstSpecial = 62  // first key past the typing keys
	keyFirstSystem  = 65  // from here to the navigation keys: system "characters"
	keyNoASCII      = 122 // keys from here on type nothing
	keyExtra        = 128 // what scan code 122 becomes
	keyLeftShift    = 44
	keyRightShift   = 57
	keyLeftCtrl     = 58
	keyRightCtrl    = 64
	keyLeftAlt      = 60
	keyRightAlt     = 62
)

// Flags (C: g_35e6).
const (
	FlagRepeat        = 1 << 0  // a key held down queues again
	FlagNoRemap       = 1 << 1  // right Ctrl/Alt and the grey keys keep their own numbers
	FlagFlag4         = 1 << 4  // set at start, never read
	FlagNavAsIs       = 1 << 9  // the keypad types navigation codes even with Num Lock on
	FlagReleases      = 1 << 11 // key releases are queued too
	FlagKeypadMouse   = 1 << 12
	FlagKeypadButtons = 1 << 13
	initialFlags      = FlagRepeat | FlagFlag4
)

// The queue holds 128 words; a key takes one, a mouse button event three.
const (
	queueWords  = 128
	keyBits     = 128
	bitsPerByte = 8
)

type state struct {
	queue        [queueWords]int             // (C: cs:0b9h)
	head, tail   int                         // (C: cs:1b9h, cs:1bbh, as word indexes)
	down         [keyBits / bitsPerByte]byte // keys held (C: cs:23ch)
	flags        int
	buttons      int  // left 1, right 2 (C: g_2f45)
	clickX       int  // where the last button event taken happened (C: g_2f10)
	clickY       int  // (C: g_2f12)
	noMouse      bool // (C: g_2f43)
	screen       *gfx.Screen
	lastX, lastY int
	lastButtons  int
}

var in = state{flags: initialFlags}

// Init connects the input to the screen whose cursor the mouse moves and makes Poll run
// once per host tick.
func Init(s *gfx.Screen) {
	in = state{flags: initialFlags, screen: s}
	in.lastX, in.lastY = s.Mouse()
	host.AddTimer(Poll)
}

// SetNoMouse makes the game run without the mouse (C: g_2f43).
func SetNoMouse(off bool) { in.noMouse = off }

// NoMouse says whether the game runs without the mouse.
func NoMouse() bool { return in.noMouse }

// Buttons are the mouse buttons as the game last saw them: left 1, right 2 (C: g_2f45).
func Buttons() int { return in.buttons }

// Click is where the cursor was at the last button event taken from the queue
// (C: g_2f10, g_2f12).
func Click() (x, y int) { return in.clickX, in.clickY }

// SetFlags turns flags on and returns them all (C: fn_2169_04ba); turning on
// FlagReleases forgets which keys are held.
func SetFlags(f int) int {
	in.flags |= f
	if in.flags&FlagReleases != 0 {
		in.down = [len(in.down)]byte{}
	}
	return in.flags
}

// ClearFlags turns flags off and returns them all (C: fn_2169_04f7).
func ClearFlags(f int) int {
	in.flags &^= f
	return in.flags
}

func next(i int) int { return (i + 1) % queueWords }

func (s *state) push(word int) bool {
	n := next(s.tail)
	if n == s.head {
		return false
	}
	s.queue[s.tail] = word
	s.tail = n
	return true
}

func hasPosition(key int) bool {
	n := key & lowByte
	return n >= KeyMouseLeft && n <= keyMouseFourth
}

func isMouseButton(n int) bool { return n == KeyMouseLeft || n == KeyMouseRight }

// Put queues a key code, mouse events with the cursor's position, and keeps track of
// held keys; false if the queue was full (C: fn_2169_0945).
func Put(key int) bool {
	if key == 0 {
		return false
	}
	x, y := 0, 0
	if in.screen != nil {
		x, y = in.screen.Mouse()
	}
	saved := in.tail
	if !in.push(key) {
		return false
	}
	n := key & lowByte
	if n == KeyMouseMoved || isMouseButton(n) {
		if !in.push(x) || !in.push(y) {
			in.tail = saved
			return false
		}
	}
	isDown := true
	switch {
	case n == KeyMouseMoved || n == keyNone:
		in.tail = saved // only the held-key bit is kept
	case key&Released != 0:
		isDown = false
		if in.flags&FlagReleases == 0 && !isMouseButton(n) {
			in.tail = saved
		}
	}
	i, bit := (key&Number)/bitsPerByte, byte(1)<<(key%bitsPerByte)
	if isDown && in.down[i]&bit != 0 && in.flags&FlagRepeat == 0 {
		in.tail = saved
	}
	if isDown {
		in.down[i] |= bit
	} else {
		in.down[i] &^= bit
	}
	return true
}

func (s *state) held(n int) bool { return s.down[n/bitsPerByte]&(1<<(n%bitsPerByte)) != 0 }

// heldModifiers are the modifier bits the held keys make.
func (s *state) heldModifiers() int {
	m := 0
	if s.held(keyLeftCtrl) || s.held(keyRightCtrl) {
		m |= Ctrl
	}
	if s.held(keyLeftAlt) || s.held(keyRightAlt) {
		m |= Alt
	}
	if s.held(keyLeftShift) || s.held(keyRightShift) {
		m |= Shift
	}
	return m
}

// Ctrl+Alt+Del and Ctrl+Alt+keypad-dot went to the BIOS (a reboot) and were not queued.
const (
	ctrlAltDel   = Ctrl | Alt | 76
	ctrlAltPoint = Ctrl | Alt | 104
)

// scanned is the keyboard interrupt for one key event (C: fn_2169_0c35).
func scanned(e host.ScanEvent) {
	key := 0
	if e.Released {
		key |= Released
	}
	var n int
	switch {
	case e.Extended:
		i := indexOf(extendedScans[:], int(e.Scan))
		if i < 0 {
			return
		}
		n = extendedKeys[i]
	case e.Scan == scanExtra:
		n = keyExtra
	default:
		n = scanKeys[e.Scan&Number]
	}
	key |= n
	if in.held(keyRightCtrl) || in.held(keyLeftCtrl) {
		key |= Ctrl
	}
	if in.held(keyLeftAlt) || in.held(keyRightAlt) {
		key |= Alt
	}
	key = remap(key)
	if in.held(keyLeftShift) || in.held(keyRightShift) {
		key |= Shift
	}
	if key == ctrlAltDel || key == ctrlAltPoint {
		return
	}
	Put(key)
}

// scanExtra is the scan code the interrupt maps to keyExtra.
const scanExtra = 122

func indexOf(table []int, v int) int {
	for i, t := range table {
		if t == v {
			return i
		}
	}
	return -1
}

// remap gives right Ctrl and Alt and the grey keys the numbers of their left and keypad
// twins, unless FlagNoRemap (C: the routine at 2169:048e).
func remap(key int) int {
	if in.flags&FlagNoRemap != 0 {
		return key
	}
	// the last entry, 0, is never matched: the original's search counted it as missing
	if i := indexOf(grey[:len(grey)-1], key&lowByte); i >= 0 {
		return key&^lowByte | twins[i]
	}
	return key
}

// Poll runs the keyboard and mouse interrupts for what happened since the last tick.
func Poll() {
	for _, e := range host.TakeScanEvents() {
		scanned(e)
	}
	if in.screen == nil {
		return
	}
	buttons := 0
	if host.MouseLeft {
		buttons |= 1
	}
	if host.MouseRight {
		buttons |= 2
	}
	x, y := host.MouseX, host.MouseY
	if x == in.lastX && y == in.lastY && buttons == in.lastButtons {
		return
	}
	in.lastX, in.lastY, in.lastButtons = x, y, buttons
	mouseEvent(x, y, buttons)
}

// Mouse button bits.
const (
	ButtonLeft  = 1 << 0
	ButtonRight = 1 << 1
)

// mouseEvent is the mouse driver's call into the game (C: fn_1c24_034f): buttons first,
// queued at the old position, then the cursor moves.
func mouseEvent(x, y, buttons int) {
	if in.noMouse {
		return
	}
	changed := buttons ^ in.buttons
	if changed == 0 {
		Put(KeyMouseMoved)
	} else {
		in.buttons = buttons
		if changed&ButtonRight != 0 {
			Put(buttonKey(KeyMouseRight, buttons&ButtonRight != 0))
		}
		if changed&ButtonLeft != 0 {
			Put(buttonKey(KeyMouseLeft, buttons&ButtonLeft != 0))
		}
	}
	in.screen.MouseMoved(x, y)
}

func buttonKey(n int, pressed bool) int {
	if pressed {
		return n
	}
	return n | Released
}
