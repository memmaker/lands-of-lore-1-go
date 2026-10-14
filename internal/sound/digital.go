package sound

import "encoding/binary"

// The Sound Blaster's digital channels (C: s029, s045, o2dc6): VOC samples converted to
// one output rate, played on four channels that are summed through a clipping table.

const (
	outputTimeConstant = 211 // the card's rate as a VOC time constant, 22222 Hz (C: g_2994 = 0xd3)
	timeConstantBase   = 256
	microseconds       = 1000000
	digitalRate        = microseconds / (timeConstantBase - outputTimeConstant)
	timerRate          = 60 // the game's timer, which times the samples (C: g_384c)
	bytesPerTick       = digitalRate / timerRate
	ticksPerSecondMul  = 16666 // C: 0x411a, the bytes per tick of a rate times 256-rate
	digitalChannels    = 4
	silence            = 128 // an unsigned sample's zero
	digitalGain        = 1.0 / 128

	vocHeaderSize   = 26 // "Creative Voice File" and the offset of the first block (C: 0x1a)
	vocOffsetField  = 20
	vocBlockHeader  = 4
	vocSoundHeader  = 2 // a sound block's rate and packing before its data
	vocEnd          = 0
	vocSound        = 1
	vocContinuation = 2
	vocSilence      = 3
	vocMarker       = 4
	vocText         = 5
	vocRepeat       = 6
	vocRepeatEnd    = 7
	vocExtended     = 8
)

// Sample is a VOC converted for the mixer (C: o2dc6's converted sound): its unsigned
// 8-bit data at the output rate, and how many timer ticks it lasts.
type Sample struct {
	data  []byte
	ticks int
}

// ConvertVOC converts a VOC file to the output rate by a whole factor, repeating or
// skipping samples, taken from the rate of its first block; nil when it has a block the
// converter does not know (C: ovr_2dc6_0020, ovr_2dc6_0025, ovr_2dc6_002f).
func ConvertVOC(b []byte) *Sample {
	if len(b) < vocHeaderSize {
		return nil
	}
	off := int(binary.LittleEndian.Uint16(b[vocOffsetField:]))
	factor := resampleFactor(firstRate(b, off))
	s := &Sample{}
	step, acc := 0, 0
	addTicks := func() {
		if step != 0 {
			s.ticks += (acc + step/2) / step
			acc = 0
		}
	}
	rate := -1
	for p := off; p+vocBlockHeader <= len(b); {
		kind := b[p]
		n := int(b[p+1]) | int(b[p+2])<<8 | int(b[p+3])<<16
		body := p + vocBlockHeader
		if kind == vocEnd {
			break
		}
		p = min(body+n, len(b))
		switch kind {
		case vocSound, vocContinuation:
			data := b[body:p]
			if kind == vocSound {
				if len(data) < vocSoundHeader {
					return nil
				}
				if r := int(data[0]); r != rate {
					step = ticksPerSecondMul / (timeConstantBase - r)
					addTicks() // sic: the ticks so far are counted at the new rate
					rate = r
				}
				data = data[vocSoundHeader:]
			}
			acc += len(data)
			s.data = append(s.data, resample(data, factor)...)
		case vocMarker:
			addTicks()
		case vocText, vocSilence, vocRepeat, vocRepeatEnd, vocExtended:
		default:
			return nil
		}
	}
	if acc != 0 {
		s.ticks += (acc + step/2) / step
	}
	return s
}

// firstRate is the time constant of the first block if it is a sound block, else the
// output's (C: ovr_2dc6_0020).
func firstRate(b []byte, off int) int {
	if off == vocHeaderSize && off+vocBlockHeader < len(b) && b[off] == vocSound {
		return int(b[off+vocBlockHeader])
	}
	return outputTimeConstant
}

// resampleFactor is how many times each sample repeats, or, negative, how many samples
// make one (C: g_2995).
func resampleFactor(r int) int {
	in := microseconds / (timeConstantBase - r)
	switch {
	case r < outputTimeConstant:
		return digitalRate / in
	case r > outputTimeConstant:
		return -(in / digitalRate)
	}
	return 1
}

func resample(data []byte, factor int) []byte {
	if factor < 0 {
		out := make([]byte, 0, len(data)/-factor)
		for i := 0; i < len(data)/-factor; i++ {
			out = append(out, data[i*-factor])
		}
		return out
	}
	out := make([]byte, 0, len(data)*factor)
	for _, c := range data {
		for range factor {
			out = append(out, c)
		}
	}
	return out
}

