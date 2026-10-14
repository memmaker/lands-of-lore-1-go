package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

// LANDS.EXE is the language folder's second program: the intro, the character
// generation and the history of the Lands. The launcher starts it with INTRO, CHARGEN or
// HISTORY; it plays its part and starts MAIN.EXE again, after the character generation
// with the chosen hero's number (C: LANDS.EXE _Main_Program). It has its own file layer,
// fonts and palette, and draws on a work page it shows through a copy of the screen.

// What the launcher starts LANDS.EXE for (C: g_10e3, g_10eb, g_1105).
const (
	landsCharGen = "CHARGEN"
	landsIntro   = "INTRO"
	landsHistory = "HISTORY"
)

// LANDS.EXE's archives, in the order its file layer searches them (C: its PAK table).
var landsArchives = []string{
	"STARTUP.PAK", "INTRO1.PAK", "INTRO2.PAK", "INTRO3.PAK", "INTRO4.PAK", "INTRO5.PAK",
	"INTRO6.PAK", "INTRO7.PAK", "INTRO8.PAK", "INTRO9.PAK", "INTROVOC.PAK", "DRIVERS.PAK",
	"HISTORY.PAK",
}

// Files and settings of the start (C: g_1174, g_1181, g_118b, g_1195, g_11b2, g_11c5).
const (
	introFontFile  = "intro.fnt"
	font8File      = "NEW8P.FNT"
	introTextFile  = "LOLINTRO.DIP"
	introScript    = "LOLINTRO.TIM"
	introScore     = "LOREINTR"
	voiceExtension = ".VOC" // (C: g_11fb)

	landsWorkPage   = 8 // what the AV scripts draw on
	landsShadowPage = 4 // the copy of the screen the work page is shown through
	landsPicPage    = 2
	landsLoadPage   = 3

	landsFadeOut     = 30 // ticks of the fade at the end (C: Fade_Palette_To(.., 0x1e))
	landsFadeOutLong = 60
	scoreTitle       = 6 // the character generation's song (C: _Play_Score(6))
	scoreAfter       = 1
	landsSoundVolume = 255 // (C: __SoundVolume)
	maxSoundTrack    = 250 // tracks Play_Sound_Vol takes (C: < 0xfa)
	alwaysSounds     = 2   // tracks played with the effects off (C: param_1 < 2)
	noPriority       = -1
	voicePriority    = 255
	maxPriority      = 256 // stopping takes the stream from anything (C: 0x100)
	aiLevelShift     = 9   // the AIL volume of vol*SoundVolume (C: >> 9)
)

// landsWindow is LANDS.EXE's window 0, the whole screen, which its blits clip to
// (C: _WindowList[0]).
var landsWindow = gfx.Window{X: 0, Y: 0, W: gfx.Width / gfx.ColumnWidth, H: gfx.Height}

