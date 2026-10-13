package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/text"
)

// Message kinds of printMessage: the low bits pick a colour and a sound.
const (
	messageKeepTalk = 1 << 2  // leaves a character's speech going
	messageSilent   = 1 << 15 // without the kind's sound
	messageKindMask = messageSilent - 1
)

var (
	messageColours = [...]int{254, 162, 132, 151, 159} // (C: g_21db)
	messageSounds  = [...]int{11, 0, 43, 27, 0}        // (C: g_21d6)
)

// printMessage prints a message on the message line, or in the dialogue window while
// the dialogue box is up; on the message line it flashes in its colour
// (C: fn_0845_01c4).
func (g *Game) printMessage(flags int, format string, args ...any) {
	s := g.Screen
	t := g.Text
	s.HideMouse()
	if flags&messageKeepTalk != 0 {
		flags ^= messageKeepTalk
	} else {
		g.endTalk()
	}
	colour := messageColours[flags&messageKindMask]
	var old int
	if g.updateFlags&updateNoPortraits != 0 {
		old = t.SelectAndClear(textWindowLabel)
	} else {
		old = t.SelectAndClear(textWindowMessage)
		pal := append([]byte(nil), s.Palette[:]...)
		at := colour * 3
		s.SetPaletteColour(pal, textColourMessage, pal[at:at+3])
		colour = textColourMessage
		g.enableTimer(timerMessage, true)
	}
	s.ShowMouse()
	t.Foreground = colour
	t.Printf(text.Sprintf(format, args...))
	t.SelectWindow(old)
	if flags&messageSilent == 0 && messageSounds[flags] != 0 {
		g.playSound(messageSounds[flags], 0)
	}
	g.messageID = flags & messageKindMask
	g.messageFading = false
}
