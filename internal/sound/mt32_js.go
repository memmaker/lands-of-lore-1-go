package sound

import (
	"errors"
	"syscall/js"
)

// The MT-32 in the browser (see mt32.go): goplay's page loads munt and offers it as
// goplayMT32, which opens the generation asked for among the ROMs in dir.

var mtJS js.Value

var mtGenerations = map[MT32Generation]string{MT32Old: "old", MT32New: "new"}

func mtOpen(dir string, gen MT32Generation) error {
	g := js.Global().Get("goplayMT32")
	if g.IsUndefined() || g.Get("ready").IsUndefined() || !g.Call("ready").Bool() {
		return errors.New("the page has no MT-32 emulator")
	}
	if g.Get("generation").IsUndefined() {
		return errors.New("the page's MT-32 cannot choose a generation")
	}
	if err := g.Call("open", dir, outputRate, mtGenerations[gen]); err.Truthy() {
		return errors.New(err.String())
	}
	mtJS = g
	return nil
}

func mtMsg(msg uint32) { mtJS.Call("playMsg", msg) }

func mtSysex(msg []byte) {
	a := js.Global().Get("Uint8Array").New(len(msg))
	js.CopyBytesToJS(a, msg)
	mtJS.Call("playSysex", a)
}

var mtBytes []byte

// mtRender: goplayMT32.render returns the frames as a Uint8Array (16-bit little endian).
func mtRender(b []int16) {
	if cap(mtBytes) < len(b)*2 {
		mtBytes = make([]byte, len(b)*2)
	}
	buf := mtBytes[:len(b)*2]
	js.CopyBytesToGo(buf, mtJS.Call("render", len(b)/frameWords))
	for i := range b {
		b[i] = int16(uint16(buf[2*i]) | uint16(buf[2*i+1])<<8)
	}
}
