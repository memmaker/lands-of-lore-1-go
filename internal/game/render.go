package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
)

// The 3D view (C: o2cda). The walls are a map of 8x8 VCN blocks, 22 columns by 15 rows,
// that the wall sets of the VMP are copied into; decorations, doors, block effects,
// items and monsters are then drawn on top, slot by slot, each clipped to the columns
// the walls in front leave open.
const (
	viewColumns    = 22
	viewRows       = 15
	viewCells      = viewColumns * viewRows
	viewMapSize    = 2 * viewCells // the walls, then the floor and ceiling behind them
	viewLeft       = 112           // the view's left edge on the page
	viewCentreX    = 200
	viewBottom     = 120
	viewTop        = 0
	viewRight      = viewLeft + viewColumns*vcnPixels - 1
	vcnPixels      = 8
	vcnRowBytes    = vcnPixels / 2
	viewWindowClip = 13 // the window everything in the view is clipped to
	columnMargin   = 14 // the view's first column on the screen, in columns

	vmpWallSize = 431 // words of one wall set in the VMP (C: struct Lvl)

	// map words: a block number with these bits
	vcnOverlay = 1 << 15 // drawn see-through over the floor behind it
	vcnFlipped = 1 << 14
	vcnIndex   = vcnFlipped - 1
	vcnNibble  = 1<<4 - 1

	viewSlots = visibleBlocks
	partySlot = 16 // the party's own block
	nearSlots = 15 // the slots in front of the party: monsters are drawn only there
)

// The parts of a wall set the view copies, as word offsets into it (C: draw_walls).
const (
	wallFar       = 0x00 // the farthest row's walls seen from the side
	wallSideFar   = 0x2d
	wallSideMid   = 0x51
	wallSideNear  = 0x61
	wallSideNext  = 0x66
	wallSideFront = 0x75
	wallFront1    = 0x81
	wallFront2    = 0x9f
	wallFront3    = 0xef
	floorStart    = -viewCells // the background before the wall set
)

// Wall type flags (C: g_531e; ScummVM: _wllWallFlags).
const (
	wallIsDoor        = 1 << 3
	wallDoorFrameless = 1 << 4 // a door that does not clip what is behind it
	wallSeeThrough    = 1 << 7 // what is in the block behind shows
)

// Clip spans in wallSpan.
const (
	spanNone = -40
	spanAll  = -41
)

// Flags of drawObject and drawWallShape (C: draw_obj, blit).
const (
	objFlipX        = 1 << 4
	objBlend        = 1 << 5 // through the level's TLC tables
	objFlipY        = 1 << 6
	objOverlayTable = 1 << 7 // the colour table is the overlay instead

	wallShapeFloor = 1 << 0 // scaled by its distance
	wallShapeDoor  = 1 << 1 // a door, scaled by its row
)

// Shades: fadeTables[n] darkens n steps.
const (
	darkestShade     = len(level{}.fadeTables) - 1
	overlayShade     = 4 // the overlay table of see-through colour 255 (C: g_7c82)
	doorShadeExtra   = 2
	levelNoOverlay   = 22 // the level whose objects draw colour 255 as black instead
	overlayColour    = 255
	shadeRowsPerStep = 3
)

// viewState is the renderer's working state (C: o2cda's globals).
type viewState struct {
	front, left, right int                // the sides of a block facing the party (C: g_4ae8, g_4ae6, g_4ae4)
	blockMap           [viewMapSize]int   // (C: g_404c)
	parity             int                // which way the floor is mirrored (C: g_4002)
	clipCache          [2 * viewSlots]int // each slot's open columns (C: g_4a9c)
	doorTop            [viewSlots]int     // (C: g_4028)
	doorBottom         [viewSlots]int     // (C: g_4004)

	// where the last shape went (C: g_4000, g_3ffe, g_3ffc, g_3ffa, g_3ff8)
	x, y, scaleX, scaleY, width int

	distanceColours [distanceColours]byte // (C: g_0500)
}

const distanceColours = 15

// sceneOverlay is a decoration drawn over the whole view (C: struct Deco4 g_4bd0).
type sceneOverlay struct {
	shape    []byte
	x, y     int
	mirrored bool // drawn again flipped to the right of itself
}

// renderScene draws the 3D view: with page 0 it is built on the view page and the
// changes are copied to the screen (C: ovr_2cda_0061).
func (g *Game) renderScene(page int) {
	// the original swapped twice: the second only acts when the first made no
	// difference, which it never does
	for range 2 {
		if page != 0 && page != g.viewPage {
			g.viewPage, g.viewBackPage = g.viewBackPage, g.viewPage
			g.copyViewFrame()
		}
	}
	g.drawWalls(g.currentBlock, g.direction)
	g.drawViewBlocks(g.viewPage)
	g.drawViewSlots()
	if page == 0 {
		g.drawSceneOverlay(g.viewPage)
		g.hideMouseInWindow(viewWindow)
		g.copyChangedView(g.viewPage, g.viewBackPage)
		g.viewPage, g.viewBackPage = g.viewBackPage, g.viewPage
		g.Screen.ShowMouseIn()
	}
	g.playSoundAt(0, g.currentBlock)
	g.drawCompass()
	g.sceneUpdateRequired = false
}

// viewWall is the wall type of view slot n's side side.
func (g *Game) viewWall(n, side int) int {
	return g.lvl.blocks[g.visibleBlocks[n]].Walls[side]
}

// setupView finds the blocks in view and forgets the clip spans (C: setup).
func (g *Game) setupView(block, dir int) {
	for i := range g.visibleBlocks {
		g.visibleBlocks[i] = (block + viewBlockOffsets[dir*viewSlots+i]) & blockMask
	}
	for i := range g.view.clipCache {
		g.view.clipCache[i] = -1
	}
}

// Sides of a block as seen from each direction.
var (
	frontSides = [directionCount]int{2, 3, 0, 1} // (C: g_0094)
	leftSides  = [directionCount]int{1, 2, 3, 0} // (C: g_0098)
	rightSides = [directionCount]int{3, 0, 1, 2} // (C: g_009c)
)

