package emc

// An instruction word: bit 15 set is a jump to the low 15 bits; else bits 8-12 are the
// operation and its argument is the sign-extended low byte (bit 14), the next word
// (bit 13) or 0 (C: fn_0f28_04a0).
const (
	jumpFlag       = 1 << 15
	byteArgFlag    = 1 << 14
	wordArgFlag    = 1 << 13
	operationShift = 8
	operationMask  = 1<<5 - 1
	addressMask    = 1<<15 - 1
	opcodeMask     = 1<<8 - 1
	shiftCountMask = 1<<5 - 1
)

// Operations.
const (
	opJump = iota
	opSetReturn
	opPushSpecial // 0: the return value, 1: a call frame
	opPushNumber
	opPushNumberToo
	opPushRegister
	opPushLocal
	opPushParameter
	opPopSpecial // 0: into the return value, 1: return from a frame
	opPopRegister
	opPopLocal
	opPopParameter
	opDrop
	opReserve
	opCallOpcode
	opJumpIfZero
	opUnary
	opBinary
	opReturn
)

// Run steps the state until it stops; it is how the game runs a script to the end
// (C: the loops of while (fn_0f28_04a0(s));).
func (st *State) Run() {
	for st.Step() {
	}
}

// Step runs one instruction; false once the script has stopped or failed.
func (st *State) Step() bool {
	if !st.Active() {
		return false
	}
	data := st.Script.data
	if st.IP >= len(data) {
		return st.fail("ip %d past the end", st.IP)
	}
	word := int(uint16(data[st.IP]))
	st.IP++
	op := word >> operationShift & operationMask
	arg := 0
	switch {
	case word&jumpFlag != 0:
		op, arg = opJump, word&addressMask
	case word&byteArgFlag != 0:
		arg = int(int8(word))
	case word&wordArgFlag != 0:
		if st.IP >= len(data) {
			return st.fail("ip %d past the end", st.IP)
		}
		arg = int(data[st.IP])
		st.IP++
	}
	switch op {
	case opJump:
		st.IP = arg
	case opSetReturn:
		st.RetValue = arg
	case opPushSpecial:
		switch arg {
		case 0:
			st.push(st.RetValue)
		case 1:
			st.push(st.IP + 1)
			st.push(st.BP)
			st.BP = st.SP + 2
		default:
			return st.stop()
		}
	case opPushNumber, opPushNumberToo:
		st.push(arg)
	case opPushRegister:
		st.push(*st.register(arg))
	case opPushLocal:
		st.push(*st.stackAt(st.BP - (arg + 2)))
	case opPushParameter:
		st.push(*st.stackAt(arg + st.BP - 1))
	case opPopSpecial:
		switch arg {
		case 0:
			st.RetValue = st.pop()
		case 1:
			if st.SP == returnStackTop {
				return st.stop()
			}
			st.BP = st.pop()
			st.IP = st.pop()
		default:
			return st.stop()
		}
	case opPopRegister:
		*st.register(arg) = st.pop()
	case opPopLocal:
		v := st.pop()
		*st.stackAt(st.BP - (arg + 2)) = v
	case opPopParameter:
		v := st.pop()
		*st.stackAt(arg + st.BP - 1) = v
	case opDrop:
		st.SP += arg
	case opReserve:
		st.SP -= arg
	case opCallOpcode:
		n := arg & opcodeMask
		if n >= len(st.Script.Opcodes) || st.Script.Opcodes[n] == nil {
			return st.fail("opcode %d missing", n)
		}
		st.RetValue = int(int16(st.Script.Opcodes[n](st)))
	case opJumpIfZero:
		if st.pop() == 0 {
			st.IP = arg & addressMask
		}
	case opUnary:
		x := st.pop()
		switch arg {
		case 0:
			x = boolInt(x == 0)
		case 1:
			x = -x
		case 2:
			x = ^x
		default:
			return st.stop()
		}
		st.push(x)
	case opBinary:
		y, x := st.pop(), st.pop()
		r, ok := binaryOp(arg, x, y)
		if !ok {
			return st.stop()
		}
		st.push(r)
	case opReturn:
		if st.SP == returnStackTop {
			return st.stop()
		}
		st.RetValue = st.pop()
		st.IP = st.pop()
		st.Running = false
	default:
		return st.stop()
	}
	if st.Err != nil {
		return false
	}
	return true
}

// binaryOp is the two-operand operations on 16-bit values; shift counts are taken modulo
// 32 as the 386 does.
func binaryOp(op, x, y int) (int, bool) {
	a, b := int16(x), int16(y)
	switch op {
	case 0:
		return boolInt(a != 0 && b != 0), true
	case 1:
		return boolInt(a != 0 || b != 0), true
	case 2:
		return boolInt(a == b), true
	case 3:
		return boolInt(a != b), true
	case 4:
		return boolInt(a < b), true
	case 5:
		return boolInt(a <= b), true
	case 6:
		return boolInt(a > b), true
	case 7:
		return boolInt(a >= b), true
	case 8:
		return int(a + b), true
	case 9:
		return int(a - b), true
	case 10:
		return int(a * b), true
	case 11:
		if b == 0 { // fix: the original's divide by zero ended the program
			return 0, true
		}
		return int(a / b), true
	case 12:
		return int(a >> (uint(b) & shiftCountMask)), true
	case 13:
		return int(a << (uint(b) & shiftCountMask)), true
	case 14:
		return int(a & b), true
	case 15:
		return int(a | b), true
	case 16:
		if b == 0 { // fix: as for division
			return 0, true
		}
		return int(a % b), true
	case 17:
		return int(a ^ b), true
	}
	return 0, false
}

// stop ends the script the way the original's error exit does, which is also how a
// script's outermost return ends it.
func (st *State) stop() bool {
	st.IP = -1
	return false
}

func (st *State) register(n int) *int {
	if n < 0 || n >= registerCount {
		st.fail("register %d", n)
		return new(int)
	}
	return &st.Regs[n]
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
