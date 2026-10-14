package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

// LANDS.EXE's history of the Lands: Patrick Stewart's narration, ps_1a.voc on, with the
// pictures of HISTORY.PAK changing at given lines (C: LANDS.EXE _Patrick_Stewart and the
// functions before it in seg 0543).

const (
	historyFont       = "font9p.FNT"
	historyText       = "history.dip" // read and freed, never shown (C: g_31c)
	historyBackground = "backgnd.cps"
	historyFire       = "fire.wsa"
	historyPalette    = "dracpal.pal"

	historyVoiceFirst = '1' // ps_1a.voc: the digit runs to '9', the letter from 'a'
	historyVoiceLast  = '9'
	historyLineFirst  = 'a'

	historyBackPage = 8 // the background
	historyPicPage  = 2 // the background with the picture drawn on

	historyFadeIn   = 30
	historyFadeBG   = 130
	historyFadeOut  = 60
	historyFadeFire = 120
	historyStep     = 3 // ticks per picture frame
	historyFrames   = 15
	historyPause    = 30
	fireX, fireY    = 75, 51
	fireStep        = 4
	fireFrames      = 5
	fireBefore      = 12
	fireAfter       = 10
	vgaWhite        = 63
	historyWSAFlags = wsaDirect // (C: open flags 3)
)

// historyScene is a line of the table at 0x140: the voice it starts on and the picture.
type historyScene struct {
	digit, letter byte
	picture       string
}

var historyScenes = [30]historyScene{
	{'1', 'a', "hist01.wsa"}, {'1', 'b', "hist01.wsa"}, {'1', 'c', "hist01.wsa"},
	{'2', 'a', "hist02.wsa"}, {'2', 'b', "hist03.wsa"}, {'2', 'e', "hist04.wsa"},
	{'2', 'f', "hist05.wsa"}, {'3', 'a', "hist06a.wsa"}, {'3', 'b', "hist06a.wsa"},
	{'3', 'c', "hist06b.wsa"}, {'3', 'd', "hist20.wsa"}, {'4', 'a', "hist08.wsa"},
	{'4', 'c', "hist09.wsa"}, {'4', 'e', "hist10.wsa"}, {'5', 'a', "hist11.wsa"},
	{'5', 'b', "hist12.wsa"}, {'5', 'c', "hist12.wsa"}, {'5', 'd', "hist13.wsa"},
	{'6', 'a', "hist14.wsa"}, {'6', 'b', "hist15.wsa"}, {'6', 'c', "hist16.wsa"},
	{'7', 'a', "hist17.wsa"}, {'7', 'b', "hist18a.wsa"}, {'7', 'c', "hist19.wsa"},
	{'8', 'a', "hist21c.wsa"}, {'8', 'c', "hist21c.wsa"}, {'9', 'a', "hist22.wsa"},
	{'9', 'd', "hist23.wsa"}, {'9', 'f', "hist24.wsa"}, {'9', 'h', "hist25.wsa"},
}

// What a scene does to the pictures (C: the jump table at 0543:08fd).
const (
	sceneOpen      = iota // open the picture and bring it in
	sceneKeep             // play on the picture shown
	sceneOnward           // play the picture on, then the next one in
	sceneSwapWait         // take the picture out, the next in, and wait
	sceneSwapAfter        // the next picture in, with the line already going
	sceneSequence         // hist06a-e one after the other
	sceneFire             // Dracle's fire and his colours
	sceneSwap             // take the picture out and bring the next in
)

var historyActions = [30]int{
	sceneOpen, sceneKeep, sceneKeep, sceneOnward, sceneSwap, sceneSwap, sceneSwapWait, sceneOnward,
	sceneKeep, sceneSwapAfter, sceneOnward, sceneSwap, sceneSwap, sceneSwap, sceneSwap, sceneSwap,
	sceneKeep, sceneOnward, sceneSwap, sceneSwap, sceneSwap, sceneSwap, sceneSequence, sceneOnward,
	sceneSwap, sceneKeep, sceneOnward, sceneSwap, sceneSwap, sceneFire,
}

