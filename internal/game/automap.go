package game

import (
	"encoding/binary"
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// The automap (C: o2ce1): the level drawn on parchment, a legend of what was found,
// buttons to page through the levels visited, the party an arrow in cycling colours.
const (
	automapPicture     = "PARCH.CPS"
	automapShapeFile   = "AUTOBUT.SHP"
	automapLegendFont  = "FONT6PN.FNT"
	automapTitleFont   = "FONT9PN.FNT"
	automapLegendFile  = "LEVEL%d.XXX"
	automapPage        = 2
	automapFirstShape  = 11  // the shapes the map uses start at this one of the file
	automapShapes      = 109 // (C: f_250c_0404)
	automapArrowShape  = 48  // four arrows, one for each direction (C: g_0710)
	automapSavedTypes  = 80  // the wall types' automap data kept while other levels are shown
	automapTypeMask    = 1<<5 - 1
	automapNoSymbol    = automapTypeMask
	automapCrossedType = 13 // a type drawn across the neighbour's side unless that is seen
	automapShadeLevel  = 52 // the darkened floor: of 64
	automapCellWidth   = 7
	automapCellHeight  = 6
	automapLeftMargin  = 5
	automapTopMargin   = 4
	automapLegends     = 32
	automapLegendX     = 244
	automapLegendY     = 22
	automapLegendRow   = 8
	automapLegendShape = 231
	automapSymbolX     = 232
	automapSymbolShade = 235
	automapTextColour  = 239
	automapTitleX      = 236
	automapTitleY      = 8
	automapHelpString  = 0x4033
	automapHelpX       = 295
	automapHelpY       = 182
	automapHelpColour  = 172
	automapArrowSize   = 16
	automapArrowTicks  = 8
	automapFadeTicks   = 10
	automapSound       = 105
	automapCheatKey    = 48 // reveals the whole level when the debug flag is set
	levelNameStrings   = 0x4210
	noLegend           = -1
)

// The three buttons painted on the parchment (C: g_06ec).
const (
	automapNext = iota
	automapPrevious
	automapExit
	automapButtons
)

var automapButtonRects = [automapButtons]struct{ x, y, w, h int }{
	{252, 175, 21, 25},
	{231, 175, 21, 25},
	{275, 175, 40, 22},
}

// Where the parts of a wall's mark go in its cell, by side north, east, south, west
// (C: f_2501_0000 to f_250c_0000).
var (
	crossedAX    = [directionCount]int{0, 7, 0, -5}
	crossedAY    = [directionCount]int{-5, 0, 6, 0}
	crossedAW    = [directionCount]int{7, 5, 7, 1}
	crossedAH    = [directionCount]int{5, 6, 4, 6}
	crossedBX    = [directionCount]int{0, 7, 0, -1}
	crossedBY    = [directionCount]int{-3, 0, 6, 0}
	crossedCX    = [directionCount]int{6, 7, 6, -3}
	crossedCY    = [directionCount]int{-3, 5, 6, 5}
	crossedW     = [directionCount]int{1, 5, 1, 1}
	crossedH     = [directionCount]int{3, 1, 3, 1}
	wallSymbolDX = [directionCount]int{-1, 6, -1, -8}
	wallSymbolDY = [directionCount]int{-7, -1, 5, -1}
	// the arrow's colours: 0 and the 24 that cycle; the last is the first byte after the
	// table (C: g_075c, g_0774)
	arrowColours = [...]byte{0, 128, 129, 130, 131, 132, 133, 134, 135, 136, 1, 1, 1, 1, 1, 136, 135, 134, 133, 132, 131, 130, 129, 128, 128}
)

// automapSymbol is a mark the legend explains once the map shows one (C: struct Sym,
// g_0720).
type automapSymbol struct {
	shape, dy, str int
	seen           bool
}

var initialAutomapSymbols = []automapSymbol{
	{shape: 14, str: 0x4207}, {shape: 25, str: 0x4208}, {shape: 26, str: 0x4209},
	{shape: 13, str: 0x420a}, {shape: 10, str: 0x420b}, {shape: 11, str: 0x420c},
	{shape: 15, str: 0x420d}, {shape: 22, str: 0x420e}, {shape: 23, str: 0x420f},
	{shape: 17, str: 0x4210}, {shape: 18, dy: -5, str: 0x4251},
}

// automapLegend is an entry of a level's legend file: a block that, once seen, names
// a place and marks the block at (x2, y2) (C: struct Legend).
type automapLegend struct {
	x, y, str, x2, y2, shape int
}

// automapState is what the automap keeps while it is open (C: g_4aee, g_4af0, g_4af2,
// g_4aea, g_4aec, g_4af4, g_4af8, g_4afc).
type automapState struct {
	level        int
	redraw, done bool
	left, top    int // the screen position of the map's first column and row
	palette      gfx.Palette
	shade        [fadeTableSize]byte
	arrow        [fadeTableSize]byte // (C: g_075c; its 256-byte read runs past the 25 entries into other data, the arrow uses only these)
	shapes       [][]byte
	titleFont    *gfx.Font
	legendFont   *gfx.Font
	legend       []automapLegend
	symbols      []automapSymbol
	savedTypes   [automapSavedTypes]int
}

// showAutomap shows the map until the exit button or Escape (C: ovr_2ce1_0020). The
// legend went into the scene palette's memory (sic: g_4edc); here it has its own.
func (g *Game) showAutomap() {
	a := &g.automap
	g.playSound(automapSound, 1)
	g.setIcon(iconAutomap, iconPress)
	a.level = g.level
	g.saveSparePage()
	copy(a.savedTypes[:], g.lvl.automapData[:])
	if err := g.loadAutomap(); err != nil {
		g.fail(err)
		return
	}
	clock.Disable(playClockSlot)
	if err := g.saveLevelState(); err != nil {
		g.fail(err)
		return
	}
	g.clearLevelObjects(true)
	g.resetMonsters()
	a.done, a.redraw = false, false
	g.restoreLevelState(a.level)
	g.loadAutomapLegend(a.level)
	g.fadeToBlack(automapFadeTicks)
	g.drawAutomapPage(automapPage)
	g.showAutomapPage()
	g.Screen.FadePalette(a.palette[:], automapFadeTicks, nil)
	clock.Set(clock.Countdown, automapArrowTicks)
	for !a.done && !g.quit {
		if a.redraw {
			g.drawAutomapPage(automapPage)
			g.showAutomapPage()
			a.redraw = false
		}
		if clock.Get(clock.Countdown) == 0 {
			g.drawAutomapArrow()
			clock.Set(clock.Countdown, automapArrowTicks)
		}
		g.automapButtons()
		key := 0
		if input.Peek() != 0 {
			key = input.Get()
			g.GUI.MarkChanged()
		}
		switch key {
		case automapCheatKey:
			if g.revealAutomap {
				for i := range g.lvl.blocks {
					g.lvl.blocks[i].Flags |= blockRevealed
				}
				a.redraw = true
			}
		case keyEscape:
			a.done = true
		}
		host.Yield()
	}
	g.fadeToBlack(automapFadeTicks)
	if err := g.loadWalls(g.level, false); err != nil {
		g.fail(err)
	}
	copy(g.lvl.automapData[:], a.savedTypes[:])
	g.restoreLevelState(g.level)
	g.placeLevelItems()
	g.GUI.MarkChanged()
	g.restoreSparePage()
	clock.Enable(playClockSlot)
}

// showAutomapPage puts the map drawn on its page on the screen.
func (g *Game) showAutomapPage() {
	s := g.Screen
	s.HideMouse()
	s.CopyRectColumns(0, 0, 0, 0, gfx.Width/gfx.ColumnWidth, gfx.Height, automapPage, pageScreen)
	s.ShowMouse()
}

// fail keeps the first error of a file the original took for granted.
func (g *Game) fail(err error) {
	if g.scriptErr == nil {
		g.scriptErr = err
	}
}

// loadAutomap loads the parchment, the shapes and the fonts and makes the shading
// table (C: ovr_2ce1_004d).
func (g *Game) loadAutomap() error {
	a := &g.automap
	if err := g.loadPicture(automapPicture, automapPage, a.palette[:]); err != nil {
		return err
	}
	shapes, err := g.loadShapes(automapShapeFile)
	if err != nil {
		return err
	}
	a.shapes = make([][]byte, automapShapes)
	for i := range a.shapes {
		if automapFirstShape+i < len(shapes) {
			a.shapes[i] = shapes[automapFirstShape+i]
		}
	}
	if a.legendFont, err = g.loadFont(automapLegendFont); err != nil {
		return err
	}
	if a.titleFont, err = g.loadFont(automapTitleFont); err != nil {
		return err
	}
	copy(a.arrow[:], arrowColours[:])
	shadePalette(a.palette[:], a.shade[:], automapShadeLevel, 0, 0, 0, fadeTableSize, false)
	a.symbols = append(a.symbols[:0], initialAutomapSymbols...)
	return nil
}

// shadePalette makes dst map each of n colours to the nearest one of the palette
// scaled by mul/64 and tinted by r, g, b (C: ovr_2ce1_0043).
func shadePalette(pal, dst []byte, mul, r, gr, b, n int, skipUI bool) {
	buf := make([]byte, 3*n)
	add := [3]int{r, gr, b}
	for i := range buf {
		buf[i] = byte(min(int(pal[i])*mul/64+add[i%3], 63))
	}
	for i := range n {
		dst[i] = byte(gfx.NearestColour(buf[3*i:], pal, n, skipUI))
	}
}

// automapShape is shape i of the map's, nil past the end (fix: the original read on
// past its table).
func (g *Game) automapShape(i int) []byte {
	if i < 0 || i >= len(g.automap.shapes) {
		return nil
	}
	return g.automap.shapes[i]
}

// drawAutomapPage draws the parchment with the map, the legend and the help line,
// twice so the blocks the legend reveals show (C: ovr_2ce1_0025).
func (g *Game) drawAutomapPage(page int) {
	a := &g.automap
	for range 2 {
		if err := g.loadPicture(automapPicture, page, a.palette[:]); err != nil {
			g.fail(err)
			return
		}
		g.drawAutomap(page, g.levelName(a.level))
		g.drawAutomapLegend()
	}
	g.drawAutomapHelp()
}

// levelName is the name of level n (C: f_24fd_0000).
func (g *Game) levelName(n int) string {
	if n <= 0 {
		return ""
	}
	return g.text(levelNameStrings + n)
}

// shadeCell runs a rectangle of the current page through the shading table.
func (g *Game) shadeCell(x, y, w, h int) {
	s := g.Screen
	s.CopyRectRemappedIn(s.Current, x, y, s.Current, x, y, w, h, 0, g.automap.shade[:])
}

// drawCrossedWall marks wall w of block b's side d when it is of the crossed type and
// the side is not seen (C: the static of o2ce1 before ovr_2ce1_008e).
func (g *Game) drawCrossedWall(b, w, x, y, d int) {
	if g.lvl.blocks[b].Flags&(1<<d) != 0 || g.lvl.automapData[w]&automapTypeMask != automapCrossedType {
		return
	}
	g.shadeCell(x+crossedAX[d], y+crossedAY[d], crossedAW[d], crossedAH[d])
	g.shadeCell(x+crossedBX[d], y+crossedBY[d], crossedW[d], crossedH[d])
	g.shadeCell(x+crossedCX[d], y+crossedCY[d], crossedW[d], crossedH[d])
}

// drawWallSymbol draws wall w's symbol on side d of the cell and tells the legend it
// was seen (C: ovr_2ce1_008e).
func (g *Game) drawWallSymbol(w, x, y, d int) {
	t := g.lvl.automapData[w] & automapTypeMask
	if t == automapNoSymbol {
		return
	}
	s := g.Screen
	s.DrawShape(s.Current, g.automapShape(t*directionCount+d), x+wallSymbolDX[d]-2, y+wallSymbolDY[d]-2, 0, 0, gfx.ShapeOptions{})
	g.markSymbolSeen(t)
}

// markSymbolSeen puts the symbol of automap type t in the legend (C: ovr_2ce1_0089).
func (g *Game) markSymbolSeen(t int) {
	for i := range g.automap.symbols {
		if g.automap.symbols[i].shape == t {
			g.automap.symbols[i].seen = true
			return
		}
	}
}

// drawAutomap draws the title and every block seen from all sides with no wall that
// hides (C: ovr_2ce1_002a).
func (g *Game) drawAutomap(page int, title string) {
	a := &g.automap
	s := g.Screen
	pg := s.SetPage(page)
	font := s.SetFont(a.titleFont)
	s.DrawText(title, automapTitleX, automapTitleY, 1, 0)
	x0 := g.automapLeft()
	y0 := g.automapTop()
	x, y := a.left, a.top
	hides := func(w int) bool { return g.lvl.automapData[w]&automapWall != 0 }
	for b := y0*mapSize + x0; b < mapSize*mapSize; b++ {
		blk := &g.lvl.blocks[b]
		if blk.Flags&blockRevealed == blockRevealed && !hides(blk.Walls[0]) && !hides(blk.Walls[2]) &&
			!hides(blk.Walls[1]) && !hides(blk.Walls[3]) {
			n := g.blockInDirection(b, 0)
			e := g.blockInDirection(b, 1)
			so := g.blockInDirection(b, 2)
			w := g.blockInDirection(b, 3)
			wn := g.lvl.blocks[n].Walls[2]
			ws := g.lvl.blocks[so].Walls[0]
			we := g.lvl.blocks[e].Walls[3]
			ww := g.lvl.blocks[w].Walls[1]
			g.shadeCell(x, y, automapCellWidth, automapCellHeight)
			g.drawCrossedWall(w, ww, x, y, 3)
			g.drawWallSymbol(ww, x, y, 3)
			if hides(ww) {
				g.shadeCell(x, y, 1, automapCellHeight)
			}
			g.drawCrossedWall(e, we, x, y, 1)
			g.drawWallSymbol(we, x, y, 1)
			if hides(we) {
				g.shadeCell(x+automapCellWidth-1, y, 1, automapCellHeight)
			}
			g.drawCrossedWall(n, wn, x, y, 0)
			g.drawWallSymbol(wn, x, y, 0)
			if hides(wn) {
				g.shadeCell(x, y, automapCellWidth, 1)
			}
			g.drawCrossedWall(so, ws, x, y, 2)
			g.drawWallSymbol(ws, x, y, 2)
			if hides(ws) {
				g.shadeCell(x, y+automapCellHeight-1, automapCellWidth, 1)
			}
		}
		x += automapCellWidth
		if b%mapSize == mapSize-1 {
			x = a.left
			y += automapCellHeight
			b += x0
		}
	}
	s.SetPage(pg)
	s.SetFont(font)
}

// drawAutomapHelp prints the help line under the map (C: ovr_2ce1_002f).
func (g *Game) drawAutomapHelp() {
	pg := g.Screen.SetPage(automapPage)
	g.printAt(g.text(automapHelpString), automapHelpX, automapHelpY, automapHelpColour, 0, printCentred|printShadow)
	g.Screen.SetPage(pg)
}

// automapButtons runs a click on one of the parchment's buttons, once it is let go
// (C: ovr_2ce1_0034).
func (g *Game) automapButtons() {
	if input.Buttons() != 1 {
		return
	}
	mx, my := g.Screen.Mouse()
	i := 0
	for ; i < automapButtons; i++ {
		r := automapButtonRects[i]
		if mx >= r.x && my >= r.y && mx <= r.x+r.w && my <= r.y+r.h {
			break
		}
	}
	if i == automapButtons {
		return
	}
	g.drawAutomapHelp()
	for input.Buttons() != 0 && !g.quit {
		host.Yield()
	}
	switch i {
	case automapNext:
		g.automapNextLevel(1)
	case automapPrevious:
		g.automapNextLevel(-1)
	case automapExit:
		g.automap.done = true
	}
	g.drawAutomapHelp()
}

// levelVisited says whether level n was entered before (C: ovr_2ce1_005c).
func (g *Game) levelVisited(n int) bool {
	return n > 0 && g.visitedLevels&(1<<(n-1)) != 0
}

// automapNextLevel shows the next visited level up or down (C: ovr_2ce1_0039,
// ovr_2ce1_003e).
func (g *Game) automapNextLevel(step int) {
	const levelMask = 1<<5 - 1
	a := &g.automap
	l := (a.level + step) & levelMask
	for !g.levelVisited(l) {
		l = (l + step) & levelMask
		if l == (a.level+step)&levelMask {
			return // fix: with no level visited the original looped for ever
		}
	}
	if l == a.level {
		return
	}
	for i := range a.symbols {
		a.symbols[i].seen = false
	}
	a.level = l
	if err := g.loadWalls(l, false); err != nil {
		g.fail(err)
		return
	}
	g.restoreLevelState(l)
	g.loadAutomapLegend(l)
	a.redraw = true
}

// drawAutomapLegend lists the places seen and the symbols on the map
// (C: ovr_2ce1_0061).
func (g *Game) drawAutomapLegend() {
	a := &g.automap
	s := g.Screen
	row := 0
	x0 := g.automapLeft()
	y0 := g.automapTop()
	font := s.SetFont(a.legendFont)
	for _, l := range a.legend {
		if g.lvl.blocks[l.x+l.y*mapSize].Flags&blockRevealed != blockRevealed || l.str == noLegend {
			continue
		}
		g.automapLegendText(l.str, automapLegendX, row*automapLegendRow+automapLegendY)
		if l.shape != noLegend {
			g.lvl.blocks[l.x2+l.y2*mapSize].Flags |= blockRevealed
			shape := g.automapShape(l.shape * directionCount)
			s.DrawShape(automapPage, shape, (l.x2-x0)*automapCellWidth+a.left-3, (l.y2-y0)*automapCellHeight+a.top-3, 0, 0, gfx.ShapeOptions{})
			s.DrawShape(automapPage, shape, automapLegendShape, row*automapLegendRow+19, 0, 0, gfx.ShapeOptions{})
		}
		row++
	}
	pg := s.SetPage(automapPage)
	for _, sym := range a.symbols {
		if !sym.seen {
			continue
		}
		g.shadeCell(automapSymbolShade, row*automapLegendRow+21, automapCellWidth, automapCellHeight)
		s.DrawShape(s.Current, g.automapShape(sym.shape*directionCount), automapSymbolX, row*automapLegendRow+sym.dy+18, 0, 0, gfx.ShapeOptions{})
		g.automapLegendText(sym.str, automapLegendX, row*automapLegendRow+automapLegendY)
		row++
	}
	s.SetPage(pg)
	s.SetFont(font)
}

// automapLegendText prints string id on the map's page (C: ovr_2ce1_0066).
func (g *Game) automapLegendText(id, x, y int) {
	pg := g.Screen.SetPage(automapPage)
	g.Screen.DrawText(g.text(id), x, y, automapTextColour, 0)
	g.Screen.SetPage(pg)
}

// automapLeft is the first column with a block that has any flag and sets where the
// map starts so it is centred (C: ovr_2ce1_006b).
func (g *Game) automapLeft() int {
	used := func(x int) bool {
		for y := range mapSize {
			if g.lvl.blocks[y*mapSize+x].Flags != 0 {
				return true
			}
		}
		return false
	}
	x0 := 0
	for x0 < mapSize && !used(x0) {
		x0++
	}
	x1 := mapSize - 1
	for x1 > 0 && !used(x1) {
		x1--
	}
	if x1 <= x0 {
		g.automap.left = automapLeftMargin
		return 0
	}
	g.automap.left = (mapSize-(x1-x0))>>1*automapCellWidth + automapLeftMargin
	return x0
}

// automapTop is automapLeft for the rows (C: ovr_2ce1_0070).
func (g *Game) automapTop() int {
	used := func(y int) bool {
		for x := range mapSize {
			if g.lvl.blocks[y*mapSize+x].Flags != 0 {
				return true
			}
		}
		return false
	}
	y0 := 0
	for y0 < mapSize && !used(y0) {
		y0++
	}
	y1 := mapSize - 1
	for y1 > 0 && !used(y1) {
		y1--
	}
	if y1 <= y0 {
		g.automap.top = automapTopMargin
		return 0
	}
	g.automap.top = (mapSize-(y1-y0))>>1*automapCellHeight + automapTopMargin
	return y0
}

// drawAutomapArrow draws the party's arrow on the map of its level and moves its
// colours on (C: ovr_2ce1_007a).
func (g *Game) drawAutomapArrow() {
	a := &g.automap
	x0 := g.automapLeft()
	y0 := g.automapTop()
	if g.level != a.level {
		return
	}
	s := g.Screen
	pg := s.SetPage(automapPage)
	s.FillRect(0, 0, automapArrowSize, automapArrowSize, 0)
	s.DrawShape(automapPage, g.automapShape(automapArrowShape+g.direction), 0, 0, 0, 0, gfx.ShapeOptions{})
	x := a.left + (g.currentBlock-x0)%mapSize*automapCellWidth
	y := a.top + (g.currentBlock-y0*mapSize)/mapSize*automapCellHeight
	s.HideMouseIn(x, y, x+automapArrowSize, y+automapArrowSize)
	s.CopyRect(x, y, x, y, automapArrowSize, automapArrowSize, automapPage, pageScreen)
	s.CopyRectRemappedIn(automapPage, 0, 0, pageScreen, x-3, y-2, automapArrowSize, automapArrowSize, 0, a.arrow[:])
	s.ShowMouseIn()
	s.SetPage(pg)
	last := len(arrowColours) - 1
	for i := 1; i < last; i++ {
		if i == 1 {
			a.arrow[last] = a.arrow[i]
		}
		a.arrow[i] = a.arrow[i+1] // sic: the last colour is lost, the first then shows twice
	}
}

// loadAutomapLegend reads level n's legend; without a file it is empty
// (C: ovr_2ce1_0084).
func (g *Game) loadAutomapLegend(n int) {
	a := &g.automap
	a.legend = a.legend[:0]
	b, err := g.Files.Read(fmt.Sprintf(automapLegendFile, n))
	if err != nil {
		return
	}
	const entrySize = 12
	word := func(o int) int { return int(int16(binary.LittleEndian.Uint16(b[o:]))) }
	for o := 0; o+entrySize <= len(b) && len(a.legend) < automapLegends; o += entrySize {
		a.legend = append(a.legend, automapLegend{x2: word(o), y2: word(o + 2), shape: word(o + 4), x: word(o + 6), y: word(o + 8), str: word(o + 10)})
	}
}
