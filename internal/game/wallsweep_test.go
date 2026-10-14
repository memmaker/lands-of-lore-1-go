package game

import (
	"fmt"
	"os"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// TestWallClicks (LOLSWEEP=1: the clickable walls, LOLSWEEP=all: every decorated one;
// LOLBLOCK=n only block n) stands the party before each wall of the first level and
// clicks over the view: a script that panics fails the test, one that never returns
// times it out with the game goroutine's stack.
func TestWallClicks(t *testing.T) {
	if os.Getenv("LOLSWEEP") == "" {
		t.Skip("set LOLSWEEP=1")
	}
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	g.SkipCharGen = true
	var runErr error
	host.StartHeadless(func() { runErr = g.Run() })
	steps(300)
	for i := 0; i < newGameSteps && steps(1); i++ {
		if i%returnEvery == 0 {
			host.PushScan(host.ScanEvent{Scan: scanReturn})
			host.PushScan(host.ScanEvent{Scan: scanReturn, Released: true})
		}
	}
	if g.party[0].MaxHP == 0 {
		t.Fatalf("no game (%v)", runErr)
	}
	for b := range g.lvl.blocks {
		if os.Getenv("LOLBLOCK") != "" && fmt.Sprint(b) != os.Getenv("LOLBLOCK") {
			continue
		}
		for side := range directionCount {
			w := g.lvl.blocks[b].Walls[side]
			if g.lvl.wallSpecialType[w] == 0 && (os.Getenv("LOLSWEEP") != "all" || g.lvl.wallShape[w] <= 0) {
				continue
			}
			p := g.blockInDirection(b, side)
			t.Logf("block %d side %d (party %d facing %d), wall %d", b, side, p, side^2, g.lvl.blocks[b].Walls[side])
			g.currentBlock, g.direction = p, side^2
			g.sceneUpdateRequired = true
			steps(20)
			savePNG(t, host.ShowFrame(), fmt.Sprintf("wall%d_%d.png", b, side))
			for _, y := range []int{30, 60, 90} {
				for _, x := range []int{viewLeft + 40, viewLeft + 88, viewLeft + 136} {
					host.MouseX, host.MouseY = x, y
					steps(3)
					host.MouseLeft = true
					steps(5)
					host.MouseLeft = false
					for i := 0; i < 400 && steps(1); i++ {
						if i%100 == 50 {
							host.PushScan(host.ScanEvent{Scan: scanReturn})
							host.PushScan(host.ScanEvent{Scan: scanReturn, Released: true})
						}
					}
				}
			}
		}
	}
	t.Logf("ended with %v", runErr)
}
