package sound

// The program opcodes of ALFX.DRV (C: the table at 0x1a9a). Each gets its first argument;
// si is just past it, and handlers taking no argument step back over it. They return 0 to
// go on reading, 1 when the channel's note lasts, 2 to stop for this tick, without the
// effects.

const (
	opSetLoop         = 0
	opLoop            = 1
	opStartProgram    = 2
	opKeyOffEarly     = 3
	opJump            = 4
	opCall            = 5
	opReturn          = 6
	opOctave          = 7
	opEnd             = 8
	opRest            = 9
	opWriteRegister   = 10
	opTie             = 11
	opTranspose       = 12
	opLevelCycle      = 13
	opStopChannel     = 14
	opWaitProgram     = 15
	opInstrument      = 16
	opSlide           = 17
	opSlideOff        = 18
	opFineTune        = 19
	opVibrato         = 21
	opSkip            = 25
	opPriority        = 26
	opSetBeat         = 28
	opWaitBeat        = 29
	opVolumeAdjust    = 30
	opHold            = 32
	opRetrigger       = 33
	opFraction        = 36
	opGlobalTempo     = 38
	opLevelCycleOff   = 39
	opTempo           = 41
	opVolume          = 43
	opSetLevelAdjust  = 44
	opAddLevelAdjust  = 45
	opTremoloDepth    = 46
	opVibratoDepth    = 47
	opAddVolumeAdjust = 48
	opClearChannel    = 51
	opRandomPitch     = 53
	opVibratoOff      = 54
	opBend            = 57
	opUseGlobalTempo  = 58
	opNop             = 59
	opRandomDuration  = 60
	opAddTempo        = 61
	opBeatTables      = 63
	opBeatOff         = 64
	opDrumInstruments = 65
	opKeyDrums        = 66
	opDrumsOff        = 67
	opDrumLevel       = 68
	opAddDrumLevel    = 69
	opDrumLevelAdjust = 70
	opMarker          = 71
	opFollowTempo     = 72
	opUnused          = 73

	goOn, lasts, wait = 0, 1, 2

	drumHiHat, drumCymbal, drumTom, drumSnare, drumBass = 0, 1, 2, 3, 4
	drumKeyMask                                         = 47 // key on and the block bits of a drum's B0
	durationFractionMask                                = 7
	instrumentLevel1, instrumentLevel2                  = 5, 6
)