// Ticks is how long the sample lasts in 60ths of a second.
func (s *Sample) Ticks() int { return s.ticks }

// digitalChannel is one of the mixer's channels (C: struct Chan).
type digitalChannel struct {
	sample   *Sample
	active   bool
	priority int
	volume   int // 100, 75, 50, 25, 12 or 0 percent
	pos      int
	elapsed  int // timer ticks since it started
	frames   int // output frames into the current tick
}

// volumeSteps are the levels the mixer scales a sample to, by the driver volume
// (C: g_2964, g_2970).
var volumeSteps = []struct{ above, percent int }{
	{223, 100}, {159, 75}, {95, 50}, {47, 25}, {15, 12},
}

// volumePercent is the level of a driver volume (C: fn_16bf_00d6).
func volumePercent(v int) int {
	for _, s := range volumeSteps {
		if v > s.above {
			return s.percent
		}
	}
	return 0
}

// scaleSample scales an unsigned sample to a level (C: fn_1bbe_050c).
func scaleSample(c byte, percent int) byte {
	v := int8(c ^ silence)
	shift := 0
	switch percent {
	case 100:
		return c
	case 75:
		return threeQuarters[c^silence] ^ silence
	case 50:
		shift = 1
	case 25:
		shift = 2
	case 12:
		shift = 3
	default:
		return silence
	}
	if v < 0 {
		v = -(-v >> shift) // sic: -128 stays -128 when negated, so it comes out positive
	} else {
		v >>= shift
	}
	return byte(v) ^ silence
}

// CanPlayDigital says whether a channel would take a sample of a priority: one is idle,
// or plays one of no higher priority (C: fn_16cf_01c8).
func (p *Player) CanPlayDigital(priority int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.digitalSlot(priority) >= 0
}

func (p *Player) digitalSlot(priority int) int {
	for i, c := range p.digital {
		if !c.active {
			return i
		}
	}
	for i, c := range p.digital {
		if c.priority <= priority {
			return i
		}
	}
	return -1
}

// PlayDigital plays a sample on the first idle channel, else the first playing one of no
// higher priority, at a driver volume; false if none takes it or the volume is 0
// (C: fn_16cf_0022, fn_16cf_0ae2).
func (p *Player) PlayDigital(s *Sample, priority, vol int) bool {
	if s == nil || vol == 0 {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	i := p.digitalSlot(min(priority, maxVolume))
	if i < 0 {
		return false
	}
	p.digital[i] = digitalChannel{sample: s, active: true, priority: min(priority, maxVolume), volume: volumePercent(vol)}
	return true
}

// StopDigital stops the channels playing a sample, or all with nil; it says whether it
// stopped any (C: fn_16cf_00b9).
func (p *Player) StopDigital(s *Sample) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	stopped := false
	for i := range p.digital {
		c := &p.digital[i]
		if c.active && (s == nil || c.sample == s) {
			c.active = false
			stopped = true
		}
	}
	return stopped
}

// DigitalPosition is how many ticks of a sample have played, -1 when it is not playing
// (C: fn_16cf_057a).
func (p *Player) DigitalPosition(s *Sample) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.digital {
		if c.sample == s {
			if c.active {
				return c.pos / bytesPerTick
			}
			return -1
		}
	}
	return -1
}

// mixDigital adds the digital channels into the stereo frames (C: fn_16cf_02b7,
// fn_16cf_05d0, fn_1bbe_05e1): each channel's next sample at its level, idle ones
// silent, summed through the clipping table. A channel ends once its data is out and its
// time is up.
func (p *Player) mixDigital(out []float32) {
	framesPerTick := outputRate / timerRate
	for i := 0; i < len(out); i += outChannels {
		p.digitalDue += float64(digitalRate) / outputRate
		next := p.digitalDue >= 1
		if next {
			p.digitalDue--
		}
		sum := 0
		for ch := range p.digital {
			c := &p.digital[ch]
			if !c.active {
				sum += silence
				continue
			}
			if c.frames++; c.frames >= framesPerTick {
				c.frames = 0
				c.elapsed++
			}
			if c.pos >= len(c.sample.data) {
				sum += silence
				if c.elapsed > c.sample.ticks {
					c.active = false
				}
				continue
			}
			sum += int(scaleSample(c.sample.data[c.pos], c.volume))
			if next {
				c.pos++
			}
		}
		v := (float32(mixTable[sum]) - silence) * digitalGain
		out[i] += v
		out[i+1] += v
	}
}
