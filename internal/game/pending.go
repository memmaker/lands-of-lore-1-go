package game

// Functions the ported code already calls but which are not ported yet. Each fails
// loudly with its original's name, so a run shows what to port next; the list shrinks as
// the port grows.

import "fmt"

// notPortedError is the panic a not yet ported function raises; Run turns it into its
// error.
type notPortedError struct{ name string }

func (e notPortedError) Error() string { return fmt.Sprintf("not ported yet: %s", e.name) }

func notPorted(name string) { panic(notPortedError{name}) }

func (g *Game) playCharacterSound(c, sound int) { notPorted("fn_0bbc_0744 (character sound)") }

// Combat and character effects the monsters call (CHECKLIST: Combat).
func (g *Game) drainParty(a, b, c int) { notPorted("ovr_2d00_002a (drain the party)") }
func (g *Game) takeCharacterItem(ch, slots, flags int) int {
	notPorted("ovr_2d46_0048 (take a character's item)")
	return 0
}
func (g *Game) poisonCharacter(ch, a, b, c, d int) { notPorted("ovr_2d00_0025 (poison a character)") }
func (g *Game) paralyseCharacter(ch int)           { notPorted("ovr_2d00_0020 (paralyse a character)") }
func (g *Game) swapLevelPaletteBack()              { notPorted("ovr_2d1b_0043 (swap the level palette back)") }
func (g *Game) launchObject(kind, item, x, y, power, dir, a, owner, flags int) bool {
	notPorted("fn_0b6b_032f (launch a flying object)")
	return false
}
func (g *Game) freezeParty()                     { notPorted("ovr_2da0_002a (freeze the party)") }
func (g *Game) playSoundEffect(n int)            { notPorted("fn_17d6_0006 (sound effect)") }
func (g *Game) shakeScreen(a, b, c, d int)       { notPorted("ovr_2d28_0034 (shake the screen)") }
func (g *Game) moveFlyingObject(f *flyingObject) { notPorted("fn_0b6b_0472 (move a flying object)") }
