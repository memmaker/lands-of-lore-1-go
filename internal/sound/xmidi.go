package sound

import "encoding/binary"

// The MIDI side of the sound: the Audio Interface Library's XMIDI driver (MT32MPU.ADV)
// as the game uses it for the Roland MT-32 (C: s030, s032 and the AIL calls fn_1b79_*).
// A sound file is an XMIDI file (FORM XDIR, CAT of FORM XMID); a track number is a
// sequence index in it. The game registers a sequence and starts it for every play: one
// at a time for the music, four in turn for the effects. The sequences run at XMIDI's
// fixed 120 Hz; their notes carry their own lengths.

const (
	xmidiRate       = 120 // XMIDI ticks a second
	midiChannels    = 16
	effectHandles   = 4    // C: g_28e2[4]
	relativeScale   = 90   // the percentage of a play's volume byte (C: (vol & 255) * 90 >> 8)
	fadeMillis      = 2000 // the music's and the effects' fade (C: fn_1b79_0301(.., 0, 2000))
	fullPercent     = 100
	controlMax      = 127
	pitchCentre     = 1 << 13
	panCentre       = 64
	lockOn          = 64 // controller values from this lock or protect
	lowestLockable  = 1  // AIL's MT-32 driver locks MIDI channels 2-9 (0-based 1-8)
	highestLockable = 8
)

// MIDI status nibbles and the controllers the sequencer itself handles.
const (
	midiNoteOff     = 128
	midiNoteOn      = 144
	midiAftertouch  = 160
	midiControl     = 176
	midiProgram     = 192
	midiPressure    = 208
	midiPitch       = 224
	midiSystem      = 240
	midiSysex       = 240
	midiSysexCont   = 247
	midiMeta        = 255
	metaEnd         = 47
	ccVolume        = 7
	ccPan           = 10
	ccExpression    = 11
	ccSustain       = 64
	ccAllNotesOff   = 123
	ccChannelLock   = 110 // XMIDI: take a free channel for this sequence's channel
	ccLockProtect   = 111 // XMIDI: keep this channel from being locked
	ccVoiceProtect  = 112
	ccTimbreProtect = 113
	ccPatchBank     = 114
	ccIndirect      = 115
	ccForLoop       = 116
	ccNextLoop      = 117
	ccClearBeat     = 118
	ccCallback      = 119
	ccBranch        = 120
	statusMask      = 240
	channelMask     = 15
	dataBits        = 7
	dataMask        = 1<<dataBits - 1
)

// Synth is a MIDI synthesiser: short messages, system exclusive messages and stereo
// frames at outputRate.
type Synth interface {
	Message(status, d1, d2 int)
	Sysex(msg []byte)
	Render(out []float32) // adds len(out)/2 stereo frames
}

// xmidiFile is a sound file's sequences.
type xmidiFile [][]byte

// parseXMIDI returns the EVNT chunks of an XMIDI file, nil if it is none.
func parseXMIDI(b []byte) xmidiFile {
	var seqs xmidiFile
	var walk func(b []byte)
	walk = func(b []byte) {
		for len(b) >= 8 {
			id := string(b[:4])
			n := min(int(binary.BigEndian.Uint32(b[4:8])), len(b)-8)
			body := b[8 : 8+n]
			switch {
			case id == "FORM" && len(body) >= 4 && string(body[:4]) == "XMID":
				for c := body[4:]; len(c) >= 8; {
					m := min(int(binary.BigEndian.Uint32(c[4:8])), len(c)-8)
					if string(c[:4]) == "EVNT" {
						seqs = append(seqs, c[8:8+m])
					}
					c = c[min(len(c), 8+m+m&1):]
				}
			case (id == "FORM" || id == "CAT ") && len(body) >= 4:
				walk(body[4:])
			}
			b = b[min(len(b), 8+n+n&1):]
		}
	}
	walk(b)
	return seqs
}