// drawWalls fills the block map with the floor and the walls in view (C: draw_walls).
func (g *Game) drawWalls(block, dir int) {
	v := &g.view
	v.front, v.left, v.right = frontSides[g.direction], leftSides[g.direction], rightSides[g.direction]
	clear(v.blockMap[:viewCells])
	v.parity = (block>>5 + block&columnMask + g.direction) & 1
	if v.parity == 0 {
		g.copyWall(0, viewRows, 1, floorStart, viewColumns, viewRows)
	} else {
		g.copyWallMirrored(0, viewRows, 1, floorStart, viewColumns, viewRows)
	}
	g.setupView(block, dir)

	front, left, right := v.front, v.left, v.right
	w := g.viewWall
	isDoor := func(t int) bool { return g.lvl.wallFlags[t]&wallIsDoor != 0 }
	solid := func(t int) bool { return t != 0 && !isDoor(t) } // (C: ovr_2cda_0034)

	if t := w(0, left); t != 0 {
		g.copyWall(-2, 3, t, wallSideNext, 3, 5)
	}
	if t := w(6, right); t != 0 {
		g.copyWallMirrored(21, 3, t, wallSideNext, 3, 5)
	}
	if t, u := w(1, left), w(2, front); solid(t) && !isDoor(u) {
		g.copyWall(2, 3, t, wallSideNext, 3, 5)
	} else if t != 0 && isDoor(u) {
		g.copyWall(2, 3, u, wallSideNext, 3, 5)
	}
	if t, u := w(5, right), w(4, front); solid(t) && !isDoor(u) {
		g.copyWallMirrored(17, 3, t, wallSideNext, 3, 5)
	} else if t != 0 && isDoor(u) {
		g.copyWallMirrored(17, 3, u, wallSideNext, 3, 5)
	}
	if t := w(2, left); t != 0 {
		g.copyWall(8, 3, t, wallSideNear, 1, 5)
	}
	if t := w(4, right); t != 0 {
		g.copyWallMirrored(13, 3, t, wallSideNear, 1, 5)
	}
	for _, c := range []struct{ slot, x int }{{1, -4}, {5, 20}, {2, 2}, {4, 14}} {
		if t := w(c.slot, front); solid(t) {
			g.copyWall(c.x, 3, t, wallFront1, 6, 5)
		}
	}
	if t := w(3, front); t != 0 {
		g.copyWall(8, 3, t, wallFront1, 6, 5)
	}
	if t := w(7, left); t != 0 {
		g.copyWall(0, 3, t, wallSideFront, 2, 6)
	}
	if t := w(11, right); t != 0 {
		g.copyWallMirrored(20, 3, t, wallSideFront, 2, 6)
	}
	if t := w(8, left); t != 0 {
		g.copyWall(6, 2, t, wallSideMid, 2, 8)
	}
	if t := w(10, right); t != 0 {
		g.copyWallMirrored(14, 2, t, wallSideMid, 2, 8)
	}
	if t := w(8, front); solid(t) {
		g.copyWall(-4, 2, t, wallFront2, 10, 8)
	}
	if t := w(10, front); solid(t) {
		g.copyWall(16, 2, t, wallFront2, 10, 8)
	}
	if t := w(9, front); t != 0 {
		g.copyWall(6, 2, t, wallFront2, 10, 8)
	}
	if t := w(12, left); t != 0 {
		g.copyWall(3, 1, t, wallSideFar, 3, 12)
	}
	if t := w(14, right); t != 0 {
		g.copyWallMirrored(16, 1, t, wallSideFar, 3, 12)
	}
	if t := w(12, front); !isDoor(t) {
		g.copyWall(-13, 1, t, wallFront3, 16, 12)
	}
	if t := w(14, front); !isDoor(t) {
		g.copyWall(19, 1, t, wallFront3, 16, 12)
	}
	if t := w(13, front); t != 0 {
		g.copyWall(3, 1, t, wallFront3, 16, 12)
	}
	t, u := w(15, left), w(17, right)
	if t != 0 {
		g.copyWall(0, 0, t, wallFar, 3, 15)
	}
	if u != 0 {
		g.copyWallMirrored(19, 0, u, wallFar, 3, 15)
	}
}

// wallWord is word at of wall type t's wall set, 0 past the VMP's end.
func (g *Game) wallWord(t, at int) int {
	i := (g.lvl.wallVMP[t]-1)*vmpWallSize + viewCells + at
	if i < 0 || i >= len(g.lvl.vmpData) {
		return 0
	}
	return g.lvl.vmpData[i]
}

// copyWall copies a rows x cols part of wall type t's set into the block map at
// column x, row y; empty words leave what is there (C: ovr_2cda_002a).
func (g *Game) copyWall(x, y, t, at, cols, rows int) {
	if g.lvl.wallVMP[t] == 0 {
		return
	}
	src := at
	for i := range rows {
		for j := range cols {
			if v := g.wallWord(t, src); x+j >= 0 && x+j < viewColumns && v != 0 {
				g.view.blockMap[(y+i)*viewColumns+x+j] = v
			}
			src++
		}
	}
}

// copyWallMirrored is copyWall mirrored left to right (C: ovr_2cda_002f).
func (g *Game) copyWallMirrored(x, y, t, at, cols, rows int) {
	if g.lvl.wallVMP[t] == 0 {
		return
	}
	for i := range rows {
		for j := range cols {
			if x+j < 0 || x+j >= viewColumns {
				continue
			}
			v := g.wallWord(t, at+i*cols+cols-1-j)
			if v != 0 {
				g.view.blockMap[(y+i)*viewColumns+x+j] = v ^ vcnFlipped
			}
		}
	}
}

// drawViewBlocks draws the block map onto a page: each word names a VCN block, 0
// takes the background's, and an overlay word is drawn see-through over the
// background (C: fn_1aff_00b9).
func (g *Game) drawViewBlocks(page int) {
	dst := g.Screen.Page(page)
	for row := range viewRows {
		for col := range viewColumns {
			i := row*viewColumns + col
			at := row*vcnPixels*gfx.Width + viewLeft + col*vcnPixels
			v := g.view.blockMap[i]
			overlay := 0
			if v&vcnOverlay != 0 {
				overlay = v &^ vcnOverlay
				v = 0
			}
			flipped := v&vcnFlipped != 0
			v &= vcnIndex
			if v == 0 {
				v = g.view.blockMap[i+viewCells]
				if v&vcnFlipped != 0 {
					flipped = true
				}
				v &= vcnIndex
			}
			g.drawVCNBlock(dst, at, v, flipped, false)
			if overlay != 0 {
				g.drawVCNBlock(dst, at, overlay&vcnIndex, overlay&vcnFlipped != 0, true)
			}
		}
	}
}