func (d *alfx) opcode(c *alfxChannel, op int, arg byte, si *int) int {
	next := func() byte { b := d.byteAt(*si); *si++; return b }
	// number is a program or instrument number: the argument, or with wide numbers it and
	// the next byte.
	number := func() int {
		if d.wideNumbers == 0 {
			return int(arg)
		}
		*si--
		n := d.wordAt(*si)
		*si += 2
		return n
	}
	jump := func() int {
		*si--
		off := int16(d.wordAt(*si))
		*si += 2
		return int(off)
	}
	other := func() *alfxChannel { return &d.ch[int(arg)%alfxChannels] } // fix: in range
	switch op {
	case opSetLoop:
		c.loopCount = arg
	case opLoop:
		*si++
		c.loopCount--
		if c.loopCount != 0 {
			*si -= 2
			off := int16(d.wordAt(*si))
			*si += 2 + int(off)
		}
	case opStartProgram:
		n := number()
		if d.wideNumbers != 0 && n >= maxProgram {
			return goOn
		}
		p := d.program(n)
		ch := int(d.byteAt(p))
		if ch < alfxChannels && int8(d.byteAt(p+1)) >= d.ch[ch].priority {
			d.begun = 1
			d.flags |= programBegun
			d.start(ch, p+1, true)
		}
	case opKeyOffEarly:
		c.keyOffEarly = arg
	case opJump:
		off := jump()
		*si += off
	case opCall:
		off := jump()
		if c.depth < alfxStackSize { // fix: the driver wrote on past its stack
			c.stack[c.depth] = *si
			c.depth++
		}
		*si += off
	case opReturn:
		if c.depth > 0 { // fix: the driver read below its stack
			c.depth--
		}
		*si = c.stack[c.depth]
	case opOctave:
		c.octaveAdd = arg
	case opRest:
		d.setDuration(c, arg)
		d.keyOff(c)
		return d.noteDone()
	case opWriteRegister:
		d.write(int(arg), next())
	case opTie:
		d.setNote(c, arg)
		d.setDuration(c, next())
		return d.noteDone()
	case opTranspose:
		c.transpose = int8(arg)
	case opLevelCycle:
		c.cycleCounter, c.cycleSpeed = arg, arg
		n := next()
		c.cycleLength, c.cycleIndex = n, int8(n)
		c.cycleRegister = next()
		c.cycleTable = d.wordAt(*si)
		*si += 2
		c.levelEffect = effectLevelCycle
	case opStopChannel:
		t := other()
		t.duration, t.priority, t.pos = 0, 0, 0
	case opWaitProgram:
		n := number()
		if d.ch[int(d.byteAt(d.program(n)))%alfxChannels].pos == 0 {
			return goOn
		}
		*si -= 2 // sic: with wide numbers this lands on the argument's second byte
		return wait
	case opInstrument:
		d.writeInstrument(c, d.instrument(number()), d.opOff)
	case opSlide:
		c.slideSpeed = arg
		hi := next()
		c.slideStep = int16(uint16(hi)<<8 | uint16(next()))
		c.effect = effectSlide
		c.slideCounter = fastest
	case opSlideOff:
		*si--
		c.effect = effectNone
		c.slideStep = 0
	case opFineTune:
		c.fineTune = arg
	case opVibrato:
		c.vibratoSpeed = arg
		c.vibratoDepth = next()
		n := next()
		c.vibratoCount = n + 1
		c.vibratoLength = n << 1
		c.vibratoDelay = next()
		c.effect = effectVibrato
	case opSkip:
		if d.skipFlag != 0 {
			lo := d.byteAt(*si)
			*si += 2
			if int8(lo) < 0 {
				*si -= 2
			}
		}
	case opPriority:
		c.priority = int8(arg)
	case opSetBeat:
		d.beatLength = arg >> 1
		d.beatLeft = d.beatLength
		d.beatCounter = fastest
		d.beatCount, d.beatSeen = 0, 0
	case opWaitBeat:
		hit := arg&d.beatCount != 0
		if d.beatSeen == 0 {
			if !hit {
				d.beatSeen++
			}
		} else if hit {
			d.beatSeen = 0
			return goOn
		}
		*si -= 2
		c.duration = 1
		return wait
	case opVolumeAdjust:
		c.volumeAdjust = arg
		d.updateLevels(c)
	case opHold:
		d.setDuration(c, arg)
		return d.noteDone()
	case opRetrigger:
		d.setDuration(c, arg)
		d.keyOnNote(c)
		return d.noteDone()
	case opFraction:
		c.durationFraction = arg & durationFractionMask
	case opGlobalTempo:
		d.tempo = arg
	case opLevelCycleOff:
		*si--
		c.levelEffect = effectNone
	case opTempo:
		c.tempo = arg
	case opVolume:
		c.volume = arg
	case opSetLevelAdjust, opAddLevelAdjust:
		v := next()
		old := d.current
		d.current = int(arg) % alfxChannels
		t := &d.ch[d.current]
		if op == opAddLevelAdjust {
			v += t.levelAdjust
		}
		t.levelAdjust = v
		d.updateLevels(t)
		d.current = old
	case opTremoloDepth:
		d.drumShadow = d.drumShadow&^tremoloDepth | (arg&1)<<7
		d.write(regDrums, d.drumShadow)
	case opVibratoDepth:
		d.drumShadow = d.drumShadow&^vibratoDepth | (arg&1)<<6
		d.write(regDrums, d.drumShadow)
	case opAddVolumeAdjust:
		c.volumeAdjust += arg
		d.updateLevels(c)
	case opClearChannel:
		if arg == 0 {
			d.beatMode = 0
		}
		old := d.current
		d.current = int(arg) % alfxChannels
		t := &d.ch[d.current]
		t.duration, t.priority, t.pos, t.levelAdjust = 0, 0, 0, 0
		if d.current != controlChannel {
			o := int(operatorOffset[d.current])
			d.write(regConnection+d.current, 0)
			d.write(regOpLevel2+o, silent)
			d.write(regOpSustain2+o, fastest)
			d.write(regKeyBlock+d.current, 0)
		}
		d.current = old
	case opRandomPitch:
		mask := uint16(arg)<<8 | uint16(next())
		f := uint16(c.keyBlock&drumBits)<<8 | uint16(c.fnum)
		f += d.random() & mask
		d.write(regFnum+d.current, byte(f))
		d.write(regKeyBlock+d.current, byte(f>>8)|c.keyBlock&keyOn)
	case opVibratoOff:
		*si--
		c.effect = effectNone
	case opBend:
		c.bend = int8(arg)
		d.writeNote(c)
	case opUseGlobalTempo:
		*si--
		c.tempo = d.tempo
	case opNop:
		*si--
	case opRandomDuration:
		c.randomDuration = arg
	case opAddTempo:
		t := int(c.tempo) + int(int8(arg))
		switch {
		case int8(arg) >= 0 && t > fastest:
			t = fastest
		case int8(arg) < 0 && t < 0:
			t = 1
		}
		c.tempo = byte(t)
	case opBeatTables:
		d.beatMode = arg
		n := int(next()) % (len(beatTables) - 1) // fix: in range
		d.beatTable, d.beatFnum = beatTables[n], beatTables[n+1]
		if arg == 2 {
			d.beatIndex = 0
			d.beatValue = alfxTables[d.beatTable]
			d.write(regFnum, alfxTables[d.beatFnum])
		}
	case opBeatOff:
		*si--
		d.beatMode = 0
	case opDrumInstruments:
		d.drumInstruments(c, arg, next)
	case opKeyDrums:
		d.write(regDrums, ^(arg&drumBits)&d.drumsKeyed|rhythmOn) // sic: without the depth bits
		d.drumsKeyed |= arg
		d.write(regDrums, d.drumsKeyed|d.drumShadow|rhythmOn)
	case opDrumsOff:
		*si--
		d.drumsKeyed = 0
		d.write(regDrums, d.drumShadow&(tremoloDepth|vibratoDepth))
	case opDrumLevel, opAddDrumLevel, opDrumLevelAdjust:
		v := next()
		for i := range d.drums {
			if arg&(1<<i) == 0 {
				continue
			}
			dr := &d.drums[i]
			var l byte
			switch op {
			case opDrumLevel:
				dr.extra = v
				l = drumLevel(dr.base + dr.adjust + dr.extra)
			case opAddDrumLevel:
				l = drumLevel(v + dr.base + dr.adjust + dr.extra)
				dr.adjust = l
			default:
				dr.adjust = v
				l = drumLevel(dr.base + v + dr.extra)
			}
			d.write(dr.register, l)
		}
	case opMarker:
		d.marker = arg
	case opFollowTempo:
		c.globalTempo = arg
	case opUnused:
		c.unused34 = [2]byte{arg, next()}
	default:
		c.priority = 0
		if d.current != controlChannel {
			d.keyOff(c)
		}
		*si = 0
		return wait
	}
	return goOn
}