// Sequence statuses (C: AIL_sequence_status).
const (
	seqStopped = iota
	seqPlaying
	seqDone
)

type heldNote struct{ ch, note, left int }

type loop struct{ pos, count int }

// sequence is one registered sequence (C: an AIL sequence handle).
type sequence struct {
	m      *midi
	evnt   []byte
	pos    int
	delay  int
	status int
	notes  []heldNote
	loops  []loop

	// relative volume in percent and its fade (C: AIL_set_relative_volume)
	vol, volFrom, volTo, volTicks, volLeft int

	// what the sequence last set on each of its channels
	shadow [midiChannels]struct {
		program, vol, expr, pan, bend int
		set                           bool
	}
	remap [midiChannels]int // the physical channel each channel plays on (-1 itself)
}

// midi is the driver with its synthesiser.
type midi struct {
	synth     Synth
	seqs      []*sequence
	locked    [midiChannels]*sequence // a channel some sequence took for itself
	protected [midiChannels]bool
	sounding  [midiChannels]int
	tickDue   float64
}

func newMIDI(s Synth) *midi { return &midi{synth: s} }

// register makes a sequence of sequence n of a file, nil if it has none
// (C: AIL_register_sequence).
func (m *midi) register(f xmidiFile, n int) *sequence {
	if n < 0 || n >= len(f) {
		return nil
	}
	s := &sequence{m: m, evnt: f[n], vol: fullPercent}
	for c := range s.remap {
		s.remap[c] = -1
	}
	m.seqs = append(m.seqs, s)
	return s
}

// release stops a sequence and forgets it (C: AIL_release_sequence_handle).
func (m *midi) release(s *sequence) {
	if s == nil {
		return
	}
	s.stop()
	for i, o := range m.seqs {
		if o == s {
			m.seqs = append(m.seqs[:i], m.seqs[i+1:]...)
			break
		}
	}
}

// start plays a sequence from its beginning (C: AIL_start_sequence).
func (s *sequence) start() {
	s.stop()
	s.pos, s.delay, s.loops = 0, 0, nil
	s.status = seqPlaying
}

// stop silences the sequence's notes and gives back the channels it locked
// (C: AIL_stop_sequence).
func (s *sequence) stop() {
	for _, n := range s.notes {
		s.m.noteOff(n.ch, n.note)
	}
	s.notes = s.notes[:0]
	for c, p := range s.remap {
		if p >= 0 {
			s.m.unlock(p)
			s.remap[c] = -1
		}
	}
	if s.status == seqPlaying {
		s.status = seqStopped
	}
}

// setVolume sets the relative volume to pct percent over ms milliseconds
// (C: AIL_set_relative_volume).
func (s *sequence) setVolume(pct, ms int) {
	ticks := ms * xmidiRate / 1000
	if ticks <= 0 {
		s.vol, s.volLeft = pct, 0
		s.sendVolumes()
		return
	}
	s.volFrom, s.volTo, s.volTicks, s.volLeft = s.vol, pct, ticks, ticks
}

// channel is where channel c of the sequence plays.
func (s *sequence) channel(c int) int {
	if p := s.remap[c]; p >= 0 {
		return p
	}
	return c
}

// owns says whether the sequence may play on physical channel p.
func (s *sequence) owns(p int) bool {
	l := s.m.locked[p]
	return l == nil || l == s
}

func (s *sequence) scaled(c int) int {
	return min(controlMax, s.shadow[c].vol*s.vol/fullPercent)
}

func (s *sequence) sendVolumes() {
	for c := range s.shadow {
		if p := s.channel(c); s.shadow[c].set && s.owns(p) {
			s.m.synth.Message(midiControl|p, ccVolume, s.scaled(c))
		}
	}
}

func (m *midi) noteOff(p, note int) {
	m.synth.Message(midiNoteOff|p, note, 0)
	if m.sounding[p] > 0 {
		m.sounding[p]--
	}
}

