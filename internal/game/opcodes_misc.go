package game

import "github.com/memmaker/lands-of-lore-1-go/internal/emc"

// opGetWall is the wall type of block arg 0 on side arg 1 (C: fn_0fbe_00dd).
func (g *Game) opGetWall(s *emc.State) int {
	return g.lvl.blocks[s.Arg(0)&blockMask].Walls[s.Arg(1)&(directionCount-1)]
}

// opGetWallFlags is the flags of that wall (C: fn_0fbe_0110).
func (g *Game) opGetWallFlags(s *emc.State) int {
	return g.lvl.wallFlags[g.opGetWall(s)]
}

// opGetDirection (C: fn_0fbe_0a4a).
func (g *Game) opGetDirection(*emc.State) int { return g.direction }

// opHideMouse, opShowMouse (C: fn_0fbe_0c40, fn_0fbe_0c4d).
func (g *Game) opHideMouse(*emc.State) int {
	g.Screen.HideMouse()
	return 1
}

func (g *Game) opShowMouse(*emc.State) int {
	g.Screen.ShowMouse()
	return 1
}

// opFadeToBlack (C: fn_0fbe_0c5a).
func (g *Game) opFadeToBlack(*emc.State) int {
	g.fadeToBlack(levelFadeTicks)
	return 1
}

// opGetLevelVar (C: fn_0fbe_0db9).
func (g *Game) opGetLevelVar(s *emc.State) int {
	if n := s.Arg(0); n >= 0 && n < levelVarCount { // fix: the original read anywhere
		return g.levelVars[n]
	}
	return 0
}

// opHasCredits says whether the party has arg 0 credits (C: fn_0fbe_17b8).
func (g *Game) opHasCredits(s *emc.State) int {
	if s.Arg(0) > g.credits {
		return 0
	}
	return 1
}

// opSetLevelTimer starts level timer arg 0 with interval arg 1, or stops it for 0
// (C: fn_0fbe_17d9).
func (g *Game) opSetLevelTimer(s *emc.State) int {
	n := s.Arg(0) + timerLevel5
	if s.Arg(1) == 0 {
		g.enableTimer(n, false)
	} else {
		g.startTimer(n, s.Arg(1))
	}
	return 1
}
