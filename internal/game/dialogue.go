package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/text"
)

// Dialogue and message printing (C: s020 fn_130a_07be, fn_130a_0a7c).

// Colours of printed text.
const (
	textColourDialogue = 254
	textColourMessage  = 192 // flashes: it starts as the dialogue colour and fades
)

// formatArgs collects the arguments of a game string's conversions, from the script's
// arguments from idx on: %s takes a string id, %n a character whose name it prints
// (made a %s) and %a the value scripts set (made a %d) (C: fn_130a_07be). It parses
// conversions as loosely as the original did.
func (g *Game) formatArgs(format string, arg func(int) int, idx int) (string, []any) {
	f := []byte(format)
	var args []any
	for i := 0; i < len(f); {
		c := f[i]
		i++
		if c != '%' {
			continue
		}
		at := func() byte {
			if i < len(f) {
				return f[i]
			}
			return 0
		}
		done := false
		switch at() {
		case 0:
			done = true
		case ' ', '+', '-':
			i++
		case '#':
			i++
			switch at() {
			case 'E', 'G', 'X', 'c', 'd', 'e', 'f', 'g', 'i', 's', 'u', 'x':
				i++
			default:
				done = true
			}
		}
		if !done {
			switch at() {
			case 0:
				done = true
			case '0':
				i++
			default:
				for at() >= '0' && at() <= '9' {
					i++
				}
			}
		}
		if done {
			continue
		}
		switch at() {
		case 'n':
			args = append(args, g.party[arg(idx)].Name)
			idx++
			f[i] = 's'
			i++
		case 's':
			args = append(args, g.text(arg(idx)))
			idx++
			i++
		case 'a':
			args = append(args, g.scriptValue)
			f[i] = 'd'
			i++
		case 'X', 'd', 'u', 'x':
			args = append(args, arg(idx))
			idx++
			i++
		}
	}
	return string(f), args
}

// printText prints format into window n with its arguments; window 3, the message
// line, takes the dialogue window while the dialogue box is up, and otherwise prints
// in the flashing colour and starts the message timer (C: fn_130a_0a7c).
func (g *Game) printText(n int, format string, arg func(int) int, idx int) {
	t := g.Text
	s := g.Screen
	s.HideMouse()
	var old int
	if n == textWindowMessage {
		if g.updateFlags&updateNoPortraits != 0 {
			old = t.SelectAndClear(textWindowLabel)
			t.Foreground = textColourDialogue
		} else {
			old = t.SelectAndClear(textWindowMessage)
			t.Foreground = textColourMessage
			pal := append([]byte(nil), s.Palette[:]...)
			at := textColourDialogue * 3
			s.SetPaletteColour(pal, textColourMessage, pal[at:at+3])
			g.enableTimer(timerMessage, true)
			g.messageID = 0
			g.messageFading = false
		}
	} else {
		old = t.SelectWindow(n)
		t.Foreground = textColourDialogue
	}
	s.ShowMouse()
	page := s.SetPage(pageScreen)
	font := t.SetFont(g.SystemFont)
	f, args := g.formatArgs(format, arg, idx)
	if len(args) != 0 {
		out := text.Sprintf(f, args...)
		g.printedLength = len(out)
		t.Printf(out)
	} else {
		g.printedLength = len(format)
		t.Printf(format)
	}
	t.SelectWindow(old)
	s.SetPage(page)
	t.SetFont(font)
	g.messageFading = false
}

// scriptArgs reads a script's arguments.
func scriptArgs(s *emc.State) func(int) int { return s.Arg }

// opSay plays a line's speech and, unless spoken without the text area, prints it in
// the dialogue window (C: fn_0fbe_24e9).
func (g *Game) opSay(s *emc.State) int {
	spoken := g.playSpeech(s.Arg(0), 0, false)
	if !spoken || g.textEnabled {
		g.printText(textWindowLabel, g.text(s.Arg(0)), scriptArgs(s), 1)
	}
	return 1
}

// playSpeech (C: fn_154c_01d3): speech is silent until the sound milestone.
func (g *Game) playSpeech(id, character int, wait bool) bool { return false }

// Dialogue choices: up to three buttons along the bottom of the dialogue window
// (whose X and W are in pixels), or none when a single choice is spoken.
const (
	choiceWidth       = 74
	choiceHeight      = 9
	choiceHalfWidth   = 37
	choiceSingleRight = 77 // a lone button's distance from the window's right edge
	choiceTextY       = 2
	choiceColour      = 254
	choiceColourHot   = 144
	maxChoices        = 3
	frameShadow       = 136 // a raised box's left and bottom edges
	frameHighlight    = 251 // its top and right
)