// drawVCNBlock draws one 8x8 block of 4-bit pixels, coloured by its shift; see-through
// leaves colour 0.
func (g *Game) drawVCNBlock(dst []byte, at, n int, flipped, seeThrough bool) {
	lv := &g.lvl
	shift := 0
	if n < len(lv.vcnShift) {
		shift = int(lv.vcnShift[n])
	}
	src := n * vcnBlockSize
	colour := func(nibble int) byte {
		i := nibble | shift
		if i < len(lv.vcnColours) {
			return lv.vcnColours[i]
		}
		return 0
	}
	for y := range vcnPixels {
		for x := range vcnRowBytes {
			b := 0
			if k := src + y*vcnRowBytes + x; !flipped && k < len(lv.vcnData) {
				b = int(lv.vcnData[k])
			} else if k := src + y*vcnRowBytes + vcnRowBytes - 1 - x; flipped && k < len(lv.vcnData) {
				b = int(lv.vcnData[k])
			}
			first, second := colour(b>>4), colour(b&vcnNibble)
			if flipped {
				first, second = second, first
			}
			p := at + y*gfx.Width + 2*x
			if !seeThrough || first != 0 {
				dst[p] = first
			}
			if !seeThrough || second != 0 {
				dst[p+1] = second
			}
		}
	}
}

// copyChangedView copies to the screen the 4-pixel groups of the view in which page
// a differs from page b (C: fn_19c9_0c24).
func (g *Game) copyChangedView(a, b int) {
	const group = 4
	pa, pb, screen := g.Screen.Page(a), g.Screen.Page(b), g.Screen.Page(pageScreen)
	for y := range viewBottom {
		for x := viewLeft; x < viewLeft+viewColumns*vcnPixels; x += group {
			at := y*gfx.Width + x
			if string(pa[at:at+group]) != string(pb[at:at+group]) {
				copy(screen[at:at+group], pa[at:at+group])
			}
		}
	}
}

// drawViewSlots draws decorations, doors, effects, items and monsters, slot by slot
// from the back (C: ovr_2cda_0048).
func (g *Game) drawViewSlots() {
	for _, n := range drawOrder {
		t := g.viewWall(n, g.view.front)
		a, b := g.clipColumns(n, viewWindowClip)
		if b <= a {
			continue
		}
		g.drawDecorations(n)
		flags := g.lvl.wallFlags[t]
		if n == partySlot {
			flags |= wallSeeThrough
		}
		g.drawBlockEffects(n, false)
		if g.lvl.blocks[g.visibleBlocks[n]].Objects != 0 && flags&wallSeeThrough != 0 {
			g.drawBlockObjects(n)
		}
		g.drawBlockEffects(n, true)
		if g.lvl.wallFlags[t]&wallIsDoor != 0 {
			const doorStepHeight = 20
			raised := (t - doorFirstType[t]) * doorStepHeight
			g.clipRows(n, viewWindowClip)
			const doorColumn = 10
			g.drawWallShape(g.lvl.doorShapes[doorShapeIndex[t]], nil, n, doorColumn, 0, -raised, wallShapeDoor)
			g.clipColumns(n, viewWindowClip)
		}
	}
}

// setViewWindow sets window win to columns x1-x2 and 8-pixel rows y1-y2
// (C: ovr_2cda_0020).
func (g *Game) setViewWindow(win, x1, y1, x2, y2 int) {
	g.Screen.SetWindow(win, x1, y1*vcnPixels, x2-x1, (y2-y1)*vcnPixels)
}

// rowsOf are the rows a door in slot n clips to; for the door's own frame (C: clip
// with win -1) the next row's. With a window it is clipped to them too (C: clip).
func (g *Game) rowsOf(n int, frame bool) (top, bottom int) {
	k := viewDepth[n]
	if frame && k != len(viewClipTop)-1 {
		k++
	}
	return viewClipTop[k], viewClipBottom[k]
}

func (g *Game) clipRows(n, win int) {
	top, bottom := g.rowsOf(n, false)
	w := g.Screen.Windows[win]
	g.Screen.SetWindow(win, w.X, top, w.W, bottom-top)
}

// clipColumns finds the columns of slot n the walls in front leave open, sets window
// win to them and returns them (C: ovr_2cda_0025).
func (g *Game) clipColumns(n, win int) (a, b int) {
	v := &g.view
	if v.clipCache[2*n] != -1 {
		a, b = v.clipCache[2*n], v.clipCache[2*n+1]
	} else {
		a, b = 0, viewColumns
		top, bottom := 0, viewBottom
		for i := range viewSlots {
			t := g.viewWall(i, v.front)
			if g.lvl.wallFlags[t]&wallIsDoor == 0 {
				d := wallSpan[n*viewSlots+i]
				if g.lvl.wallVMP[t] == 0 || d == spanNone {
					continue
				}
				if d == spanAll {
					a, b = viewColumns, 0
					break
				}
				if d > 0 && b > d {
					b = d
				}
				if d < 0 && -d > a {
					a = -d
				}
			} else {
				j := (n*viewSlots + i) * 2
				if doorSpan[j] > a {
					a = doorSpan[j]
					if g.lvl.wallFlags[t]&wallDoorFrameless == 0 {
						top, bottom = g.rowsOf(n, true)
					}
				}
				if doorSpan[j+1] < b {
					b = doorSpan[j+1]
					if g.lvl.wallFlags[t]&wallDoorFrameless == 0 {
						top, bottom = g.rowsOf(n, true)
					}
				}
			}
			if b < a {
				break
			}
		}
		a += columnMargin
		b += columnMargin
		v.doorTop[n], v.doorBottom[n] = top, bottom
		v.clipCache[2*n], v.clipCache[2*n+1] = a, b
	}
	g.setViewWindow(win, a, 0, b, viewRows)
	return a, b
}

// fixedMul multiplies by an 8.8 factor, signed, keeping 16 bits (C: fn_1aff_0002,
// fn_1aff_0386).
func fixedMul(v, factor int) int {
	return int(int16(int32(int16(v)) * int32(int16(factor)) >> byteShift))
}

// toView turns map position (x, y) into the view's: x to the right, y ahead of the
// party (C: fn_1aff_0341).
func (g *Game) toView(x, y int) (int, int) { return g.toViewFacing(x, y, g.direction) }

func (g *Game) toViewFacing(x, y, dir int) (int, int) {
	return relativePosition(g.partyX, g.partyY, x, y, dir)
}

