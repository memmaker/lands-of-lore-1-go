package game

// Doors: a door's wall types are its steps from closed to open; a moving door steps
// one type on each tick of the doors timer until it reaches a type flagged as an end
// (C: g_4ed0, fn_0bbc_004a, fn_13e8_0358, fn_13e8_03e9).
const (
	movingDoors     = 3
	wallBlocksWay   = 1 << 0
	wallDoorOpen    = 1 << 4 // the open end of a door's steps
	wallDoorClosed  = 1 << 5 // the closed end
	wallDoorSidesNS = 1 << 3 // the door lies across the north-south sides
	soundDoorOpen   = 31
	soundDoorClose  = 32
	soundDoorShut   = 33
	doorOpening     = 1
	doorClosing     = -1
)

type movingDoor struct {
	block int // 0 for a free slot
	delta int // the step: doorOpening or doorClosing
	side  int // the wall it changes, and the one opposite
}

// doorEnd is the wall flag that ends a door moving by delta (C: g_2849).
func doorEnd(delta int) int {
	if delta == doorClosing {
		return wallDoorClosed
	}
	return wallDoorOpen
}

func doorSound(delta int) int {
	if delta == doorClosing {
		return soundDoorClose
	}
	return soundDoorOpen
}

// moveDoors moves each moving door one step (C: fn_0bbc_004a).
func (g *Game) moveDoors(int) {
	for i := range g.doors {
		d := &g.doors[i]
		if d.block == 0 {
			continue
		}
		walls := &g.lvl.blocks[d.block].Walls
		walls[d.side] += d.delta
		walls[d.side^2] += d.delta
		flags := g.lvl.wallFlags[walls[d.side]]
		sound := doorSound(d.delta)
		if flags&wallDoorClosed != 0 {
			sound = soundDoorShut
		}
		if g.updateFlags&updateInScene == 0 {
			g.playDoorSound(sound, d.block)
		}
		if flags&(wallDoorOpen|wallDoorClosed) != 0 {
			d.block = 0
		}
	}
}

// playDoorSound plays a door's sound where the door is; one out of sight sounds
// from where the party stands as well.
func (g *Game) playDoorSound(sound, block int) {
	if g.noteBlockChanged(block) {
		g.playSoundAt(sound, block)
	} else {
		g.playSoundAt(sound, block)
		g.playSoundAt(0, g.currentBlock)
	}
}

// toggleDoor opens a closed door or closes an open one, or turns one that moves; delta
// 0 picks the way (C: fn_13e8_0358). Doors where the party or a monster stands stay.
func (g *Game) toggleDoor(block, side, delta int) {
	if g.currentBlock == block || g.lvl.blocks[block].Objects&monsterFlag != 0 {
		return
	}
	if delta == 0 {
		for _, d := range g.doors {
			if d.block == block {
				delta = -d.delta
				break
			}
		}
	}
	if delta == 0 {
		side = g.doorSide(block)
		if g.lvl.wallFlags[g.lvl.blocks[block].Walls[side]]&wallBlocksWay != 0 {
			delta = doorOpening
		} else {
			delta = doorClosing
		}
	}
	g.startDoor(block, delta)
}

// doorSide is the side of block a door's walls are on.
func (g *Game) doorSide(block int) int {
	if g.lvl.wallFlags[g.lvl.blocks[block].Walls[0]]&wallDoorSidesNS != 0 {
		return 0
	}
	return 1
}

// startDoor sets the door of block moving by delta, unless it is at that end already;
// with no free slot it jumps to the end at once (C: fn_13e8_03e9).
func (g *Game) startDoor(block, delta int) {
	side := g.doorSide(block)
	walls := &g.lvl.blocks[block].Walls
	w := walls[side]
	if doorEnd(delta)&g.lvl.wallFlags[w] != 0 {
		return
	}
	free, found := -1, -1
	for i, d := range g.doors {
		if d.block == block {
			found = i
			break
		}
		if d.block == 0 && free == -1 {
			free = i
		}
	}
	if found == -1 && free == -1 {
		for doorEnd(delta)&g.lvl.wallFlags[w] == 0 {
			w += delta
		}
		walls[side] = w
		walls[side^2] = w
		g.noteBlockChanged(block)
		return
	}
	if found == -1 {
		found = free
	}
	g.doors[found] = movingDoor{block: block, delta: delta, side: side}
	if doorEnd(-delta)&g.lvl.wallFlags[w] != 0 {
		walls[side] += delta
		walls[side^2] += delta
		g.playDoorSound(doorSound(delta), block)
	}
	g.enableTimer(timerDoors, true)
}
