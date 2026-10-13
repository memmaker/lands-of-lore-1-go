package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// Party movement (C: s009 fn_09aa_*). A step runs the scripts of the block left and
// of the block entered, each twice: before the walk animation and after it.
const (
	eventEnter  = 1 << 0 // about to step in
	eventArrive = 1 << 1 // stepped in
	eventLeave  = 1 << 2 // about to step out
	eventLeft   = 1 << 3 // stepped out
	eventTurn   = 1 << 4 // the party turned on the block
	eventTurned = 1 << 14

	// the movement icons (C: the key codes of the buttons)
	iconTurnLeft    = 79
	iconForward     = 80
	iconTurnRight   = 81
	iconStrafeLeft  = 82
	iconBack        = 83
	iconStrafeRight = 84

	wallVanishing     = 26 // a wall type that goes once the party stands in its block
	stringCannotGo    = 0x403f
	soundBump         = 19
	walkFrameTicks    = 2
	turnFrameTicks    = 1
	blockCentreSide   = 128
	optionSmoothMoves = 1 << 3
)

// How a step animates.
const (
	walkForward = iota
	walkBack
	walkLeft
	walkRight
)

// fastMachine is true where the original found a 486 or better (C: g_1f3a); on slower
// machines the animations dropped every other frame.
const fastMachine = true

// buttonFromView says whether a button is one of the invisible ones over the view,
// which only work with arrow cursors on (C: w38).
func (g *Game) ignoresViewButton(b *gui.Button) bool {
	return b.Arg != 0 && g.options&optionArrowCursors == 0
}

// The C passed the move functions a second argument from the button's state (right
// button and key, 0x1080) that canEnter never looked at; it is left out.

// clickMoveForward (C: fn_09aa_0197).
func (g *Game) clickMoveForward(b *gui.Button) bool {
	if g.ignoresViewButton(b) {
		return false
	}
	g.moveParty(g.direction, walkForward, iconForward)
	return true
}

// clickMoveBack (C: fn_09aa_01e4).
func (g *Game) clickMoveBack(b *gui.Button) bool {
	if g.ignoresViewButton(b) {
		return false
	}
	g.moveParty(g.direction^2, walkBack, iconBack)
	return true
}

// clickStrafeLeft (C: fn_09aa_0219).
func (g *Game) clickStrafeLeft(b *gui.Button) bool {
	if g.ignoresViewButton(b) {
		return false
	}
	g.moveParty((g.direction-1)&(directionCount-1), walkLeft, iconStrafeLeft)
	return true
}

// clickStrafeRight (C: fn_09aa_026a).
func (g *Game) clickStrafeRight(b *gui.Button) bool {
	if g.ignoresViewButton(b) {
		return false
	}
	g.moveParty((g.direction+1)&(directionCount-1), walkRight, iconStrafeRight)
	return true
}

// clickTurnLeft (C: fn_09aa_02bb).
func (g *Game) clickTurnLeft(b *gui.Button) bool {
	if g.ignoresViewButton(b) {
		return false
	}
	g.setIcon(iconTurnLeft, iconPress)
	g.direction = (g.direction - 1) & (directionCount - 1)
	g.walkTransition = 1
	g.runBlockScript(g.currentBlock, eventTurned)
	g.fadeMessage(2, false)
	if g.walkTransition != 0 {
		g.animateTurnLeft(turnFrameTicks)
	} else {
		g.drawScene(pageScreen)
	}
	g.setIcon(iconTurnLeft, iconRelease)
	g.runBlockScript(g.currentBlock, eventTurn)
	return true
}

// clickTurnRight (C: fn_09aa_033f). It releases the icon before the animation, the
// left turn after it (sic).
func (g *Game) clickTurnRight(b *gui.Button) bool {
	if g.ignoresViewButton(b) {
		return false
	}
	g.setIcon(iconTurnRight, iconPress)
	g.direction = (g.direction + 1) & (directionCount - 1)
	g.walkTransition = 1
	g.runBlockScript(g.currentBlock, eventTurned)
	g.fadeMessage(2, false)
	g.setIcon(iconTurnRight, iconRelease)
	if g.walkTransition != 0 {
		g.animateTurnRight(turnFrameTicks)
	} else {
		g.drawScene(pageScreen)
	}
	g.runBlockScript(g.currentBlock, eventTurn)
	return true
}

