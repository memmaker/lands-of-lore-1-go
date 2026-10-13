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
func (g *Game) waitTicks(n int)                 { notPorted("fn_0845_0bb6 (wait, running the game's idle work)") }

// Combat and character effects the monsters call (CHECKLIST: Combat).
func (g *Game) drainParty(a, b, c int) { notPorted("ovr_2d00_002a (drain the party)") }
func (g *Game) takeCharacterItem(ch, slots, flags int) int {
	notPorted("ovr_2d46_0048 (take a character's item)")
	return 0
}
func (g *Game) characterSays(id, character int, wait bool) bool {
	notPorted("fn_154c_0057 (a character speaks)")
	return false
}
func (g *Game) printMessage(flags int, format string, args ...any) {
	notPorted("fn_0845_01c4 (print a message)")
}
func (g *Game) poisonCharacter(ch, a, b, c, d int) { notPorted("ovr_2d00_0025 (poison a character)") }
func (g *Game) paralyseCharacter(ch int)           { notPorted("ovr_2d00_0020 (paralyse a character)") }
func (g *Game) characterFacing(x, y int) int {
	notPorted("fn_070c_03b1 (the character a monster attacks)")
	return 0
}
func (g *Game) attackHits(who, target, a int) int {
	notPorted("fn_070c_044a (does an attack hit)")
	return 0
}
func (g *Game) attackDamage(who, target, hit int) int {
	notPorted("fn_070c_0793 (attack damage)")
	return 0
}
func (g *Game) scaledDamage(d int) int {
	notPorted("fn_070c_07e3 (damage by difficulty)")
	return d
}
func (g *Game) damageCharacter(ch, damage, source, a, how int) {
	notPorted("fn_070c_07f5 (damage a character)")
}
func (g *Game) launchObject(kind, item, x, y, power, dir, a, owner, flags int) bool {
	notPorted("fn_0b6b_032f (launch a flying object)")
	return false
}
func (g *Game) freezeParty()               { notPorted("ovr_2da0_002a (freeze the party)") }
func (g *Game) playSoundEffect(n int)      { notPorted("fn_17d6_0006 (sound effect)") }
func (g *Game) shakeScreen(a, b, c, d int) { notPorted("ovr_2d28_0034 (shake the screen)") }
