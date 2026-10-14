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

// The spells' effects, overlay by overlay (CHECKLIST: Monsters, combat, magic).
