package game

// Walking to a cell picked on the automap, an addition of the port: the way goes over
// the blocks the map shows, one step per main loop round with the party's own turns and
// steps, so block scripts, doors and traps act as they would for the player. A key or a
// click, a fighting monster coming next to the party or a step that does not land where planned ends it.

// findPath is the directions of the shortest way from block from to block to over the
// revealed blocks; nil when there is none.
func (g *Game) findPath(from, to int) []int {
	const unvisited = -1
	if g.lvl.blocks[to].Flags&blockRevealed != blockRevealed {
		return nil
	}
	cameFrom := make([]int, mapSize*mapSize)
	for i := range cameFrom {
		cameFrom[i] = unvisited
	}
	cameFrom[from] = from
	queue := []int{from}
	for len(queue) > 0 && cameFrom[to] == unvisited {
		b := queue[0]
		queue = queue[1:]
		for dir := range directionCount {
			n := g.blockInDirection(b, dir)
			if cameFrom[n] != unvisited || (n != to && g.lvl.blocks[n].Flags&blockRevealed != blockRevealed) ||
				g.wallsBlock(n, dir, blockedByWalls) {
				continue
			}
			cameFrom[n] = b
			queue = append(queue, n)
		}
	}
	if cameFrom[to] == unvisited {
		return nil
	}
	var path []int
	for b := to; b != from; b = cameFrom[b] {
		p := cameFrom[b]
		for dir := range directionCount {
			if g.blockInDirection(p, dir) == b {
				path = append(path, dir)
				break
			}
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// stepAutoWalk takes the next step of the way, or ends the walk when it cannot.
func (g *Game) stepAutoWalk() {
	if len(g.autoWalk) == 0 {
		return
	}
	if g.updateFlags&updateGameMenu != 0 || g.screenFrozen || g.controlsLocked != 0 ||
		g.spellTarget != noCharacter || g.monstersNear(g.currentBlock) > g.autoWalkMonsters {
		g.autoWalk = nil
		return
	}
	g.autoWalkMonsters = g.monstersNear(g.currentBlock)
	dir := g.autoWalk[0]
	g.autoWalk = g.autoWalk[1:]
	level, next := g.level, g.blockInDirection(g.currentBlock, dir)
	g.turnTo(dir)
	g.moveParty(dir, walkForward, iconForward)
	if g.level != level || g.currentBlock != next {
		g.autoWalk = nil
	}
	g.autoWalkMonsters = g.monstersNear(g.currentBlock)
}

// monstersNear counts the monsters fighting the party on block b and those beside it.
func (g *Game) monstersNear(b int) int {
	count := 0
	near := []int{b}
	for dir := range directionCount {
		near = append(near, g.blockInDirection(b, dir))
	}
	for _, n := range near {
		for o := g.lvl.blocks[n].Objects; o != 0; o = *g.objectNext(o) {
			if o&monsterFlag == 0 {
				continue
			}
			switch g.monsters[o&monsterIndex].Mode {
			case monsterModeChase, monsterModeAttack, monsterModeRetreat:
				count++
			}
		}
	}
	return count
}
