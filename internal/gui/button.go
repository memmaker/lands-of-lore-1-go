// Package gui is the Westwood library's button lists (C: s118, s025): buttons in a
// window that react to the mouse and to keys, and draw themselves per state.
package gui

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// Button flags. The event bits say which events trigger a button: those in EventLeft*
// come from the left mouse button and the button's first key, those in EventRight*
// from the right mouse button and its second key.
const (
	FlagNeedsInside  = 1 << 0 // a release counts only over the button
	FlagSticky       = 1 << 2 // pressing it latches (state Down|Hover|Latched)
	FlagDisabled     = 1 << 3
	FlagDrawDisabled = 1 << 4 // disabled, it still draws, dithered
	FlagStopAfter    = 1 << 5 // after firing, later buttons see nothing
	FlagCapture      = 1 << 6 // once grabbed, keeps the mouse until released

	EventLeftPress    = 1 << 8
	EventLeftHeld     = 1 << 9
	EventLeftRelease  = 1 << 10
	EventLeftUp       = 1 << 11
	EventRightPress   = 1 << 12
	EventRightHeld    = 1 << 13
	EventRightRelease = 1 << 14
	EventRightUp      = 1 << 15

	leftEvents    = EventLeftPress | EventLeftHeld | EventLeftRelease | EventLeftUp
	rightEvents   = EventRightPress | EventRightHeld | EventRightRelease | EventRightUp
	pressEvents   = EventLeftPress | EventRightPress
	heldEvents    = EventLeftHeld | EventRightHeld
	upEvents      = EventLeftUp | EventRightUp
	releaseEvents = EventLeftRelease | EventRightRelease
	// pressOrHeld are the events that light a sticky button under the cursor.
	pressOrHeld = pressEvents | heldEvents
)

// Button states.
const (
	StateDown        = 1 << 0
	StateHover       = 1 << 1
	StateLatched     = 1 << 2
	statePrevDown    = 1 << 3
	statePrevLatched = 1 << 4 // after Release; Process keeps the previous StateHover here
	stateByKey       = 1 << 7
	stateBits        = 1<<8 - 1
	stateCurrent     = StateDown | StateHover
	statePrevious    = statePrevDown | statePrevLatched
	stateLit         = StateHover | StateLatched
	prevShift        = 3
)

// Draw types.
const (
	DrawNothing = 0
	DrawShape   = 1
	DrawText    = 2
	DrawSkip    = 3
	DrawFunc    = 4
	DrawOutline = 5
	DrawXorFill = 6
)

// Button is one button (C: struct Btn).
type Button struct {
	Next    *Button
	ID      int
	Key1    int    // a key that acts as the left button
	Key2    int    // a key that acts as the right button
	Types   [3]int // how it draws: normally, down, latched
	Flags   int
	Data    [3]any // per draw type: a shape ([]byte), a string, or a func(*Button)
	Window  int
	X, Y    int // negative: from the window's right or bottom edge
	W, H    int
	Colours [3][2]int
	Extra   int // (C: +2ch)
	State   int
	Func    func(*Button) bool // called when it fires; true stops the list

	// The game's buttons carry two more words (C: struct Btn of o2ceb, +38h and +3ah).
	Arg   int // what the callback acts on: a character, a slot, a direction
	Index int // the definition it was made from
}

// System holds the button system's state.
type System struct {
	Screen *gfx.Screen
	// Changed makes the next Process redraw the list (C: g_372d).
	Changed bool
	grabbed *Button // (C: g_2890)
	current *Button // the list Process saw last (C: g_2894)
	held    int     // mouse buttons held: EventLeftHeld, EventRightHeld (C: g_abd2)
	// DisabledColour is what disabled buttons are dithered with (C: g_370b).
	DisabledColour int
}

// New makes a button system drawing on a screen.
func New(s *gfx.Screen) *System { return &System{Screen: s} }

// Append adds b at the end of list and returns the list's head (C: fn_20bf_0006).
func (g *System) Append(list, b *Button) *Button {
	if b == nil {
		return list
	}
	b.Next = nil
	head := b
	if list != nil {
		head = list
		for list.Next != nil {
			list = list.Next
		}
		list.Next = b
	}
	g.Changed = true
	return head
}

