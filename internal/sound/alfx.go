package sound

// ALFX.DRV, Westwood's AdLib driver (from DRIVERS.PAK), ported from its disassembly. It
// plays programs of byte code out of a sound file: each program starts with the channel
// it runs on and a priority, then pairs of bytes, a note and its duration or an opcode
// (128 and up) and its first argument. Channels 0-8 are the chip's, channel 9 runs control
// programs that start the others. The game's timer calls tick 72 times a second.
//
// The driver kept its state in its own code segment; here it is fields. A program's
// position is an index into the sound data, 0 for none, as the driver's offset was.

// Driver functions, by the number MAIN.EXE puts in BX (C: the table at the dispatcher).
const (
	alfxTickRate = 72 // C: the driver table entry for ALFX.DRV

	alfxChannels   = 10
	controlChannel = 9
	rhythmChannel  = 6 // channels 6-8 are the drums in rhythm mode
	alfxQueueSize  = 16
	alfxStackSize  = 4 // calls a program can nest (C: +0xc..+0x13)
	octave         = 12
	blockStep      = 16 // a note's high nibble is its block, in the program's notes
	maxOpcode      = 74 // higher opcodes run the last one's handler

	instrumentTable = 1000 // the instrument offsets follow the 500 program offsets
	maxProgram      = 500  // with 16-bit program numbers, higher ones do nothing
	opcodeBase      = 128

	levelMask    = 63 // total level bits of a level register
	keyScaleBits = 192
	keyOn        = 1 << 5
	blockBits    = 28 // block bits of the B0 register
	fnumHighBits = 3
	fnumBits     = 1023
	rhythmOn     = 1 << 5
	drumBits     = 31
	tremoloDepth = 1 << 7
	vibratoDepth = 1 << 6
	beatRemember = 1 << 15 // beat table index bit the driver drops (C: and 0x7fff)

	slideTop    = 734 // a slide past this halves the frequency number a block up
	slideBottom = 388 // and below this doubles it a block down

	blockChannels = 1 << 0 // flags: programs of channels 0-8 (the effects) are refused
	blockControl  = 1 << 1 // programs of the control channel (the music) are refused
	programBegun  = 1 << 3 // a program was queued since the last tick

	randomSeed = 4660
	randomStep = 37448
)

// OPL register numbers the driver writes (per operator add the operator offset, per
// channel the channel).
const (
	regWaveSelect   = 1
	regTimerControl = 8
	regOpChar       = 32
	regOpLevel1     = 64
	regOpLevel2     = 67
	regOpAttack1    = 96
	regOpAttack2    = 99
	regOpSustain1   = 128
	regOpSustain2   = 131
	regFnum         = 160
	regKeyBlock     = 176
	regDrums        = 189
	regConnection   = 192
	regWave1        = 224
	regWave2        = 227

	waveSelectOn = 1 << 5
	silent       = 63
	fastest      = 255
)

// operatorOffset is each channel's first operator register offset; the control channel
// reads the byte after the table, the frequency table's first (sic).
var operatorOffset = [alfxChannels]byte{0, 1, 2, 8, 9, 10, 16, 17, 18, 52}

// noteFnum is the frequency number of each note of an octave.
var noteFnum = [octave]int{308, 327, 346, 367, 388, 412, 436, 462, 489, 519, 549, 582}

// The bend tables in alfxTables, 32 bytes each, and the beat tables.
const bendTableSize = 32

var beatTables = [...]int{460, 590, 460, 590, 718, 590}

// Effects a channel runs each tick (C: the function pointers at +0x2b and +0x2d).
const (
	effectNone = iota
	effectVibrato
	effectSlide
	effectLevelCycle
)

