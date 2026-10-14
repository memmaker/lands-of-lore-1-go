package sound

import "testing"

func TestDigitalMix(t *testing.T) {
	voc := make([]byte, vocHeaderSize)
	copy(voc, "Creative Voice File\x1a")
	voc[vocOffsetField] = vocHeaderSize
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(silence + 100*(i%2*2-1))
	}
	n := len(data) + vocSoundHeader
	voc = append(voc, vocSound, byte(n), byte(n>>8), 0, 166, 0) // 11111 Hz
	voc = append(voc, data...)
	voc = append(voc, vocEnd)
	s := ConvertVOC(voc)
	if len(s.data) != 2*len(data) || s.Ticks() != 5 {
		t.Fatalf("converted %d bytes, %d ticks", len(s.data), s.Ticks())
	}
	if scaleSample(0, 50) != silence+64 {
		t.Errorf("quirk lost: %d", scaleSample(0, 50))
	}
	p := New()
	p.offline = true
	if !p.PlayDigital(s, 10, maxVolume) {
		t.Fatal("not played")
	}
	out := make([]float32, outChannels*outputRate/10)
	p.mixDigital(out)
	loud := false
	for _, v := range out {
		loud = loud || v > 0.1
	}
	if !loud || p.DigitalPosition(s) != -1 {
		t.Errorf("loud %v, position %d", loud, p.DigitalPosition(s))
	}
}
