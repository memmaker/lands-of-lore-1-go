// Package sound is Lands of Lore's sound: the AdLib card played by ALFX.DRV on an OPL2
// emulator (github.com/memmaker/opl2), for the music and the effects, and the digital channels of the Sound
// Blaster for the sampled effects and speech, mixed into one Ebiten audio stream.
//
// The driver runs on its own 72 Hz clock in the audio stream, as the timer interrupt
// ran it; the game calls in under the same lock.
package sound

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/memmaker/opl2"
)

const (
	outputRate    = 44100
	outChannels   = 2
	bytesPerFrame = 8 // two float32 channels
	chipGain      = 1.0 / 16384
	playerBuffer  = 50 * time.Millisecond

	headerSize  = 500 // the track table before a sound file's data (C: ovr_2db9_0039)
	trackCount  = headerSize / 2
	noProgram   = 65535
	programMask = 255 // ALFX.DRV takes program numbers in a byte (C: n &= 0xff)
	noSaved     = -1

	priorityByte = 1 // the bytes of a program fn_178d_01ef scales by the volume
	volumeByte   = 3
	maxVolume    = 255
	levelTop     = 63

	startWait = 100 * time.Millisecond // the longest the game waits for a tick
)

// Player is the sound card. The zero value is not usable; use New.
type Player struct {
	mu      sync.Mutex
	chip    *opl2.Chip
	out     *opl2.Resampler // the chip at the output rate
	drv     *alfx
	running bool // the audio stream is playing and ticking the driver

	// tracks are the track tables of the two driver slots, the music's and the
	// effects' (C: g_28fa.hdr, g_28b8.hdr); both load into the one driver, the last
	// load's data is what it plays (C: g_2908 == g_28c6).
	tracks [2][]byte

	saved                     int  // the program whose bytes were scaled (C: g_2a9a)
	savedPriority, savedLevel byte // their old values (C: g_abd6, g_abd4)

	tickDue float64

	// the MIDI driver, when a slot plays on the MT-32 (C: g_28fa.h, g_28b8 != -1)
	midi       *midi
	midiSlot   [2]bool
	xmidi      [2]xmidiFile
	song       *sequence                // C: g_28fa.voice
	effects    [effectHandles]*sequence // C: g_28e2
	nextEffect int                      // C: g_28e0

	digital    [digitalChannels]digitalChannel // C: g_chan
	digitalDue float64
	offline    bool // rendered by a test, without the stream
}

// Driver slots.
const (
	Music   = 0
	Effects = 1
)

// New makes the card with the driver initialised (C: ovr_2db6_0025 with mode 72:
// driver function 2).
func New() *Player {
	p := &Player{chip: opl2.New(), saved: noSaved}
	p.out = opl2.NewResampler(p.chip, outputRate)
	p.drv = newALFX(p.chip)
	p.drv.init()
	return p
}

// Start opens the audio stream, once.
func (p *Player) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	ctx := audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(outputRate)
	}
	pl, err := ctx.NewPlayerF32(stream{p})
	if err != nil {
		return
	}
	pl.SetBufferSize(playerBuffer)
	pl.Play()
	p.mu.Lock()
	p.running = true
	p.mu.Unlock()
}

// UseMIDI plays the music, the effects or both on a MIDI synthesiser through the XMIDI
// driver instead of the AdLib's (C: MT32MPU.ADV in the driver slots).
func (p *Player) UseMIDI(s Synth, music, effects bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.midi = newMIDI(s)
	p.midiSlot = [2]bool{music, effects}
}

// MIDI says whether a slot plays on the MIDI synthesiser.
func (p *Player) MIDI(slot int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.midiSlot[slot]
}