// Insert adds b to list sorted by ID and returns the head (C: fn_20bf_0225).
func (g *System) Insert(list, b *Button) *Button {
	if list == nil {
		return b
	}
	if b == nil {
		return list
	}
	head, prev := list, list
	if uint16(b.ID) <= uint16(head.ID) {
		b.Next = list
		return b
	}
	for uint16(b.ID) > uint16(list.ID) && list.Next != nil {
		prev = list
		list = list.Next
	}
	if uint16(b.ID) > uint16(list.ID) {
		g.Append(head, b)
	} else {
		prev.Next = b
		b.Next = list
	}
	g.Changed = true
	return head
}

// Disable disables b and redraws it (C: fn_20bf_0073).
func (g *System) Disable(b *Button) *Button {
	if b != nil && b.Flags&FlagDisabled == 0 {
		b.Flags |= FlagDisabled
		g.Draw(b)
	}
	return b
}

// Enable enables b and redraws it (C: fn_20bf_00a8).
func (g *System) Enable(b *Button) *Button {
	if b != nil && b.Flags&FlagDisabled != 0 {
		b.Flags &^= FlagDisabled
		g.Draw(b)
	}
	return b
}

// Find returns the button with an ID (C: fn_20bf_00dd).
func Find(list *Button, id int) *Button {
	for ; list != nil; list = list.Next {
		if list.ID == id {
			break
		}
	}
	return list
}

func (b *Button) rememberPrevious() {
	if b.State&StateDown != 0 {
		b.State |= statePrevDown
	} else {
		b.State &^= statePrevDown
	}
}

// Release lets b go, calling its function if call (C: fn_20bf_010c).
func (g *System) Release(b *Button, call bool) *Button {
	if b == nil || b.Flags&FlagDisabled != 0 {
		return b
	}
	b.rememberPrevious()
	b.State &^= StateDown | StateHover
	if b.State&StateLatched != 0 {
		b.State |= statePrevLatched
	} else {
		b.State &^= statePrevLatched
	}
	b.State &^= StateLatched
	g.Draw(b)
	if call && b.Func != nil {
		b.Func(b)
	}
	return b
}

// Press holds b down, calling its function if call (C: fn_20bf_01b2).
func (g *System) Press(b *Button, call bool) *Button {
	if b == nil || b.Flags&FlagDisabled != 0 {
		return b
	}
	b.rememberPrevious()
	b.State |= StateDown
	g.Draw(b)
	if call && b.Func != nil {
		b.Func(b)
	}
	return b
}

// Enabled says whether b is enabled (C: fn_20bf_02f6).
func Enabled(b *Button) bool { return b != nil && b.Flags&FlagDisabled == 0 }

// Down says whether b is down (C: fn_20bf_0317).
func Down(b *Button) bool { return b != nil && b.State&StateDown != 0 }

// Hovered says whether b is lit (C: fn_20bf_033b).
func Hovered(b *Button) bool { return b != nil && b.State&StateHover != 0 }

// DrawList draws every button of a list (C: fn_20bf_035f).
func (g *System) DrawList(list *Button) {
	for ; list != nil; list = list.Next {
		g.Draw(list)
	}
}

// MarkChanged makes the next Process redraw its list (C: fn_163d_0004).
func (g *System) MarkChanged() { g.Changed = true }

// keyFromCharacter marks keys that came through the character reader.
const keyFromCharacter = 1 << 15

const (
	keyMask  = keyFromCharacter - 1
	lowByte  = 1<<8 - 1
	wordMask = 1<<16 - 1
)

// ProcessCharacters feeds the next waiting key, as a character, to the list
// (C: fn_163d_000f).
func (g *System) ProcessCharacters(list *Button) int {
	key := 0
	if input.CheckWithReleases() != 0 {
		key = input.WaitCharWithReleases() | keyFromCharacter
	}
	return g.Process(list, key)
}

// ProcessKeys feeds the next waiting key code to the list (C: fn_163d_0039).
func (g *System) ProcessKeys(list *Button) int {
	key := 0
	if input.Peek() != 0 {
		key = input.Get()
		if key == input.KeyMouseMoved {
			return 0
		}
	}
	return g.Process(list, key)
}

// Contains says whether screen point (x, y) is on button b, as Process tests it.
func (g *System) Contains(b *Button, x, y int) bool {
	bx, by := g.position(b)
	return !(x < bx || y < by || bx+b.W < x || by+b.H < y)
}

// position is a button's screen position.
func (g *System) position(b *Button) (x, y int) {
	w := g.Screen.Windows[b.Window]
	x, y = b.X, b.Y
	if x < 0 {
		x += w.W * gfx.ColumnWidth
	}
	if y < 0 {
		y += w.H
	}
	return x + w.Left(), y + w.Top()
}

