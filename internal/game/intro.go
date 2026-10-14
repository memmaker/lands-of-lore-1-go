package game

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

// LANDS.EXE's AV engine: TIM files of the same IFF form as MAIN.EXE's, played by their
// own interpreter (C: LANDS.EXE _Load_AV_File, _Process_AV, _Set_Next_Event and the
// opcodes of seg 0543).

const (
	avLines          = 10
	avSamples        = 125
	randMax          = 32768 // the C library's rand() range
	introTextTop     = 160   // the text's rows (C: _Intro_Print)
	introTextRows    = gfx.Height - introTextTop
	introTextBottomY = 188 // where a one-line caption goes (col < 0)
	introVoiceVolume = 208 // (C: 0xd0)
	animDelay        = 10 // the delay LOL_Open_Animation is given by the opcode
)

// avScript is a loaded AV file with its lines (C: struct AVHdr).
type avScript struct {
	avtl  []int
	text  []byte
	lines [avLines]avLine
}

type avLine struct {
	ip, last, next, loop, start int
}

// loadAV reads an AV file (C: _Load_AV_File).
func (g *Game) loadAV(name string) (*avScript, error) {
	file, err := g.lands.files.Read(name)
	if err != nil {
		return nil, err
	}
	a := &avScript{}
	avtl := readIFFChunk(file, tagAVTL)
	a.avtl = make([]int, len(avtl)/2)
	for i := range a.avtl {
		a.avtl[i] = int(int16(binary.LittleEndian.Uint16(avtl[2*i:])))
	}
	a.text = append([]byte(nil), readIFFChunk(file, tagText)...)
	a.lines[0].start = a.word(0)
	return a, nil
}

func (a *avScript) word(i int) int {
	if i < 0 || i >= len(a.avtl) {
		return 0
	}
	return a.avtl[i]
}

func (a *avScript) str(n int) string {
	t := &timScript{text: a.text}
	return t.timString(n)
}

// Results of an opcode.
const (
	avInput    = -3
	avStopLine = -2
	avStopAll  = -1
	avLoopDone = 22
)

// processAV runs every line's due commands, showing the work page after each
// (C: _Process_AV).
func (g *Game) processAV(a *avScript) {
	now := clock.Ticks()
	if l := &a.lines[0]; l.ip == 0 {
		l.ip = l.start
		l.next, l.last = now, now
	}
	run := true
	for i := range a.lines {
		l := &a.lines[i]
		for run && l.ip != 0 && l.next <= clock.Ticks() {
			op := int(int8(a.word(l.ip + timCmdOp)))
			r := g.avOp(a, i, op, l.ip+timCmdParams)
			switch r {
			case avStopAll:
				run = false
				continue
			case avStopLine:
				g.Screen.CopyChangedView(landsWorkPage, landsShadowPage)
				goto nextLine
			case avLoopDone:
				l.loop = 0
			}
			g.setNextEvent(a, l)
			g.Screen.CopyChangedView(landsWorkPage, landsShadowPage)
		}
	nextLine:
	}
}

// setNextEvent moves a line to its next command (C: _Set_Next_Event).
func (g *Game) setNextEvent(a *avScript, l *avLine) {
	if l.ip == 0 {
		return
	}
	l.ip += a.word(l.ip + timCmdLength)
	l.last = l.next
	l.next += a.word(l.ip + timCmdDelay)
}

// The opcodes (C: the AV opcode table).
const (
	avRestart = iota
	avEnd
	avAnimOpen
	avAnimClose
	avBgStart
	avBgEnd
	avFrame
	avText
	avSmpLoad
	avSmpClose
	avSmpPlay
	avSmpStop
	avScoreLoad
	avScoreClose
	avScorePlay
	avScoreStop
	avSndLoad
	avSndClose
	avSndPlay
	avSndStop
	avLoopStart
	avLoopEnd
	avStopLoop
	avTimeStart
	avTimeStop
	avUser
	avStartTL
	avStopTL
	avInputOp
	avButton
	avTrap
)

const avPageFlag = 1 << 14 // a frame drawn on the picture page

