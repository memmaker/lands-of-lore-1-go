// Package emc runs Westwood's EMC scripts (.EMC, .INF): IFF FORM files whose bytecode
// drives the game's levels, items and start-up (C: s018 loading and interpreting, o2de9
// reading the IFF chunks).
package emc

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Script is a loaded EMC file (C: struct EmcData).
type Script struct {
	text    []byte  // TEXT: a table of big-endian offsets into itself, then the strings
	order   []int   // ORDR: entry points into Data, by function number, -1 for none
	data    []int16 // DATA: the bytecode
	Opcodes []Opcode
}

// Opcode is a function the script calls with the opcode instruction; its result becomes
// the return value register (C: the opcode tables passed to fn_0f28_0114).
type Opcode func(s *State) int

// IFF layout (C: ovr_2de9_0020, ovr_2de9_002a, ovr_2de9_002f).
const (
	formID         = "FORM"
	chunksStart    = 12 // past FORM, its size and the form type
	chunkIDSize    = 4
	chunkSizeSize  = 4
	chunkHeader    = chunkIDSize + chunkSizeSize
	noFunction     = -1
	stackSize      = 100
	registerCount  = 15
	initialSP      = 100
	initialBP      = 102
	returnStackTop = stackSize
)

// chunk finds a chunk by id, nil if there is none. The original's reader skips a zero
// word where an id should be (padding some files have).
func chunk(file []byte, id string) []byte {
	at := chunksStart
	for at+chunkHeader <= len(file) {
		if binary.BigEndian.Uint32(file[at:]) == 0 {
			at += chunkIDSize
			continue
		}
		size := int(binary.BigEndian.Uint32(file[at+chunkIDSize:]))
		body := at + chunkHeader
		if string(file[at:at+chunkIDSize]) == id {
			return file[body:min(body+size, len(file))]
		}
		at = body + (size+1)&^1
	}
	return nil
}

// Load reads an EMC file; scripts without ORDR or DATA are refused (C: fn_0f28_0114).
func Load(file []byte, opcodes []Opcode) (*Script, error) {
	if len(file) < chunksStart || string(file[:chunkIDSize]) != formID {
		return nil, errors.New("emc: not an IFF FORM")
	}
	order, data := chunk(file, "ORDR"), chunk(file, "DATA")
	if len(order) == 0 || len(data) == 0 {
		return nil, errors.New("emc: no ORDR or DATA")
	}
	s := &Script{text: chunk(file, "TEXT"), Opcodes: opcodes}
	for i := 0; i+1 < len(order); i += 2 {
		s.order = append(s.order, int(int16(binary.BigEndian.Uint16(order[i:]))))
	}
	for i := 0; i+1 < len(data); i += 2 {
		s.data = append(s.data, int16(binary.BigEndian.Uint16(data[i:])))
	}
	return s, nil
}

// String returns the text chunk's string at a table index (C: the STR macro).
func (s *Script) String(index int) string {
	at := 2 * index
	if at < 0 || at+1 >= len(s.text) {
		return ""
	}
	start := int(binary.BigEndian.Uint16(s.text[at:]))
	end := start
	for end < len(s.text) && s.text[end] != 0 {
		end++
	}
	return string(s.text[min(start, end):end])
}

// State is one running script (C: struct EmcState). IP is -1 when it has stopped.
type State struct {
	Script   *Script
	IP       int
	RetValue int
	BP, SP   int
	Regs     [registerCount]int
	Stack    [stackSize]int
	Running  bool // inside a function started by Call
	Err      error
}

// Init attaches a state to a script, stopped (C: fn_0f28_032c).
func (st *State) Init(s *Script) {
	*st = State{Script: s, IP: -1, BP: initialBP, SP: initialSP}
}

// Start points the state at a function's start, or stops it if the script has no such
// function (C: fn_0f28_035f). The entry point's first word is skipped.
func (st *State) Start(function int) {
	st.IP = -1
	if s := st.Script; s != nil && function >= 0 && function < len(s.order) && s.order[function] != noFunction {
		st.IP = s.order[function] + 1
	}
}

// Active says whether the state has an instruction to run (C: fn_0f28_0478).
func (st *State) Active() bool { return st.Script != nil && st.IP >= 0 }

// Call runs a function as a subroutine of the current position: the return position and
// value are pushed and the function's return comes back here (C: fn_0f28_03d2).
func (st *State) Call(function int) {
	if !st.Active() || st.Running {
		return
	}
	st.Running = true
	st.push(st.IP)
	st.push(st.RetValue)
	st.IP = st.Script.order[function] + 1
}

// Arg is the n-th argument an opcode was called with (C: the ARG macro).
func (st *State) Arg(n int) int {
	if i := st.SP + n; i >= 0 && i < stackSize {
		return st.Stack[i]
	}
	return 0
}

// StringArg is the script string the n-th argument indexes (C: the STR macro).
func (st *State) StringArg(n int) string { return st.Script.String(st.Arg(n)) }

func (st *State) fail(format string, args ...any) bool {
	st.IP = -1
	st.Err = fmt.Errorf("emc: "+format, args...)
	return false
}

// push and pop keep the stack's 16-bit words. fix: the original wrote past the stack
// on overflow; here the script stops.
func (st *State) push(v int) {
	if st.SP <= 0 {
		st.fail("stack overflow")
		return
	}
	st.SP--
	st.Stack[st.SP] = int(int16(v))
}

func (st *State) pop() int {
	if st.SP >= stackSize {
		st.fail("stack underflow")
		return 0
	}
	v := st.Stack[st.SP]
	st.SP++
	return v
}

func (st *State) stackAt(i int) *int {
	if i < 0 || i >= stackSize {
		st.fail("stack index %d", i)
		return new(int)
	}
	return &st.Stack[i]
}
