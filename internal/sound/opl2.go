package sound

// OPL2 (Yamaha YM3812) emulator: 9 two-operator channels, the four waveforms, the envelope
// generator with its rate tables, key scaling of level and rate, tremolo and vibrato,
// feedback and both connections, and rhythm mode (ALFX.DRV plays drums). Levels are worked in the chip's log domain (a log-sine table and an exponent table), so
// attenuations add.
//
// Licence: written for the Ultima Underworld Go port (~/Projects/UltimaUnderworld,
// internal/snd/opl.go, same author as this port) after the public descriptions of the chip
// in the DOSBox and Nuked OPL cores; reused through the Ultima VI port with readable names,
// rhythm mode added here after the same descriptions. No code from those GPL/LGPL cores is
// copied.

import "math"

// oplRate is the chip's own sample rate: its 14.31818 MHz clock divided by 288.
const oplRate = 49716

// Register numbers and bit layout.
const (
	regTest          = 1   // bit 5 enables the waveform select registers
	regRhythm        = 189 // tremolo depth (bit 7), vibrato depth (bit 6), rhythm mode
	regOpFlags       = 32  // per operator: tremolo, vibrato, sustain, key-scale rate, multiple
	regOpLevel       = 64  // per operator: key-scale level, total level
	regOpAttackDecay = 96  // per operator: attack rate, decay rate
	regOpSustain     = 128 // per operator: sustain level, release rate
	regFreqLow       = 160 // per channel: low 8 bits of the frequency number
	regFreqHigh      = 176 // per channel: key on (bit 5), block (bits 2-4), high frequency bits
	regFeedback      = 192 // per channel: feedback (bits 1-3), additive connection (bit 0)
	regWaveform      = 224 // per operator: waveform 0-3
	opGroupSize      = 32  // operator registers come in groups of 32 numbers
	channels         = 9

	bit0, bit4, bit5, bit6, bit7 = 1 << 0, 1 << 4, 1 << 5, 1 << 6, 1 << 7

	maxAttenuation = 511  // envelope fully released, in 0.1875 dB steps
	sineSteps      = 256  // a quarter sine wave
	silentLevel    = 4096 // a log level that the exponent table maps to nothing
	maxLogLevel    = 8191 // the loudest log level still audible
	tremoloTop     = 105  // the tremolo counter's triangle peak

	exponentScale       = 1024 // the exponent table's 1.0
	byteRange           = 256  // one register's worth of a wider value
	levelRange          = 64   // total level is the low 6 bits of the level register
	fnumTopBit          = 512  // the frequency number's highest bit
	fnumKeyScaleUnit    = 64   // frequency number per key-scale level table step
	maxRate             = 63
	instantAttackRate   = 60   // attack rates from here on jump straight to full volume
	octaveKeyScaleLevel = 32   // key-scale level lost per block below the top
	tremoloPeriod       = 64   // samples per tremolo step
	vibratoPeriod       = 1024 // samples per vibrato step
	vibratoFnumUnit     = 128  // frequency number per unit of vibrato depth
)

var (
	logSine  [sineSteps]int
	exponent [sineSteps]int
	// twice each multiple setting's frequency multiple
	doubleMultiple = [16]int{1, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 20, 24, 24, 30, 30}
	keyScaleLevel  = [16]int{0, 32, 40, 45, 48, 51, 53, 55, 56, 58, 59, 60, 61, 62, 63, 64}
	keyScaleShift  = [4]uint{8, 1, 2, 0}
	// the envelope increments, eight steps a row, as the chip's
	envelopeIncrement = [15][8]int{
		{0, 1, 0, 1, 0, 1, 0, 1}, {0, 1, 0, 1, 1, 1, 0, 1}, {0, 1, 1, 1, 0, 1, 1, 1}, {0, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 2, 1, 1, 1, 2}, {1, 2, 1, 2, 1, 2, 1, 2}, {1, 2, 2, 2, 1, 2, 2, 2},
		{2, 2, 2, 2, 2, 2, 2, 2}, {2, 2, 2, 4, 2, 2, 2, 4}, {2, 4, 2, 4, 2, 4, 2, 4}, {2, 4, 4, 4, 2, 4, 4, 4},
		{4, 4, 4, 4, 4, 4, 4, 4}, {8, 8, 8, 8, 8, 8, 8, 8}, {0, 0, 0, 0, 0, 0, 0, 0},
	}
	// the register offset (within a group) of each operator slot, and each channel's first slot
	slotOffset  = [18]int{0, 1, 2, 3, 4, 5, 8, 9, 10, 11, 12, 13, 16, 17, 18, 19, 20, 21}
	channelSlot = [channels]int{0, 1, 2, 6, 7, 8, 12, 13, 14}
)

