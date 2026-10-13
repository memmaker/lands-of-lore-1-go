package game

import "github.com/memmaker/lands-of-lore-1-go/internal/gfx"

// The view blitters of the walk and turn animations (C: s038 fn_19c9_*). A page holds
// the view at viewLeft with the screen's stride; a buffer holds it unstrided.
const (
	viewWidth  = 176
	viewHeight = 120

	zoomSplitRow  = 46 // the zoom treats the rows above and below the horizon apart
	zoomShift     = 8  // the zoom steps are fractions of 256
	fractionCarry = 1 << 16
	overlayRow    = 97 // the scene overlay's rows in the view
	overlayRows   = 23
	overlayColumn = 19
	overlayHalf   = 69 // the overlay is mirrored around its right edge
)

// bufferToView copies an unstrided view to the page (C: fn_19c9_095d).
func bufferToView(buf, page []byte) {
	for y := range viewHeight {
		copy(page[y*gfx.Width+viewLeft:][:viewWidth], buf[y*viewWidth:])
	}
}

// viewToBuffer copies the page's view out unstrided; buf may be the same page
// (C: fn_19c9_09a4).
func viewToBuffer(page, buf []byte) {
	for y := range viewHeight {
		copy(buf[y*viewWidth:][:viewWidth], page[y*gfx.Width+viewLeft:])
	}
}

// copyView copies the view from one page to another (C: fn_19c9_09cf).
func copyView(src, dst []byte) {
	for y := range viewHeight {
		at := y*gfx.Width + viewLeft
		copy(dst[at:at+viewWidth], src[at:])
	}
}

// zoomViewTop draws the rows above the horizon of an unstrided view zoomed in: margin
// columns are cut from each side and rows rows from the top, and the rest is stretched
// back to full size (C: fn_19c9_07e2).
func zoomViewTop(src, dst []byte, margin, rows int) {
	zoomView(src, dst, margin+rows*viewWidth, 0, margin, zoomSplitRow-rows, rows)
}

// zoomViewBottom is zoomViewTop for the rows from the horizon down, cut at the bottom
// (C: fn_19c9_089f). It draws one row more than the view has (sic).
func zoomViewBottom(src, dst []byte, margin, rows int) {
	zoomView(src, dst, margin+zoomSplitRow*viewWidth, zoomSplitRow*viewWidth, margin, viewHeight-zoomSplitRow-rows, rows)
}

// zoomView stretches srcRows rows of 176-2*margin pixels from src at si to full rows
// at di in dst. Each step's fraction is added up in 16 bits and a carry repeats the
// pixel or the row; the column fraction restarts each row and a repeated row restarts
// the row fraction instead of keeping what was left (sic).
func zoomView(src, dst []byte, si, di, margin, srcRows, cut int) {
	width := viewWidth - 2*margin
	colStep := ((2*margin + 1) << zoomShift / width) << zoomShift
	rowStep := ((cut + 1) << zoomShift / srcRows) << zoomShift
	rowFraction := 0
	for srcRows > 0 {
		colFraction := 0
		for range width - 1 {
			colFraction += colStep
			n := 1
			if colFraction >= fractionCarry {
				colFraction -= fractionCarry
				n++
			}
			p := src[si]
			si++
			for ; n > 0; n-- {
				dst[di] = p
				di++
			}
		}
		dst[di] = src[si]
		di++
		si++
		si += 2 * margin
		rowFraction += rowStep
		if rowFraction >= fractionCarry {
			si -= viewWidth
			rowFraction = 0
			continue
		}
		srcRows--
	}
}

// The turn frames build a full view in buf from the edge of one view stretched and
// the rest of the other squeezed (C: fn_19c9_0a01, fn_19c9_0a61, fn_19c9_0ab1).
const (
	turnEdgeColumns = 15 // the edge stretched to 44 pixels: one doubled, the rest tripled
	turnEdgeWidth   = 44
	turnHalfColumns = 44 // in the half frame both are doubled
	turnRestColumns = 99 // squeezed to 132: one in three doubled
)

// stretchThirds copies 15 pixels as 44: the first doubled, the rest tripled; with
// last the one doubled is the last instead.
func stretchEdge(dst, src []byte, last bool) {
	d := 0
	put := func(p byte, n int) {
		for range n {
			dst[d] = p
			d++
		}
	}
	for i := range turnEdgeColumns {
		n := 3
		if (i == 0 && !last) || (i == turnEdgeColumns-1 && last) {
			n = 2
		}
		put(src[i], n)
	}
}

