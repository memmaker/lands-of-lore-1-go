package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
)

// Portrait layout (C: fn_0a21_0a0c): small portraits across the screen in the compact
// control mode without text, else large ones beside the compass.
const (
	compactPortraitsWidth = 280
	compactPortraitWidth  = 33
	compactPortraitsLeft  = 10
	portraitsWidth        = 235
	portraitWidth         = 66
	portraitsLeft         = 83
)

// faceShapes is how many shapes a character's face file holds; FACEnn.SHP is named by
// the character's id (C: g_0fb6).
const (
	faceShapes     = 28
	faceFileFormat = "FACE%02d.SHP"
)

// A new member's first face animation comes 7 to 18 ticks after joining; members after
// the first greet the party with face 2 and sound 6.
const (
	joinAnimDice      = 12
	joinAnimBase      = 6
	joinGreetingFace  = 2
	joinGreetingSound = 6
)

// activeCharacters counts the party (C: ovr_2d74_0025).
func (g *Game) activeCharacters() int {
	n := 0
	for i := range g.party {
		if g.party[i].Flags&charPresent != 0 {
			n++
		}
	}
	return n
}

// layOutPortraits works out where each member's portrait goes (C: fn_0a21_0a0c).
func (g *Game) layOutPortraits() {
	n := g.activeCharacters()
	width, each, left := portraitsWidth, portraitWidth, portraitsLeft
	if g.controlMode != 0 && !g.textEnabled {
		width, each, left = compactPortraitsWidth, compactPortraitWidth, compactPortraitsLeft
	}
	gap := (width - n*each) / (n + 1)
	for i := range n {
		g.portraitX[i] = gap*(i+1) + i*each + left
	}
}

// addCharacter lets the template with an id join the party; false if the party is full
// or there is no such character (C: ovr_2d74_0020).
func (g *Game) addCharacter(id int) (bool, error) {
	c := g.activeCharacters()
	if c == partySize {
		return false, nil
	}
	t := -1
	for i := range characterTemplates {
		if characterTemplates[i].ID == id {
			t = i
			break
		}
	}
	if t < 0 {
		return false, nil
	}
	g.party[c] = characterTemplates[t]
	if err := g.loadFaces(c, g.party[c].ID); err != nil {
		return false, err
	}
	g.party[c].AnimCountdown = dice(1, joinAnimDice) + joinAnimBase
	for i, itemType := range g.party[c].Items {
		if itemType == 0 {
			continue
		}
		item := g.newItem(itemType, 0, 0)
		g.party[c].Items[i] = item
		if err := g.runItemScript(c, item, itemEventEquip, 0, 0); err != nil {
			return false, err
		}
	}
	g.layOutPortraits()
	if c != 0 {
		g.setTemporaryFace(c, joinGreetingFace, joinGreetingSound, false)
	}
	return true, nil
}

// loadFaces loads a character's face shapes (C: ovr_2d46_0034).
func (g *Game) loadFaces(c, id int) error {
	faces, err := g.loadShapesUpTo(fmt.Sprintf(faceFileFormat, abs(id)), faceShapes)
	if err != nil {
		return err
	}
	g.faces[c] = faces
	return nil
}

// setTemporaryFace shows a face for a while, with a sound (C: fn_0a21_0674).
func (g *Game) setTemporaryFace(c, face, sound int, redraw bool) {
	g.party[c].FaceDefault = face
	if face != 0 && sound != 0 {
		g.playCharacterSound(c, sound)
	}
	if redraw {
		g.drawPortrait(c)
	}
}

// opCharacterJoinsOrLeaves: arg 0 positive adds that character, negative removes it
// (C: ovr_2d74_002a).
func (g *Game) opCharacterJoinsOrLeaves(s *emc.State) int {
	id := s.Arg(0)
	if id < 0 {
		id = abs(id)
		for i := range g.party {
			if g.party[i].Flags&charPresent != 0 && g.party[i].ID == id {
				g.party[i].Flags &^= charPresent
				g.layOutPortraits()
				if g.updateFlags == 0 {
					g.buildButtons()
					g.drawPortraits()
				}
				if g.selectedCharacter == i {
					g.selectedCharacter = 0
				}
				break
			}
		}
		return 1
	}
	if _, err := g.addCharacter(id); err != nil {
		s.Err = err
	}
	if g.updateFlags == 0 {
		g.buildButtons()
		g.drawPortraits()
	}
	return 1
}

// dice rolls n dice of a number of sides (C: fn_0845_0636).
func dice(n, sides int) int {
	const randScale = 1 << 15
	sum := 0
	if sides > 0 && n > 0 {
		for range n {
			r := random.Rand()*sides/randScale + 1
			if r > sides {
				r -= sides
			}
			sum += r
		}
	}
	return sum
}
