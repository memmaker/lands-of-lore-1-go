package game

import (
	"fmt"
	"os"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"

	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

const introSteps = 40000

func TestIntro(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	done := false
	var runErr error
	host.ShowFrame = g.Screen.Frame
	host.StartHeadless(func() {
		clock.Reset()
		input.Init(g.Screen)
		if runErr = g.setUpVideo(); runErr == nil {
			_, runErr = g.runLands(landsIntro)
		}
		done = true
	})
	for i := 0; i < introSteps && !done && steps(1); i++ {
		if os.Getenv("LOLROWS") != "" {
			f := host.ShowFrame()
			sum := 0
			for _, c := range f.Pixels[195*320 : 196*320] {
				sum = sum*31 + int(c)
			}
			t.Logf("%d %d %d", i, clock.Ticks(), sum&0xffff)
		}
		if i%300 == 0 {
			savePNG(t, host.ShowFrame(), fmt.Sprintf("intro%04d.png", i))
		}
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	t.Logf("done %v", done)
}