// alfxChannel is a channel's state (C: the 70-byte structs at 0x3da).
type alfxChannel struct {
	levelAdjust      byte // +0: added to the levels; clearing the channel keeps it
	pos              int  // +3
	duration         byte // +7: ticks until the next note
	loopCount        byte // +8
	octaveAdd        byte // +9
	priority         int8 // +0xa
	depth            int  // +0xb
	stack            [alfxStackSize]int
	transpose        int8 // +0x14
	slideSpeed       byte // +0x15
	slideCounter     byte
	slideStep        int16 // +0x17
	vibratoStep      uint16
	vibratoDepth     byte // +0x1b
	vibratoCount     byte
	vibratoLength    byte
	vibratoDelay     byte // +0x1e
	vibratoSpeed     byte
	vibratoCounter   byte
	vibratoWait      byte // +0x21
	volumeAdjust     byte // +0x22
	keyOffAt         byte // +0x23
	fineTune         byte // +0x26
	tempo            byte // +0x27
	tempoCounter     byte
	fnum             byte // +0x29: the A0 register's value
	keyBlock         byte // +0x2a: the B0 register's value
	effect           int  // +0x2b
	levelEffect      int  // +0x2d
	durationFraction byte // +0x2f
	level1, level2   byte // +0x30, +0x31: the instrument's level registers
	volume           byte // +0x32
	additive         byte // +0x33
	unused34         [2]byte
	keyOffEarly      byte // +0x37
	randomDuration   byte // +0x38
	cycleSpeed       byte // +0x3b
	cycleCounter     byte
	cycleLength      byte
	cycleIndex       int8 // +0x3e
	cycleRegister    byte
	cycleTable       int  // +0x40
	globalTempo      byte // +0x42
	note             byte // +0x43
	bend             int8 // +0x44
	master           byte // +0x45: the music or effects volume
}

// drum is one rhythm instrument's level parts (C: 0x38a..0x398).
type drum struct {
	register            int
	base, adjust, extra byte
}

// The drums, in the order of the level opcodes' bits: hi-hat, cymbal, tom, snare, bass.
const drumCount = 5

// alfx is the driver.
type alfx struct {
	chip opl
	data []byte

	ch      [alfxChannels]alfxChannel
	current int  // C: 0x37d, the channel being worked on
	opOff   byte // C: 0x389, its operator offset

	queue      [alfxQueueSize]byte
	head, tail int

	flags        uint16 // C: 0x375
	begun        byte   // C: 0x373
	musicVolume  byte   // C: 0x377
	effectVolume byte   // C: 0x378
	skipFlag     byte   // C: 0x379
	wideNumbers  byte   // C: 0x37a: program and instrument numbers take two bytes
	lastDuration byte   // C: 0x374
	tempo        byte   // C: 0x37f, the global tempo
	marker       byte   // C: 0x372, set by programs for the game to read
	drumsKeyed   byte   // C: 0x37e: the drums keyed on; not 0 means rhythm mode
	drumShadow   byte   // C: 0x388, the depth bits of the drum register
	seed         uint16 // C: 0x385
	drums        [drumCount]drum

	beatMode            byte // C: 0x380
	beatValue           byte // C: 0x381
	beatIndex           uint16
	beatTable, beatFnum int

	beatLength, beatLeft, beatCounter, beatCount, beatSeen byte // C: 0x399..0x39d
}

type opl interface{ write(reg int, val byte) }

func newALFX(chip opl) *alfx {
	d := &alfx{chip: chip, musicVolume: fastest, effectVolume: fastest, tempo: 1,
		beatMode: 1, beatValue: fastest, seed: randomSeed, beatCounter: fastest}
	d.drums = [drumCount]drum{{register: 81}, {register: 85}, {register: 82}, {register: 84}, {register: 83}}
	return d
}

func (d *alfx) write(reg int, val byte) { d.chip.write(reg&fastest, val) }

// byteAt reads the sound data; past its end reads 0 (fix: the driver read on).
func (d *alfx) byteAt(i int) byte {
	if i < 0 || i >= len(d.data) {
		return 0
	}
	return d.data[i]
}

func (d *alfx) wordAt(i int) int { return int(d.byteAt(i)) | int(d.byteAt(i+1))<<8 }

// program is where program n starts.
func (d *alfx) program(n int) int { return d.wordAt(2 * n) }

// instrument is where instrument n starts.
func (d *alfx) instrument(n int) int { return d.wordAt(instrumentTable + 2*n) }

// random steps the driver's generator (C: 0x1988).
func (d *alfx) random() uint16 {
	v := d.seed + randomStep
	d.seed = v>>3 | v<<13
	return d.seed
}

