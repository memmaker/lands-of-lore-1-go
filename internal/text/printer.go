// Package text is the game's text window printer (C: s022): formatted output into one of
// the screen's windows with word wrap, tabs, colour codes, scrolling and a <MORE> prompt.
package text

import (
	"strconv"

	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// Control codes in printed strings.
const (
	CodeMore       = 1  // show the <MORE> prompt now
	CodeBackground = 2  // the next byte is the new background colour
	CodeForeground = 6  // the next byte is the new foreground colour
	CodeTab        = 9  // to the next tab stop
	CodeClear      = 12 // clear the window
	CodeNewline    = 13
	CodeClearTab   = 19 // to the next tab stop, clearing on the way
	CodeColumn     = 24 // the next byte is the character column to go to
	CodeRow        = 25 // the next byte is the row to go to
	CodeHigh       = 27 // the next byte plus 127 is the character
	CodeLiteral    = 29 // the next byte is the character, even a control code
)

const (
	tabStop     = 8
	highOffset  = 127
	space       = ' '
	defaultMore = "<MORE>"
	// defaultMoreColumn is how many character cells from the right edge the prompt sits.
	defaultMoreColumn = 7
)

// MoreFunc shows the <MORE> prompt and waits (C: the function at g_370d).
type MoreFunc func(p *Printer, prompt string, column, foreground, background int)

// Printer is the printer's state (C: the globals of s022 and the current-window copies
// g_ad8b..g_ad95, g_3711..g_371b).
type Printer struct {
	Screen *gfx.Screen

	Window     int // (C: g_3715)
	X, Y, W, H int // the window; X and W in 8-pixel columns (C: g_ad8d, g_ad8b, g_ad8f, g_ad91)
	Foreground int // (C: g_3713)
	Background int // (C: g_3711)
	Column     int // pen position in pixels from the window's left (C: g_ad95)
	Row        int // (C: g_ad93)
	Rows       int // text rows that fit (C: g_3717)
	Columns    int // character cells per row (C: g_3719)
	PixelWidth int // (C: g_371b)
	Cell       int // character cell index of the pen, for tabs (C: g_ab49)
	Printed    int // characters printed since the last <MORE> (C: g_2172)

	MoreEnabled    bool // (C: g_3644)
	More           MoreFunc
	morePrompt     string
	moreColumn     int  // (C: g_284f)
	moreFg         int  // (C: g_2851)
	moreBg         int  // (C: g_2853)
	linesSinceMore int  // (C: g_284c)
	wrapped        bool // a word was just moved whole to the next line (C: g_284e)

	line      []byte // the word-wrap buffer (C: g_ab4f, g_ab4d)
	lineWidth int    // (C: g_ab4b)

	// the input stream: an argument being spliced in, the format, and a lookahead byte
	arg       []byte // (C: g_ab41)
	argActive bool
	format    []byte // (C: g_ab45)
	ahead     byte   // (C: g_ab3f[0])
	afterArg  byte   // (C: g_ab3e)
}

// New makes a printer for a screen, with the library's default <MORE> prompt.
func New(s *gfx.Screen) *Printer {
	p := &Printer{Screen: s, More: DefaultMore}
	p.SetMorePrompt("", 0, 0, 0)
	return p
}

func (p *Printer) fontHeight() int {
	if p.Screen.Font == nil {
		return 0
	}
	return p.Screen.Font.Height()
}

// cellWidth is a character cell: the widest glyph and the spacing.
func (p *Printer) cellWidth() int {
	w := p.Screen.CharSpacing
	if p.Screen.Font != nil {
		w += p.Screen.Font.MaxWidth()
	}
	return w
}

func (p *Printer) lineHeight() int { return p.fontHeight() + p.Screen.LineSpacing }

// div is the original's unsigned division, which a zero divisor would have crashed.
func div(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

// SetFont makes f the current font, recomputing the window's rows and columns as the
// library did, without line spacing (C: fn_1f2d_0002).
func (p *Printer) SetFont(f *gfx.Font) *gfx.Font {
	old := p.Screen.SetFont(f)
	if f != nil {
		p.Rows = div(p.H, f.Height())
		p.PixelWidth = p.W * gfx.ColumnWidth
		p.Columns = div(p.PixelWidth, f.MaxWidth())
	}
	return old
}

// SetMorePrompt sets what the <MORE> prompt shows; an empty prompt restores the default
// (C: fn_14b3_00c8).
func (p *Printer) SetMorePrompt(prompt string, column, foreground, background int) {
	if prompt == "" {
		prompt, column, foreground, background = defaultMore, defaultMoreColumn, 0, 0
	}
	p.morePrompt, p.moreColumn, p.moreFg, p.moreBg = prompt, column, foreground, background
}

// EnableMore makes a full window stop at a <MORE> prompt (C: fn_14b3_0108).
func (p *Printer) EnableMore() {
	p.MoreEnabled = true
	p.linesSinceMore = 0
}

// DisableMore lets text scroll on (C: fn_14b3_0118).
func (p *Printer) DisableMore() { p.MoreEnabled = false }

// SelectWindow makes window n the printer's and returns the previous one
// (C: fn_14b3_0122).
func (p *Printer) SelectWindow(n int) int {
	old := p.Window
	p.Window = n
	w := p.Screen.Windows[n]
	p.X, p.Y, p.W, p.H = w.X, w.Y, w.W, w.H
	p.Foreground, p.Background = w.Foreground, w.Background
	p.Column, p.Row = w.Column, w.Row
	p.linesSinceMore = 0
	p.Cell = div(p.Column, p.cellWidth())
	p.Rows = div(p.H-p.Screen.LineSpacing, p.lineHeight())
	p.PixelWidth = p.W * gfx.ColumnWidth
	p.Columns = div(p.PixelWidth, p.cellWidth())
	return old
}

// SelectAndClear selects window n and clears it (C: fn_14b3_01fa).
func (p *Printer) SelectAndClear(n int) int {
	old := p.SelectWindow(n)
	p.Clear()
	return old
}

// Clear fills the window with the background and puts the pen at the top left
// (C: fn_14b3_0215).
func (p *Printer) Clear() {
	left := p.X * gfx.ColumnWidth
	right := (p.X + p.W) * gfx.ColumnWidth
	p.Screen.FillRect(left, p.Y, right-1, p.Y+p.H-1, p.Background)
	p.Column, p.Cell, p.Row = 0, 0, 0
	p.linesSinceMore = 0
}

// PrintNumber prints a number (C: fn_14b3_0270).
func (p *Printer) PrintNumber(n int) { p.Printf("%d", n) }

// onScreen says whether the printer draws to the visible page, where the cursor must
// keep out of the way.
func (p *Printer) onScreen() bool { return p.Screen.Current == 0 }

func (p *Printer) hideMouse() {
	if !p.onScreen() {
		return
	}
	w := p.Screen.Windows[p.Window]
	p.Screen.HideMouseIn(w.Left(), w.Top(), w.Right()-1, w.Bottom()-1) // C: fn_234e_0003
}

func (p *Printer) showMouse() {
	if p.onScreen() {
		p.Screen.ShowMouseIn() // C: fn_234e_0062
	}
}

func (p *Printer) callMore() {
	if p.More == nil {
		return
	}
	p.showMouse()
	p.More(p, p.morePrompt, p.moreColumn, p.moreFg, p.moreBg)
	p.hideMouse()
	p.Printed = 0
}

// fill reads one byte of input into dst: from the spliced argument, else the format
// (C: fn_14b3_08e1).
func (p *Printer) fill(dst *byte) {
	c := byte(0)
	if p.argActive {
		if len(p.arg) == 0 {
			p.argActive = false
			c = p.afterArg
		} else {
			c, p.arg = p.arg[0], p.arg[1:]
		}
	}
	if c == 0 && p.format != nil {
		if len(p.format) == 0 {
			p.format = nil
		} else {
			c, p.format = p.format[0], p.format[1:]
		}
	}
	*dst = c
}

// read takes the next input byte, keeping one byte of lookahead (C: fn_14b3_095d).
func (p *Printer) read() byte {
	if p.ahead == 0 {
		p.fill(&p.ahead)
	}
	c := p.ahead
	p.ahead = 0
	p.fill(&p.ahead)
	return c
}

// splice makes the input continue with s before the rest of the format.
func (p *Printer) splice(s []byte) {
	p.arg, p.argActive = s, true
	p.afterArg = 0
	p.ahead = 0
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// Printf prints into the window; %s and %d take strings and numbers from args, the
// control codes above steer the pen (C: fn_14b3_0284).
func (p *Printer) Printf(format string, args ...any) {
	p.wrapped = false
	p.line = p.line[:0]
	p.lineWidth = 0
	p.ahead = 0
	p.format = []byte(format)
	p.argActive = false
	savedFg, savedBg := p.Foreground, p.Background
	p.hideMouse()
	next := 0
	for {
		c := p.read()
		if c == 0 {
			break
		}
		if !p.argActive && c == '%' {
			switch lower(p.ahead) {
			case 's':
				if next < len(args) {
					s, _ := args[next].(string)
					next++
					p.splice([]byte(s))
					c = p.read()
				}
			case 'd':
				n := 0
				if next < len(args) {
					n, _ = args[next].(int)
					next++
				}
				p.splice([]byte(strconv.Itoa(int(int16(n)))))
				c = p.read()
			}
		}
		p.control(c)
	}
	if len(p.line) != 0 {
		p.flushLine()
	}
	w := &p.Screen.Windows[p.Window]
	w.Column, w.Row = p.Column, p.Row
	p.Foreground, p.Background = savedFg, savedBg
	w.Foreground, w.Background = p.Foreground, p.Background
	p.showMouse()
}

// control handles one input byte.
func (p *Printer) control(c byte) {
	switch c {
	case CodeColumn:
		p.flushLine()
		p.Cell = min(max(int(int8(p.read())), 0), p.Columns-1)
		p.Column = p.Cell * p.cellWidth()
	case CodeRow:
		p.flushLine()
		p.Row = min(int(int8(p.read())), p.Rows-1)
	case CodeMore:
		p.flushLine()
		if p.More != nil {
			p.callMore()
		}
	case CodeClear:
		p.Clear()
	case CodeNewline:
		p.flushLine()
		p.linesSinceMore++
		p.Column = 0
		p.Row++
	case CodeBackground:
		p.flushLine()
		p.Background = int(p.read())
	case CodeForeground:
		p.flushLine()
		p.Foreground = int(p.read())
	case CodeTab:
		p.flushLine()
		p.Cell = (p.Cell+tabStop)&^(tabStop-1) - 1
		if p.Cell >= p.Columns {
			p.Cell = 0
		}
		p.Column = p.Cell * p.cellWidth()
	case CodeClearTab:
		p.flushLine()
		from := p.Column
		left := p.X * gfx.ColumnWidth
		y := p.Y + p.Row*p.lineHeight()
		bottom := y + p.lineHeight() - 1
		p.Cell = (p.Cell+tabStop)&^(tabStop-1) - 1
		if p.Cell >= p.Columns {
			p.Column, p.Cell = 0, 0
			p.Screen.FillRect(left+from, y, left+p.PixelWidth-1, bottom, p.Background)
			p.linesSinceMore++
			p.Row++
		} else {
			p.Column = p.Cell * p.cellWidth()
			p.Screen.FillRect(left+from, y, left+p.Column-1, bottom, p.Background)
		}
	case CodeHigh:
		p.addChar(highOffset + p.read())
	case CodeLiteral:
		p.addChar(p.read())
	default:
		p.addChar(c)
	}
}

// addChar puts a character into the wrap buffer, printing when the line is full.
func (p *Printer) addChar(c byte) {
	p.lineWidth += p.Screen.CharWidth(c)
	p.line = append(p.line, c)
	if p.Column+p.lineWidth > p.PixelWidth {
		p.flushLine()
	}
}

// scroll moves the window's text up a line, stopping at <MORE> first if the window has
// filled since the last one (C: fn_14b3_062d).
func (p *Printer) scroll() {
	if p.Rows-1 <= p.linesSinceMore && p.MoreEnabled {
		p.linesSinceMore = 0
		p.callMore()
	}
	lh := p.lineHeight()
	last := (div(p.H, lh) - 1) * lh
	if p.H-lh != 0 {
		page := p.Screen.Current
		p.Screen.CopyRectColumns(p.X, p.Y+lh, p.X, p.Y, p.W, p.H-lh, page, page)
	}
	left := p.X * gfx.ColumnWidth
	right := (p.X + p.W) * gfx.ColumnWidth
	p.Screen.FillRect(left, p.Y+last, right-1, p.Y+p.H-1, p.Background)
}

// flushLine prints what fits of the wrap buffer, breaking at a space, and carries the
// rest to the next line (C: fn_14b3_0731).
func (p *Printer) flushLine() {
	for p.Row >= p.Rows {
		p.scroll()
		if p.Row >= p.Rows {
			p.Row--
		}
	}
	x := p.X*gfx.ColumnWidth + p.Column
	y := p.Y + p.Row*p.lineHeight()
	limit := p.PixelWidth
	n := len(p.line)
	w := p.lineWidth
	if p.Column+w > limit {
		if p.Rows-1 <= p.linesSinceMore && p.MoreEnabled {
			limit -= p.moreColumn * p.cellWidth() // keep clear of the prompt
		}
		limit -= p.Column
		cut := 0
		i := n - 1
		for ; i > 0; i-- {
			ch := p.line[i]
			w -= p.Screen.CharWidth(ch)
			if cut == 0 && w <= limit {
				cut = i
			}
			if cut != 0 && ch == space {
				n = i
				p.wrapped = false
				break
			}
		}
		if i == 0 {
			if p.Column != 0 && !p.wrapped {
				n, w = 0, 0 // the word goes whole to the next line
				p.wrapped = true
			} else {
				n = cut
			}
		}
		// fix: a first character too wide for an empty line looped forever.
		if n == 0 && p.Column == 0 && len(p.line) > 0 {
			n = 1
		}
	}
	p.Screen.DrawText(string(p.line[:n]), x, y, p.Foreground, p.Background)
	p.Column += w
	p.Printed += n
	if n < len(p.line) && p.line[n] == space {
		n++
	}
	if n < len(p.line) && p.line[n] == space {
		n++
	}
	p.line = append(p.line[:0], p.line[n:]...)
	p.lineWidth = p.Screen.TextWidth(string(p.line))
	if len(p.line) != 0 || p.Column >= p.PixelWidth {
		p.Column, p.Cell = 0, 0
		p.Row++
		p.linesSinceMore++
		p.flushLine()
	}
}

// DefaultMore is the library's prompt: drawn at the window's bottom right in the
// window's colours swapped, it waits for a key and then wipes itself (C: fn_14b3_0001).
func DefaultMore(p *Printer, prompt string, column, foreground, background int) {
	w := (column - 1) * p.cellWidth()
	x := (p.X+p.W)*gfx.ColumnWidth - w
	y := p.Y + (p.Row-1)*p.lineHeight()
	win := p.Screen.Windows[p.Window]
	if foreground == 0 {
		foreground = win.Background
	}
	if background == 0 {
		background = win.Foreground
	}
	p.Screen.DrawText(prompt, x, y, foreground, background)
	p.showMouse()
	input.Flush()
	input.WaitChar()
	p.hideMouse()
	p.Screen.FillRect(x, y, x+w-1, y+p.lineHeight()-1, p.Background)
}
