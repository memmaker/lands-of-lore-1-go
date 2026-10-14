package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gui"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// Resting: the party sleeps with the view dark while the world goes on, healing hit
// points and magic a point at a time, until everyone is whole, a key is pressed or a
// monster wakes them; poison keeps hurting meanwhile (C: o2d7a).
const (
	iconRest         = iconFirstSceneButton + 1
	charResting      = charMute // asleep, a resting character cannot speak either
	charPoisoned     = charFaceTinted
	restTime         = 600 // ticks a whole bar takes at most, split by the largest maximum
	restMaxStep      = 30
	restPoisonStep   = 15
	restWarmUp       = 32 // rounds of the world before the healing starts (16 on slow machines)
	restRound        = 8  // rounds between key checks (4 on slow machines)
	faceAsleep       = 4
	faceWokenHurt    = 5
	faceWokenSound   = 6
	stringRestBegin  = 0x4057
	stringRestDone   = 0x4059
	stringRestNoNeed = 0x4058
	stringRestPoison = 0x405a
	faceNotNeeded    = 2
	soundNotNeeded   = 4
	facePoisoned     = 3
	soundPoisoned    = 8
	restFadeTicks    = 40
	restPoisonDamage = 1
)

// clickRestParty rests the party (C: ovr_2d7a_0020).
func (g *Game) clickRestParty(*gui.Button) bool {
	maxHP, maxMP, poisonHP := -1, -1, -1
	needMP, needHP, poisoned := 0, 0, 0
	g.setIcon(iconRest, iconPress)
	if g.screenFrozen {
		g.clickExitCharacterInventory(&scriptButton)
	}
	for i := range g.party {
		c := &g.party[i]
		if c.Flags&charPresent == 0 || c.Flags&charDead != 0 {
			continue
		}
		maxHP = max(maxHP, c.MaxHP)
		maxMP = max(maxMP, c.MaxMP)
		if c.Flags&charPoisoned != 0 {
			poisoned |= 1 << i
			poisonHP = max(poisonHP, c.HP)
		} else if c.HP < c.MaxHP {
			needHP |= 1 << i
		}
		if c.MP < c.MaxMP {
			needMP |= 1 << i
		}
		c.Flags |= charResting
	}
	if needHP == 0 && needMP == 0 {
		for i := range g.party {
			g.party[i].Flags &^= charResting
		}
		if poisoned != 0 {
			g.setPartyFaces(0, 0, false)
			for i := range g.party {
				if poisoned&(1<<i) != 0 {
					g.setTemporaryFace(i, facePoisoned, soundPoisoned, false)
				}
			}
			g.printMessage(messageSilent, g.text(stringRestPoison))
			g.drawPortraits()
		} else {
			g.setPartyFaces(faceNotNeeded, soundNotNeeded, true)
			g.printMessage(messageSilent, g.text(stringRestNoNeed))
		}
		g.setIcon(iconRest, iconRelease)
		return true
	}
	s := g.Screen
	s.HideMouse()
	s.FillRect(viewLeft, 0, viewLeft+viewWidth, viewHeight, 1) // sic: one wider and taller
	s.ShowMouse()
	g.drawPortraits()
	g.printMessage(messageSilent, g.text(stringRestBegin))
	g.setIcon(iconRest, iconRelease)
	// fix: a maximum of 0 (no magic in the party, a poisoned member at 0 hit points)
	// divided by zero
	stepHP := min(restTime/max(maxHP, 1)+1, restMaxStep)
	stepMP := min(restTime/max(maxMP, 1)+1, restMaxStep)
	if poisonHP == 0 {
		poisonHP = 1
	}
	stepPoison := min((restTime/poisonHP+1)>>1, restPoisonStep) // sic: negative without poison
	now := clock.Ticks()
	nextHP, nextMP, nextPoison := now+stepHP, now+stepMP, now+stepPoison
	g.sceneReady = false
	g.updateFlags |= updateInScene
	round := func(n int) {
		for range n {
			g.moveMonsters(0)
			g.moveMonsters(1)
			g.moveDoors(0)
			g.moveFlyingObjects(0)
			if g.sceneReady {
				break
			}
		}
	}
	warmUp, rounds := restWarmUp, restRound
	if !fastMachine {
		warmUp, rounds = warmUp/2, rounds/2
	}
	round(warmUp)
	g.updateAutomapFlags()
	for (needMP != 0 || needHP != 0) && !g.sceneReady {
		host.Yield() // the original ran as fast as the machine
		round(rounds)
		if input.Peek() != 0 {
			k := input.Get()
			if k != 0 && k&input.Released == 0 {
				g.putKeyBack(k)
				break
			}
			g.GUI.MarkChanged()
		}
		if g.sceneReady {
			break
		}
		now := clock.Ticks()
		if now > nextPoison {
			for i := range g.party {
				if poisoned&(1<<i) != 0 {
					g.damageFighter(i, restPoisonDamage, poisonSource, 1, poisonDamageKind)
					if g.party[i].Flags&charDead != 0 {
						poisoned &^= 1 << i
					}
				}
			}
			nextPoison = now + stepPoison
		}
		if now > nextHP {
			for i := range g.party {
				if needHP&(1<<i) != 0 {
					g.heal(i, 1, false)
					g.drawPortrait(i)
					if g.party[i].HP == g.party[i].MaxHP {
						needHP &^= 1 << i
					}
				}
			}
			nextHP = now + stepHP
		}
		if now > nextMP {
			for i := range g.party {
				c := &g.party[i]
				if needMP&(1<<i) != 0 {
					c.MP++
					g.drawPortrait(i)
					if c.MP == c.MaxMP {
						needMP &^= 1 << i
					}
				}
			}
			nextMP = now + stepMP
		}
	}
	for i := range g.party {
		c := &g.party[i]
		switch {
		case c.Flags&charResting != 0:
			c.Flags &^= charResting
			if !g.sceneReady {
				g.setTemporaryFace(i, 0, 0, true)
				break
			}
			face := faceAsleep
			if c.DamageSuffered != 0 {
				face = faceWokenHurt
				g.playSound(c.Scream, 0)
			}
			g.setTemporaryFace(i, face, faceWokenSound, true)
		case c.DamageSuffered == 0:
			g.setTemporaryFace(i, faceAsleep, 0, true)
		}
	}
	g.updateFlags &^= updateInScene
	g.sceneReady = true
	g.copyViewFrame()
	g.drawScene(0)
	g.printMessage(messageSilent, g.text(stringRestDone))
	g.fadeInDark(restFadeTicks)
	return true
}

// setPartyFaces gives every member a temporary face (C: fn_0a21_06b8).
func (g *Game) setPartyFaces(face, sound int, redraw bool) {
	for i := range g.party {
		if g.party[i].Flags&charPresent != 0 {
			g.setTemporaryFace(i, face, sound, false)
		}
	}
	if redraw {
		g.drawPortraits()
	}
}
