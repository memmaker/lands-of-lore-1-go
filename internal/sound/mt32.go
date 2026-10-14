package sound

import "fmt"

// The Roland MT-32: munt's emulator on the ROMs the player supplies, loaded at run time
// (libmt32emu through purego on the desktop, mt32_dl.go; goplay's page in the browser,
// mt32_js.go).

// MT32Generation is which MT-32 the emulator is to be: the game was written for the first
// one; the CM-32L and the MT-32 with control ROM 2.x have a changed LA engine that can
// make its custom patches (LOLSYSEX) sound distorted or wrong.
type MT32Generation int

const (
	MT32Old MT32Generation = iota // MT-32 with control ROM 1.04-1.07 or Blue Ridge
	MT32New                       // CM-32L, CM-32LN, MT-32 with control ROM 2.x
)

func (g MT32Generation) String() string {
	if g == MT32Old {
		return "MT-32 (old)"
	}
	return "CM-32L / MT-32 (new)"
}

// munt's machine ids for a generation, best first.
var mt32Machines = map[MT32Generation][]string{
	MT32Old: {"mt32_1_07", "mt32_1_06", "mt32_1_05", "mt32_1_04", "mt32_bluer"},
	MT32New: {"cm32l_1_02", "cm32l_1_00", "mt32_2_07", "mt32_2_06", "mt32_2_04", "mt32_2_03", "cm32ln_1_00"},
}

const (
	mt32Gain   = 1.0 / 32768
	frameWords = 2
)

// MT32 is the emulated synthesiser.
type MT32 struct {
	Generation MT32Generation
	buf        []int16
}

// OpenMT32 starts the emulator as an MT-32 of the generation with the ROMs in dir; it
// fails when the emulator or that generation's ROMs are missing.
func OpenMT32(dir string, gen MT32Generation) (*MT32, error) {
	if err := mtOpen(dir, gen); err != nil {
		return nil, fmt.Errorf("%v: %w", gen, err)
	}
	return &MT32{Generation: gen}, nil
}

// Message sends a short message.
func (m *MT32) Message(status, d1, d2 int) {
	mtMsg(uint32(status) | uint32(d1)<<8 | uint32(d2)<<16)
}

// Sysex sends a system exclusive message.
func (m *MT32) Sysex(msg []byte) {
	if len(msg) > 0 {
		mtSysex(msg)
	}
}

// Render adds the emulator's next frames.
func (m *MT32) Render(out []float32) {
	if cap(m.buf) < len(out) {
		m.buf = make([]int16, len(out))
	}
	b := m.buf[:len(out)/frameWords*frameWords]
	if len(b) == 0 {
		return
	}
	mtRender(b)
	for i, v := range b {
		out[i] += float32(v) * mt32Gain
	}
}