// moveParty steps the party one block towards dir if it can, running the blocks'
// scripts and the walk animation of kind (C: fn_09aa_0458).
func (g *Game) moveParty(dir, kind, icon int) {
	if icon != 0 {
		g.setIcon(icon, iconPress)
	}
	old := g.currentBlock
	next := g.blockInDirection(old, dir)
	if !g.canEnter(next, dir) {
		g.bumpWall(kind == walkForward)
		g.setIcon(icon, iconRelease)
		return
	}
	g.lastMoveDirection = dir
	g.currentBlock = next
	g.walkTransition = 1
	g.partyX, g.partyY = blockPosition(g.currentBlock, blockCentreSide, blockCentreSide)
	g.setFlag(flagMovedThisTurn, false)
	g.runBlockScript(old, eventLeave)
	g.runBlockScript(next, eventEnter)
	if !g.flag(flagMovedThisTurn) {
		g.fadeMessage(2, false)
		if g.walkTransition != 0 {
			switch kind {
			case walkForward:
				g.animateWalkForward(walkFrameTicks)
			case walkBack:
				g.animateWalkBack(walkFrameTicks)
			case walkLeft:
				g.animateStrafeLeft(turnFrameTicks)
			case walkRight:
				g.animateStrafeRight(turnFrameTicks)
			}
		} else {
			g.drawScene(pageScreen)
		}
		g.setIcon(icon, iconRelease)
		if g.currentBlock == next {
			g.runBlockScript(old, eventLeft)
			g.runBlockScript(next, eventArrive)
			b := &g.lvl.blocks[next]
			if b.Walls[0] == wallVanishing {
				b.Walls = [directionCount]int{}
			}
		}
	}
	g.revealAround(g.currentBlock)
}

// bumpWall is the party walking into a wall: with forward the view bounces
// (C: fn_09aa_03c3).
func (g *Game) bumpWall(forward bool) {
	g.flushInput()
	if forward {
		g.animateBump(walkFrameTicks)
	}
	g.speechKey(1)
	g.printMessage(messageSilent|2, g.text(stringCannotGo))
	g.playSound(soundBump, 0)
}

// turnTo turns the party towards dir the short way round (C: fn_130a_0001).
func (g *Game) turnTo(dir int) {
	d := dir - g.direction
	if abs(d) > 2 {
		d = -(d ^ 2)
	}
	for d != 0 {
		if d > 0 {
			g.clickTurnRight(&scriptButton)
			d--
		} else {
			g.clickTurnLeft(&scriptButton)
			d++
		}
	}
}

// scriptButton is what scripts press the movement buttons with (C: f_27b1_0010).
var scriptButton gui.Button

// Party moves a script asks for (C: fn_0fbe_0168).
const (
	scriptMoveRelative = 6  // 6-9: forward, right, back, left of the way the party faces
	scriptTurnTo       = 10 // 10-13: turn to face north, east, south, west
)

// opMoveParty: 0 forward, 1 strafe right, 2 back, 3 strafe left, 4 turn left, 5 turn
// right, 6-9 those of a direction, 10-13 turn to a direction (C: fn_0fbe_0168).
func (g *Game) opMoveParty(s *emc.State) int {
	d := s.Arg(0)
	if d >= scriptMoveRelative && d < scriptTurnTo {
		d = (d - scriptMoveRelative - g.direction) & (directionCount - 1)
	}
	switch d {
	case 0:
		g.clickMoveForward(&scriptButton)
	case 1:
		g.clickStrafeRight(&scriptButton)
	case 2:
		g.clickMoveBack(&scriptButton)
	case 3:
		g.clickStrafeLeft(&scriptButton)
	case 4:
		g.clickTurnLeft(&scriptButton)
	case 5:
		g.clickTurnRight(&scriptButton)
	case 10, 11, 12, 13:
		g.turnTo(d - scriptTurnTo)
	}
	return 1
}

// The walk and turn animations (C: s011 fn_0acb_*). They only play with the smooth
// moves option, and never in a scene. They work on 176x120 views kept without the
// page's stride in pages 2 and 6, and in a buffer at the end of page 2.
const (
	viewBufferAt  = 42240        // (C: 0xa500)
	pageOtherView = portraitPage // the view page besides pageBuild (C: 6)
	walkFrames    = 5
	bumpFrames    = 2
)

var (
	zoomColumns    = [walkFrames]int{5, 10, 15, 20, 25} // (C: g_2270)
	zoomTopRows    = [walkFrames]int{2, 4, 7, 9, 12}    // (C: g_227a)
	zoomBottomCols = [walkFrames]int{5, 10, 15, 20, 25} // (C: g_2284)
	zoomBottomRows = [walkFrames]int{4, 8, 12, 16, 20}  // (C: g_228e)
)

// smoothMoves says whether the animations play (C: g_7fda & 8, fn_0acb_000f).
func (g *Game) smoothMoves(inScene bool) bool {
	return g.options&optionSmoothMoves != 0 && !(inScene && g.sceneOn)
}