// stretchRest copies 99 pixels as 132: of each three, the third is doubled.
func stretchRest(dst, src []byte) {
	d := 0
	for i := 0; i < turnRestColumns; i += 3 {
		dst[d], dst[d+1], dst[d+2], dst[d+3] = src[i], src[i+1], src[i+2], src[i+2]
		d += 4
	}
}

// doubleColumns copies n pixels each twice.
func doubleColumns(dst, src []byte, n int) {
	for i := range n {
		dst[2*i], dst[2*i+1] = src[i], src[i]
	}
}

// turnQuarter: a's right edge stretched on the left, b's left part on the right
// (C: fn_19c9_0a01).
func turnQuarter(a, b, buf []byte) {
	for y := range viewHeight {
		row := buf[y*viewWidth:]
		stretchEdge(row, a[y*gfx.Width+viewLeft+viewWidth-turnEdgeColumns:], false)
		stretchRest(row[turnEdgeWidth:], b[y*gfx.Width+viewLeft:])
	}
}

// turnHalf: a's right quarter and b's left quarter, both doubled (C: fn_19c9_0a61).
func turnHalf(a, b, buf []byte) {
	for y := range viewHeight {
		row := buf[y*viewWidth:]
		doubleColumns(row, a[y*gfx.Width+viewLeft+viewWidth-turnHalfColumns:], turnHalfColumns)
		doubleColumns(row[2*turnHalfColumns:], b[y*gfx.Width+viewLeft:], turnHalfColumns)
	}
}

// turnThreeQuarters: a's right part squeezed on the left, b's left edge stretched on
// the right (C: fn_19c9_0ab1).
func turnThreeQuarters(a, b, buf []byte) {
	for y := range viewHeight {
		row := buf[y*viewWidth:]
		stretchRest(row, a[y*gfx.Width+viewLeft+viewWidth-turnRestColumns:])
		stretchEdge(row[viewWidth-turnEdgeWidth:], b[y*gfx.Width+viewLeft:], true)
	}
}

// copyViewColumns copies 2*w pixels of each view row from src at sx to dst at dx
// (C: fn_19c9_0b15).
func copyViewColumns(src []byte, sx int, dst []byte, dx, w int) {
	for y := range viewHeight {
		at := y*gfx.Width + viewLeft
		copy(dst[at+dx:at+dx+2*w], src[at+sx:])
	}
}

// slideView moves 2*w pixels of each view row from x to nx, through the start of the
// page row, which it overwrites (C: fn_19c9_0b55).
func slideView(page []byte, x, nx, w int) {
	for y := range viewHeight {
		row := page[y*gfx.Width:]
		copy(row[:2*w], row[viewLeft+x:])
		copy(row[viewLeft+nx:viewLeft+nx+2*w], row[:2*w])
	}
}

// captureSceneOverlay draws the scene overlay's left half alone on page and keeps its
// rows for drawOverlayRows; false without an overlay (C: ovr_2cda_005c,
// fn_19c9_0730, fn_19c9_075b). Each row is kept from the pixel after its first one
// that is not 0 (sic: that one is lost).
func (g *Game) captureSceneOverlay(page int) bool {
	o := &g.sceneOverlay
	if o.shape == nil {
		return false
	}
	p := g.Screen.Page(page)
	for y := overlayRow; y < overlayRow+overlayRows; y++ {
		clear(p[y*gfx.Width+viewLeft:][:viewWidth])
	}
	g.Screen.DrawShape(page, o.shape, o.x, o.y, viewWindow, 0, gfx.ShapeOptions{})
	g.overlayRows = g.overlayRows[:0]
	for y := overlayRow; y < overlayRow+overlayRows; y++ {
		row := p[y*gfx.Width+viewLeft+overlayColumn:]
		skip := 1
		for skip <= overlayHalf && row[skip-1] == 0 {
			skip++
		}
		if skip > overlayHalf {
			skip = overlayHalf // fix: the original ran on into the next row
		}
		g.overlayRows = append(g.overlayRows, overlayLine{skip, append([]byte(nil), row[skip:overlayHalf]...)})
	}
	return true
}

// overlayLine is one row of the captured overlay: where it starts and its pixels.
type overlayLine struct {
	skip   int
	pixels []byte
}

// drawOverlayRows draws the captured overlay and its mirror image into an unstrided
// view (C: fn_19c9_079a).
func (g *Game) drawOverlayRows(buf []byte) {
	for i, l := range g.overlayRows {
		row := buf[(overlayRow+i)*viewWidth+overlayColumn+l.skip:]
		n := copy(row, l.pixels)
		for j := range n {
			row[n+j] = l.pixels[n-1-j]
		}
	}
}
