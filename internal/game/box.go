package game

import "github.com/memmaker/lands-of-lore-1-go/internal/gfx"

// boxStyle is a bevelled box's colours (C: g_084e).
type boxStyle struct{ fill, light, dark, corner int }

var boxStyles = []boxStyle{
	{225, 227, 227, 227},
	{223, 225, 204, 223},
	{223, 204, 225, 223},
	{225, 227, 223, 225},
	{225, 223, 227, 225},
	{44, 25, 72, 44},
}

// buttonFrame is a button kind's inner frame: its style and inset, 0 for none (C: g_087e).
type buttonFrame struct{ style, inset int }

var buttonFrames = []buttonFrame{{0, 0}, {2, 4}, {1, 1}, {2, 1}, {0, 0}, {0, 0}}

// Box styles used by name.
const boxMenu = 5

// highlightStyles is how far a highlighted button's styles are from the plain ones.
const highlightStyles = 2

// drawBox draws a bevelled w x h box at (x, y) on the current page: light bottom and
// right edges, dark top and left, corner pixels, optionally filled (C: ovr_2cfa_0020).
func (g *Game) drawBox(x, y, w, h, style int, fill bool) {
	s := g.Screen
	w--
	h--
	b := boxStyles[style]
	if fill {
		s.FillRect(x, y, x+w, y+h, b.fill)
	}
	s.DrawLine(x, y+h, x+w, y+h, b.light)
	s.DrawLine(x+w, y, x+w, y+h, b.light)
	s.DrawLine(x, y, x+w, y, b.dark)
	s.DrawLine(x, y, x, y+h, b.dark)
	s.SetPixel(x, y+h, b.corner)
	s.SetPixel(x+w, y, b.corner)
}

// drawWindowBox draws window n as a button of a kind, highlighted or not
// (C: ovr_2cfa_0025).
func (g *Game) drawWindowBox(n, kind int, highlighted bool) {
	s := g.Screen
	win := s.Windows[n]
	x, y, w, h := win.Left(), win.Top(), win.W*gfx.ColumnWidth, win.H
	shift := 0
	if highlighted {
		shift = highlightStyles
	}
	if s.Current == 0 {
		s.HideMouseIn(x, y, x+w, y+h)
	}
	g.drawBox(x, y, w, h, shift+1, true)
	if f := buttonFrames[kind]; f.inset != 0 {
		g.drawBox(x+f.inset, y+f.inset, w-f.inset*2, h-f.inset*2, f.style+shift, false)
	}
	if s.Current == 0 {
		s.ShowMouseIn()
	}
}
