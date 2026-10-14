package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// textMenu is a vertical list of choices in the current text window (C: struct at g_20dc).
type textMenu struct {
	x, y     int // hit area left edge in font widths, first line's offset from the window top
	w        int // hit area width in font widths
	n        int // entries shown
	selected int
	normal   int // colours
	bright   int
	unused   int
}

// The menus as the program starts (C: g_20dc); each game works on its own copy, whose
// main menu's entry count is set when it opens and whose selection it keeps.
var startTextMenus = []textMenu{{x: 1, y: 4, w: 12, n: 3, normal: 61, bright: 159}}

// The menu helpers' state (C: g_4b4e, g_4b50, g_0f88).
type menuState struct {
	spacing     int  // extra pixels between lines, from the last draw
	lastKey     int  // the key that did not choose an entry
	mouseHovers bool // the entry under the mouse is highlighted
}

// showAll is the entry mask that shows every entry.
const showAll = 1<<32 - 1

// Keys the menus act on (key numbers).
const (
	keyMenuUp        = 96
	keyMenuDown      = 98
	keyMenuHome      = 91
	keyMenuHomePad   = 101
	keyMenuEnd       = 93
	keyMenuEndPad    = 103
	keyMenuReturn    = 43
	keyMenuSpace     = 61
	keyMenuCentrePad = 97
)

// menuKeyBits is what of a key code the menu looks at: the number and the release bit.
const menuKeyBits = lowByte | input.Released

// flashes and flashTicks: a chosen entry blinks three times.
const (
	flashes    = 3
	flashTicks = 2
)

// menuEntry is the index of the n-th shown entry (C: fn_2d43_0000).
func menuEntry(n int, mask uint32, shift int) int {
	if mask == showAll {
		return n
	}
	i := 0
	for ; n != 0; i++ {
		if mask&(1<<(i+shift)) != 0 {
			n--
		}
	}
	for mask&(1<<(i+shift)) == 0 {
		i++
	}
	return i
}

// textWindow returns the current text window's rectangle (C: g_ad8d, g_ad8b, g_ad8f,
// g_ad91 as the text layer set them).
func (g *Game) textWindow() gfx.Window { return g.Screen.Windows[g.Text.Window] }

func (g *Game) menuOrigin(m *textMenu) (x, y int) {
	win := g.textWindow()
	return (win.X + win.W>>1) * gfx.ColumnWidth, win.Y + m.y
}

// drawMenu draws a menu's entries centred in the text window, step pixels apart beyond
// the font height (C: ovr_2d43_0020).
func (g *Game) drawMenu(menu int, items []string, mask uint32, shift, step int) {
	m := &g.textMenus[menu]
	x0, y0 := g.menuOrigin(m)
	selected := menuEntry(m.selected, mask, shift)
	g.Screen.HideMouse()
	for i := range m.n {
		k := menuEntry(i, mask, shift)
		y := y0 + i*g.Screen.Font.Height() + i*step
		colour := m.normal
		if k == selected && (g.menu.mouseHovers || input.NoMouse()) {
			colour = m.bright
		}
		g.printAt(items[k], x0, y, colour, 0, printCentred|printShadow)
	}
	g.menu.spacing = step
	g.Screen.ShowMouse()
	input.Flush()
}

// flashEntry blinks an entry (C: fn_2d43_004a).
func (g *Game) flashEntry(text string, x, y, normal, bright, background int) {
	for range flashes {
		g.Screen.HideMouse()
		g.printAt(text, x, y, bright, background, printCentred|printShadow)
		clock.Delay(flashTicks)
		g.printAt(text, x, y, normal, background, printCentred|printShadow)
		g.Screen.ShowMouse()
		clock.Delay(flashTicks)
	}
}

func inRect(x, y, x1, y1, x2, y2 int) bool { return x >= x1 && x <= x2 && y >= y1 && y <= y2 }

// pollMenu handles one key or mouse event for a menu drawn by drawMenu and returns the
// entry chosen, -1 for none yet (C: ovr_2d43_0025).
func (g *Game) pollMenu(menu int, items []string, mask uint32, shift int) int {
	s := g.Screen
	m := &g.textMenus[menu]
	last := m.n - 1
	current := m.selected % (last + 1)
	old := current
	result := -1
	lineHeight := s.Font.Height() + g.menu.spacing
	half := g.menu.spacing >> 1
	x0, y0 := g.menuOrigin(m)
	key := 0
	g.menu.lastKey = 0
	if input.Peek() != 0 {
		key = input.Get() & menuKeyBits
	}
	win := g.textWindow()
	x1 := win.Left() + m.x*s.Font.MaxWidth()
	y1 := win.Y + m.y - half
	x2 := x1 + m.w*s.Font.MaxWidth() - 1
	y2 := y1 + m.n*lineHeight - 1
	if !input.NoMouse() {
		mx, my := s.Mouse()
		if inRect(mx, my, x1, y1, x2, y2) && g.menu.mouseHovers {
			current = (my - y1) / lineHeight
		}
	}
	switch key {
	case keyMenuUp:
		if current--; current < 0 {
			current = last
		}
	case keyMenuDown:
		if current++; current > last {
			current = 0
		}
	case keyMenuHome, keyMenuHomePad:
		current = 0
	case keyMenuEnd, keyMenuEndPad:
		current = last
	case input.KeyMouseLeft, input.KeyMouseRight:
		cx, cy := input.Click()
		if !inRect(cx, cy, x1, y1, x2, y2) {
			g.menu.lastKey = key
			break
		}
		current = (cy - y1) / lineHeight
		result = current
	case keyMenuReturn, keyMenuSpace, keyMenuCentrePad:
		result = current
	default:
		typed := upper(byte(input.ASCII(key & lowByte)))
		for i := range m.n {
			if upper(firstByte(items[menuEntry(i, mask, shift)])) == typed {
				current, result = i, i
				break
			}
		}
		g.menu.lastKey = key
	}
	if current != old {
		s.HideMouse()
		g.printAt(items[menuEntry(old, mask, shift)], x0, y0+old*lineHeight, m.normal, 0, printCentred|printShadow)
		g.printAt(items[menuEntry(current, mask, shift)], x0, y0+current*lineHeight, m.bright, 0, printCentred|printShadow)
		s.ShowMouse()
	}
	if result != -1 {
		entry := menuEntry(result, mask, shift)
		s.HideMouse()
		g.flashEntry(items[entry], x0, y0+current*lineHeight, m.normal, m.bright, 0)
		s.ShowMouse()
		result = entry
	}
	m.selected = current
	return result
}

// lowByte is a key code's character byte.
const lowByte = 1<<8 - 1

// upper is C's toupper.
func upper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// firstByte is a string's first byte, 0 for an empty one as for a C string.
func firstByte(s string) byte {
	if s == "" {
		return 0
	}
	return s[0]
}