// Character codes of the mouse buttons as the character reader returns them.
const (
	charMouseLeft  = input.KeyMouseLeft + 133
	charMouseRight = input.KeyMouseRight + 133
	releaseShift   = 2
)

// eventIndex picks the strongest of a button's triggered events: press, held, release,
// up (C: g_2898).
var eventIndex = [16]int{
	0, EventLeftPress, EventLeftHeld, EventLeftPress, EventLeftRelease, EventLeftPress,
	EventLeftRelease, EventLeftPress, EventLeftUp, EventLeftPress, EventLeftHeld,
	EventLeftPress, EventLeftRelease, EventLeftPress, EventLeftRelease, EventLeftPress,
}

// Process runs a key and the mouse through a button list: buttons light, latch and fire;
// it returns the fired button's ID with bit 15 set, else the key (C: fn_163d_0068).
func (g *System) Process(list *Button, key int) int {
	if list == nil {
		return key & keyMask
	}
	if list != g.current || g.Changed {
		g.grabbed = nil
		g.held = 0
		if input.Down(input.KeyMouseLeft) {
			g.held |= EventLeftHeld
		}
		if input.Down(input.KeyMouseRight) {
			g.held |= EventRightHeld
		}
		g.current = list
		g.Changed = false
		g.DrawList(list)
	}
	mx, my := g.Screen.Mouse()
	ev := 0
	if !input.NoMouse() {
		m := 0
		k := key & lowByte
		if key&keyFromCharacter != 0 {
			switch k {
			case charMouseRight:
				m = EventRightPress
			case charMouseLeft:
				m = EventLeftPress
			}
		} else {
			switch k {
			case input.KeyMouseRight:
				m = EventRightPress
			case input.KeyMouseLeft:
				m = EventLeftPress
			}
		}
		if key&input.Released != 0 {
			m <<= releaseShift
		}
		ev |= m
		if m != 0 {
			mx, my = input.Click()
		}
		g.held &^= (m & releaseEvents) >> 1
		g.held |= (m & pressEvents) << 1
		ev |= g.held
		ev |= (g.held << releaseShift) ^ upEvents
		ev &= wordMask
	}
	b := g.current
	if g.grabbed != nil {
		b = g.grabbed
	}
	if g.grabbed != nil && g.grabbed.Flags&FlagDisabled != 0 {
		g.grabbed = nil
	}
	result := 0
	clearDown := false
	for ; b != nil; b = b.Next {
		if b.Flags&FlagDisabled != 0 {
			continue
		}
		b.State &^= statePrevious
		b.State |= (b.State & stateCurrent) << prevShift
		x, y := g.position(b)
		inside := !(mx < x || my < y || x+b.W < mx || y+b.H < my)
		b.State &^= stateByKey
		if k := key & keyMask; k != 0 {
			matched := false
			switch {
			case b.Key1 == k:
				ev = b.Flags & leftEvents
				matched = true
			case b.Key2 == k:
				ev = b.Flags & rightEvents
				if ev == 0 {
					ev = b.Flags & leftEvents
				}
				matched = true
			}
			if matched {
				inside = true
				b.State |= stateByKey
				key = 0
				g.grabbed = b
			}
		}
		fire := false
		if !inside {
			b.State &^= stateLit
		}
		switch {
		case ev&pressOrHeld != 0 && b.Flags&FlagSticky != 0 && inside && (b == g.grabbed || g.grabbed == nil):
			b.State |= stateLit
			if g.grabbed == nil {
				g.grabbed = b
			}
		case ev&upEvents != 0 && b.Flags&FlagSticky == 0 && inside:
			b.State |= stateLit
		default:
			b.State &^= stateLit
		}
		hit := inside
		if g.grabbed != nil && g.grabbed.Flags&FlagCapture != 0 {
			hit = g.grabbed == b
		}
		if hit {
			if ev&pressEvents != 0 && inside && g.grabbed == nil {
				key = 0
				g.grabbed = b
			}
			if b.Flags&ev != 0 && (inside || b.Flags&FlagNeedsInside == 0) {
				t := ev & b.Flags
				t = (t&leftEvents | (t&rightEvents)>>4) >> 8
				switch eventIndex[t] {
				default:
					b.State ^= StateDown
					result = b.ID | keyFromCharacter
					fire = true
					if b.Flags&FlagSticky != 0 {
						b.State |= stateLit
					}
					g.grabbed = b
				case EventLeftRelease:
					if b.Flags&FlagNeedsInside == 0 || g.grabbed == b {
						b.State ^= StateDown
						result = b.ID | keyFromCharacter
						fire = true
					}
					if b.Flags&FlagSticky == 0 {
						b.State &^= stateLit
					}
				case EventLeftUp:
					if b.Flags&FlagSticky == 0 {
						b.State |= stateLit
					}
					if b.Flags&FlagNeedsInside == 0 {
						fire = true
					}
				case EventLeftHeld:
					if b.Flags&FlagSticky != 0 {
						b.State |= stateLit
					}
					if b.Flags&FlagNeedsInside == 0 {
						fire = true
					}
				}
			}
		}
		if ev&heldEvents != 0 && inside {
			b.State |= stateLit
			if b.Flags&FlagSticky == 0 && b.State&StateDown == 0 {
				clearDown = true
				b.State |= StateDown
			}
		}
		if b.State&stateByKey != 0 || ev&upEvents == upEvents {
			g.grabbed = nil
			if !inside || b.Flags&FlagSticky != 0 {
				b.State &^= stateLit
			}
		}
		if !inside && b == g.grabbed && b.Flags&FlagCapture == 0 {
			g.grabbed = nil
		}
		if b.State&statePrevious != (b.State&stateCurrent)<<prevShift {
			g.Draw(b)
		}
		if clearDown {
			b.State &^= StateDown
		}
		if fire {
			b.State &= stateBits
			b.State |= ev
			if (b.Func != nil && b.Func(b)) || b.Flags&FlagStopAfter != 0 {
				break
			}
		}
		if g.grabbed == b && b.Flags&FlagCapture != 0 {
			break
		}
	}
	if result == 0 {
		result = key & keyMask
	}
	return result
}