// lock takes a channel for sequence s: the highest lockable one that is neither locked
// nor protected, preferring a silent one; -1 if there is none (C: AIL_lock_channel).
func (m *midi) lock(s *sequence) int {
	best := -1
	for p := highestLockable; p >= lowestLockable; p-- {
		if m.locked[p] != nil || m.protected[p] {
			continue
		}
		if m.sounding[p] == 0 {
			best = p
			break
		}
		if best < 0 {
			best = p
		}
	}
	if best < 0 {
		return -1
	}
	m.synth.Message(midiControl|best, ccAllNotesOff, 0)
	m.sounding[best] = 0
	for _, o := range m.seqs {
		o.dropNotes(best)
	}
	m.locked[best] = s
	return best
}

// unlock gives a channel back to the sequences that had it, with their settings
// (C: AIL_release_channel).
func (m *midi) unlock(p int) {
	m.synth.Message(midiControl|p, ccAllNotesOff, 0)
	m.sounding[p] = 0
	m.locked[p] = nil
	for _, o := range m.seqs {
		for c := range o.shadow {
			if o.channel(c) == p {
				o.restore(c)
			}
		}
	}
}

func (s *sequence) dropNotes(p int) {
	k := s.notes[:0]
	for _, n := range s.notes {
		if n.ch != p {
			k = append(k, n)
		}
	}
	s.notes = k
}

// restore sends channel c the settings the sequence last gave it.
func (s *sequence) restore(c int) {
	sh := &s.shadow[c]
	p := s.channel(c)
	if !sh.set || !s.owns(p) {
		return
	}
	syn := s.m.synth
	syn.Message(midiProgram|p, sh.program, 0)
	syn.Message(midiControl|p, ccVolume, s.scaled(c))
	syn.Message(midiControl|p, ccExpression, sh.expr)
	syn.Message(midiControl|p, ccPan, sh.pan)
	syn.Message(midiPitch|p, sh.bend&dataMask, sh.bend>>dataBits)
}

// send plays a channel message of the sequence, keeping its view of the channel.
func (s *sequence) send(status, d1, d2 int) {
	c := status & channelMask
	sh := &s.shadow[c]
	if !sh.set {
		sh.set = true
		sh.vol, sh.expr, sh.pan, sh.bend = controlMax, controlMax, panCentre, pitchCentre
	}
	switch status & statusMask {
	case midiControl:
		switch d1 {
		case ccVolume:
			sh.vol = d2
			d2 = s.scaled(c)
		case ccExpression:
			sh.expr = d2
		case ccPan:
			sh.pan = d2
		}
	case midiProgram:
		sh.program = d1
	case midiPitch:
		sh.bend = d1 | d2<<dataBits
	}
	if p := s.channel(c); s.owns(p) {
		s.m.synth.Message(status&statusMask|p, d1, d2)
	}
}

func (s *sequence) varlen() int {
	v := 0
	for s.pos < len(s.evnt) {
		b := s.evnt[s.pos]
		s.pos++
		v = v<<dataBits | int(b&dataMask)
		if b&(dataMask+1) == 0 {
			break
		}
	}
	return v
}

func (s *sequence) byte() int {
	if s.pos >= len(s.evnt) {
		return 0
	}
	b := s.evnt[s.pos]
	s.pos++
	return int(b)
}

// tick runs one 120 Hz tick of the sequence.
func (s *sequence) tick() {
	if s.volLeft > 0 {
		s.volLeft--
		s.vol = s.volTo + (s.volFrom-s.volTo)*s.volLeft/s.volTicks
		s.sendVolumes()
	}
	if s.status != seqPlaying {
		return
	}
	k := s.notes[:0]
	for _, n := range s.notes {
		if n.left--; n.left <= 0 {
			s.m.noteOff(n.ch, n.note)
			continue
		}
		k = append(k, n)
	}
	s.notes = k
	if s.delay > 0 {
		if s.delay--; s.delay > 0 {
			return
		}
	}
	for s.status == seqPlaying {
		if s.pos >= len(s.evnt) {
			s.end()
			return
		}
		b := int(s.evnt[s.pos])
		if b < dataMask+1 { // XMIDI delays are runs of bytes below 128
			for s.pos < len(s.evnt) && s.evnt[s.pos] <= dataMask {
				s.delay += int(s.evnt[s.pos])
				s.pos++
			}
			if s.delay > 0 {
				return
			}
			continue
		}
		s.pos++
		s.event(b)
	}
}