func (g *Game) avOp(a *avScript, line, op, p int) int {
	l := g.lands
	ln := &a.lines[line]
	arg := func(i int) int { return a.word(p + i) }
	switch op {
	case avRestart:
		for i := 1; i < landsAnimations; i++ {
			l.anims[i] = landsAnim{}
		}
		a.lines[0].ip = a.lines[0].start
		a.lines[0].last = clock.Ticks()
		// sic: Set_Next_Event follows and skips the first command
	case avEnd:
		ln.ip = 0
		if line == 0 {
			l.avEnd = true
		}
		return avStopLine
	case avAnimOpen:
		g.landsOpenAnimation(arg(0), a.str(arg(1)), arg(2), arg(3), animDelay, arg(4), arg(5))
	case avAnimClose:
		n := arg(0)
		if l.anims[n].wsa == nil {
			return 0
		}
		l.anims[n] = landsAnim{}
	case avBgStart:
		t := &a.lines[arg(0)]
		if t.start == 0 {
			t.start = a.word(arg(0))
		}
		t.ip = t.start
	case avBgEnd:
		a.lines[arg(0)].ip = 0
	case avFrame:
		an := &l.anims[arg(0)]
		page := landsWorkPage
		if an.flags&avPageFlag != 0 {
			page = landsPicPage
		}
		g.drawWSA(an.wsa, arg(1), an.x, an.y, page, an.flags&animDrawFlagsMask, nil, nil)
	case avText:
		g.introPrint(g.introText(arg(0)), arg(1))
	case avSmpLoad:
		l.samples[arg(1)] = g.loadQuickSample(a.str(arg(0)))
	case avSmpClose:
		l.samples[arg(0)] = nil
	case avSmpPlay:
		vol := arg(1) * 255 / percent
		if s := l.samples[arg(0)]; s == nil {
			g.playLandsSound(arg(0), vol)
		} else {
			g.playLongSample(s, 0, vol)
		}
	case avSmpStop:
		g.stopLongSample()
	case avScoreLoad:
		g.loadScore(strings.TrimSuffix(a.str(arg(0)), scoreExtension))
	case avScoreClose:
	case avScorePlay:
		g.playScore(arg(0))
	case avScoreStop:
		g.audio.Play(sound.Music, scoreStop, -1)
	case avSndLoad, avSndClose, avSndStop:
	case avSndPlay:
		g.playLandsSound(arg(0), 255)
	case avLoopStart:
		ln.loop = ln.ip
	case avLoopEnd:
		if ln.loop == 0 {
			return avStopLine
		}
		ln.ip = ln.loop
		ln.next += rand.IntN(randMax) * arg(0) / randMax
	case avStopLoop:
		a.lines[arg(0)].loop = 0
	case avTimeStart:
		now := clock.Ticks()
		for i := range a.lines {
			if a.lines[i].ip != 0 {
				a.lines[i].next = now
			}
		}
	case avTimeStop:
	case avUser:
		return g.avUser(a, p)
	case avStartTL:
		t := &a.lines[arg(0)]
		t.ip = t.start
		t.next, t.last = clock.Ticks(), clock.Ticks()
	case avStopTL:
		a.lines[arg(0)] = avLine{start: a.lines[arg(0)].start}
	case avInputOp, avButton:
		return 1
	default:
		return avStopAll
	}
	return 0
}

// avUser runs one of the custom functions (C: the User opcode's table).
func (g *Game) avUser(a *avScript, p int) int {
	l := g.lands
	arg := func(i int) int { return a.word(p + 1 + i) }
	switch a.word(p) {
	case 0: // C: _Custom_Start_Fade
		g.startBump(arg(0))
	case 1:
	case 2: // C: _Custom_Load_Palette
		if b, err := l.files.Read(a.str(arg(0))); err == nil {
			copy(l.palette[:], b)
		}
	case 3: // C: _Custom_Fade_To_Black
		l.palette = l.black
		g.startBump(arg(0))
	case 4: // C: _Custom_Scale_Blit
		an := &l.anims[arg(0)]
		pct := max(arg(4), 0)
		g.drawWSA(an.wsa, arg(1), an.x, an.y, landsPicPage, 0, nil, nil)
		w, h := an.wsa.width(), an.wsa.height()
		g.Screen.ScaleBlit(an.x, an.y, arg(2), arg(3), w, h, w*pct/percent, h*pct/percent, landsPicPage, landsWorkPage, 0)
		g.Screen.CopyChangedView(landsWorkPage, landsShadowPage)
	case 5: // C: _Custom_Display_Text
		g.introPrint(g.introText(arg(0)), arg(1))
	case 6:
		if err := g.history(); err != nil {
			return avStopAll
		}
	default:
		return avStopAll
	}
	return 0
}

// Open flags of LOL_Open_Animation.
const (
	landsAnimFade    = 1 << 1
	landsAnimShow    = 1 << 2
	landsAnimDirect  = 1 << 4
	landsAnimPicPage = 1 << 5
	fadeInTicks      = 30
	fadeBlackTicks   = 15
)

