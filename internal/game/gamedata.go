package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// Files the data load reads (C: g_110d, g_107e, g_108a, g_1096, g_1119, g_1125, g_0b62,
// g_08ce).
const (
	playfieldPicture = "PLAYFLD.CPS"
	itemShapesFile   = "ITEMSHP.SHP"
	gameShapesFile   = "GAMESHP.SHP"
	thrownShapesFile = "THROWN.SHP"
	itemTypesScript  = "ONETIME.INF"
	itemScriptFile   = "ITEM.INF"
	iceShapesFile    = "ice.shp"
	fireShapesFile   = "fireball.shp"
	iceShapes        = 4
	fireShapes       = 8
)

// The icon the cursor shows while the game is busy (an hourglass).
const busyCursor = 133

// The <MORE> prompt of the text window (C: g_113a and its fn_14b3_00c8 arguments).
const (
	morePrompt     = "<MORE>"
	moreColumn     = 10
	moreForeground = 31
)

// Colour tables built from the playfield palette (C: ovr_2d50_0048's fn_1b43_000e calls):
// the palette is first made white in its lower half and in colours 192-195, except
// colour 1, so that only the rest can be chosen.
const (
	paletteWhite      = 63
	whiteColours      = 128
	whiteBlockFirst   = 192
	whiteBlockColours = 4
	shadowTarget      = 1
	shadowAmount      = 96
	tintTarget        = 144
	tintAmount        = 65
)

// loadGameData loads what play needs after the menu (C: ovr_2d50_0048). The original's
// buffer allocations, the shape cache and the speech cache set-up have no counterpart, and
// the block of globals it clears (g_7fd0, 563 bytes) starts out zero here.
func (g *Game) loadGameData() error {
	file, err := g.Files.Read(playfieldPicture)
	if err != nil {
		return err
	}
	playfield, err := data.DecodeCPS(file)
	if err != nil {
		return err
	}
	copy(g.Screen.Page(pageScratch), playfield.Pixels)
	copy(g.Palette[:], playfield.Palette)
	g.buildColourTables()

	if g.cursors, err = g.loadShapes(itemIcons); err != nil {
		return err
	}
	g.setBusyCursor(busyCursor)
	if g.itemShapes, err = g.loadShapes(itemShapesFile); err != nil {
		return err
	}
	if g.gameShapes, err = g.loadShapes(gameShapesFile); err != nil {
		return err
	}
	if g.thrownShapes, err = g.loadShapes(thrownShapesFile); err != nil {
		return err
	}
	g.freeAllItems()
	if err := g.runScriptOnce(itemTypesScript, 0); err != nil {
		return err
	}
	if g.itemScript, err = g.loadScript(itemScriptFile); err != nil {
		return err
	}
	g.textReady = true // C: g_20d9
	if g.iceShapes, err = g.loadShapesUpTo(iceShapesFile, iceShapes); err != nil {
		return err
	}
	if g.fireShapes, err = g.loadShapesUpTo(fireShapesFile, fireShapes); err != nil {
		return err
	}
	g.Text.SetMorePrompt(morePrompt, moreColumn, moreForeground, 0)
	g.Text.EnableMore()
	g.restoreCursor()
	return nil
}

// buildColourTables makes the shadow and tint tables (C: f_2803_030a, f_2803_020a).
func (g *Game) buildColourTables() {
	pal := g.Palette
	for i := range whiteColours * 3 {
		pal[i] = paletteWhite
	}
	copy(pal[shadowTarget*3:shadowTarget*3+3], g.Palette[shadowTarget*3:])
	for i := whiteBlockFirst * 3; i < (whiteBlockFirst+whiteBlockColours)*3; i++ {
		pal[i] = paletteWhite
	}
	g.shadowTable = gfx.MixTable(pal[:], shadowTarget, shadowAmount)
	g.tintTable = gfx.MixTable(pal[:], tintTarget, tintAmount)
}

// loadShapes reads every shape of a shape file (C: the loops of ovr_2d50_002a, 002f,
// 0034 over ovr_2d46_0025).
func (g *Game) loadShapes(name string) ([][]byte, error) {
	return g.loadShapesUpTo(name, -1)
}

// loadShapesUpTo reads the first n shapes of a file, all of them for n < 0
// (C: ovr_2d1b_0020, ovr_2d03_0020).
func (g *Game) loadShapesUpTo(name string, n int) ([][]byte, error) {
	file, err := g.Files.Read(name)
	if err != nil {
		return nil, err
	}
	img, err := data.DecodeCPS(file)
	if err != nil {
		return nil, err
	}
	collection := img.Pixels
	count := gfx.ShapeCount(collection)
	if n >= 0 {
		count = n
	}
	shapes := make([][]byte, count)
	for i := range shapes {
		if frame := gfx.ShapeFrame(collection, i); frame != nil {
			shapes[i] = append([]byte(nil), frame...)
		}
	}
	return shapes, nil
}

// tintedTable maps the first n colours of pal to the nearest of them after scaling by
// mul/64 and adding (r, g, b) (C: ovr_2ce1_0043).
func tintedTable(pal []byte, mul, r, gr, b, n int, skipUI bool) []byte {
	const maxLevel = 63
	tinted := make([]byte, n*3)
	add := [3]int{r, gr, b}
	for i := range n * 3 {
		v := int(pal[i])*mul/64 + add[i%3]
		tinted[i] = byte(min(v, maxLevel))
	}
	table := make([]byte, fadeTableSize)
	for i := range n {
		table[i] = byte(gfx.NearestColour(tinted[i*3:], pal, n, skipUI))
	}
	return table
}