// showDialogueChoices lays out and draws the choice buttons; a single choice while
// speech plays gets none (C: fn_13e8_0903).
func (g *Game) showDialogueChoices(n int, choices []string) {
	t := g.Text
	if n == 1 && g.speechVolume != 0 {
		t.SelectWindow(textWindowMain)
		g.dialogueChoices = 0
		g.choiceLabels = [maxChoices]string{}
		return
	}
	g.choiceSelected = 1
	g.dialogueChoices = n
	copy(g.choiceLabels[:], choices)
	t.SelectWindow(textWindowMain)
	g.choiceY = t.Y + t.H - choiceHeight
	if n == 1 {
		g.choiceStep = 0
		g.choiceX = t.X + t.W - choiceSingleRight
	} else {
		g.choiceStep = t.W / n
		g.choiceX = t.X + g.choiceStep>>1 - choiceHalfWidth
	}
	g.drawDialogueChoices()
	g.flushInput()
}

// drawDialogueChoices draws the buttons, the selected one's label lit (C: fn_13e8_05a4).
// The original showed the cursor fully while it drew and hid it as often again after.
func (g *Game) drawDialogueChoices() {
	s := g.Screen
	page := s.SetPage(pageScreen)
	font := s.SetFont(g.SmallFont)
	x := g.choiceX
	for i := 1; i <= g.dialogueChoices; i++ {
		label := g.choiceLabels[i-1]
		colour := choiceColour
		if i == g.choiceSelected {
			colour = choiceColourHot
		}
		s.HideMouse()
		g.drawFrame(x, g.choiceY, choiceWidth, choiceHeight, frameShadow, frameHighlight, noFill)
		s.DrawText(label, x+choiceHalfWidth-s.TextWidth(label)>>1, g.choiceY+choiceTextY, colour, 0)
		s.ShowMouse()
		x += g.choiceStep
	}
	s.SetPage(page)
	s.SetFont(font)
}

// Keys of the dialogue.
const (
	keyChoose      = 43 // Return
	keyChooseSpace = 61
	keyChoiceLeft  = 79
	keyPadLeft     = 92
	keyChoiceRight = 89
	keyPadRight    = 102
	keyCodeMask    = 1<<12 - 1
	labelRows      = 9
)

// choiceAt is the button under (x, y), 0 for none.
func (g *Game) choiceAt(x, y int) int {
	bx := g.choiceX
	for i := 1; i <= g.dialogueChoices; i++ {
		if x >= bx && x <= bx+choiceWidth && y >= g.choiceY && y <= g.choiceY+choiceHeight {
			return i
		}
		bx += g.choiceStep
	}
	return 0
}

// dialogueAnswer takes keys and clicks for the dialogue and returns the choice made, 0
// while none is; without buttons a key or the end of the speech answers 1
// (C: fn_13e8_06a1).
func (g *Game) dialogueAnswer() int {
	r := 0
	old := g.choiceSelected
	if i := g.choiceAt(g.Screen.Mouse()); i != 0 {
		g.choiceSelected = i
	}
	if g.dialogueChoices == 0 {
		if input.Peek() != 0 {
			k := input.Get() & keyCodeMask
			g.GUI.MarkChanged()
			if k == keyChoose || k == keyChooseSpace {
				g.speechKey(1)
				g.talkWaitUntil = 0
			}
		}
		if g.updateSpeech() != speechPlaying && g.talkWaitUntil <= clock.Ticks() {
			r = 1
			g.flushInput()
			g.GUI.MarkChanged()
		}
	} else if input.Peek() != 0 {
		k := input.Get() & keyCodeMask
		g.GUI.MarkChanged()
		switch k {
		case keyChoiceLeft, keyPadLeft:
			if g.dialogueChoices > 1 {
				g.choiceSelected--
				if g.choiceSelected < 1 {
					g.choiceSelected = g.dialogueChoices
				}
			}
		case keyChoiceRight, keyPadRight:
			if g.dialogueChoices > 1 {
				g.choiceSelected++
				if g.choiceSelected > g.dialogueChoices {
					g.choiceSelected = 1
				}
			}
		case keyChoose, keyChooseSpace:
			g.speechKey(1)
			g.talkWaitUntil = 0
			r = g.choiceSelected
		case input.Released | input.KeyMouseLeft, input.Released | input.KeyMouseRight:
			if i := g.choiceAt(input.Click()); i != 0 {
				g.choiceSelected = i
				r = i
			}
		}
	}
	if old != g.choiceSelected {
		g.drawDialogueChoices()
	}
	if r != 0 {
		g.endTalk()
		t := g.Text
		s := g.Screen
		if !g.textEnabled && g.controlMode != 0 {
			t.SelectWindow(textWindowMain)
			s.HideMouse()
			s.FillRect(t.X, t.Y+t.H-labelRows, t.X+t.W-1, t.Y+t.H-1, t.Background)
			s.ShowMouse()
		} else {
			s.HideMouse()
			s.FillRect(t.X, t.Y, t.X+t.W-2, t.Y+t.H-1, t.Background)
			t.SelectAndClear(textWindowLabel)
			s.ShowMouse()
			w := &s.Windows[textWindowLabel]
			w.Column, w.Row = 0, 0
		}
	}
	return r
}