func init() {
	for i := range logSine {
		logSine[i] = int(math.Round(-math.Log2(math.Sin((float64(i)+0.5)*math.Pi/(2*sineSteps))) * sineSteps))
		exponent[i] = int(math.Round(math.Pow(2, float64(sineSteps-1-i)/sineSteps) * exponentScale))
	}
}

const (
	envAttack = iota
	envDecay
	envSustain
	envRelease
)

type operator struct {
	tremolo, vibrato, sustained, keyScaleRate bool
	multiple                                  int
	keyScaleLevel, totalLevel                 int
	attack, decay, sustainLevel, release      int
	waveform                                  int

	phase                uint32
	keys                 int // keyChannel and keyDrum: who holds the operator keyed on
	envelope             int // attenuation, 0 to maxAttenuation
	envState             int
	lastOut, previousOut int // for feedback
}

// The two ways an operator is keyed on: by its channel or as a drum.
const (
	keyChannel = 1 << 0
	keyDrum    = 1 << 1
)

type channel struct {
	fnum, block int
	feedback    int
	additive    bool
	ops         [2]*operator // modulator, carrier
}

type opl2 struct {
	ops                      [18]operator
	ch                       [channels]channel
	waveSelect               bool
	deepTremolo, deepVibrato bool
	envCounter               uint32
	tremoloPos, tremoloDir   int
	vibratoPos               int
	sampleCount              uint32
	rhythm                   bool
	drums                    int    // the drum key bits of the rhythm register
	noise                    uint32 // the 23-bit noise generator
}

// Rhythm mode: channel 6 is the bass drum; channel 7's operators are the hi-hat and the
// snare, channel 8's the tom and the cymbal. The rhythm register keys them by these bits.
const (
	drumChannel  = 6
	keyHiHat     = 1 << 0
	keyCymbal    = 1 << 1
	keyTom       = 1 << 2
	keySnare     = 1 << 3
	keyBass      = 1 << 4
	rhythmEnable = 1 << 5
	noiseTap     = 14
	noiseTopBit  = 22
	hiHatSlot    = 13
	tomSlot      = 14
	bassSlots    = 12 // and 15
	snareSlot    = 16
	cymbalSlot   = 17
	drumGain     = 2  // the drums sound at twice an operator's output
	hiHatBase    = 52 // the low phase bits the hi-hat's waveform is read at
	cymbalBase   = 128
)

func newOPL2() *opl2 {
	o := &opl2{tremoloDir: 1, noise: 1}
	for c := range o.ch {
		o.ch[c].ops[0] = &o.ops[channelSlot[c]]
		o.ch[c].ops[1] = &o.ops[channelSlot[c]+3]
	}
	for i := range o.ops {
		o.ops[i].envelope = maxAttenuation
		o.ops[i].envState = envRelease
	}
	return o
}

// write sets register reg to val.
func (o *opl2) write(reg int, val byte) {
	v := int(val)
	switch {
	case reg == regTest:
		o.waveSelect = v&bit5 != 0
	case reg == regRhythm:
		o.deepTremolo = v&bit7 != 0
		o.deepVibrato = v&bit6 != 0
		o.rhythm = v&rhythmEnable != 0
		o.drums = v % rhythmEnable
		if !o.rhythm {
			o.drums = 0
		}
		o.keyDrum(keyBass, bassSlots, bassSlots+3)
		o.keyDrum(keySnare, snareSlot)
		o.keyDrum(keyTom, tomSlot)
		o.keyDrum(keyCymbal, cymbalSlot)
		o.keyDrum(keyHiHat, hiHatSlot)
	case reg >= regFreqLow && reg < regFreqLow+channels:
		c := &o.ch[reg-regFreqLow]
		c.fnum = c.fnum/byteRange*byteRange + v
	case reg >= regFreqHigh && reg < regFreqHigh+channels:
		c := &o.ch[reg-regFreqHigh]
		c.fnum = c.fnum%byteRange + v%4*byteRange
		c.block = v / 4 % 8
		for _, op := range c.ops {
			op.key(keyChannel, v&bit5 != 0)
		}
	case reg >= regFeedback && reg < regFeedback+channels:
		c := &o.ch[reg-regFeedback]
		c.feedback = v / 2 % 8
		c.additive = v&bit0 != 0
	case reg >= regOpFlags:
		group, offset := reg-reg%opGroupSize, reg%opGroupSize
		slot := -1
		for i, off := range slotOffset {
			if off == offset {
				slot = i
			}
		}
		if slot < 0 {
			return
		}
		op := &o.ops[slot]
		switch group {
		case regOpFlags:
			op.tremolo, op.vibrato = v&bit7 != 0, v&bit6 != 0
			op.sustained, op.keyScaleRate = v&bit5 != 0, v&bit4 != 0
			op.multiple = v % 16
		case regOpLevel:
			op.keyScaleLevel, op.totalLevel = v/levelRange, v%levelRange
		case regOpAttackDecay:
			op.attack, op.decay = v/16, v%16
		case regOpSustain:
			op.sustainLevel, op.release = v/16, v%16
		case regWaveform:
			op.waveform = v % 4
		}
	}
}