// Load loads a sound file into a slot: its track table, and its data into the driver
// (C: ovr_2db9_0039, driver function 4). Nil unloads the slot.
func (p *Player) Load(slot int, file []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if file == nil {
		p.tracks[slot] = nil
		p.xmidi[slot] = nil
		return
	}
	if p.midiSlot[slot] { // an XMIDI file, without the track table (C: d->h != -1)
		p.tracks[slot] = file
		p.xmidi[slot] = parseXMIDI(file)
		return
	}
	if len(file) < headerSize {
		file = append(file, make([]byte, headerSize-len(file))...)
	}
	p.tracks[slot] = file
	p.drv.setData(append([]byte(nil), file[headerSize:]...))
}

// Loaded says whether the slot has a file.
func (p *Player) Loaded(slot int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tracks[slot] != nil
}

// Play plays track n of a slot's file: it waits until the programs queued last have
// started, puts back the bytes the last scaled play changed, scales the new program's
// priority and volume bytes by vol unless vol is -1, and queues it (C: fn_178d_01ef for
// ALFX.DRV). It returns true when it queued program 0, the stop, which the game follows
// with a pause.
func (p *Player) Play(slot, n, vol int) bool {
	if p.MIDI(slot) {
		p.playMIDI(slot, n, vol)
		return false
	}
	p.waitStarted()
	p.mu.Lock()
	defer p.mu.Unlock()
	t := p.tracks[slot]
	if n == -1 || t == nil {
		return false
	}
	prog := 0
	if 2*n+1 < len(t) {
		prog = int(binary.LittleEndian.Uint16(t[2*n:]))
	}
	if prog == noProgram {
		return false
	}
	prog &= programMask
	d := p.drv
	if p.saved != noSaved {
		d.setProgramByte(p.saved, priorityByte, p.savedPriority)
		d.setProgramByte(p.saved, volumeByte, p.savedLevel)
	}
	r := d.programByte(prog, 0)
	p.saved = noSaved
	if vol != -1 && r != controlChannel {
		p.saved = prog
		p.savedPriority = d.programByte(prog, priorityByte)
		p.savedLevel = d.programByte(prog, volumeByte)
		v := byte(vol)
		level := byte(levelTop - (int(levelTop-p.savedLevel)*int(v))>>8)
		d.setProgramByte(prog, volumeByte, level)
		d.setProgramByte(prog, priorityByte, byte(int(p.savedPriority)*int(v)>>8))
	}
	d.queueProgram(prog)
	return prog == 0
}

// playMIDI starts sequence n of a slot's file: the music replaces its sequence, the
// effects take their four handles in turn (C: fn_178d_00a4, fn_17d6_0019).
func (p *Player) playMIDI(slot, n, vol int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pct := (vol & maxVolume) * relativeScale >> 8
	if slot == Music {
		p.midi.release(p.song)
		p.song = p.midi.register(p.xmidi[slot], n)
		if p.song != nil {
			p.song.start()
			p.song.setVolume(pct, 0)
		}
		return
	}
	h := &p.effects[p.nextEffect]
	p.midi.release(*h)
	if *h = p.midi.register(p.xmidi[slot], n); *h != nil {
		(*h).start()
		(*h).setVolume(pct, 0)
	}
	p.nextEffect = (p.nextEffect + 1) % effectHandles
}

