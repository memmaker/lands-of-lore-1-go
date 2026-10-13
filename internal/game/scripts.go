package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
)

// opcodeCount is the size of the game's opcode table (C: g_2494).
const opcodeCount = 192

// Opcodes with known numbers.
const (
	opcodeAllocItemTypes = 13
	opcodeDefineItemType = 14
	opcodeNothing        = 23
	opcodeNothingToo     = 191
)

// opcodes is the table level, item and start-up scripts call into (C: g_2494). Entries
// still nil are ported along with the systems they drive; a script reaching one stops
// with an error.
func (g *Game) opcodes() []emc.Opcode {
	ops := make([]emc.Opcode, opcodeCount)
	ops[opcodeAllocItemTypes] = g.opAllocItemTypes
	ops[opcodeDefineItemType] = g.opDefineItemType
	ops[opcodeNothing] = opNothing
	ops[opcodeNothingToo] = opNothing
	return ops
}

// opNothing is an empty opcode (C: fn_0fbe_004c).
func opNothing(*emc.State) int { return 0 }

// loadScript reads an EMC file with the game's opcodes.
func (g *Game) loadScript(name string) (*emc.Script, error) {
	file, err := g.Files.Read(name)
	if err != nil {
		return nil, err
	}
	s, err := emc.Load(file, g.opcodes())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return s, nil
}

// runFunction runs a function of a script to its end (C: fn_0fbe_000c on the level
// script slot).
func runFunction(s *emc.Script, function int) error {
	var st emc.State
	st.Init(s)
	st.Start(function)
	st.Run()
	return st.Err
}

// runScriptOnce loads a script into the level script slot, runs its function 0 and then
// function, if not 0, and drops it again (C: ovr_2d2c_007f).
func (g *Game) runScriptOnce(name string, function int) error {
	g.runningOnce = true
	defer func() { g.runningOnce = false }()
	s, err := g.loadScript(name)
	if err != nil {
		return err
	}
	g.levelScript = s
	defer func() { g.levelScript = nil }()
	if err := runFunction(s, 0); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if function != 0 {
		if err := runFunction(s, function); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
