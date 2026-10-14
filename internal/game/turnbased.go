package game

import "github.com/memmaker/lands-of-lore-1-go/internal/clock"

// Turn-based play, an addition of the port: Return switches between it and the
// original real time. In turn-based play the play clock (which drives the monsters,
// doors, missiles, recoveries, poison and the lamp) stands still until the party acts;
// each action then lets it run for one turn while further input waits.
//
// A turn is turnTicks of the play clock, the time a monster of the usual pace (thinking
// every other 6-tick round and stepping 32 of a block's 256 units) takes to walk one
// block, so the monsters keep pace with a party taking a step a turn. A step, an
// attack, a spell, using an item on a character or waiting with Z each take a turn;
// turning on the spot takes half of one. A character's recovery after an attack still
// runs on the play clock, about two turns, so the party's members take turns to hit.
const (
	turnTicks     = 96
	turnHalfTicks = turnTicks / 2
	keyWaitZ      = 46
)

const (
	turnBasedText = "Turn-based: each action takes a turn (Z waits)."
	realTimeText  = "Real time."
)

// turnState is where the play clock stops in turn-based play.
type turnState struct {
	on  bool
	end int // the play clock's tick the running turn ends at
}

// toggleTurnBased switches between turn-based play and real time.
func (g *Game) toggleTurnBased() {
	g.turns.on = !g.turns.on
	if g.turns.on {
		g.turns.end = clock.Get(playClock)
		g.printMessage(messageEffect, turnBasedText)
	} else {
		clock.Enable(playClockSlot)
		g.printMessage(messageEffect, realTimeText)
	}
}

// spendTurn lets the play clock run for ticks more in turn-based play.
func (g *Game) spendTurn(ticks int) {
	if !g.turns.on {
		return
	}
	g.turns.end = max(g.turns.end, clock.Get(playClock)) + ticks
	clock.Enable(playClockSlot)
}

// turnRunning stops the play clock once the turn is over, and says whether one is
// still running; menus and scenes switch the clock on again as they close.
func (g *Game) turnRunning() bool {
	if !g.turns.on {
		return false
	}
	if clock.Get(playClock) < g.turns.end {
		clock.Enable(playClockSlot)
		return true
	}
	clock.Disable(playClockSlot)
	return false
}