// key keys the operator on or off for one of its keyers; it attacks when the first
// keys it and releases when the last lets go.
func (op *operator) key(by int, on bool) {
	old := op.keys
	if on {
		op.keys |= by
	} else {
		op.keys &^= by
	}
	switch {
	case old == 0 && op.keys != 0:
		op.envState, op.phase = envAttack, 0
	case old != 0 && op.keys == 0:
		op.envState = envRelease
	}
}

// keyDrum keys a drum's operators as its bit in the rhythm register says.
func (o *opl2) keyDrum(bit int, slots ...int) {
	for _, s := range slots {
		o.ops[s].key(keyDrum, o.drums&bit != 0)
	}
}

// effectiveRate is an envelope rate setting (0-15) scaled by the key, 0 to 63.
func (op *operator) effectiveRate(r int, c *channel) int {
	if r == 0 {
		return 0
	}
	keyCode := c.block*2 + c.fnum/fnumTopBit%2
	if !op.keyScaleRate {
		keyCode /= 4
	}
	return min(r*4+keyCode, maxRate)
}

// stepEnvelope advances an operator's envelope one sample.
func (o *opl2) stepEnvelope(op *operator, c *channel) {
	var r int
	switch op.envState {
	case envAttack:
		r = op.effectiveRate(op.attack, c)
	case envDecay:
		r = op.effectiveRate(op.decay, c)
	case envSustain:
		if op.sustained {
			return
		}
		r = op.effectiveRate(op.release, c)
	case envRelease:
		r = op.effectiveRate(op.release, c)
	}
	if r == 0 {
		return
	}
	var row, shift int
	switch group := r / 4; {
	case group <= 12:
		row, shift = r%4, 12-group
	case group == 13:
		row = 4 + r%4
	case group == 14:
		row = 8 + r%4
	default:
		row = 12
	}
	if o.envCounter%(1<<shift) != 0 {
		return
	}
	inc := envelopeIncrement[row][(o.envCounter>>shift)%8]
	switch op.envState {
	case envAttack:
		if r >= instantAttackRate {
			op.envelope = 0
		} else {
			op.envelope += (-(op.envelope + 1) * inc) >> 3
		}
		if op.envelope <= 0 {
			op.envelope, op.envState = 0, envDecay
		}
	case envDecay:
		sustain := op.sustainLevel * 16
		if op.sustainLevel == 15 {
			sustain = maxAttenuation
		}
		op.envelope += inc
		if op.envelope >= sustain {
			op.envelope, op.envState = sustain, envSustain
		}
	default:
		op.envelope = min(op.envelope+inc, maxAttenuation)
	}
}

// operatorOutput is one operator's output for a phase modulation of mod.
func (o *opl2) operatorOutput(op *operator, c *channel, mod, tremolo int) int {
	return o.waveOutput(op, int(op.phase>>10)+mod, op.attenuation(c, tremolo))
}

// attenuation is the operator's envelope with its level, key scaling and tremolo.
func (op *operator) attenuation(c *channel, tremolo int) int {
	att := op.envelope + op.totalLevel*4
	if op.keyScaleLevel != 0 {
		if k := keyScaleLevel[c.fnum/fnumKeyScaleUnit]*4 - (8-c.block)*octaveKeyScaleLevel; k > 0 {
			att += k >> keyScaleShift[op.keyScaleLevel]
		}
	}
	if op.tremolo {
		att += tremolo
	}
	return min(att, maxAttenuation)
}