// relativePosition is (x, y) as seen from (ox, oy) looking towards dir: across to the
// right and ahead (C: fn_1aff_0341).
func relativePosition(ox, oy, x, y, dir int) (int, int) {
	dx, dy := x-ox, oy-y
	switch dir {
	case 1:
		return -dy, dx
	case 2:
		return -dx, -dy
	case 3:
		return dy, -dx
	}
	return dx, dy
}

// turnInBlock turns a position inside a block to direction dir (C: fn_1aff_03b4).
func turnInBlock(x, y, dir int) (int, int) {
	switch dir {
	case 1:
		return blockUnits - y, x
	case 2:
		return blockUnits - x, blockUnits - y
	case 3:
		return y, blockUnits - x
	}
	return x, y
}

// placeObject finds where a shape at map position (x, y) goes in the view, how much
// it is scaled and its distance row; a flying one hangs from the ceiling
// (C: ovr_2cda_0039).
func (g *Game) placeObject(x, y int, shape []byte, flying bool) int {
	v := &g.view
	x, y = g.toView(x, y)
	if y < 0 {
		v.x, v.y, v.scaleX, v.scaleY = 0, 0, 0, 0
		return 0
	}
	r := y >> 5
	v.y = floorLineY(r)
	v.x = fixedMul(x, sideStepScale(r)) + viewCentreX
	v.scaleX = scaleAtY(v.y)
	v.scaleY = v.scaleX
	if !flying {
		v.y -= gfx.ScaledShapeHeight(shape, v.scaleY) >> 1
	} else {
		v.y = (viewBottom-v.y)>>1 + gfx.ScaledShapeHeight(shape, v.scaleY)>>1
	}
	return r
}

// shapeFlags turns the renderer's flip and blend flags into the drawing's.
func shapeFlags(flags int) int {
	f := 0
	if flags&objFlipX != 0 {
		f |= gfx.ShapeFlipX
	}
	if flags&objBlend != 0 {
		f |= gfx.ShapeLevels
	}
	if flags&objFlipY != 0 {
		f |= gfx.ShapeFlipY
	}
	return f
}

// blitViewShape draws a shape scaled at the last place onto the view page
// (C: blit).
func (g *Game) blitViewShape(shape, colours []byte, x, y, flags int, fade []byte) {
	v := &g.view
	f := gfx.ShapeScaled | shapeFlags(flags) | gfx.ShapeFaded
	opt := gfx.ShapeOptions{Colours: colours, Fade: fade, FadeSteps: 1, ScaleX: v.scaleX, ScaleY: v.scaleY}
	if colours != nil {
		f |= gfx.ShapeRecoloured
	}
	if f&gfx.ShapeLevels != 0 {
		opt.Levels, opt.LevelTables = g.lvl.tlcTable, g.lvl.tlcLevels
	}
	g.Screen.DrawShape(g.viewPage, shape, x, y, viewWindowClip, f, opt)
}

// drawWallShape draws a shape on the wall of slot n at column c of the wall: a door,
// or something on the floor (C: draw_wallshape).
func (g *Game) drawWallShape(shape, colours []byte, n, c, dx, dy, flags int) {
	v := &g.view
	raise := 0
	c = viewColumnFromDirection[g.direction<<5+c]
	row := viewDepth[n]
	const fifths = 5
	shade := shapeDistance[row*fifths+c/fifths]
	idx := n<<5 + c
	v.y = wallShapeY[idx] + viewBottom
	switch {
	case flags&wallShapeFloor != 0:
		v.scaleX = scaleAtY(v.y)
		v.scaleY = v.scaleX
		shade += doorShadeExtra
	case flags&wallShapeDoor != 0:
		v.scaleX, v.scaleY = doorScale[row*2], doorScale[row*2+1]
		raise = doorRaise[row]
		shade += doorShadeExtra
	}
	if v.scaleX == 0 || v.scaleY == 0 {
		return
	}
	half := gfx.ScaledShapeHeight(shape, v.scaleY) >> 1
	if dx != 0 {
		dx = fixedMul(dx, v.scaleX)
	}
	if dy != 0 {
		dy = fixedMul(dy, v.scaleY)
	}
	v.x = wallShapeX[idx] + dx + viewCentreX
	v.y = v.y + 4 - half + dy - raise
	fade := g.lvl.fadeTables[min(shade, darkestShade)]
	v.width = gfx.ScaledShapeWidth(shape, v.scaleX)
	v.x -= v.width >> 1
	v.y -= half
	g.blitViewShape(shape, colours, v.x, v.y, flags, fade)
}

// drawObject draws a shape standing at map position (x, y), moved by (dx, dy) at full
// size. dist -1 shades it by its distance; otherwise all its colours become dist. It
// returns the fade table used (C: draw_obj).
func (g *Game) drawObject(shape, colours []byte, x, y, dx, dy, flags, dist int, flying bool) []byte {
	v := &g.view
	overlay := g.lvl.fadeTables[overlayShade]
	if flags&objOverlayTable != 0 {
		flags &^= objOverlayTable
		overlay, colours = colours, nil
	}
	r := g.placeObject(x, y, shape, flying)
	var fade []byte
	if dist == -1 {
		fade = g.lvl.fadeTables[max(min(darkestShade-(r/shadeRowsPerStep-1), darkestShade), 0)]
	} else {
		for i := range v.distanceColours {
			v.distanceColours[i] = byte(dist)
		}
		colours = append([]byte{byte(doorSpan[len(doorSpan)-1])}, v.distanceColours[:]...)
		fade = g.lvl.fadeTables[darkestShade]
	}
	f := shapeFlags(flags)
	if g.level == levelNoOverlay {
		if fade != nil {
			fade[overlayColour] = 0
		}
	} else {
		f |= gfx.ShapeOverlay
	}
	if dx != 0 {
		v.x += fixedMul(dx, v.scaleX)
	}
	if dy != 0 {
		v.y += fixedMul(dy, v.scaleY)
	}
	half := gfx.ScaledShapeHeight(shape, v.scaleY) >> 1
	opt := gfx.ShapeOptions{Colours: colours, Fade: fade, FadeSteps: 1, ScaleX: v.scaleX, ScaleY: v.scaleY, Overlay: overlay}
	f |= gfx.ShapeScaled | gfx.ShapeCentred | gfx.ShapeFaded
	if colours != nil {
		f |= gfx.ShapeRecoloured
	}
	if f&gfx.ShapeLevels != 0 {
		opt.Levels, opt.LevelTables = g.lvl.tlcTable, g.lvl.tlcLevels
		opt.FadeSteps = 0
	}
	g.Screen.DrawShape(g.viewPage, shape, v.x, v.y, viewWindowClip, f, opt)
	v.width = gfx.ScaledShapeWidth(shape, v.scaleX)
	v.x -= v.width >> 1
	v.y -= half
	return fade
}

