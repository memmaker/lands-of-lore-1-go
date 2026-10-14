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

func TestSpeechIndex(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	g.loadSpeechIndex(0)
	if g.commonSpeech == nil || len(g.commonSpeech.entries) < 10 {
		t.Fatal("no 00.TLK index")
	}
	if !g.loadSpeech(stringAudioSample, 0) || g.speechSample.Ticks() <= 0 {
		t.Fatalf("audio menu sample %#x not loaded", stringAudioSample)
	}
	g.loadSpeechIndex(1)
	if g.levelSpeech == nil {
		t.Fatal("no 01.TLK index")
	}
	e := g.levelSpeech.entries[len(g.levelSpeech.entries)/2]
	if p := g.levelSpeech.findSpeech(e.id, e.voice); p < 0 {
		t.Fatalf("speech %d %c not found", e.id, e.voice)
	}
}
