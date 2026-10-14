//go:build !(darwin || linux || freebsd || js)

package sound

import "errors"

// No MT-32 emulator binding on this system (see mt32.go).

func mtOpen(dir string, gen MT32Generation) error { return errors.New("no MT-32 emulator here") }
func mtMsg(msg uint32)                            {}
func mtSysex(msg []byte)                          {}
func mtRender(b []int16)                          {}
