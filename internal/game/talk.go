package game

import (
	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
)

// Talking: a character who speaks moves its mouth on its portrait, or on the face in
// the dialogue box, until its speech ends (C: g_4bf8 and friends, fn_130a_0605).
const (
	talkPortrait  = 0 // the text goes to the message line, the face is the portrait's
	talkDialogue  = 1 // the text goes to the dialogue window, beside the face
	talkCharacter = 2 // the text goes to the dialogue window of a character's screen

	talkFaces       = 6 // mouth faces 7-12
	firstTalkFace   = 7
	talkFaceTicks   = 10
	speechStillOn   = 2
	talkFaceY       = 144
	dialogueFaceX   = 90
	dialogueFaceY   = 130
	characterFaceX  = 16
	characterFaceY  = 134
	characterFaceDY = 145
	characterFaceDX = 10
	anyoneTalks     = 1<<4 | 1<<5 | 1<<6 // a character id with these bits: the first one (C: 0x70)
	selectedTalks   = 1                  // the selected character
	noOneTalks      = 8                  // nobody needs to be in the party
)

// animateTalk moves the talking character's mouth, or with the talk over sets the face
// back (C: fn_130a_0605).
func (g *Game) animateTalk() {
	x, y, onPortrait := 0, 0, false
	ch := g.talkingCharacter
	switch g.talkStyle {
	case talkPortrait:
		x, y, onPortrait = g.portraitX[ch], talkFaceY, true
	case talkDialogue:
		if !g.textEnabled {
			x, y = g.portraitX[ch], talkFaceY
		} else {
			x, y = dialogueFaceX, dialogueFaceY
		}
	case talkCharacter:
		if !g.textEnabled {
			x, y = g.portraitX[ch]+characterFaceDX, characterFaceDY
		} else {
			x, y = characterFaceX, characterFaceY
		}
	}
	face := dice(1, talkFaces) - 1
	if g.party[ch].Face == face {
		face++
	}
	if face > talkFaces-1 {
		face -= talkFaces - 1
	}
	face += firstTalkFace
	if g.speechVolume != 0 {
		if g.updateSpeech() == speechStillOn {
			g.talkSteps = 2
		} else {
			g.talkSteps = 1
		}
	}
	g.talkSteps--
	if g.talkSteps != 0 {
		g.party[ch].Face = face
		if onPortrait {
			g.drawPortrait(ch)
		} else {
			g.drawFace(ch, x, y, pageScreen)
		}
		g.talkEnds = clock.Ticks() + talkFaceTicks
	} else if g.talkRestore {
		g.restoreFace(ch)
		if onPortrait {
			g.drawPortrait(ch)
			g.fadeMessage(0, false)
		} else {
			g.drawFace(ch, x, y, pageScreen)
		}
		g.talkingCharacter = noTalk
	}
}

// restoreFace sets a character's face back after talking (C: fn_0a21_0608).
func (g *Game) restoreFace(ch int) {
	const (
		faceHurt    = 6
		faceStunned = 5
	)
	c := &g.party[ch]
	switch {
	case c.Face == 1, c.Face == faceHurt && c.FaceDefault != faceStunned:
		g.setTemporaryFace(ch, 0, 0, false)
	case c.Face == faceHurt:
		c.Face = faceStunned
	default:
		c.Face = 0
	}
}

// endTalk ends a character's talking at once (C: fn_130a_0784).
func (g *Game) endTalk() {
	if g.talkingCharacter == noTalk {
		return
	}
	g.talkSteps = 1
	g.talkRestore = true
	g.animateTalk()
	g.talkingCharacter = noTalk
	g.talkSteps = 1
	if g.talkStyle == talkPortrait {
		g.fadeMessage(0, false)
	}
}

// characterText has character ch say str in style, with speech id speech: 1 is the
// selected character, a negative number one at random, else a character id. With
// the text on it is printed where the style puts it, formatted with arg from idx
// (C: fn_130a_03cd). It returns false when that character is not in the party.
func (g *Game) characterText(ch, style int, restore bool, str string, speech int, arg func(int) int, idx int) bool {
	skip := false
	var who int
	if ch != -1 && ch&anyoneTalks != 0 {
		ch ^= anyoneTalks
		who = 0
	} else {
		if ch == selectedTalks {
			ch = 0
			if g.selectedCharacter != 0 {
				ch = g.party[g.selectedCharacter].ID
			}
		}
		who = ch
	}
	g.endTalk()
	if ch < 0 {
		ch = random.Rand() * g.activeCharacters() / (1 << 15)
		who = ch
	} else if ch > 0 {
		i := 0
		for ; i < partySize; i++ {
			if g.party[i].ID == ch && g.party[i].Flags&charPresent != 0 {
				if who == ch {
					who = i
				}
				ch = i
				break
			}
		}
		if i == partySize {
			if ch != noOneTalks {
				return false
			}
			skip = true
		}
	}
	if !skip {
		g.talkingCharacter = ch
		g.talkStyle = style
		g.talkSteps = len(str) >> 1
		g.talkRestore = restore
	}
	if speech >= 0 {
		g.playSpeech(speech, who, false)
	}
	if g.textEnabled {
		w := &g.Screen.Windows[textWindowLabel]
		switch style {
		case talkPortrait:
			g.printText(textWindowMessage, str, arg, idx)
		case talkDialogue:
			w.Column, w.Row = 0, 0
			g.Text.SelectAndClear(textWindowLabel)
			g.Screen.SetWindow(textWindowLabel, 16, 123, 23, 47)
			g.printText(textWindowLabel, str, arg, idx)
			g.Screen.SetWindow(textWindowLabel, 11, 123, 28, 47)
		case talkCharacter:
			w.Column, w.Row = 0, 0
			save := g.Text.SelectAndClear(textWindowLabel)
			g.Screen.SetWindow(textWindowLabel, 9, 133, 30, 60)
			g.printText(textWindowLabel, str, arg, 3)
			g.Text.SelectWindow(save)
			g.Screen.SetWindow(textWindowLabel, 1, 133, 37, 60)
		}
	}
	g.messageFading = false
	if !skip {
		g.animateTalk()
	}
	return true
}

// characterSays has a character speak speech id; with show its face talks. It is
// false only when the character is not in the party or the text is off while the
// speech plays (C: fn_154c_0057).
func (g *Game) characterSays(id, ch int, show bool) bool {
	switch {
	case ch == selectedTalks:
		ch = g.selectedCharacter
	case ch < 0:
		ch = 0
	case ch != 0:
		i := 0
		for ; i < partySize; i++ {
			if g.party[i].ID == ch && g.party[i].Flags&charPresent != 0 {
				ch = i
				break
			}
		}
		if i == partySize {
			return false
		}
	}
	r := g.playSpeech(id, ch, false)
	if r && show {
		g.endTalk()
		g.talkingCharacter = ch
		g.talkStyle = talkPortrait
		g.talkRestore = true
		g.messageFading = false
		g.animateTalk()
	}
	return !(r && !g.textEnabled)
}