// Monster frames (C: ovr_2cda_003e).
const (
	mirroredFrames = 1 << 4 // in monsterFacingFrames: the frame is drawn flipped
	noFrame        = -1
	hitFrame       = 12
	monsterResting = 4 // type flag: shows frame 13 while fleeing
	attackFrames   = 13
	maxDeathFrame  = 13
	monsterSlowHit = 14
)

// monsterFrame is the shape a monster shows, by the kind of animation its type has
// (C: ovr_2cda_003e).
func (g *Game) monsterFrame(m *Monster, base int) int {
	switch g.lvl.monsterAnimType[m.Type.shapeIndex] {
	case 0:
		return monsterFrame0(m, base)
	case 1:
		return g.monsterFrame1(m)
	case 2:
		return min(m.Counter, maxDeathFrame)
	case 3:
		return monsterFrame3(m)
	}
	return noFrame // fix: the original returned whatever was in a register
}

// monsterFrame0 animates by facing (C: frame0).
func monsterFrame0(m *Monster, base int) int {
	switch {
	case base != 0:
		if m.Mode == monsterDead {
			return noFrame
		}
		return base + m.AnimFrame
	case m.Damage != 0:
		return hitFrame
	}
	switch m.Mode {
	case monsterDead:
		return noFrame
	case monsterModeAttack:
		return m.Counter + attackFrames
	case monsterModeHit:
		return monsterSlowHit
	case monsterModeRetreat:
		if m.Type.flags&monsterResting != 0 {
			return attackFrames
		}
		return 0
	}
	return base + m.AnimFrame
}

// monsterFrame1 shows the monster's wounds: four frames per quarter of hit points
// (C: frame1).
func (g *Game) monsterFrame1(m *Monster) int {
	const woundFrames = 4
	hp := scale(m.Type.hitPoints, difficultyHitPoints[g.difficulty()])
	b := 0
	switch {
	case m.HitPoints > hp>>1:
	case m.HitPoints > hp>>2:
		b = woundFrames
	default:
		b = 2 * woundFrames
	}
	switch m.Mode {
	case monsterModeHit:
		return hitFrame
	case monsterDead:
		return m.Counter + hitFrame
	case monsterModeAttack:
		return b + m.Counter
	}
	return b
}

// monsterFrame3 (C: frame3).
func monsterFrame3(m *Monster) int {
	const (
		hit    = 5
		attack = 6
	)
	switch m.Mode {
	case monsterModeAttack:
		return m.Counter + attack
	case monsterModeHit:
		return hit
	case monsterModeRetreat:
		if m.Damage != 0 {
			return hit
		}
		return attack
	}
	if m.Damage != 0 {
		return hit
	}
	return m.AnimFrame
}

// Monster type flags the renderer reads.
const (
	monsterFlies      = 1 << 9
	monsterBlended    = 1 << 11
	monsterBloodMask  = 3 << 14
	monsterBloodGreen = 1 << 14
	monsterBloodWhite = 2 << 14
	monsterBloodOther = 3 << 14
	monsterEquipMask  = 1<<3 - 1
	gameShapeHit      = 6 // (C: f_2803_00be)
	hitColoursFirst   = 2
	hitColoursLast    = 7
	maxHitDivisor     = 4
)

// tickOffset is a monster's shift by its animation tick. A tick of 255 (never moved)
// read the two bytes before the table, the end of the distance colours
// (C: g_050f[c0b], a signed byte).
func (g *Game) tickOffset(tick int) (int, int) {
	t := int(int8(tick))
	switch {
	case t >= 0 && t < len(monsterTickOffset):
		return monsterTickOffset[t][0], monsterTickOffset[t][1]
	case t == -1:
		d := g.view.distanceColours
		return int(int8(d[len(d)-2])), int(int8(d[len(d)-1]))
	}
	return 0, 0
}

// drawMonster draws monster n, its equipment and the splat of a hit (C: draw_monster).
func (g *Game) drawMonster(n int) {
	m := &g.monsters[n]
	mt := m.Type
	if mt == nil {
		return
	}
	v := &g.view
	lv := &g.lvl
	set := mt.shapeIndex
	facing := monsterFacingFrames[g.direction][m.Facing&(directionCount-1)]
	frame := g.monsterFrame(m, facing&^mirroredFrames)
	flying := mt.flags&monsterFlies != 0
	ox, oy := g.tickOffset(m.Shift)
	var ref []byte
	if frame != noFrame {
		equip := m.Flags & monsterEquipMask
		flags := facing & mirroredFrames
		ref = g.monsterShape(set, frame)
		if mt.flags&monsterBlended != 0 {
			flags |= objBlend
		}
		var colours []byte
		if equip != 0 {
			pal := lv.monsterPalettes[set][frame&(monsterShapeCount-1)] // (C: f_2803_040a[c00][frame & 15])
			if at := shapeColourCount(ref) * (equip - 1); at < len(pal) {
				colours = pal[at:]
			}
		}
		fade := g.drawObject(ref, colours, m.X+ox, m.Y+oy, 0, 0, flags|1, -1, flying)
		for _, e := range m.Equipment {
			k := e - 1
			if k == -1 {
				break
			}
			if k >= monsterAnimations || frame >= monsterShapeCount {
				continue
			}
			// the original worked out the overlay's offsets and then drew it at the
			// monster's corner anyway
			if f := lv.monsterFrames[set][k][frame]; f.shape != nil {
				g.blitViewShape(f.shape, nil, v.x, v.y, flags|1, fade)
			}
		}
	} else {
		ref = lv.monsterShapes[set][0]
		g.placeObject(m.X+ox, m.Y+oy, ref, flying)
	}
	if m.Damage == 0 {
		return
	}
	dx := gfx.ScaledShapeWidth(ref, v.scaleX) >> 1
	dy := gfx.ScaledShapeHeight(ref, v.scaleY) >> 1
	var size int
	if m.Mode != monsterDead {
		if hit := m.Damage & monsterIndex; hit != 0 { // fix: a hit of 32768 divided by 0
			size = int(int16(mt.hitPoints)) / hit
		}
	} else {
		size = m.Counter << 1
	}
	colour := 0
	switch mt.flags & monsterBloodMask {
	case monsterBloodGreen:
		colour = 63
	case monsterBloodWhite:
		colour = 15
	case monsterBloodOther:
		colour = 74
	}
	var table []byte
	if colour != 0 {
		table = identityTable()
		for c := hitColoursFirst; c <= hitColoursLast; c++ {
			table[c] += byte(colour)
		}
	}
	dx += m.DX
	dy += m.DY
	size = max(min(size, maxHitDivisor), 1)
	steps := 0
	if colour != 0 {
		steps = 1
	}
	g.Screen.DrawShape(g.viewPage, g.gameShapes[gameShapeHit], v.x+dx, v.y+dy, viewWindowClip,
		gfx.ShapeScaled|gfx.ShapeCentred|gfx.ShapeFaded,
		gfx.ShapeOptions{Fade: table, FadeSteps: steps, ScaleX: v.scaleX / size, ScaleY: v.scaleY / size})
}

