package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// TestFinale plays the ending for Conrad at the hardest difficulty without the voices (so
// the text shows), saving a picture every LOLEVERY steps (default 600), and presses Escape
// after LOLFINSTEPS steps (default 40000) until it ends.
const scanEscape = 1

func TestFinale(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	g.Config.DigitalSound = os.Getenv("LOLVOICES") != ""
	done := false
	var runErr error
	host.ShowFrame = g.Screen.Frame
	host.StartHeadless(func() {
		clock.Reset()
		input.Init(g.Screen)
		if runErr = g.setUpVideo(); runErr == nil {
			runErr = g.runFinale(3, finaleHardest)
		}
		done = true
	})
	env := func(k string, d int) int {
		if n, err := strconv.Atoi(os.Getenv(k)); err == nil {
			return n
		}
		return d
	}
	every, total := env("LOLEVERY", 600), env("LOLFINSTEPS", 40000)
	// LOLCMP: a parity.py dump folder; each dump's closest frame is reported
	var dumps [][]byte
	var names []string
	if dir := os.Getenv("LOLCMP"); dir != "" {
		names, _ = filepath.Glob(filepath.Join(dir, "*.raw"))
		for _, n := range names {
			b, _ := os.ReadFile(n)
			dumps = append(dumps, b)
		}
	}
	best := make([]int, len(dumps))
	bestAt := make([]int, len(dumps))
	for i := range best {
		best[i] = 1 << 30
	}
	for i := 0; !done && steps(1); i++ {
		if len(dumps) > 0 && i%2 == 0 {
			f := host.ShowFrame().Pixels
			for k, d := range dumps {
				n := 0
				for j := range d {
					if d[j] != f[j] {
						n++
					}
				}
				if n < best[k] {
					best[k], bestAt[k] = n, i
				}
			}
		}
		if i%every == 0 {
			savePNG(t, host.ShowFrame(), fmt.Sprintf("finale%05d.png", i))
		}
		if i >= total && i%100 == 0 {
			host.PushScan(host.ScanEvent{Scan: scanEscape})
			host.PushScan(host.ScanEvent{Scan: scanEscape, Released: true})
		}
		if i > total+20000 {
			t.Fatal("the ending did not end")
		}
	}
	for k := range dumps {
		t.Logf("%s: %d pixels differ at step %d", filepath.Base(names[k]), best[k], bestAt[k])
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
}