// waitFrame waits for the countdown, keeping the speaking face going.
func (g *Game) waitFrame(face bool) {
	for clock.Get(clock.Countdown) != 0 {
		if face {
			g.updateSpeechFace()
		}
		host.Yield()
	}
}

// zoomFrame draws step i of the zoom from view src into dst, both unstrided.
func (g *Game) zoomFrame(src, dst []byte, i int) {
	zoomViewTop(src, dst, zoomColumns[i], zoomTopRows[i])
	zoomViewBottom(src, dst, zoomBottomCols[i], zoomBottomRows[i])
}

// animateWalkForward zooms into the old view, then shows the new one
// (C: fn_0acb_001e).
func (g *Game) animateWalkForward(ticks int) {
	if !g.smoothMoves(true) {
		return
	}
	s := g.Screen
	pg2, pg6, pg0 := s.Page(pageBuild), s.Page(pageOtherView), s.Page(pageScreen)
	buf := pg2[viewBufferAt:]
	var overlay bool
	if g.viewBackPage == pageBuild {
		overlay = g.captureSceneOverlay(pageOtherView)
		g.drawScene(pageOtherView)
		viewToBuffer(pg6, buf)
		viewToBuffer(pg2, pg6)
	} else {
		overlay = g.captureSceneOverlay(pageBuild)
		g.drawScene(pageBuild)
		viewToBuffer(pg2, buf)
		viewToBuffer(pg6, pg6)
	}
	g.hideMouseInWindow(viewWindow)
	for i := 0; i < walkFrames; i++ {
		clock.Set(clock.Countdown, ticks)
		g.zoomFrame(pg6, pg2, i)
		if overlay {
			g.drawOverlayRows(pg2)
		}
		bufferToView(pg2, pg0)
		g.waitFrame(true)
		if !fastMachine {
			i++
		}
	}
	if overlay {
		g.drawOverlayRows(buf)
	}
	if g.walkTransition != 2 {
		bufferToView(buf, pg0)
	}
	s.ShowMouseIn()
	g.copyViewFrame()
}

// animateBump zooms a little into the view and back (C: fn_0acb_01b4).
func (g *Game) animateBump(ticks int) {
	if !g.smoothMoves(true) {
		return
	}
	s := g.Screen
	pg2, pg6, pg0 := s.Page(pageBuild), s.Page(pageOtherView), s.Page(pageScreen)
	if g.viewBackPage == pageBuild {
		viewToBuffer(pg2, pg6)
	} else {
		viewToBuffer(pg6, pg6)
	}
	g.hideMouseInWindow(viewWindow)
	i := 0
	for ; i < bumpFrames; i++ {
		clock.Set(clock.Countdown, ticks)
		g.zoomFrame(pg6, pg2, i)
		bufferToView(pg2, pg0)
		g.waitFrame(false)
		if !fastMachine {
			i++
		}
	}
	for i > 0 {
		if !fastMachine {
			i--
		}
		clock.Set(clock.Countdown, ticks)
		g.zoomFrame(pg6, pg2, i) // the way back starts a step deeper than the way in went
		bufferToView(pg2, pg0)
		g.waitFrame(false)
		i--
	}
	if g.walkTransition != 2 {
		bufferToView(pg6, pg0)
	}
	s.ShowMouseIn()
	g.copyViewFrame()
}

// animateWalkBack draws the new view and zooms out of it (C: fn_0acb_032e).
func (g *Game) animateWalkBack(ticks int) {
	if g.options&optionSmoothMoves == 0 {
		return
	}
	s := g.Screen
	pg2, pg6, pg0 := s.Page(pageBuild), s.Page(pageOtherView), s.Page(pageScreen)
	overlay := g.captureSceneOverlay(pageBuild)
	g.drawScene(pageBuild)
	viewToBuffer(pg2, pg6)
	g.hideMouseInWindow(viewWindow)
	for i := walkFrames - 1; i >= 0; i-- {
		clock.Set(clock.Countdown, ticks)
		g.zoomFrame(pg6, pg2, i)
		if overlay {
			g.drawOverlayRows(pg2)
		}
		bufferToView(pg2, pg0)
		g.waitFrame(true)
		if !fastMachine {
			i--
		}
	}
	if overlay {
		g.drawOverlayRows(pg6)
	}
	if g.walkTransition != 2 {
		bufferToView(pg6, pg0)
	}
	s.ShowMouseIn()
	g.copyViewFrame()
}

