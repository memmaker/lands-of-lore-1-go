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

// startLands runs LANDS.EXE with arg headless; done and the hero are set when it ends.
func startLands(t *testing.T, arg string) (g *Game, done *bool, hero *int, runErr *error) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	done, hero, runErr = new(bool), new(int), new(error)
	host.ShowFrame = g.Screen.Frame
	host.StartHeadless(func() {
		clock.Reset()
		input.Init(g.Screen)
		input.SetNoMouse(false)
		if *runErr = g.setUpVideo(); *runErr == nil {
			*hero, *runErr = g.runLands(arg)
		}
		*done = true
	})
	return g, done, hero, runErr
}

func TestCharGen(t *testing.T) {
	_, done, hero, runErr := startLands(t, landsCharGen)
	click := func(x, y int, name string) {
		host.MouseX, host.MouseY = x, y
		steps(5)
		host.MouseLeft = true
		steps(10)
		host.MouseLeft = false
		steps(60)
		savePNG(t, host.ShowFrame(), name)
	}
	steps(200)
	savePNG(t, host.ShowFrame(), "chargen_king.png")
	click(charHeroes[1].x+10, charHeroes[1].y+10, "chargen_michael.png")
	steps(300)
	savePNG(t, host.ShowFrame(), "chargen_michael_talk.png")
	click((yesX1+yesX2)/2, (answerY1+answerY2)/2, "chargen_yes.png")
	for i := 0; i < 3000 && !*done && steps(1); i++ {
	}
	if *runErr != nil {
		t.Fatal(*runErr)
	}
	if !*done || *hero != 1 {
		t.Fatalf("done %v hero %d", *done, *hero)
	}
}
