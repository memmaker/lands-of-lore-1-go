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
func (g *Game) takeCharacterItem(ch, slots, flags int) int {
	notPorted("ovr_2d46_0048 (take a character's item)")
	return 0
}
func (g *Game) swapLevelPaletteBack() { notPorted("ovr_2d1b_0043 (swap the level palette back)") }
func (g *Game) launchObject(kind, item, x, y, power, dir, a, owner, flags int) bool {
	notPorted("fn_0b6b_032f (launch a flying object)")
	return false
}
func (g *Game) freezeParty()                     { notPorted("ovr_2da0_002a (freeze the party)") }
func (g *Game) moveFlyingObject(f *flyingObject) { notPorted("fn_0b6b_0472 (move a flying object)") }
func (g *Game) shatterWall()                     { notPorted("ovr_2d1b_0052 (shatter a crystal wall)") }
func (g *Game) showAutomap()                     { notPorted("ovr_2ce1_0020 (automap)") }

// The spells' effects, overlay by overlay (CHECKLIST: Monsters, combat, magic).
func (g *Game) sparkSpell(level, caster int)     { notPorted("ovr_2d83_0020 (spark spell)") }
func (g *Game) iceSpell(caster, level int)       { notPorted("ovr_2d1b_0039 (ice spell)") }
func (g *Game) lightningSpell(caster, level int) { notPorted("ovr_2d03_002a (lightning spell)") }
func (g *Game) fireballSpell(level int)          { notPorted("ovr_2d15_002a (fireball spell)") }
func (g *Game) mistSpell(caster, level int)      { notPorted("ovr_2d4d_002a (mist spell)") }
func (g *Game) quakeSpell(level, caster int)     { notPorted("ovr_2d28_002a (quake spell)") }
func (g *Game) freezeSpell(n, caster int)        { notPorted("ovr_2d99_002a (freeze spell)") }