// waveOutput is an operator's output at a phase (10 bits a wave) and attenuation.
func (o *opl2) waveOutput(op *operator, phase, att int) int {
	secondQuarter := phase&sineSteps != 0
	secondHalf := phase&(2*sineSteps) != 0
	idx := phase % sineSteps
	if idx < 0 {
		idx += sineSteps
	}
	mirrored := idx
	if secondQuarter {
		mirrored = sineSteps - 1 - idx
	}
	waveform := op.waveform
	if !o.waveSelect {
		waveform = 0
	}
	negative := false
	var level int
	switch waveform {
	case 0: // sine
		negative, level = secondHalf, logSine[mirrored]
	case 1: // half sine
		level = logSine[mirrored]
		if secondHalf {
			level = silentLevel
		}
	case 2: // absolute sine
		level = logSine[mirrored]
	case 3: // pulse sine: rising quarters only
		level = logSine[idx]
		if secondQuarter {
			level = silentLevel
		}
	}
	level += att * 8
	if level > maxLogLevel {
		return 0
	}
	v := exponent[level%sineSteps] * 2 >> (level / sineSteps)
	if negative {
		return -v
	}
	return v
}

// sample runs the chip one sample and returns the sum of its channels.
func (o *opl2) sample() int {
	o.sampleCount++
	o.envCounter++
	if o.sampleCount%tremoloPeriod == 0 {
		o.tremoloPos += o.tremoloDir
		if o.tremoloPos >= tremoloTop {
			o.tremoloDir = -1
		} else if o.tremoloPos <= 0 {
			o.tremoloDir = 1
		}
	}
	if o.sampleCount%vibratoPeriod == 0 {
		o.vibratoPos = (o.vibratoPos + 1) % 8
	}
	if (o.noise^o.noise>>noiseTap)&1 != 0 {
		o.noise = o.noise>>1 | 1<<noiseTopBit
	} else {
		o.noise >>= 1
	}
	tremolo := o.tremoloPos / 16
	if o.deepTremolo {
		tremolo = o.tremoloPos / 4
	}
	sum := 0
	for ci := range o.ch {
		c := &o.ch[ci]
		if o.rhythm && ci > drumChannel {
			o.advance(c)
			continue
		}
		mod, car := c.ops[0], c.ops[1]
		if mod.envelope >= maxAttenuation && car.envelope >= maxAttenuation &&
			mod.envState == envRelease && car.envState == envRelease {
			continue
		}
		o.advance(c)
		feedback := 0
		if c.feedback != 0 {
			feedback = (mod.lastOut + mod.previousOut) >> (9 - c.feedback)
		}
		m := o.operatorOutput(mod, c, feedback, tremolo)
		mod.previousOut, mod.lastOut = mod.lastOut, m
		v := 0
		if c.additive {
			v = m + o.operatorOutput(car, c, 0, tremolo)
		} else {
			v = o.operatorOutput(car, c, m, tremolo)
		}
		if o.rhythm && ci == drumChannel {
			v *= drumGain
		}
		sum += v
	}
	if o.rhythm {
		sum += o.drumOutput(tremolo)
	}
	return sum
}

// advance steps the envelopes and phases of a channel's operators.
func (o *opl2) advance(c *channel) {
	for _, op := range c.ops {
		o.stepEnvelope(op, c)
		f := c.fnum
		if op.vibrato {
			r := f / vibratoFnumUnit % 8
			switch {
			case o.vibratoPos%4 == 0:
				r = 0
			case o.vibratoPos%2 != 0:
				r /= 2
			}
			if !o.deepVibrato {
				r /= 2
			}
			if o.vibratoPos >= 4 {
				r = -r
			}
			f += r
		}
		op.phase += uint32((f << c.block) * doubleMultiple[op.multiple] / 2)
	}
}

// drumOutput is the four single-operator drums: the tom a plain wave, the hi-hat, snare
// and cymbal read at phases made of the hi-hat's and cymbal's phase bits and the noise.
func (o *opl2) drumOutput(tremolo int) int {
	bit := func(slot, n int) int { return int(o.ops[slot].phase>>10) >> n & 1 }
	noise := int(o.noise & 1)
	ring := (bit(hiHatSlot, 2) ^ bit(hiHatSlot, 7)) | (bit(hiHatSlot, 3) ^ bit(cymbalSlot, 5)) |
		(bit(cymbalSlot, 3) ^ bit(cymbalSlot, 5))
	hh, tc := &o.ch[drumChannel+1], &o.ch[drumChannel+2]
	out := func(slot int, c *channel, phase int) int {
		op := &o.ops[slot]
		if op.envelope >= maxAttenuation && op.envState == envRelease {
			return 0
		}
		return o.waveOutput(op, phase, op.attenuation(c, tremolo))
	}
	sum := out(hiHatSlot, hh, ring<<9|hiHatBase<<((ring^noise)<<1))
	snareBit := bit(hiHatSlot, 8)
	sum += out(snareSlot, hh, snareBit<<9|(snareBit^noise)<<8)
	sum += out(tomSlot, tc, int(o.ops[tomSlot].phase>>10))
	sum += out(cymbalSlot, tc, ring<<9|cymbalBase)
	return sum * drumGain
}