// init empties the queue and resets the chip (C: function 2, 0xcf6).
func (d *alfx) init() {
	d.head, d.tail = 0, 0
	d.reset()
}

// reset silences the chip and clears the channels (C: function 3, 0xca2).
func (d *alfx) reset() {
	d.seed = randomSeed
	d.write(regWaveSelect, waveSelectOn)
	d.write(regTimerControl, 0)
	d.write(regDrums, 0)
	d.beatMode = 0
	for c := alfxChannels - 1; c >= 0; c-- {
		if c != controlChannel {
			d.write(regOpLevel1+int(operatorOffset[c]), silent)
			d.write(regOpLevel2+int(operatorOffset[c]), silent)
		}
		d.ch[c].clear()
	}
}

// clear resets a channel for a new program (C: 0x823).
func (c *alfxChannel) clear() {
	*c = alfxChannel{levelAdjust: c.levelAdjust, tempo: fastest, keyOffEarly: 1}
}

// setData points the driver at the sound data (C: function 4, 0xdf7).
func (d *alfx) setData(b []byte) { d.data = b }

// queueProgram queues program n to start on the next tick, unless the flags refuse its
// channel; program 0 is never refused (C: function 6, 0xd57).
func (d *alfx) queueProgram(n int) {
	d.begun = 1
	d.flags |= programBegun
	ch := d.byteAt(d.program(n))
	if n != 0 {
		if ch == controlChannel {
			if d.flags&blockControl != 0 {
				return
			}
		} else if d.flags&blockChannels != 0 {
			return
		}
	}
	d.queue[d.tail] = byte(n)
	d.tail = (d.tail + 1) % alfxQueueSize
}

// channelProgram is where channel n's program is, 0 if it is idle (C: function 7).
func (d *alfx) channelProgram(n int) int { return d.ch[n&fastest%alfxChannels].pos }

// stopChannels stops channel n, or all of them for a negative n (C: function 8, 0x17e9).
func (d *alfx) stopChannels(n int) {
	first, last := n, n
	if n < 0 {
		first, last = 0, controlChannel
	}
	for i := first; i <= last; i++ {
		d.current = i
		c := &d.ch[i]
		c.priority, c.pos = 0, 0
		if i != controlChannel {
			d.keyOff(c)
		}
	}
}

// programByte reads byte off of program n (C: function 9, 0x7db).
func (d *alfx) programByte(n, off int) byte { return d.byteAt(d.program(n) + off) }

// setProgramByte writes it and returns the old one (C: function 10, 0x7b7).
func (d *alfx) setProgramByte(n, off int, v byte) byte {
	i := d.program(n) + off
	old := d.byteAt(i)
	if i >= 0 && i < len(d.data) {
		d.data[i] = v
	}
	return old
}

// setBeat moves the beat tables to index v and writes channel 0's frequency from it
// (C: function 12, 0xdb3).
func (d *alfx) setBeat(v uint16) {
	d.beatIndex = v
	i := int(v &^ beatRemember)
	d.beatValue = alfxTables[d.beatTable+i]
	d.write(regFnum, alfxTables[d.beatFnum+i])
}

// setFlags sets bits of the flags and returns the old ones (C: function 16, 0x791).
func (d *alfx) setFlags(mask uint16) uint16 { old := d.flags; d.flags |= mask; return old }

// clearFlags clears them (C: function 17, 0x7a3).
func (d *alfx) clearFlags(mask uint16) uint16 { old := d.flags; d.flags &^= mask; return old }

// setMusicVolume sets the volume of channels 0-5 (C: function 18, 0x6f3).
func (d *alfx) setMusicVolume(v byte) {
	d.musicVolume = v
	d.setVolumes(0, rhythmChannel, v)
}

// setEffectVolume sets the volume of channels 6-8 (C: function 19, 0x742).
func (d *alfx) setEffectVolume(v byte) {
	d.effectVolume = v
	d.setVolumes(rhythmChannel, controlChannel, v)
}

func (d *alfx) setVolumes(from, to int, v byte) {
	for i := from; i < to; i++ {
		c := &d.ch[i]
		c.master = v
		op := int(operatorOffset[i])
		d.write(regOpLevel2+op, c.carrierLevel())
		d.write(regOpLevel1+op, c.modulatorLevel())
	}
}

