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

// Combat and character effects the monsters call (CHECKLIST: Combat).
func (g *Game) takeCharacterItem(ch, slots, flags int) int {
	notPorted("ovr_2d46_0048 (take a character's item)")
	return 0
}
func (g *Game) freezeParty() { notPorted("ovr_2da0_002a (freeze the party)") }
func (g *Game) showAutomap() { notPorted("ovr_2ce1_0020 (automap)") }

// The spells' effects, overlay by overlay (CHECKLIST: Monsters, combat, magic).
