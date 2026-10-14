package sound

import (
	"os"
	"path/filepath"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"testing"
)

// TestMT32Generations opens each generation on the ROMs in LOLMT32OLD and LOLMT32NEW
// (folders with an old MT-32's and a CM-32L's ROMs): each folder serves only its own.
func TestMT32Generations(t *testing.T) {
	old, nw := os.Getenv("LOLMT32OLD"), os.Getenv("LOLMT32NEW")
	if old == "" || nw == "" {
		t.Skip("LOLMT32OLD and LOLMT32NEW not set")
	}
	for _, c := range []struct {
		dir  string
		gen  MT32Generation
		want bool
	}{{old, MT32Old, true}, {old, MT32New, false}, {nw, MT32Old, false}, {nw, MT32New, true}} {
		_, err := OpenMT32(c.dir, c.gen)
		if (err == nil) != c.want {
			t.Errorf("%s as %v: %v", c.dir, c.gen, err)
		}
	}
}

// TestMT32Music sets the old MT-32 in LOLMT32OLD up with LOLSYSEX, as the game's start
// does, and plays a song and an effect: each must end or sound.
func TestMT32Music(t *testing.T) {
	roms := os.Getenv("LOLMT32OLD")
	if roms == "" {
		t.Skip("LOLMT32OLD not set")
	}
	game := filepath.Join("..", "..", "game")
	startup := data.OpenArchives(game, []string{"STARTUP.PAK", "GENERAL.PAK"})
	sysex, err := startup.Read("LOLSYSEX.XMI")
	if err != nil {
		t.Skip("no game files:", err)
	}
	mt, err := OpenMT32(roms, MT32Old)
	if err != nil {
		t.Fatal(err)
	}
	p := New()
	p.offline = true
	p.UseMIDI(mt, true, true)
	p.Load(Music, sysex)
	p.Play(Music, 0, maxVolume)
	second := make([]float32, outChannels*outputRate)
	for i := 0; p.Playing(); i++ {
		if i > 30 {
			t.Fatal("LOLSYSEX still plays after 30 seconds")
		}
		p.render(second)
	}
	song, err := data.OpenArchives(game, []string{"MUSIC.PAK"}).Read("LORE01A.XMI")
	if err != nil {
		t.Skip(err)
	}
	fx, err := startup.Read("LORESFX.XMI")
	if err != nil {
		t.Fatal(err)
	}
	p.Load(Music, song)
	p.Load(Effects, fx)
	peak := func() float32 {
		pk := float32(0)
		for i := 0; i < 2; i++ {
			clear(second)
			p.render(second)
			for _, v := range second {
				pk = max(pk, v, -v)
			}
		}
		return pk
	}
	p.Play(Music, 3, maxVolume) // a song file's sequence 0 is empty
	if pk := peak(); pk < 0.01 || !p.Playing() {
		t.Errorf("song: peak %v, playing %v", pk, p.Playing())
	}
	p.Stop(Music)
	p.Play(Effects, 2, maxVolume)
	if pk := peak(); pk < 0.01 {
		t.Errorf("effect: peak %v", pk)
	}
}
