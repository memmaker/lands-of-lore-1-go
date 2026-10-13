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
func (g *Game) mainLoop() error                 { notPorted("fn_0ca0_0c33 (main loop)"); return nil }