// monsterShape is shape frame of set; frames past the set's 16 are the next set's,
// as the original's table ran on.
func (g *Game) monsterShape(set, frame int) []byte {
	i := set*monsterShapeCount + frame
	if frame < 0 || i >= monsterShapeSets*monsterShapeCount {
		return nil
	}
	return g.lvl.monsterShapes[i/monsterShapeCount][i%monsterShapeCount]
}

func shapeColourCount(shape []byte) int {
	if len(shape) <= colourCountOffset {
		return 0
	}
	return int(shape[colourCountOffset])
}

// Item type flags the renderer reads.
const (
	itemFlying      = 1 << 12 // drawn with the thrown shapes
	itemBolt        = 1 << 11 // drawn as thrown shape 7
	itemGameShape   = 1 << 6  // drawn with a game shape
	thrownBolt      = 7
	itemHeightStep  = 6
	itemDepthOffset = 5
	itemSidewaysMul = 2
	alternateBit    = 1 << 5 // of x ^ y: which items use the alternate flags
)

// floorIcon is the item shape an item kind shows on the floor (C: f_27b1_004c[shape][0],
// the byte before portraitWeapons and each pair's first).
func floorIcon(shape int) int {
	const firstFloorIcon = 21 // (C: f_27b1_004c)
	if shape == 0 {
		return firstFloorIcon
	}
	return int(portraitWeapons[2*shape-1])
}

// itemShape is the shape item it shows on the floor.
func (g *Game) itemShape(it *Item) []byte {
	t := &g.itemTypes[it.Type]
	if t.Flags&itemGameShape != 0 {
		return shapeAt(g.gameShapes, int(t.Shape))
	}
	return shapeAt(g.itemShapes, floorIcon(int(t.Shape)))
}

func shapeAt(shapes [][]byte, i int) []byte {
	if i < 0 || i >= len(shapes) {
		return nil
	}
	return shapes[i]
}

// drawBlockObjects draws the items and monsters of slot n back to front, sorting them
// first when the party has turned (C: draw_block).
func (g *Game) drawBlockObjects(n int) {
	blk := &g.lvl.blocks[g.visibleBlocks[n]]
	if blk.Direction != g.direction {
		blk.DrawObjects = 0
		blk.Direction = g.direction
		for o := blk.Objects; o != 0; o = *g.objectNext(o) {
			g.insertDrawObject(g.direction, o, blk, true)
		}
	}
	o := blk.DrawObjects
	for o != 0 {
		if o&monsterFlag != 0 {
			o &= monsterIndex
			if n < nearSlots {
				g.drawMonster(o)
			}
			o = g.monsters[o].DrawNext
			continue
		}
		it := &g.items[o]
		x := itemOffsets[o&7] * itemSidewaysMul
		y := itemOffsets[o>>1&7] + itemDepthOffset
		if it.Sub < 2 || n < nearSlots {
			if it.Sub >= 2 {
				y -= (it.Sub - 1) * itemHeightStep
			}
			flags := 0
			var shape []byte
			t := &g.itemTypes[it.Type]
			if t.Flags&itemFlying != 0 && it.Flags&(itemOnFloor|itemFree) == 0 {
				views := int(t.Shape)
				if t.Flags&itemBolt != 0 {
					views = thrownBolt
				}
				tv := thrownViewAt(views)
				f := 0
				for f < flyingObjects && !(g.flying[f].Active && g.flying[f].Item == o) {
					f++
				}
				if tv.alternate != 0 && (it.X^it.Y)&alternateBit != 0 {
					flags |= tv.alternate
				}
				flags |= tv.flags
				if f != flyingObjects {
					var view int
					switch g.direction - g.flying[f].Dir>>1 {
					case 0:
						view = tv.away
					case -3, 1:
						view = tv.side
					case -1, 3:
						flags |= objFlipX
						view = tv.side
					case -2, 2:
						view = tv.towards
					}
					shape = shapeAt(g.thrownShapes, view)
				}
				y += gfx.ShapeHeight(shape) >> 2
			} else {
				shape = g.itemShape(it)
			}
			if shape != nil {
				g.drawObject(shape, nil, it.X, it.Y, x, y, flags, -1, false)
			}
		}
		o = it.DrawNext
	}
}

// thrownViewAt reads thrownViews; the original's table had no end check.
func thrownViewAt(i int) thrownView {
	if i < 0 || i >= len(thrownViews) {
		return thrownView{}
	}
	return thrownViews[i]
}

// drawSortKey orders a block's objects: lower first, then farthest first
// (C: fn_080e_01f3).
func (g *Game) drawSortKey(o, dir int) int {
	x, y, sub := g.objectPosition(o)
	_, y = g.toViewFacing(x, y, dir)
	const farthest = 1<<12 - 1
	return sub<<12 | (farthest - max(y, 0))
}

// objectPosition is an item's or monster's map position and height.
func (g *Game) objectPosition(o int) (x, y, sub int) {
	if o&monsterFlag != 0 {
		m := &g.monsters[o&monsterIndex]
		return m.X, m.Y, m.Sub
	}
	it := &g.items[o]
	return it.X, it.Y, it.Sub
}

