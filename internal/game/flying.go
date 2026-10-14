package game

// Flying objects: thrown items, missiles and spell projectiles (C: s012).
const (
	flyingDrops      = 0 // an item that lands where it stops
	flyingVanishes   = 1 // gone once it stops, unless it still flies
	flyingNoTarget   = 8 // stopped by its range or a newer launch
	flyingHitWall    = placeWall
	flyingHitMonster = placeMonster
	flyingHitParty   = placeParty

	eventItemHit     = 1 << 15 // an item hits something (C: 0x8000)
	itemHitsArea     = 1 << 14 // it hits everything on the block (C: ItemType.flags 0x4000)
	flyingSizeArea   = 256
	flyingSizeSingle = 1<<6 - 1
	flyingStopsAll   = blockedByAll
	flyingWallMask   = 2
	launchExperience = 2
	launchSkill      = 1
)

// placeFlying puts a flying object's item where it is, in the air while it has power
// (C: fn_0b6b_0006).
func (g *Game) placeFlying(f *flyingObject) {
	switch f.Type {
	case flyingDrops:
		g.dropItem(f.Item, f.X, f.Y, f.Power, f.Power == 0)
	case flyingVanishes:
		if f.Power != 0 {
			g.dropItem(f.Item, f.X, f.Y, f.Power, false)
		} else {
			g.freeItem(f.Item)
			g.noteBlockChanged(positionBlock(f.X, f.Y))
		}
	}
}

// liftFlying takes a flying object's item off its block.
func (g *Game) liftFlying(f *flyingObject) int {
	b := positionBlock(f.X, f.Y)
	g.removeObject(&g.lvl.blocks[b].Objects, f.Item)
	g.removeFromDrawList(&g.lvl.blocks[b], f.Item)
	return b
}

// stepFlying moves a flying object on to (x, y) (C: fn_0b6b_008a).
func (g *Game) stepFlying(f *flyingObject, x, y int) {
	b := g.liftFlying(f)
	f.X, f.Y = x, y
	g.placeFlying(f)
	g.noteBlockChanged(b)
}

// flyingHits runs the item's hit script on what it hit at (x, y) (C: fn_0b6b_00f9).
func (g *Game) flyingHits(f *flyingObject, x, y, what int) {
	it := &g.items[f.Item]
	area := g.itemTypes[it.Type].Flags&itemHitsArea != 0
	target := 0
	switch what {
	case flyingHitWall:
		b := g.blockInDirection(it.Block, f.Dir>>1)
		g.runBlockScriptWith(b, eventItemHit, -1, f.Item, 0, 0)
		return
	case flyingHitMonster:
		if area {
			for o := g.lvl.blocks[it.Block].Objects; o&monsterFlag != 0; {
				next := *g.objectNext(o)
				g.runItemScript(f.Owner, f.Item, eventItemHit, o, 0)
				o = next
			}
			return
		}
		target = g.nearestMonster(x, y)
	case flyingHitParty:
		g.sceneReady = true
		if area {
			for ch := range partySize {
				if g.party[ch].Flags&charPresent != 0 {
					g.runItemScript(f.Owner, f.Item, eventItemHit, ch, 0)
				}
			}
			return
		}
		target = g.characterFacing(x, y)
	default:
		return
	}
	g.runItemScript(f.Owner, f.Item, eventItemHit, target, 0)
}

// endFlying stops a flying object at (x, y) after hitting what: an item drops to
// the floor there (C: fn_0b6b_0272).
func (g *Game) endFlying(f *flyingObject, x, y, what int) {
	g.liftFlying(f)
	if what == flyingHitWall {
		x, y = f.X, f.Y
	}
	switch f.Type {
	case flyingDrops, flyingVanishes:
		g.flyingHits(f, x, y, what)
		f.X = x&itemSubSnap | itemSubCentre
		f.Y = y&itemSubSnap | itemSubCentre
		f.Power = 0
		g.placeFlying(f)
	}
	f.Active = false
}

// launchObject sends an item flying from (x, y) at height power towards dir (0-7);
// with every slot taken the one farthest from the party is stopped for it, if it is
// farther than the new one. owner is a character, a monster or -1 (C: fn_0b6b_032f).
func (g *Game) launchObject(kind, item, x, y, power, dir, _, owner, flags int) bool {
	best := distance(g.partyX, g.partyY, x, y)
	i, far := 0, -1
	for ; i < flyingObjects; i++ {
		f := &g.flying[i]
		if !f.Active {
			best = -1
			break
		}
		if d := distance(g.partyX, g.partyY, f.X, f.Y); d > best {
			best, far = d, i
		}
	}
	if best != -1 && far != -1 {
		i = far
		g.endFlying(&g.flying[i], x, y, flyingNoTarget)
	}
	if i == flyingObjects {
		return false
	}
	f := &g.flying[i]
	*f = flyingObject{Active: true, Type: kind, Item: item, X: x, Y: y, Power: power, Dir: dir,
		Range: -1, Owner: owner, Flags: flyingStopsAll, Field16: flyingWallMask, Unknown: flags}
	if owner != -1 {
		if isMonster(owner) {
			f.Flags &^= blockedByMonsters
		} else {
			f.Flags &^= blockedByParty
			g.addExperience(owner, launchSkill, launchExperience)
		}
	}
	g.placeFlying(f)
	return true
}

// moveFlyingObject moves a flying object a step, ending it where something stops it
// or its range runs out (C: fn_0b6b_0472).
func (g *Game) moveFlyingObject(f *flyingObject) {
	nx, ny := nextStep(f.X, f.Y, f.Dir)
	size := flyingSizeSingle
	// sic: the item types are looked up by the item's number
	if f.Item < len(g.itemTypes) && g.itemTypes[f.Item].Flags&itemHitsArea != 0 {
		size = flyingSizeArea
	}
	if r := g.placementBlocked(nx, ny, size, f.Flags, f.Field16); r != placeFree {
		g.endFlying(f, nx, ny, r)
	} else if f.Range = int(int8(f.Range - 1)); f.Range == 0 { // a byte: -1 runs out after 255 steps
		g.endFlying(f, nx, ny, flyingNoTarget)
	} else {
		g.stepFlying(f, nx, ny)
	}
}
