package game

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

// Speech (C: s023, o2da3): the voices are VOC files in archives named NN.TLK, 00.TLK
// for the strings of every level and one per level for its own. Each member is named
// after its string, the speaker and the part, so a long speech plays as several parts.

const (
	commonSpeechFlag    = 0x4000 // strings from 00.TLK (C: id & 0x4000)
	levelSpeechOnly     = 1000   // "@NNNN" members are strings from 1000 on
	anyVoice            = '_'    // a member any speaker says
	noVoice             = '0'    // the voice of the archive's first member
	speechPriority      = 255
	speechFile          = "%02d.TLK"
	speechSampleRate    = 22000 // the rate the speech length is reckoned at (C: ovr_2da3_002f)
	notSpeaking         = -1
	timerTicksPerSecond = 60
)

// speechEntry is one member of a speech archive (C: struct VocEnt).
type speechEntry struct {
	id    int
	voice byte
	part  int // nonzero for the parts after a speech's first
	off   int
}

// speechIndex is an archive's members (C: g_abc2, g_abc6).
type speechIndex struct {
	file    *os.File
	size    int
	entries []speechEntry
}

// loadSpeechIndex reads the members of NN.TLK: the level's for a level, 00.TLK's for 0;
// without it the speech goes off and the text on (C: ovr_2da3_0020).
func (g *Game) loadSpeechIndex(level int) {
	slot := &g.levelSpeech
	if level == 0 {
		slot = &g.commonSpeech
	} else {
		for g.levelSpeech != nil && g.updateSpeech() != 0 {
			clock.Delay(1)
		}
	}
	if *slot != nil {
		(*slot).file.Close()
		*slot = nil
	}
	ix, err := g.readSpeechIndex(fmt.Sprintf(speechFile, level), level == 0)
	if err != nil {
		g.speechVolume = 0
		g.textEnabled = true
		return
	}
	*slot = ix
}

// readSpeechIndex reads an archive's directory: offsets and names up to an offset of 0
// (C: ovr_2da3_0034).
func (g *Game) readSpeechIndex(name string, common bool) (*speechIndex, error) {
	f, err := g.Files.Open(name)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	ix := &speechIndex{file: f, size: int(fi.Size())}
	r := io.NewSectionReader(f, 0, fi.Size())
	var off [4]byte
	for {
		if _, err := io.ReadFull(r, off[:]); err != nil {
			break
		}
		o := int(binary.LittleEndian.Uint32(off[:]))
		if o == 0 {
			break
		}
		var name []byte
		for c := [1]byte{}; ; {
			if _, err := r.Read(c[:]); err != nil || c[0] == 0 {
				break
			}
			name = append(name, c[0])
		}
		ix.entries = append(ix.entries, parseSpeechName(strings.ToUpper(string(name)), common, o))
	}
	return ix, nil
}

// parseSpeechName reads a member's string, speaker and part from its name: three hex
// digits in 00.TLK, else three decimal ones or "@" and four (C: "%3X%c%c", "%3d%c%c",
// "@%4d%c").
func parseSpeechName(name string, common bool, off int) speechEntry {
	e := speechEntry{off: off}
	at := func(i int) byte {
		if i < len(name) {
			return name[i]
		}
		return 0
	}
	switch {
	case common:
		e.id = scanNumber(name, 3, 16) | commonSpeechFlag
		e.voice, e.part = at(3), int(at(4))-'0'
	case strings.HasPrefix(name, "@"):
		e.id = scanNumber(name[1:], 4, 10) + levelSpeechOnly
		e.voice = at(5)
	default:
		e.id = scanNumber(name, 3, 10)
		e.voice, e.part = at(3), int(at(4))-'0'
	}
	return e
}

// scanNumber reads up to width digits as sscanf does.
func scanNumber(s string, width, base int) int {
	s = s[:min(width, len(s))]
	n := 0
	for i := range len(s) {
		if _, err := strconv.ParseInt(s[:i+1], base, 32); err != nil {
			break
		}
		n = i + 1
	}
	v, _ := strconv.ParseInt(s[:n], base, 32)
	return int(v)
}

func (g *Game) speechIndexOf(id int) *speechIndex {
	if id&commonSpeechFlag != 0 {
		return g.commonSpeech
	}
	return g.levelSpeech
}

// findSpeech is the first part of string id in voice v: a binary search by string, then
// a walk toward the voice and back to the first part; -1 if not found (C: fn_154c_040d).
func (ix *speechIndex) findSpeech(id int, v byte) int {
	// the search leaves out the first and the last member (C: hdr[1] - 2 from base + 8)
	lo, n := 1, len(ix.entries)-2
	p := -1
	for n > 0 {
		mid := lo + n/2
		switch c := id - ix.entries[mid].id; {
		case c == 0:
			p = mid
			n = 0
		case c > 0:
			lo = mid + 1
			n -= n/2 + 1
		default:
			n /= 2
		}
	}
	if p < 0 {
		return -1
	}
	e := ix.entries[p]
	if e.voice != anyVoice {
		step := 1
		if e.voice > v {
			step = -1
		}
		for e.voice != v && e.id == id {
			p += step
			if p < 0 || p >= len(ix.entries) {
				return -1
			}
			if e = ix.entries[p]; e.voice == noVoice {
				return -1
			}
		}
	}
	for e.part != 0 && p > 0 {
		p--
		e = ix.entries[p]
	}
	return p
}