// drawNextOf is the link to the object drawn after o.
func (g *Game) drawNextOf(o int) *int {
	if o&monsterFlag != 0 {
		return &g.monsters[o&monsterIndex].DrawNext
	}
	return &g.items[o].DrawNext
}

// insertDrawObject puts object o into block b's draw list by its sort key; a block
// sorted for another direction is marked to be sorted again (C: fn_080e_0245).
func (g *Game) insertDrawObject(dir, o int, b *Block, ascending bool) {
	if b.Direction != dir {
		b.Direction = noDirection
		return
	}
	key := g.drawSortKey(o, dir)
	list := &b.DrawObjects
	for *list != 0 {
		k := g.drawSortKey(*list, dir)
		if ascending && k >= key || !ascending && k > key {
			break
		}
		list = g.drawNextOf(*list)
	}
	*g.drawNextOf(o) = *list
	*list = o
}

// Block effects: the block flags 16 << k draw ice shape k on the floor or in the air
// (C: ovr_2cda_0066).
const (
	blockEffectFirst = 4
	blockEffects     = 4
	redEffect        = 3 // drawn through the red tint
	risingEffect     = 1 // drawn 20 units higher
	effectRise       = -20
)

// drawBlockEffects draws slot n's effects 2 and 3 (before its objects) or 0 and 1
// (after them).
func (g *Game) drawBlockEffects(n int, after bool) {
	b := g.visibleBlocks[n]
	fl := g.lvl.blocks[b].Flags
	if fl&((1<<blockEffects-1)<<blockEffectFirst) == 0 {
		return
	}
	k := 2
	if after {
		k = 0
	}
	for range 2 {
		if fl&(1<<(blockEffectFirst+k)) != 0 {
			x, y := turnInBlock(blockCentre, blockEffectY[k], g.direction)
			flags := objBlend
			var table []byte
			if k == redEffect {
				flags, table = objOverlayTable, g.redTint
			}
			x = (b&columnMask)<<byteShift | x
			y = (b>>5)<<byteShift | y
			dy := 0
			if k == risingEffect {
				dy = effectRise
			}
			g.drawObject(shapeAt(g.iceShapes, k), table, x, y, 0, dy, flags, -1, false)
		}
		k++
	}
}

// drawDecorations draws the wall decorations of slot n, back layer first
// (C: ovr_2cda_0043).
func (g *Game) drawDecorations(n int) {
	lv := &g.lvl
	var fade []byte
	for i := 1; i >= 0; i-- {
		si := n<<1 + i
		w, h := decorationScaleX[si], decorationScaleY[si]
		s := decorationView[si]
		c := abs(s)
		shade := min(shapeDistance[viewDepth[n]*5+4]+doorShadeExtra, darkestShade)
		if w == 0 || h == 0 {
			continue
		}
		side := (decorationSide[si] + g.direction) & (directionCount - 1)
		d := lv.wallShape[g.viewWall(n, side)]
		for d > 0 && d < len(lv.decorations) {
			deco := &lv.decorations[d]
			const (
				decoTwin      = 1 << 0 // drawn again mirrored beside itself
				decoMirrored  = 1 << 1
				decoParity    = 1 << 2 // mirrored on alternate blocks
				decoFrontOnly = 1 << 3 // only on the walls straight ahead
			)
			if deco.flags&decoFrontOnly != 0 && n != 3 && n != 9 && n != 13 {
				d = deco.next
				continue
			}
			if decorationViewBase[c] == 1 && (deco.flags&decoMirrored != 0 || deco.flags&decoParity != 0 && g.view.parity != 0) {
				s = -s
			}
			ox, oy := 0, 0
			if deco.viewFlags[c]&1 != 0 {
				ox, oy = deco.x[c], deco.y[c]
				c = decorationViewBase[c]
				fade = lv.fadeTables[shade]
			} else if deco.shapes[c] != noShape {
				w, h = fixedOne, fixedOne
				fade = lv.fadeTables[darkestShade]
			}
			if deco.shapes[c] != noShape {
				shape := shapeAt(lv.decorationShapes, deco.shapes[c])
				if shape != nil {
					var x, flags int
					x = decorationX[si] + fixedMul(deco.x[c], w) + ox
					if s >= 0 {
						flags = gfx.ShapeScaled | gfx.ShapeFaded
					} else {
						if decorationView[si] == s {
							x = decorationX[si] - fixedMul(deco.x[c], w)
							x -= gfx.ScaledShapeWidth(shape, w) + ox
						}
						flags = gfx.ShapeScaled | gfx.ShapeFaded | gfx.ShapeFlipX
					}
					y := decorationY[si] + fixedMul(deco.y[c], h) + oy
					opt := gfx.ShapeOptions{Fade: fade, FadeSteps: 1, ScaleX: w, ScaleY: h}
					g.Screen.DrawShape(g.viewPage, shape, x+viewLeft, y, viewWindowClip, flags, opt)
					const twinViews = 3
					if deco.flags&decoTwin != 0 && c >= 0 && c <= twinViews {
						x += gfx.ScaledShapeWidth(shape, w)
						flags ^= gfx.ShapeFlipX
						g.Screen.DrawShape(g.viewPage, shape, x+viewLeft, y, viewWindowClip, flags, opt)
					}
				}
			}
			d = deco.next
			c = abs(decorationView[si])
		}
	}
}

// drawSceneOverlay draws the scene overlay onto page (C: ovr_2cda_0057).
func (g *Game) drawSceneOverlay(page int) {
	o := &g.sceneOverlay
	if o.shape == nil {
		return
	}
	g.Screen.DrawShape(page, o.shape, o.x, o.y, viewWindow, 0, gfx.ShapeOptions{})
	if o.mirrored {
		g.Screen.DrawShape(page, o.shape, o.x+gfx.ShapeWidth(o.shape), o.y, viewWindow, gfx.ShapeFlipX, gfx.ShapeOptions{})
	}
}