// historyState is the open picture and the fire.
type historyState struct {
	pic            *wsaPlayer
	x, y, w, h     int // the picture's rectangle (C: _sx, _sy, _sw, _sh)
	frame          int // (C: g_13e)
	fire           *wsaPlayer
	fireFrame      int // (C: g_134)
	fireNext       int // (C: g_136)
	sequenceLetter byte
}

// history tells the history of the Lands (C: _Patrick_Stewart).
func (g *Game) history() error {
	l := g.lands
	s := g.Screen
	h := &historyState{}
	font, err := l.loadFont(historyFont)
	if err != nil {
		return err
	}
	l.files.Read(historyText)
	var black, pal gfx.Palette
	s.FadePalette(black[:], historyFadeIn, nil)
	if err := g.loadPictureLands(historyBackground, historyBackPage, pal[:]); err != nil {
		return err
	}
	s.CopyPage(historyBackPage, pageScreen)
	s.CopyPage(historyBackPage, historyPicPage)
	s.FadePalette(pal[:], historyFadeBG, nil)
	s.CopyRect(h.x, h.y, h.x, h.y, h.w, h.h, historyPicPage, pageScreen)
	s.SetFont(font)

	scene := 0
	voice := []byte("ps_1a.voc")
	const digitAt, letterAt = 3, 4
	for voice[digitAt] <= historyVoiceLast && input.Check() == 0 {
		for {
			sample := g.loadQuickSample(string(voice))
			if sample == nil {
				break
			}
			if scene < len(historyScenes) && historyScenes[scene].digit == voice[digitAt] && historyScenes[scene].letter == voice[letterAt] {
				g.historyScene(h, scene, sample)
				scene++
			} else {
				g.playLongSample(sample, voicePriority, introVoiceVolume)
			}
			for g.longSampleStatus() && input.Check() == 0 {
				host.Yield()
			}
			g.stopLongSample()
			voice[letterAt]++
			if input.Check() != 0 {
				break
			}
		}
		voice[letterAt] = historyLineFirst
		voice[digitAt]++
	}

	var white gfx.Palette
	for i := range white {
		white[i] = vgaWhite
	}
	var idle func()
	if h.fire != nil {
		idle = func() { g.dracleStatus(h) }
	}
	s.FadePalette(white[:], historyFadeOut, idle)
	s.ClearPage(pageScreen)
	s.FadePalette(black[:], historyFadeOut, nil)
	for input.Check() != 0 {
		input.Get()
	}
	return nil
}

// historyScene plays a line that changes the picture (C: _Patrick_Stewart's switch).
func (g *Game) historyScene(h *historyState, n int, sample *sound.Sample) {
	play := func() { g.playLongSample(sample, voicePriority, introVoiceVolume) }
	name := historyScenes[n].picture
	switch historyActions[n] {
	case sceneOpen:
		g.openHistoryPicture(h, name)
		g.animateHistory(h, true)
		play()
	case sceneKeep:
		play()
		g.animateHistory(h, true)
	case sceneOnward:
		play()
		g.animateHistory(h, true)
		g.restoreHistoryBackground(h)
		g.openHistoryPicture(h, name)
		g.animateHistory(h, true)
	case sceneSwapWait:
		play()
		g.animateHistory(h, false)
		g.restoreHistoryBackground(h)
		g.openHistoryPicture(h, name)
		g.animateHistory(h, true)
		g.historyWait(historyPause)
		g.animateHistory(h, true)
	case sceneSwapAfter:
		play()
		g.openHistoryPicture(h, name)
		g.animateHistory(h, true)
	case sceneSequence:
		g.animateHistory(h, false)
		g.restoreHistoryBackground(h)
		until := clock.Ticks() + historyPause
		g.openHistoryPicture(h, name)
		g.historyWaitUntil(until)
		play()
		g.animateHistory(h, true)
		// hist06a.wsa to hist06d.wsa, then hist06e.wsa stays open; the original wrote
		// the letters into its copy of the table
		const letterAt = 6
		b := []byte(name)
		for k := byte(1); k < 4; k++ {
			b[letterAt] = 'a' + k
			until := clock.Ticks() + historyPause
			g.openHistoryPicture(h, string(b))
			g.historyWaitUntil(until)
			g.animateHistory(h, true)
		}
		b[letterAt] = 'e'
		g.openHistoryPicture(h, string(b))
	case sceneFire:
		play()
		g.animateHistory(h, false)
		g.restoreHistoryBackground(h)
		g.openHistoryPicture(h, name)
		if file, err := g.lands.files.Read(historyFire); err == nil {
			h.fire = newWSAPlayer(file, historyWSAFlags, nil)
		}
		g.longSampleStatus()
		g.animateHistory(h, true)
		h.fireFrame = 0
		for range fireBefore {
			g.fireFrame(h, true)
			h.fireFrame++
		}
		var pal gfx.Palette
		if b, err := g.lands.files.Read(historyPalette); err == nil {
			copy(pal[:], b)
		}
		g.Screen.FadePalette(pal[:], historyFadeFire, func() { g.dracleStatus(h) })
		for g.longSampleStatus() {
			h.fireFrame++
			g.fireFrame(h, true)
		}
		h.fireFrame = 0
		for range fireAfter {
			g.fireFrame(h, false)
			h.fireFrame++
		}
	default: // sceneSwap
		play()
		g.animateHistory(h, false)
		g.restoreHistoryBackground(h)
		g.openHistoryPicture(h, name)
		g.animateHistory(h, true)
	}
}