func (s *sequence) event(b int) {
	c := b & channelMask
	switch b & statusMask {
	case midiNoteOff:
		n, _ := s.byte(), s.byte()
		s.send(b, n, 0)
	case midiNoteOn:
		n, v := s.byte(), s.byte()
		dur := s.varlen()
		p := s.channel(c)
		if !s.owns(p) {
			return
		}
		s.send(b, n, v)
		if v == 0 {
			return
		}
		s.m.sounding[p]++
		if dur <= 0 {
			s.m.noteOff(p, n)
		} else {
			s.notes = append(s.notes, heldNote{p, n, dur})
		}
	case midiAftertouch, midiPitch:
		d1, d2 := s.byte(), s.byte()
		s.send(b, d1, d2)
	case midiControl:
		s.control(c, s.byte(), s.byte())
	case midiProgram, midiPressure:
		s.send(b, s.byte(), 0)
	case midiSystem:
		switch b {
		case midiMeta:
			typ := s.byte()
			n := s.varlen()
			s.pos += n
			if typ == metaEnd {
				s.end()
			}
		case midiSysex, midiSysexCont:
			n := s.varlen()
			end := min(s.pos+n, len(s.evnt))
			msg := append([]byte{byte(b)}, s.evnt[s.pos:end]...)
			if b == midiSysexCont {
				msg = msg[1:]
			}
			s.pos = end
			s.m.synth.Sysex(msg)
		}
	}
}

// control handles a controller: XMIDI's own ones here, the others go to the synth.
func (s *sequence) control(c, d1, d2 int) {
	switch d1 {
	case ccForLoop:
		s.loops = append(s.loops, loop{s.pos, d2})
	case ccNextLoop:
		n := len(s.loops)
		if n == 0 || d2 < lockOn {
			return
		}
		l := &s.loops[n-1]
		switch {
		case l.count == 0: // forever
			s.pos = l.pos
		case l.count > 1:
			l.count--
			s.pos = l.pos
		default:
			s.loops = s.loops[:n-1]
		}
	case ccChannelLock:
		if d2 >= lockOn {
			if s.remap[c] < 0 {
				if p := s.m.lock(s); p >= 0 {
					s.remap[c] = p
				}
			}
		} else if p := s.remap[c]; p >= 0 {
			s.remap[c] = -1
			s.m.unlock(p)
		}
	case ccLockProtect:
		s.m.protected[s.channel(c)] = d2 >= lockOn
	case ccVoiceProtect, ccTimbreProtect, ccPatchBank, ccIndirect, ccClearBeat, ccCallback, ccBranch:
		// timbre banks, callbacks and branches: the game uses none of them
	default:
		s.send(midiControl|c, d1, d2)
	}
}

func (s *sequence) end() {
	s.stop()
	s.status = seqDone
}

// render adds frames of the synthesiser, ticking the sequences on the way.
func (m *midi) render(out []float32) {
	samplesPerTick := float64(outputRate) / xmidiRate
	for start := 0; start < len(out); {
		if m.tickDue <= 0 {
			for _, s := range m.seqs {
				s.tick()
			}
			m.tickDue += samplesPerTick
		}
		frames := min((len(out)-start)/outChannels, int(m.tickDue)+1)
		m.synth.Render(out[start : start+frames*outChannels])
		m.tickDue -= float64(frames)
		start += frames * outChannels
	}
}
