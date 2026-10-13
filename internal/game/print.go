package game

// printAt flags (C: fn_0845_07b3).
const (
	printCentred = 1 << 0 // x is the middle of the text
	printRight   = 1 << 1 // x is its right end
	printShadow  = 1 << 2 // a colour 1 shadow left and below
	printOutline = 1 << 3 // the same in colour 227
)

const (
	shadowColour  = 1
	outlineColour = 227
)

// printAt draws text at (x, y) on the current page (C: fn_0845_07b3; its printf
// formatting is the caller's fmt.Sprintf).
func (g *Game) printAt(text string, x, y, foreground, background, flags int) {
	s := g.Screen
	if flags&printCentred != 0 {
		x -= s.TextWidth(text) >> 1
	}
	if flags&printRight != 0 {
		x -= s.TextWidth(text)
	}
	if flags&printShadow != 0 {
		s.DrawText(text, x-1, y, shadowColour, background)
		s.DrawText(text, x, y+1, shadowColour, background)
	}
	if flags&printOutline != 0 {
		s.DrawText(text, x-1, y, outlineColour, background)
		s.DrawText(text, x, y+1, outlineColour, background)
	}
	s.DrawText(text, x, y, foreground, background)
}