// speechVoice is the letter of a speaker: a letter itself, else the first of the party
// member's name (C: fn_154c_057b).
func (g *Game) speechVoice(ch int) byte {
	if ch >= 'A' {
		return byte(ch)
	}
	if ch < 0 || ch >= partySize || g.party[ch].Flags&charPresent == 0 || g.party[ch].Name == "" {
		return 0
	}
	return g.party[ch].Name[0]
}

// loadSpeech converts the next part of string id in voice v into the speech sample,
// keeping where the part after it is (C: fn_154c_029f).
func (g *Game) loadSpeech(id int, v byte) bool {
	ix := g.speechIndexOf(id)
	if ix == nil {
		return false
	}
	p := g.speechNext
	if p == 0 {
		if p = ix.findSpeech(id, v); p < 0 {
			return false
		}
	}
	e := ix.entries[p]
	end := ix.size // fix: the last member's end was read past the index
	g.speechNext = 0
	if p+1 < len(ix.entries) {
		next := ix.entries[p+1]
		end = next.off
		if next.id == e.id && next.voice == e.voice {
			g.speechNext, g.speechNextIn = p+1, ix
		}
	}
	b := make([]byte, max(0, end-e.off))
	if _, err := ix.file.ReadAt(b, int64(e.off)); err != nil && err != io.EOF {
		return false
	}
	g.speechSample = sound.ConvertVOC(b)
	return g.speechSample != nil
}

// playSpeech plays string id in a speaker's voice after the speech before it; true if it
// plays, or was the last played (C: fn_154c_01d3).
func (g *Game) playSpeech(id, ch int, _ bool) bool {
	v := g.speechVoice(ch)
	if g.lastSpeech == id && g.lastSpeechVoice == v {
		return true
	}
	if g.speechIndexOf(id) == nil || g.speechVolume == 0 {
		return false
	}
	for g.audio.DigitalActive() && g.audio.DigitalPosition(g.speechSample) != -1 {
		g.idleWork()
		clock.Delay(1)
	}
	if !g.loadSpeech(id, v) {
		return false
	}
	g.lastSpeech, g.lastSpeechVoice = id, v
	r := g.digitalEffectsOn || g.digitalSpeechOn
	if r {
		r = g.audio.PlayDigital(g.speechSample, speechPriority, effectDriverVolume(g.speechVolume))
	}
	g.soundVar = 0
	return r
}

// updateSpeech keeps the speech going: the next part, else the queued speech. It is 2
// while speech plays, 1 while other samples do, else 0 (C: fn_154c_0136).
func (g *Game) updateSpeech() int {
	if g.speechSample != nil && g.audio.DigitalPosition(g.speechSample) != -1 {
		return speechPlaying
	}
	t := clock.Ticks()
	g.lastSpeech, g.lastSpeechVoice = notSpeaking, 0
	r := false
	if g.speechNext != 0 && g.speechNextIn != nil {
		e := g.speechNextIn.entries[g.speechNext]
		r = g.playSpeech(e.id, int(e.voice), e.part != 0)
	}
	if !r && g.queuedSpeech != 0 {
		r = g.playSpeech(g.queuedSpeech, g.queuedSpeaker, g.queuedSpeechStarted)
		g.queuedSpeech = 0
	}
	if r {
		g.speechDelay = clock.Ticks() - t
		return speechPlaying
	}
	if g.audio.DigitalActive() {
		return 1
	}
	return 0
}

// stopSpeech stops the speech playing and what was to follow; with hold the script sees
// it was cut (C: fn_154c_000f).
func (g *Game) stopSpeech(hold bool) {
	if g.speechSample == nil || g.audio.DigitalPosition(g.speechSample) < 0 {
		return
	}
	g.queuedSpeech = 0
	g.speechNext = 0
	g.talkWaitUntil = 0
	g.audio.StopDigital(g.speechSample)
	if hold {
		g.soundVar = 1
	}
}

// playSpeechSample loads a string's speech for the audio menu's speech slider to play
// (C: ovr_2da3_002a).
func (g *Game) playSpeechSample(id int) {
	g.endTalk()
	g.speechNext = 0
	g.loadSpeech(id, 0)
}

// speechLength is how long the speech last played lasts, in ticks (C: ovr_2da3_002f).
func (g *Game) speechLength() int {
	if g.lastSpeech == notSpeaking {
		return 0
	}
	ix := g.speechIndexOf(g.lastSpeech)
	if ix == nil {
		return 0
	}
	p := ix.findSpeech(g.lastSpeech, g.lastSpeechVoice)
	if p < 0 {
		return 0
	}
	first := ix.entries[p]
	end := ix.size
	for p++; p < len(ix.entries); p++ {
		if e := ix.entries[p]; e.id != first.id || e.voice != first.voice {
			end = e.off
			break
		}
	}
	return (end - first.off) * timerTicksPerSecond / speechSampleRate
}