// landsOpenAnimation opens an animation, or shows the CPS of that name when there is no
// WSA (C: _LOL_Open_Animation).
func (g *Game) landsOpenAnimation(n int, name string, x, y, delay, p4, flags int) {
	l := g.lands
	s := g.Screen
	an := &l.anims[n]
	*an = landsAnim{x: x, y: y, delay: delay, flags: flags}
	open := 0
	if flags&landsAnimDirect != 0 {
		open = wsaDirect
	}
	var pal []byte
	if n == 1 {
		pal = l.palette[:]
	}
	wsaName := fmt.Sprintf("%s.WSA", name)
	file, err := l.files.Read(wsaName)
	if err != nil {
		if flags&landsAnimFade != 0 {
			s.FadePalette(l.black[:], fadeBlackTicks, nil)
			s.ClearPage(landsWorkPage)
			s.CopyChangedView(landsWorkPage, landsShadowPage)
		}
		g.loadPictureLands(fmt.Sprintf("%s.CPS", name), landsLoadPage, l.palette[:])
		s.CopyPage(landsLoadPage, landsWorkPage)
		s.CopyChangedView(landsWorkPage, landsShadowPage)
		if flags&landsAnimFade != 0 {
			s.FadePalette(l.palette[:], fadeInTicks, nil)
		}
		return
	}
	an.wsa = newWSAPlayer(file, open, pal)
	if an.x == -1 {
		an.x = 0
	}
	if an.y == -1 {
		an.y = 0
	}
	if flags&landsAnimFade != 0 {
		s.FadePalette(l.black[:], fadeBlackTicks, nil)
		s.ClearPage(landsWorkPage)
		s.CopyChangedView(landsWorkPage, landsShadowPage)
	}
	if flags&landsAnimShow != 0 {
		if l.files.Exists(name + ".CPS") {
			g.loadPictureLands(name+".CPS", landsLoadPage, l.palette[:])
			s.CopyPage(landsLoadPage, landsWorkPage)
		}
		g.drawWSA(an.wsa, 0, an.x, an.y, landsWorkPage, 0, nil, nil)
		s.CopyChangedView(landsWorkPage, landsShadowPage)
	}
	if flags&landsAnimFade != 0 {
		s.FadePalette(l.palette[:], fadeInTicks, nil)
	}
}

// introText is string n of LOLINTRO.DIP: an offset table and the strings it points at
// (C: _Intro_Print's RawText lookup).
func (g *Game) introText(n int) string {
	t := &timScript{text: g.lands.rawText}
	return t.timString(n)
}

// introColours are the text colour ramps (C: g_114c).
var introColours = [][3]int{{0, 0, 0}, {100, 100, 100}, {97, 81, 48}, {41, 72, 100}, {0, 75, 59}}

const (
	textColourFirst = 240
	textColours     = 15
	vgaLevels       = 4 // 15 steps of 4 make the 0-63 range
)

// rawPaletteFade sets the text colours to ramp c, at once or as a background fade
// (C: _Raw_Palette_Fade).
func (g *Game) rawPaletteFade(c, speed int) {
	l := g.lands
	c = max(0, min(c, len(introColours)-1))
	for i := range textColours {
		for k := range 3 {
			l.palette[(textColourFirst+i)*3+k] = byte((textColours - i) * vgaLevels * introColours[c][k] / percent)
		}
	}
	if speed == 0 && l.bumpRate == 0 {
		g.Screen.SetPalette(l.palette[:])
	} else {
		g.startBump(speed)
	}
}

// introPrint shows a caption, or plays its voice when it has one and speech is on
// (C: _Intro_Print).
func (g *Game) introPrint(text string, col int) {
	l := g.lands
	s := g.Screen
	screen := s.Page(pageScreen)
	if l.textUp {
		copy(screen[introTextTop*gfx.Width:], l.textRows)
		l.textUp = false
	}
	if strings.HasPrefix(text, "$") {
		if end := strings.IndexByte(text[1:], '$'); end >= 0 {
			name := text[1 : end+1]
			text = text[end+2:]
			l.voice = g.loadQuickSample(name + voiceExtension)
			if l.voice != nil && g.Config.DigitalSound && col >= 0 {
				g.playLongSample(l.voice, 255, introVoiceVolume)
				return
			}
		}
	}
	if text == "" {
		return
	}
	c := col
	if col < 0 {
		c = 1
		s.SetFont(l.font8)
		s.SetFontPalette(font8Palette)
		s.CharSpacing = -2
	}
	g.rawPaletteFade(c, 0)
	s.LineSpacing = -4
	copy(l.textRows, screen[introTextTop*gfx.Width:])
	l.textUp = true
	old := s.SetPage(pageScreen)
	for n, line := range strings.Split(text, "\r") {
		x := (gfx.Width - s.TextWidth(line)) / 2
		y := introTextTop + n*s.Font.Height()
		if col < 0 {
			y = introTextBottomY
		}
		s.DrawText(line, x, y, textColourFirst, 0)
	}
	s.SetPage(old)
	s.LineSpacing = 0
	if col < 0 {
		s.SetFont(l.introFont)
		s.SetFontPalette(introFontPalette)
		s.CharSpacing = 0
	}
}