// highlightFloorItems draws the items lying close by in the two slots by the party in
// one colour each, so they can be told apart (C: ovr_2cda_006b).
func (g *Game) highlightFloorItems() {
	const highlightReach = 320
	g.setupView(g.currentBlock, g.direction)
	g.Screen.FillRect(viewLeft, viewTop, viewRight, viewBottom-1, 0)
	for i, n := range [...]int{13, 16} { // (C: g_06e9)
		g.clipColumns(n, viewWindowClip)
		o := g.lvl.blocks[g.visibleBlocks[n]].DrawObjects
		colour := i<<7 + 1
		for o != 0 {
			if o&monsterFlag != 0 {
				o = g.monsters[o&monsterIndex].DrawNext
				continue
			}
			it := &g.items[o]
			if it.Flags&itemOnFloor != 0 && distance(it.X, it.Y, g.partyX, g.partyY) < highlightReach {
				x := itemOffsets[o&7] * itemSidewaysMul // (C: g_06e1, the same as g_053f)
				y := itemOffsets[o>>1&7] + itemDepthOffset
				if it.Sub >= 2 {
					y -= (it.Sub - 1) * itemHeightStep
				}
				g.drawObject(g.itemShape(it), nil, it.X, it.Y, x, y, 0, colour, false)
			}
			o = it.DrawNext
			colour++
		}
	}
}

// The compass in the top right: the rose for the language and a needle per 32nd of a
// turn, which swings towards the party's direction (C: ovr_2cda_004d, ovr_2cda_0052).
const (
	gameShapeCompass       = 22 // plus the language (C: f_2803_00fe)
	gameShapeNeedle        = 25 // (C: f_2803_010a)
	compassRoseX           = 294
	compassRoseY           = 3
	needleShadowX          = 298
	needleShadowY          = 9
	needleX                = 299
	needleY                = 8
	compassLeft            = 293
	compassTop             = 5
	compassRight           = 316
	compassBottom          = 24
	compassUnitsPerQuarter = 64
	compassTurn            = 4 * compassUnitsPerQuarter
	compassNeedles32       = 32
	compassSwingTicks      = 3
	maxCompassSwing        = 24
	updateGameMenu         = 1 << 2 // the game menu is open (C: g_7fd6 & 4)
)

// drawCompass draws the compass needle if it has moved (C: ovr_2cda_004d).
func (g *Game) drawCompass() {
	if !g.flag(flagHasCompass) {
		return
	}
	if g.compassDirection == noCompass {
		g.compassNeedle = noCompass
		g.compassDirection = g.direction * compassUnitsPerQuarter
	}
	n := (g.compassDirection + 4) >> 3 & (compassNeedles32 - 1)
	if n == g.compassNeedle {
		return
	}
	g.compassNeedle = n
	s := g.Screen
	if s.Current == pageScreen {
		s.HideMouseIn(compassLeft, compassTop, compassRight, compassBottom)
	}
	s.DrawShape(s.Current, shapeAt(g.gameShapes, gameShapeCompass+g.language), compassRoseX, compassRoseY, 0, 0, gfx.ShapeOptions{})
	needle := compassNeedles[n]
	shape := shapeAt(g.gameShapes, gameShapeNeedle+needle.shape)
	s.DrawShape(s.Current, shape, needle.x+needleShadowX, needle.y+needleShadowY, 0,
		needle.flags|gfx.ShapeFaded|gfx.ShapePredator, gfx.ShapeOptions{Fade: g.shadowTable, FadeSteps: 1})
	s.DrawShape(s.Current, shape, needle.x+needleX, needle.y+needleY, 0, needle.flags, gfx.ShapeOptions{})
	if s.Current == pageScreen {
		s.ShowMouseIn()
	}
}

// swingCompass moves the needle a step towards the party's direction, or anywhere
// when the compass is broken (C: ovr_2cda_0052).
func (g *Game) swingCompass() {
	if !g.flag(flagHasCompass) || g.updateFlags&updateGameMenu != 0 {
		return
	}
	if g.compassDirection == noCompass {
		g.compassSwing = 0
		g.drawCompass()
		return
	}
	now := clock.Ticks()
	if g.compassNextTick >= now {
		return
	}
	if g.direction*compassUnitsPerQuarter == g.compassDirection && g.compassSwing == 0 {
		return
	}
	g.compassNextTick = now + compassSwingTicks
	sign := 1
	if g.compassSwing < 0 {
		sign = -1
	}
	if g.compassSwing != 0 {
		g.compassSwing -= (abs(g.compassSwing)>>4 + 2) * sign
	}
	var dx int
	if g.compassBroken != 0 {
		dx = random.Byte() - g.compassDirection
	} else {
		dx = g.direction*compassUnitsPerQuarter - g.compassDirection
	}
	if dx <= -compassTurn/2 {
		dx += compassTurn
	}
	if dx >= compassTurn/2 {
		dx -= compassTurn
	}
	dx >>= 2
	g.compassSwing = max(min(g.compassSwing+dx, maxCompassSwing), -maxCompassSwing)
	g.compassDirection += g.compassSwing
	if g.compassDirection < 0 {
		g.compassDirection += compassTurn
	}
	if g.compassDirection >= compassTurn {
		g.compassDirection -= compassTurn
	}
	if g.compassDirection/compassUnitsPerQuarter == g.direction && g.compassSwing < 2 && abs(dx) < 4 {
		g.compassDirection = g.direction * compassUnitsPerQuarter
		g.compassSwing = 0
	}
	g.drawCompass()
}

// Distances past the ends of rowY and scaleByY read on into the tables after them, as
// the original did (rowY, rowScale, viewClipTop and viewClipBottom lie one after the
// other in DGROUP; wallShapeY, scaleByY and thrownViews in the far data).
var (
	rowBytes   = concat(rowY[:], rowScale[:], viewClipTop[:], viewClipBottom[:])
	scaleWords = concat(wallShapeY[:], scaleByY[:], flatThrownViews())
)

func concat(parts ...[]int) []int {
	var all []int
	for _, p := range parts {
		all = append(all, p...)
	}
	return all
}

func flatThrownViews() []int {
	var all []int
	for _, t := range thrownViews {
		all = append(all, t.towards, t.away, t.side, t.flags, t.alternate)
	}
	return all
}

// floorLineY is the floor line's y r steps of 32 ahead (C: g_00f0[r]).
func floorLineY(r int) int { return tableAt(rowBytes, r) }

// sideStepScale is the scale of a step sideways r steps ahead (C: g_010e[r]).
func sideStepScale(r int) int { return tableAt(rowBytes, len(rowY)+r) }

// scaleAtY is the scale of a shape whose floor line is at y (C: f_24dc_0000[y - 56]).
func scaleAtY(y int) int {
	const firstY = 56
	return tableAt(scaleWords, len(wallShapeY)+y-firstY)
}

func tableAt(t []int, i int) int {
	if i < 0 || i >= len(t) {
		return 0
	}
	return t[i]
}