// scaledLevel is a level register's value: the instrument's level, with the adjustments
// and the volumes added (C: 0xacd, 0xb1f).
func (c *alfxChannel) scaledLevel(reg byte, scaled bool) byte {
	l := reg & levelMask
	if scaled {
		base := l + c.volumeAdjust + c.levelAdjust
		l = (c.volume ^ levelMask)
		if v := int(l) * int(c.master); v != 0 {
			l = byte((v + levelMask) >> 8)
		}
		l = (l ^ levelMask) + base
	}
	if l > levelMask {
		if int8(l) < 0 {
			l = 0
		} else {
			l = levelMask
		}
	}
	if c.master == 0 {
		l = levelMask
	}
	return l | reg&keyScaleBits
}

// modulatorLevel is scaled only in additive connection, when the modulator sounds.
func (c *alfxChannel) modulatorLevel() byte { return c.scaledLevel(c.level1, c.additive != 0) }

func (c *alfxChannel) carrierLevel() byte { return c.scaledLevel(c.level2, true) }

// updateLevels writes the current channel's levels (C: 0xd27).
func (d *alfx) updateLevels(c *alfxChannel) {
	op := int(operatorOffset[d.current])
	d.write(regOpLevel2+op, c.carrierLevel())
	if c.additive != 0 {
		d.write(regOpLevel1+op, c.modulatorLevel())
	}
}

// tick is the timer interrupt (C: entry 3, 0xc76).
func (d *alfx) tick() {
	if d.begun > 0 {
		d.begun--
	}
	if d.begun == 0 {
		d.flags &^= programBegun
	}
	d.startQueued()
	d.updateChannels()
	d.countBeat()
}

// startQueued starts the queued programs whose priority is high enough (C: 0x8bf).
func (d *alfx) startQueued() {
	for d.head != d.tail {
		p := d.program(int(d.queue[d.head]))
		d.head = (d.head + 1) % alfxQueueSize
		ch := int(d.byteAt(p))
		if ch == 0 {
			d.beatMode = 0
		}
		if ch >= alfxChannels {
			continue // fix: the driver took a channel past its table
		}
		d.start(ch, p+1, ch != controlChannel)
	}
}

// start starts the program at p (its priority) on channel ch if the priority is enough;
// silence keys the channel's sound off first (C: 0x8f5, 0xf05).
func (d *alfx) start(ch, p int, silence bool) {
	c := &d.ch[ch]
	pri := int8(d.byteAt(p))
	if pri < c.priority {
		return
	}
	c.clear()
	c.priority = pri
	c.pos = p + 1
	c.tempo, c.tempoCounter = fastest, fastest
	c.duration = 1
	if ch >= rhythmChannel {
		c.master = d.effectVolume
	} else {
		c.master = d.musicVolume
	}
	if silence {
		d.silence(ch)
	}
}

// silence sets the channel's envelopes to the fastest and keys it on and off, except
// for the drums in rhythm mode (C: 0x86e). It keys on before the end (sic).
func (d *alfx) silence(ch int) {
	if d.drumsKeyed != 0 && ch >= rhythmChannel {
		return
	}
	op := int(operatorOffset[ch])
	d.write(regOpAttack1+op, fastest)
	d.write(regOpAttack2+op, fastest)
	d.write(regOpSustain1+op, fastest)
	d.write(regOpSustain2+op, fastest)
	d.write(regKeyBlock+ch, 0)
	d.write(regKeyBlock+ch, keyOn)
}

// keyOff releases the current channel's note (C: 0x952).
func (d *alfx) keyOff(c *alfxChannel) {
	if d.current == controlChannel || (d.drumsKeyed != 0 && d.current >= rhythmChannel) {
		return
	}
	c.keyBlock &^= keyOn
	d.write(regKeyBlock+d.current, c.keyBlock)
}

// setNote sets the current channel's frequency for a note: its high nibble is the block,
// the low one the note (C: 0x97c).
func (d *alfx) setNote(c *alfxChannel, note byte) {
	c.note = note
	d.writeNote(c)
}