// fireFrame shows the next frame of the fire and waits for its time; the last frames run
// without asking after the voice (C: the loops at 0543:064c, 06d8, 0742).
func (g *Game) fireFrame(h *historyState, poll bool) {
	until := clock.Ticks() + historyStep
	if h.fireFrame > fireFrames-1 {
		h.fireFrame = 0
	}
	g.drawWSA(h.fire, h.fireFrame, fireX, fireY, pageScreen, 0, nil, nil)
	for clock.Ticks() < until {
		if poll {
			g.longSampleStatus()
		}
		host.Yield()
	}
}

// dracleStatus moves the fire on during a fade (C: _Dracle_Status).
func (g *Game) dracleStatus(h *historyState) bool {
	if clock.Ticks() > h.fireNext {
		g.drawWSA(h.fire, h.fireFrame, fireX, fireY, pageScreen, 0, nil, nil)
		h.fireFrame++
		if h.fireFrame > fireFrames-1 {
			h.fireFrame = 0
		}
		h.fireNext = clock.Ticks() + fireStep
	}
	g.longSampleStatus()
	return true
}

// openHistoryPicture opens a picture and takes its colours (C: _Open_Anim).
func (g *Game) openHistoryPicture(h *historyState, name string) {
	h.pic = nil
	file, err := g.lands.files.Read(name)
	if err != nil {
		return
	}
	var pal gfx.Palette
	h.pic = newWSAPlayer(file, historyWSAFlags, pal[:])
	if h.pic == nil {
		return
	}
	if h.pic.anim.Palette != nil {
		g.Screen.SetPalette(pal[:])
	}
	h.x, h.y, h.w, h.h = h.pic.anim.X, h.pic.anim.Y, h.pic.width(), h.pic.height()
	h.frame = 1
}

// restoreHistoryBackground puts the background back under the picture (C: _Restore_BG).
func (g *Game) restoreHistoryBackground(h *historyState) {
	s := g.Screen
	s.CopyRect(h.x, h.y, h.x, h.y, h.w, h.h, historyBackPage, pageScreen)
	s.CopyRect(h.x, h.y, h.x, h.y, h.w, h.h, historyBackPage, historyPicPage)
}

// animateHistory plays 15 frames of the picture onward or back, bringing it in or
// taking it out (C: _Animate).
func (g *Game) animateHistory(h *historyState, onward bool) {
	for range historyFrames {
		until := clock.Ticks() + historyStep
		g.drawWSA(h.pic, h.frame, 0, 0, historyPicPage, 0, nil, nil)
		g.Screen.CopyRect(h.x, h.y, h.x, h.y, h.w, h.h, historyPicPage, pageScreen)
		g.historyWaitUntil(until)
		if onward {
			h.frame++
		} else {
			h.frame--
		}
	}
}

func (g *Game) historyWait(n int) { g.historyWaitUntil(clock.Ticks() + n) }

func (g *Game) historyWaitUntil(t int) {
	for clock.Ticks() < t {
		g.longSampleStatus()
		host.Yield()
	}
}