// Draw draws a button as its state wants (C: fn_163d_05cc).
func (g *System) Draw(b *Button) {
	if b == nil {
		return
	}
	s := g.Screen
	if b.Flags&FlagDisabled != 0 {
		if b.Flags&FlagDrawDisabled != 0 {
			g.drawDisabled(b)
		}
		return
	}
	look := 0
	switch b.State & (StateDown | StateLatched) {
	case StateLatched, StateLatched | StateDown:
		look = 2
	case StateDown:
		look = 1
	}
	kind, data := b.Types[look], b.Data[look]
	c1, c2 := b.Colours[look][0], b.Colours[look][1]
	x, y := g.position(b)
	w := s.Windows[b.Window]
	x0, y0 := x-w.Left(), y-w.Top()
	x2, y2 := x+b.W-1, y+b.H-1
	onScreen := s.Current == 0
	hide := func() {
		if onScreen {
			s.HideMouseIn(x, y, x2, y2)
		}
	}
	show := func() {
		if onScreen {
			s.ShowMouseIn()
		}
	}
	switch kind {
	case DrawShape:
		shape, _ := data.([]byte)
		hide()
		s.DrawShape(s.Current, shape, x0, y0, b.Window, gfx.ShapeWindowRelative, gfx.ShapeOptions{})
		show()
	case DrawText:
		text, _ := data.(string)
		hide()
		s.DrawText(text, x, y, c1, c2)
		show()
	case DrawFunc:
		if f, ok := data.(func(*Button)); ok && f != nil {
			f(b)
		}
	case DrawOutline:
		hide()
		s.DrawRectOutline(x, y, x2, y2, c1)
		show()
	case DrawXorFill:
		// The original XORed the rectangle with bytes from wherever its source register
		// pointed; no button of the game uses this type.
	}
}

// drawDisabled draws a disabled button's normal shape at its raw position and dithers it
// (C: fn_1db2_0003, which reads the button as a shape object: Data[0] at (X, Y) in window
// Window).
func (g *System) drawDisabled(b *Button) {
	s := g.Screen
	onScreen := s.Current == 0
	if onScreen {
		s.HideMouseIn(b.X, b.Y, b.X+b.W, b.Y+b.H)
	}
	shape, _ := b.Data[0].([]byte)
	s.DrawShape(s.Current, shape, b.X, b.Y, b.Window, 0, gfx.ShapeOptions{})
	s.DitherRect(b.X, b.Y, b.W, b.H, g.DisabledColour)
	if onScreen {
		s.ShowMouseIn()
	}
}