// turnPages returns the old view's page, the new one's and the buffer, and draws
// the new view (C: the start of fn_0acb_0461 and fn_0acb_0604).
func (g *Game) turnPages() (old, new, buf []byte, overlay bool) {
	s := g.Screen
	old, new = s.Page(g.viewBackPage), s.Page(g.viewPage)
	if g.viewBackPage == pageBuild {
		buf = old[viewBufferAt:]
	} else {
		buf = new[viewBufferAt:]
	}
	overlay = g.captureSceneOverlay(g.viewPage)
	g.drawScene(g.viewPage)
	g.hideMouseInWindow(viewWindow)
	return old, new, buf, overlay
}

// turnFrame shows one frame of a turn built into buf.
func (g *Game) turnFrame(ticks int, buf []byte, overlay bool, build func()) {
	clock.Set(clock.Countdown, ticks)
	build()
	if overlay {
		g.drawOverlayRows(buf)
	}
	bufferToView(buf, g.Screen.Page(pageScreen))
	g.waitFrame(true)
}

// endTurn shows the new view unless a script asked otherwise.
func (g *Game) endTurn(new []byte) {
	if g.walkTransition != 2 {
		g.drawSceneOverlay(g.viewPage)
		copyView(new, g.Screen.Page(pageScreen))
	}
	g.Screen.ShowMouseIn()
}

// animateTurnLeft swings the new view in from the left (C: fn_0acb_0461).
func (g *Game) animateTurnLeft(ticks int) {
	if g.options&optionSmoothMoves == 0 {
		return
	}
	ticks *= 2
	old, new, buf, overlay := g.turnPages()
	g.turnFrame(ticks, buf, overlay, func() { turnQuarter(new, old, buf) })
	g.turnFrame(ticks, buf, overlay, func() { turnHalf(new, old, buf) })
	g.turnFrame(ticks, buf, overlay, func() { turnThreeQuarters(new, old, buf) })
	g.endTurn(new)
}

// animateTurnRight swings the new view in from the right (C: fn_0acb_0604).
func (g *Game) animateTurnRight(ticks int) {
	if g.options&optionSmoothMoves == 0 {
		return
	}
	ticks *= 2
	old, new, buf, overlay := g.turnPages()
	g.turnFrame(ticks, buf, overlay, func() { turnThreeQuarters(old, new, buf) })
	g.turnFrame(ticks, buf, overlay, func() { turnHalf(old, new, buf) })
	g.turnFrame(ticks, buf, overlay, func() { turnQuarter(old, new, buf) })
	g.endTurn(new)
}

// strafePages draws the new view and swaps the view pages; it returns the old view's
// page and the new one's (C: the start of fn_0acb_07a7 and fn_0acb_089c).
func (g *Game) strafePages() (old, new []byte) {
	s := g.Screen
	old, new = s.Page(g.viewBackPage), s.Page(g.viewPage)
	g.drawScene(g.viewPage)
	g.viewPage, g.viewBackPage = g.viewBackPage, g.viewPage
	g.hideMouseInWindow(viewWindow)
	return old, new
}

// The strafe steps: how far the old view slides and how much of the new one shows.
const (
	strafeStart = 88
	strafeStep  = 22
	strafeEnd   = 22
)

// animateStrafeLeft slides the new view in from the left (C: fn_0acb_07a7).
func (g *Game) animateStrafeLeft(ticks int) {
	if g.options&optionSmoothMoves == 0 {
		return
	}
	ticks *= 2
	old, new := g.strafePages()
	screen := g.Screen.Page(pageScreen)
	for x, y := strafeStart, strafeStart; x > strafeEnd; x, y = x-strafeStep, y+strafeStep {
		clock.Set(clock.Countdown, ticks)
		slideView(old, 66, y, x/2)
		copyViewColumns(new, x, old, 0, y/2)
		copyView(old, screen)
		g.waitFrame(true)
	}
	if g.walkTransition != 2 {
		copyView(new, screen)
	}
	g.Screen.ShowMouseIn()
}

// animateStrafeRight slides the new view in from the right (C: fn_0acb_089c).
func (g *Game) animateStrafeRight(ticks int) {
	if g.options&optionSmoothMoves == 0 {
		return
	}
	ticks *= 2
	old, new := g.strafePages()
	screen := g.Screen.Page(pageScreen)
	frame := func(build func()) {
		clock.Set(clock.Countdown, ticks)
		build()
		copyView(old, screen)
		g.waitFrame(true)
	}
	frame(func() { copyViewColumns(new, 0, old, 110, 33) })
	frame(func() {
		slideView(old, 22, 0, 33)
		copyViewColumns(new, 0, old, 88, 44)
	})
	frame(func() {
		slideView(old, 44, 0, 11)
		copyViewColumns(new, 0, old, 66, 55)
	})
	if g.walkTransition != 2 {
		copyView(new, screen)
	}
	g.Screen.ShowMouseIn()
}
