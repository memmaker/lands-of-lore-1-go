package game

import (
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

type nullSynth struct{}

func (nullSynth) Message(status, d1, d2 int) {}
func (nullSynth) Sysex([]byte)               {}
func (nullSynth) Render([]float32)           {}

// TestMT32Files checks that the MT-32 gets the XMIDI files: LORESFX for the effects and
// the song for the music, with the music on and off.
func TestMT32Files(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	g.UseMT32(nullSynth{})
	g.options |= optionMusic
	g.queueTrack(firstMusicTrack)
	if !g.audio.Loaded(sound.Music) || !g.audio.Loaded(sound.Effects) {
		t.Fatalf("music on: music %v, effects %v", g.audio.Loaded(sound.Music), g.audio.Loaded(sound.Effects))
	}
	g.options &^= optionMusic
	g.queueTrack(firstMusicTrack)
	if g.audio.Loaded(sound.Music) || !g.audio.Loaded(sound.Effects) {
		t.Fatalf("music off: music %v, effects %v", g.audio.Loaded(sound.Music), g.audio.Loaded(sound.Effects))
	}
}
