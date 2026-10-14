//go:build darwin || linux || freebsd

package sound

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ebitengine/purego"
)

// The MT-32 on the desktop (see mt32.go): munt's libmt32emu, loaded at run time with
// purego (no cgo).

var mt32Libs = []string{
	"libmt32emu.dylib", "/opt/homebrew/lib/libmt32emu.dylib", "/usr/local/lib/libmt32emu.dylib",
	"libmt32emu.so.2", "libmt32emu.so",
}

// munt's return codes for an added ROM (C: MT32EMU_RC_ADDED_CONTROL_ROM, ..._PCM_ROM).
const (
	addedControlROM = 1
	addedPCMROM     = 2
)

var mtAPI struct {
	loaded     bool
	createCtx  func(handler, data uintptr) uintptr
	freeCtx    func(ctx uintptr)
	addMachine func(ctx uintptr, machine, name string) int32
	setRate    func(ctx uintptr, rate float64)
	openSynth  func(ctx uintptr) int32
	playMsg    func(ctx uintptr, msg uint32)
	playSysex  func(ctx uintptr, sysex *byte, n uint32)
	render     func(ctx uintptr, out *int16, frames uint32)
	version    func() string
}

func loadMT32Lib() error {
	if mtAPI.loaded {
		return nil
	}
	var lib uintptr
	var err error
	for _, name := range mt32Libs {
		if lib, err = purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			break
		}
	}
	if lib == 0 {
		return fmt.Errorf("libmt32emu not found: %v", err)
	}
	purego.RegisterLibFunc(&mtAPI.createCtx, lib, "mt32emu_create_context")
	purego.RegisterLibFunc(&mtAPI.freeCtx, lib, "mt32emu_free_context")
	purego.RegisterLibFunc(&mtAPI.addMachine, lib, "mt32emu_add_machine_rom_file")
	purego.RegisterLibFunc(&mtAPI.setRate, lib, "mt32emu_set_stereo_output_samplerate")
	purego.RegisterLibFunc(&mtAPI.openSynth, lib, "mt32emu_open_synth")
	purego.RegisterLibFunc(&mtAPI.playMsg, lib, "mt32emu_play_msg")
	purego.RegisterLibFunc(&mtAPI.playSysex, lib, "mt32emu_play_sysex")
	purego.RegisterLibFunc(&mtAPI.render, lib, "mt32emu_render_bit16s")
	purego.RegisterLibFunc(&mtAPI.version, lib, "mt32emu_get_library_version_string")
	mtAPI.loaded = true
	return nil
}

var mtCtx uintptr

// mtOpen starts the synth as the best machine of the generation whose control and PCM
// ROM are both among the files in dir; munt only takes a ROM that fits the machine.
func mtOpen(dir string, gen MT32Generation) error {
	if err := loadMT32Lib(); err != nil {
		return err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("ROMs: %v", err)
	}
	var names []string
	for _, e := range ents {
		if !e.IsDir() {
			names = append(names, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(names)
	for _, machine := range mt32Machines[gen] {
		ctx := mtAPI.createCtx(0, 0)
		if ctx == 0 {
			return errors.New("mt32emu: no context")
		}
		control, pcm := false, false
		for _, n := range names {
			switch mtAPI.addMachine(ctx, machine, n) {
			case addedControlROM:
				control = true
			case addedPCMROM:
				pcm = true
			}
		}
		if control && pcm {
			mtAPI.setRate(ctx, outputRate)
			if mtAPI.openSynth(ctx) == 0 {
				mtCtx = ctx
				return nil
			}
		}
		mtAPI.freeCtx(ctx)
	}
	return fmt.Errorf("mt32emu %s: no control and PCM ROM of this generation in %s", mtAPI.version(), dir)
}

func mtMsg(msg uint32)   { mtAPI.playMsg(mtCtx, msg) }
func mtSysex(msg []byte) { mtAPI.playSysex(mtCtx, &msg[0], uint32(len(msg))) }
func mtRender(b []int16) { mtAPI.render(mtCtx, &b[0], uint32(len(b)/frameWords)) }
