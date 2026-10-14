package sound

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

func gameFile(t *testing.T, name string) []byte {
	dir := os.Getenv("LOLDATA")
	if dir == "" {
		dir = "../../game"
	}
	b, err := os.ReadFile(dir + "/" + name)
	if err != nil {
		t.Skip("no game files; set LOLDATA")
	}
	return b
}

// TestMusicPlays plays a track of the first level's music and checks it sounds.
func TestMusicPlays(t *testing.T) {
	p := New()
	p.Load(Music, gameFile(t, "LORE01A.ADL"))
	p.SetMusicVolume(255)
	p.Play(Music, 2, -1)
	out := make([]float32, outChannels*outputRate*4)
	p.render(out)
	sum := 0.0
	for _, v := range out {
		if math.IsNaN(float64(v)) {
			t.Fatal("NaN")
		}
		sum += float64(v) * float64(v)
	}
	rms := math.Sqrt(sum / float64(len(out)))
	t.Logf("rms %.4f", rms)
	if rms < 0.001 {
		t.Fatal("silent")
	}
	if w := os.Getenv("LOLWAV"); w != "" {
		writeWAV(t, w, out)
	}
}

func writeWAV(t *testing.T, name string, out []float32) {
	b := make([]byte, 44+2*len(out))
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], outChannels)
	binary.LittleEndian.PutUint32(b[24:], outputRate)
	binary.LittleEndian.PutUint32(b[28:], outputRate*2*outChannels)
	binary.LittleEndian.PutUint16(b[32:], 2*outChannels)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(2*len(out)))
	for i, v := range out {
		binary.LittleEndian.PutUint16(b[44+2*i:], uint16(int16(max(-1, min(v, 1))*32767)))
	}
	if err := os.WriteFile(name, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