// writeNote works out and writes the frequency of the channel's note, with its bend
// (C: 0x97c, and 0x1320 for the bend opcode).
func (d *alfx) writeNote(c *alfxChannel) {
	block := c.note&^(blockStep-1) + c.octaveAdd
	n := int8(c.note&(blockStep-1)) + c.transpose
	switch {
	case n >= octave:
		n -= octave
		block += blockStep
	case n < 0:
		n += octave
		block -= blockStep
	}
	i := int(n) % octave // fix: the driver read past the table for larger transpositions
	if i < 0 {
		i += octave
	}
	f := noteFnum[i] + int(c.fineTune)
	v := (int(block>>2)&blockBits|f>>8)<<8 | f&fastest
	if c.bend > 0 {
		v += int(alfxTables[(int(c.note&(blockStep-1))+2)*bendTableSize+int(c.bend)])
	} else if c.bend < 0 {
		v -= int(alfxTables[int(c.note&(blockStep-1))*bendTableSize-int(c.bend)])
	}
	c.keyBlock = c.keyBlock&keyOn | byte(v>>8)
	c.fnum = byte(v)
	d.write(regFnum+d.current, c.fnum)
	d.write(regKeyBlock+d.current, c.keyBlock)
}

// keyOnNote keys the note on and starts the vibrato (C: 0xa4a).
func (d *alfx) keyOnNote(c *alfxChannel) {
	c.keyBlock |= keyOn
	d.write(regKeyBlock+d.current, c.keyBlock)
	shift := (9 - c.vibratoDepth) & 31
	f := uint16(c.fnum) | uint16(c.keyBlock)<<8
	c.vibratoStep = (f & fnumBits >> shift) & fastest
	c.vibratoWait = c.vibratoDelay
}

// setDuration sets how many ticks the note lasts, with the random part or the early key
// off (C: 0xa84).
func (d *alfx) setDuration(c *alfxChannel, dur byte) {
	d.lastDuration = dur
	if c.randomDuration != 0 {
		c.duration = dur + byte(d.random())&c.randomDuration
		return
	}
	if c.durationFraction != 0 {
		step := dur >> 3
		c.keyOffAt = 0
		for range c.durationFraction {
			c.keyOffAt += step
		}
	}
	c.duration = dur
}

// noteDone is what the note opcodes return: 1 for a note that lasts, 0 to go on.
func (d *alfx) noteDone() int {
	if d.lastDuration != 0 {
		return 1
	}
	return 0
}

// updateChannels runs each channel, from the control channel down (C: 0xb6a).
func (d *alfx) updateChannels() {
	for d.current = controlChannel; d.current >= 0; d.current-- {
		c := &d.ch[d.current]
		if c.pos == 0 {
			continue
		}
		d.opOff = operatorOffset[d.current]
		if c.globalTempo != 0 {
			c.tempo = d.tempo
		}
		sum := int(c.tempoCounter) + int(c.tempo)
		c.tempoCounter = byte(sum)
		if sum > fastest {
			c.duration--
			if c.duration == 0 {
				if !d.run(c) {
					continue
				}
			} else {
				if c.duration == c.keyOffAt {
					d.keyOff(c)
				}
				if c.duration == c.keyOffEarly && d.current != controlChannel {
					d.keyOff(c)
				}
			}
		}
		d.runEffect(c, c.effect)
		d.runEffect(c, c.levelEffect)
	}
}

// run reads the channel's program until a note that lasts or an opcode that waits;
// false skips the effects this tick.
func (d *alfx) run(c *alfxChannel) bool {
	si := c.pos
	for {
		lo, hi := d.byteAt(si), d.byteAt(si+1)
		si += 2
		if lo < opcodeBase {
			d.setNote(c, lo)
			d.keyOnNote(c)
			d.setDuration(c, hi)
			if d.lastDuration != 0 {
				c.pos = si
				return true
			}
			continue
		}
		op := min(int(lo-opcodeBase), maxOpcode)
		r := d.opcode(c, op, hi, &si)
		c.pos = si
		switch r {
		case 0:
			continue
		case 1:
			return true
		default:
			return false
		}
	}
}

