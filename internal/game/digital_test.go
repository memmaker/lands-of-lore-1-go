package game

import "testing"

func TestEffectSamples(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for n := range maxSoundEffect {
		if s := g.effectSample(n); s != nil {
			if s.Ticks() <= 0 {
				t.Errorf("effect %d: %d ticks", n, s.Ticks())
			}
			found++
		}
	}
	if found == 0 {
		t.Fatal("no effect has a sample")
	}
	t.Logf("%d effects have samples", found)
}