// drumLevel keeps a drum's level in the register's 6 bits; the key scale bits are lost
// (C: 0xeb0).
func drumLevel(v byte) byte { return min(v, levelMask) }

// drumInstruments sets the instruments of the three drum channels, the bass drum's from
// the argument and the others' from the next bytes, then their frequencies, and turns
// rhythm mode on (C: 0x1481). The instruments' channel fields go to the channel running
// the program (sic).
func (d *alfx) drumInstruments(c *alfxChannel, arg byte, next func() byte) {
	oldCurrent, oldOp := d.current, d.opOff
	n := arg
	for ch := rhythmChannel; ch < controlChannel; ch++ {
		if ch > rhythmChannel {
			n = next()
		}
		d.current = ch
		d.opOff = operatorOffset[ch]
		p := d.instrument(int(n))
		// fix: the driver read the levels of channels 7 and 8 through its code segment
		switch ch {
		case rhythmChannel:
			d.drums[drumBass].base = d.byteAt(p + instrumentLevel2)
		case rhythmChannel + 1:
			d.drums[drumHiHat].base = d.byteAt(p + instrumentLevel1)
			d.drums[drumSnare].base = d.byteAt(p + instrumentLevel2)
		default:
			d.drums[drumTom].base = d.byteAt(p + instrumentLevel1)
			d.drums[drumCymbal].base = d.byteAt(p + instrumentLevel2)
		}
		d.writeInstrument(c, p, d.opOff)
	}
	for ch := rhythmChannel; ch < controlChannel; ch++ {
		kb := next() & drumKeyMask
		d.ch[ch].keyBlock = kb
		d.write(regKeyBlock+ch, kb)
		d.write(regFnum+ch, next())
	}
	d.drumsKeyed = rhythmOn
	d.opOff, d.current = oldOp, oldCurrent
}