// countBeat runs the beat counter that programs wait on (C: 0xc3c).
func (d *alfx) countBeat() {
	sum := int(d.beatCounter) + int(d.tempo)
	d.beatCounter = byte(sum)
	if sum > fastest {
		d.beatLeft--
		if d.beatLeft == 0 {
			d.beatLeft = d.beatLength
			d.beatCount++
		}
	}
}

// writeInstrument writes the 11 bytes of the instrument at p to the current channel's
// operators at op (C: 0xe3c).
func (d *alfx) writeInstrument(c *alfxChannel, p int, op byte) {
	o := int(op)
	next := func() byte { b := d.byteAt(p); p++; return b }
	d.write(regOpChar+o, next())
	d.write(regOpChar+3+o, next())
	conn := next()
	d.write(regConnection+d.current, conn)
	c.additive = conn & 1
	d.write(regWave1+o, next())
	d.write(regWave2+o, next())
	c.level1 = next()
	d.write(regOpLevel1+o, c.modulatorLevel())
	c.level2 = next()
	d.write(regOpLevel2+o, c.carrierLevel())
	d.write(regOpAttack1+o, next())
	d.write(regOpAttack2+o, next())
	d.write(regOpSustain1+o, next())
	d.write(regOpSustain2+o, next())
}

// runEffect runs a channel's per-tick effect.
func (d *alfx) runEffect(c *alfxChannel, e int) {
	switch e {
	case effectVibrato:
		d.vibrato(c)
	case effectSlide:
		d.slide(c)
	case effectLevelCycle:
		d.levelCycle(c)
	}
}

// vibrato swings the frequency up and down (C: 0x1858).
func (d *alfx) vibrato(c *alfxChannel) {
	if c.vibratoWait != 0 {
		c.vibratoWait--
		return
	}
	sum := int(c.vibratoCounter) + int(c.vibratoSpeed)
	c.vibratoCounter = byte(sum)
	if sum <= fastest {
		return
	}
	c.vibratoCount--
	if c.vibratoCount == 0 {
		c.vibratoStep = -c.vibratoStep
		c.vibratoCount = c.vibratoLength
	}
	f := (uint16(c.fnum)|uint16(c.keyBlock)<<8)&fnumBits + c.vibratoStep
	c.fnum = byte(f)
	c.keyBlock = c.keyBlock&^fnumHighBits | byte(f>>8) // sic: a negative sum sets every bit
	d.write(regFnum+d.current, c.fnum)
	d.write(regKeyBlock+d.current, c.keyBlock)
}

// slide moves the frequency by a step, changing block at the ends (C: 0x18f5).
func (d *alfx) slide(c *alfxChannel) {
	sum := int(c.slideCounter) + int(c.slideSpeed)
	c.slideCounter = byte(sum)
	if sum <= fastest {
		return
	}
	f := int16(c.fnum) | int16(c.keyBlock&fnumHighBits)<<8
	block := c.keyBlock & blockBits
	key := c.keyBlock & keyOn
	if c.slideStep >= 0 {
		f += c.slideStep
		if f >= slideTop {
			f >>= 1
			if f&fnumBits == 0 {
				f++
			}
			block = (block + 4) & blockBits
		}
	} else {
		f += c.slideStep
		if f <= slideBottom {
			f <<= 1
			if f&fnumBits == 0 {
				f--
			}
			block = (block - 4) & blockBits
		}
	}
	f &= fnumBits
	c.fnum = byte(f)
	d.write(regFnum+d.current, c.fnum)
	c.keyBlock = byte(f>>8) | block | key
	d.write(regKeyBlock+d.current, c.keyBlock)
}

// levelCycle writes the next of a table of values to a register of the channel's
// operators (C: 0x18c3). The table is read from the sound data (fix: the driver read it
// through whichever segment the last channel left loaded).
func (d *alfx) levelCycle(c *alfxChannel) {
	sum := int(c.cycleCounter) + int(c.cycleSpeed)
	c.cycleCounter = byte(sum)
	if sum <= fastest {
		return
	}
	c.cycleIndex--
	if c.cycleIndex < 0 {
		c.cycleIndex = int8(c.cycleLength)
	}
	d.write(int(c.cycleRegister+d.opOff), d.byteAt(c.cycleTable+int(byte(c.cycleIndex))))
}