// The intro font's colours and those of the 8 point font (C: g_80c, g_81c).
var (
	introFontPalette = []byte{0, 240, 241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 0, 0, 0, 0}
	font8Palette     = []byte{0, 240, 254, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

// lands is LANDS.EXE's state.
type lands struct {
	files   *data.Archives
	palette gfx.Palette // the colours being faded to (C: _Palette)
	black   gfx.Palette // (C: _BlackPalette)

	introFont, font8 *gfx.Font // (C: _IntroFont, _Font8Ptr; FINALE.EXE: NEW6P.FNT)
	rawText          []byte    // LOLINTRO.DIP (C: _RawText)

	textUp   bool   // the text's rows are saved in textRows (C: _up)
	textRows []byte // the screen under the text (C: _TextBuffer)
	voice    *sound.Sample

	bumpRate, bumpTick, bumpAccum int // the background fade (C: _BumpRate, _BumpTick, _BumpAccum)
	fadeTimer                     int // (C: _fadetimer)

	long         *sound.Sample // the sample the stream plays (C: the long sample)
	longPriority int           // (C: g_2988)

	anims   [landsAnimations]landsAnim
	samples [avSamples]*sound.Sample // (C: g_dbc)
	av      *avScript
	avEnd   bool // (C: _AVEnd)

	finale bool // FINALE.EXE's engine, which draws straight on the screen
	hero   int  // FINALE.EXE's hero (C: g_d4)
}

// landsAnim is one of LANDS.EXE's animations (C: struct _Anim, 0x8c bytes).
type landsAnim struct {
	wsa          *wsaPlayer
	x, y         int
	delay, flags int
}

const landsAnimations = 10

// runLands is LANDS.EXE started with arg; it returns the hero the character generation
// chose (C: _Main_Program).
func (g *Game) runLands(arg string) (int, error) {
	l := &lands{files: data.OpenArchives(g.Files.LanguageDir(), landsArchives), longPriority: noPriority}
	g.lands = l
	defer func() { g.lands = nil }()
	s := g.Screen
	windows := s.Windows
	s.Windows[0] = landsWindow
	font := s.Font
	defer func() {
		s.Windows = windows
		s.SetFont(font)
		s.CharSpacing, s.LineSpacing = 0, 0
	}()
	defer s.RestoreMouseHidden(s.StartMouseHidden()) // a program of its own, its mouse fresh
	defer g.audio.SetMusicVolume(landsSoundVolume)   // MAIN.EXE loads the driver anew

	var err error
	if l.introFont, err = l.loadFont(introFontFile); err != nil {
		return 0, err
	}
	if l.font8, err = l.loadFont(font8File); err != nil {
		return 0, err
	}
	s.SetFont(l.font8)
	g.mt32Init()
	input.Flush()
	l.bumpRate = 0
	s.SetPalette(l.black[:])
	s.ClearPage(pageScreen)
	s.ClearPage(landsShadowPage)
	if l.rawText, err = l.files.Read(introTextFile); err != nil {
		return 0, err
	}
	l.textRows = make([]byte, introTextRows*gfx.Width)
	s.SetFont(l.introFont)
	s.SetFontPalette(introFontPalette)
	s.SetPalette(l.black[:])

	hero := 0
	switch arg {
	case landsHistory:
		if err := g.history(); err != nil {
			return 0, err
		}
	case landsCharGen:
	default:
		if err := g.playIntro(); err != nil {
			return 0, err
		}
	}
	if arg != landsIntro && arg != landsHistory {
		g.loadScore(introScore)
		g.playScore(scoreTitle)
		var err error
		if hero, err = g.charGen(); err != nil {
			return 0, err
		}
		g.playScore(scoreAfter)
		g.audio.Load(sound.Music, nil)
		s.FadePalette(l.black[:], landsFadeOutLong, nil)
	}
	g.audio.StopDigital(nil)
	g.audio.Load(sound.Effects, nil)
	g.audio.Load(sound.Music, nil)
	return hero, nil
}

func (l *lands) loadFont(name string) (*gfx.Font, error) {
	b, err := l.files.Read(name)
	if err != nil {
		return nil, err
	}
	return gfx.NewFont(b), nil
}

// playIntro runs LOLINTRO.TIM until it ends or a key is pressed, keeping the background
// fade going (C: _Main_Program's loop).
func (g *Game) playIntro() error {
	l := g.lands
	l.avEnd = false
	av, err := g.loadAV(introScript)
	if err != nil {
		return err
	}
	l.av = av
	for input.Check() == 0 && !l.avEnd {
		g.longSampleStatus()
		if av != nil {
			g.processAV(av)
		}
		g.bumpPalette()
		host.Yield()
	}
	l.av = nil
	g.Screen.FadePalette(l.black[:], landsFadeOut, nil)
	g.audio.Fade(sound.Music)
	g.stopLongSample()
	return nil
}

// bumpPalette takes the background fade a step when it is due (C: _Main_Program).
func (g *Game) bumpPalette() {
	l := g.lands
	if l.bumpRate == 0 || clock.Ticks() < l.fadeTimer {
		return
	}
	n := l.bumpAccum + l.bumpTick
	l.fadeTimer = clock.Ticks() + n>>fixedShift
	l.bumpAccum = n & fixedMask
	if !g.Screen.FadeStep(l.palette[:], l.bumpRate) {
		l.bumpRate = 0
	}
}

const (
	fixedShift = 8
	fixedMask  = 1<<fixedShift - 1
)

// startBump starts a background fade to the palette over ticks (C: _Determine_BumpRate
// into _BumpTick and _BumpRate).
func (g *Game) startBump(ticks int) {
	l := g.lands
	l.bumpTick, l.bumpRate = g.Screen.FadeRate(l.palette[:], ticks)
	l.bumpAccum = 0
}

// loadPictureLands reads a CPS onto a page, its palette into the working palette
// (C: _Load_Uncompress).
func (g *Game) loadPictureLands(name string, page int, pal []byte) error {
	file, err := g.lands.files.Read(name)
	if err != nil {
		return err
	}
	img, err := data.DecodeCPS(file)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	copy(g.Screen.Page(page)[:gfx.Width*gfx.Height], img.Pixels)
	if pal != nil && img.Palette != nil {
		copy(pal, img.Palette)
	}
	return nil
}

// Sound.

// loadScore loads a song file into the music and the effects, the file of each slot's
// driver; false when neither is there (C: _Load_Score_File, _Load_Sound_File with
// Data_File_Name).
func (g *Game) loadScore(name string) bool {
	found := false
	for _, slot := range []int{sound.Music, sound.Effects} {
		b, err := g.lands.files.Read(name + g.soundFileExt(slot))
		if err != nil {
			continue
		}
		g.audio.Load(slot, b)
		found = true
	}
	return found
}

// mt32Init sets up the MT-32 with LOREINTR's first song, the same patches as LOLSYSEX
// (C: MT32_Init, FINALE.EXE _MT32_Init). fix: the original printed "Initializing sound
// card." and a dot a second for the ~10 seconds the messages are spaced for a real MT-32;
// the emulator has them from the title already, or takes them at once. sic: FINALE.EXE's
// archives have no LOREINTR, so there it only clears the screen.
func (g *Game) mt32Init() {
	if !g.audio.MIDI(sound.Music) && !g.audio.MIDI(sound.Effects) {
		return
	}
	s := g.Screen
	l := g.lands
	found := !g.snd.sysexSent && g.loadScore(introScore)
	s.ClearPage(pageScreen)
	if !l.finale {
		s.ClearPage(landsShadowPage)
	}
	if found {
		g.audio.PlayThrough(0)
		g.snd.sysexSent = true
	}
	s.SetPalette(l.black[:])
}

// playScore plays a song (C: _Play_Score).
func (g *Game) playScore(n int) {
	if n >= 0 && n < maxSoundTrack+1 {
		g.audio.Play(sound.Music, n, -1)
	}
}

// playLandsSound plays an effect of the song file (C: _Play_Sound_Vol).
func (g *Game) playLandsSound(n, vol int) {
	if n >= 0 && n < maxSoundTrack && (g.effectsOn || n < alwaysSounds) {
		g.audio.Play(sound.Effects, n, vol)
	}
}

// loadQuickSample reads a VOC from the intro's sample archive (C: _Load_Quick_Sample).
func (g *Game) loadQuickSample(name string) *sound.Sample {
	b, err := g.lands.files.Read(name)
	if err != nil {
		return nil
	}
	return sound.ConvertVOC(b)
}

// longSampleStatus says whether the stream plays (C: _Long_Sample_Status).
func (g *Game) longSampleStatus() bool {
	l := g.lands
	return l.long != nil && g.audio.DigitalPosition(l.long) != -1
}

// playLongSample plays a sample on the stream unless one of higher priority plays; nil
// only stops the stream (C: _Play_Long_Sample_Vol).
func (g *Game) playLongSample(s *sound.Sample, priority, vol int) bool {
	l := g.lands
	if s == nil {
		priority = maxPriority
	} else {
		priority = min(priority, voicePriority)
	}
	if !g.longSampleStatus() {
		l.longPriority = noPriority
	}
	if l.longPriority > priority {
		return false
	}
	g.stopLongSample()
	if s == nil {
		return false
	}
	l.longPriority = priority
	l.long = s
	return g.audio.PlayDigitalLevel(s, priority, vol*landsSoundVolume>>aiLevelShift)
}

// stopLongSample stops the stream (C: _Stop_Long_Sample, _Stop_Sample).
func (g *Game) stopLongSample() {
	l := g.lands
	if l.long != nil {
		g.audio.StopDigital(l.long)
		l.long = nil
	}
}
