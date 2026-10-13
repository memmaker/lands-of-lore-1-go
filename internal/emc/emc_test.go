package emc

import (
	"os"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
)

func gameFiles(t *testing.T) *data.Files {
	dir := os.Getenv("LOLDATA")
	if dir == "" {
		dir = "../../game"
	}
	f, err := data.Open(dir)
	if err != nil {
		t.Skip("no game files; set LOLDATA")
	}
	return f
}

func TestRunOnetime(t *testing.T) {
	file, err := gameFiles(t).Read("ONETIME.INF")
	if err != nil {
		t.Fatal(err)
	}
	called := map[int]int{}
	ops := make([]Opcode, 256)
	for i := range ops {
		ops[i] = func(s *State) int { called[i]++; return 0 }
	}
	s, err := Load(file, ops)
	if err != nil {
		t.Fatal(err)
	}
	var st State
	st.Init(s)
	st.Start(0)
	steps := 0
	for st.Step() {
		steps++
	}
	if st.Err != nil {
		t.Fatal(st.Err)
	}
	t.Logf("%d steps, opcodes %v, %d functions", steps, called, len(s.order))
}

func TestArithmetic(t *testing.T) {
	// 7 - -3 into the return value; 1 << 33, which the 386 shifts by 1, into register 0.
	word := func(op, arg int) int16 { return int16(byteArgFlag | op<<operationShift | arg&0xff) }
	s := &Script{order: []int{0}, data: []int16{0,
		word(opPushNumber, 7), word(opPushNumber, -3), word(opBinary, 9), word(opPopSpecial, 0),
		word(opPushNumber, 1), word(opPushNumber, 33), word(opBinary, 13), word(opPopRegister, 0),
	}}
	var st State
	st.Init(s)
	st.Start(0)
	for st.Step() {
	}
	if st.RetValue != 10 || st.Regs[0] != 2 {
		t.Fatalf("ret %d reg %d", st.RetValue, st.Regs[0])
	}
}