// Stop stops a slot: the AdLib's stop program, or the MIDI sequences released; true when
// the game should pause after it (C: fn_178d_015e, fn_17d6_0123).
func (p *Player) Stop(slot int) bool {
	if !p.MIDI(slot) {
		return p.Loaded(slot) && p.Play(slot, 0, 0)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if slot == Music {
		p.midi.release(p.song)
		p.song = nil
		return false
	}
	for i, h := range p.effects {
		p.midi.release(h)
		p.effects[i] = nil
	}
	return false
}

// Fade fades a slot out: the AdLib's fade program, or the sequences to silence over two
// seconds (C: fn_178d_01b0, fn_17d6_01b0).
func (p *Player) Fade(slot int) {
	if !p.MIDI(slot) {
		p.Play(slot, 1, 0)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if slot == Music {
		if p.song != nil {
			p.song.setVolume(0, fadeMillis)
		}
		return
	}
	for _, h := range p.effects {
		if h != nil {
			h.setVolume(0, fadeMillis)
		}
	}
}

// PlayThrough plays sequence n of the music slot to its end at once, rendering the
// synthesiser silently on the way: a set-up of sysex messages needs no real time on the
// emulator. It gives up after maxThrough of sound.
func (p *Player) PlayThrough(n int) {
	if !p.MIDI(Music) {
		return
	}
	p.playMIDI(Music, n, maxVolume)
	scratch := make([]float32, outChannels*outputRate/throughChunks)
	for i := 0; i < maxThrough*throughChunks; i++ {
		p.mu.Lock()
		done := p.song == nil || p.song.status != seqPlaying
		if !done {
			p.midi.render(scratch)
		}
		p.mu.Unlock()
		if done {
			return
		}
	}
}

const (
	throughChunks = 10 // PlayThrough renders a tenth of a second at a time, letting the stream in between
	maxThrough    = 60 // seconds
)

// waitStarted waits for the driver to start what was queued (C: the loop on function 16
// in fn_178d_01ef); without the stream there is no tick to wait for.
func (p *Player) waitStarted() {
	deadline := time.Now().Add(startWait)
	for {
		p.mu.Lock()
		busy := p.running && p.drv.flags&programBegun != 0
		p.mu.Unlock()
		if !busy || time.Now().After(deadline) {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// Playing says whether channel 0 plays a program (C: fn_178d_0038, driver function 7).
func (p *Player) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.midiSlot[Music] { // without the stream nothing would end it
		return (p.running || p.offline) && p.song != nil && p.song.status == seqPlaying
	}
	return p.drv.channelProgram(0) != 0
}

// Marker is the value programs set for the game (C: fn_178d_0007, driver function 11).
func (p *Player) Marker() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.midiSlot[Music] {
		return 0 // the XMIDI driver has none (C: g_28fa.h != -1)
	}
	return int(p.drv.marker)
}

// Block refuses (or allows again) the programs of the music channels or of the control
// channel (C: fn_16bf_0001, fn_16bf_0058: driver functions 16 and 17).
func (p *Player) Block(effects, on bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	mask := uint16(blockControl)
	if effects {
		mask = blockChannels
	}
	if on {
		p.drv.setFlags(mask)
	} else {
		p.drv.clearFlags(mask)
	}
}

// SetMusicVolume sets the volume of the music channels, 0-255 (C: ovr_2db9_0048,
// driver function 18).
func (p *Player) SetMusicVolume(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.midiSlot[Music] {
		// sic: only the song playing takes it; the next starts at its play's volume
		if p.song != nil {
			p.song.setVolume(max(0, min(v, maxVolume))*relativeScale>>8, 0)
		}
		return
	}
	p.drv.setMusicVolume(byte(max(0, min(v, maxVolume))))
}

// SetEffectVolume sets the volume of the effect channels (C: ovr_2db9_002a, driver
// function 19).
func (p *Player) SetEffectVolume(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.drv.setEffectVolume(byte(max(0, min(v, maxVolume))))
}

// render adds the next stereo frames into out.
func (p *Player) render(out []float32) {
	samplesPerTick := float64(outputRate) / alfxTickRate
	for i := 0; i < len(out); i += outChannels {
		if p.tickDue <= 0 {
			p.drv.tick()
			p.tickDue += samplesPerTick
		}
		p.tickDue--
		v := p.out.Next()
		out[i] += float32(v) * chipGain
		out[i+1] += float32(v) * chipGain
	}
	if p.midi != nil {
		p.midi.render(out)
	}
	p.mixDigital(out)
}

type stream struct{ p *Player }

func (s stream) Read(b []byte) (int, error) {
	frames := len(b) / bytesPerFrame
	mix := make([]float32, outChannels*frames)
	s.p.mu.Lock()
	s.p.render(mix)
	s.p.mu.Unlock()
	for i, v := range mix {
		binary.LittleEndian.PutUint32(b[i*bytesPerFrame/outChannels:], math.Float32bits(v))
	}
	return frames * bytesPerFrame, nil
}
