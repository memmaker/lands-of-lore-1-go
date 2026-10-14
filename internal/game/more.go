package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/text"
)

// The in-game <MORE> button and the message fade-out (C: fn_13e8_09b7, fn_0845_11b7).

const (
	messageColour      = 192 // the message text's colour, faded into the background
	messageFadedColour = 252
	messageFadeTicks   = 60
	colourMask         = 63

	moreWidth       = 74
	moreHeight      = 9
	moreRightInset  = 77 // the button's left edge from the window's right
	moreBottomInset = 10
	moreSceneInset  = 2 // with the dialogue box of a scene
	moreBoxInset    = 5
	moreSceneShift  = 6
	moreTextY       = 2
	moreEdgeLight   = 136
	moreEdgeDark    = 251
	moreTextColour  = 144
	stringMore      = 0x4073
	moreTalkSteps   = 36 // the mouth moves at most this many times over the prompt
	keyMoreReturn   = 43
	keyMoreSpace    = 61
)

// fadeMessageColour moves colour a toward colour b over total ticks, t ticks in; at
// the end a takes b's value in the dark palette too. It says whether a still changes
// (C: fn_0845_105c).
func (g *Game) fadeMessageColour(a, b, t, total int) bool {
	s := g.Screen
	rgb := make([]byte, 3)
	changed := false
	for i := range rgb {
		pa, pb := int(s.Palette[a*3+i]), int(s.Palette[b*3+i])
		if t < total {
			d := pb&colourMask - pa&colourMask
			if d != 0 {
				changed = true
			}
			rgb[i] = byte(pa&colourMask + (d<<8)/total*t>>8)
		} else {
			rgb[i] = byte(pb)
			g.darkPalette[a*3+i] = byte(pb)
			changed = false
		}
	}
	pal := append([]byte(nil), s.Palette[:]...)
	s.SetPaletteColour(pal, a, rgb)
	return changed
}

// updateMessageFade fades the message out; when it has faded it is cleared
// (C: fn_0845_11b7).
func (g *Game) updateMessageFade() bool {
	if !g.messageFading {
		return false
	}
	r := g.fadeMessageColour(messageColour, messageFadedColour, clock.Ticks()-g.messageFadeStart, messageFadeTicks)
	if !r && !g.sceneOn {
		old := g.Text.SelectAndClear(textWindowMessage)
		g.Screen.HideMouse()
		g.Text.SelectWindow(old)
		g.Screen.ShowMouse()
		g.enableTimer(timerMessage, false)
		g.messageFading = false
	}
	return r
}

// pauseTimer switches timer n off and returns the ticks it had left, -1 if it was off
// (C: fn_0bbc_0960).
func (g *Game) pauseTimer(n int) int {
	t := &g.timers[n]
	if !t.enabled {
		return -1
	}
	t.enabled = false
	return t.next - clock.Get(t.clock)
}

// resumeTimer switches a paused timer back on with the ticks it had left (C: fn_0bbc_09a9).
func (g *Game) resumeTimer(n, left int) {
	if left != -1 {
		t := &g.timers[n]
		t.enabled = true
		t.next = clock.Get(t.clock) + left
	}
}

// morePrompt is the game's <MORE>: a button under the text that a click on it, Return or
// Space takes away; with speech on it goes by itself when the speech of the text shown
// so far is over (C: fn_13e8_09b7).
func (g *Game) morePrompt(p *text.Printer, _ string, _, _, _ int) {
	s := g.Screen
	oldPage := s.SetPage(pageScreen)
	oldFont := s.SetFont(g.SmallFont)
	old := g.pauseTimer(timerMessage)
	g.messageFading = false
	saveRestore, saveSteps := false, 0
	if g.talkingCharacter != noTalk {
		saveRestore, g.talkRestore = g.talkRestore, false
		saveSteps = g.talkSteps
		g.talkSteps = min(g.talkSteps, moreTalkSteps)
	}
	deadline := 0
	if g.speechVolume != 0 {
		if t := g.speechLength(); t != 0 && g.printedLength != 0 {
			deadline = clock.Ticks() + p.Printed*t/g.printedLength
		}
	}
	x := (p.X+p.W)*8 - moreRightInset
	y := p.Y + p.H - moreBottomInset
	switch {
	case !g.sceneOn || g.updateFlags&updateNoPortraits == 0:
	case g.controlMode == 0:
		x += moreSceneShift
		y = p.Y + p.H - moreSceneInset
	default:
		y = p.Y + p.H - moreBoxInset
	}
	s.HideMouse()
	g.drawFrame(x, y, moreWidth, moreHeight, moreEdgeLight, moreEdgeDark, noFill)
	label := g.text(stringMore)
	s.DrawText(label, x+moreWidth/2-s.TextWidth(label)/2, y+moreTextY, moreTextColour, 0)
	s.ShowMouse()
	input.Flush()
	hidden := s.MouseHidden()
	if !input.NoMouse() {
		for s.MouseHidden() > 0 {
			s.ShowMouse()
		}
	}
	clicked := false
	for wait := true; wait && !g.quit; {
		key := 0
		for input.Peek() == 0 && !g.quit {
			g.idleWork()
			host.Yield()
			if g.speechVolume != 0 && deadline != 0 && (clock.Ticks() > deadline || g.updateSpeech() != speechPlaying) {
				key = keyMoreReturn
				break
			}
		}
		if key == 0 {
			if g.quit {
				break
			}
			key = input.Get() & (input.Released | 255)
		}
		switch key {
		case input.KeyMouseLeft, input.KeyMouseRight:
			if s.MouseIn(x, y, x+moreWidth, y+moreHeight) {
				clicked = true
			}
		case input.KeyMouseLeft | input.Released, input.KeyMouseRight | input.Released:
			if clicked {
				wait = false
			}
		case keyMoreReturn, keyMoreSpace:
			wait = false
		}
	}
	if !input.NoMouse() {
		for range hidden {
			s.HideMouse()
		}
	}
	s.HideMouse()
	s.FillRect(x, y, x+moreWidth, y+moreHeight, p.Background)
	p.Clear()
	s.ShowMouse()
	g.resumeTimer(timerMessage, old)
	if g.talkingCharacter != noTalk {
		g.talkRestore = saveRestore
		if saveSteps > moreTalkSteps {
			saveSteps -= moreTalkSteps
		} else {
			saveSteps >>= 1
		}
		g.talkSteps = saveSteps
	}
	s.SetFont(oldFont)
	s.SetPage(oldPage)
	input.Flush()
}
