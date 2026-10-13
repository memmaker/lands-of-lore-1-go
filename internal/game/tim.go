package game

import (
	"encoding/binary"
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
)

// TIM scripts: timed animation and dialogue scripts, an IFF file whose AVTL chunk holds
// a table of function offsets and the commands, and whose TEXT chunk holds strings.
// Up to ten functions run side by side, each a list of commands that wait their delay
// in ticks before they run (C: s001, s002; ScummVM: TIMInterpreter).
const (
	timSlots     = 10 // scripts loaded at once (C: g_20fc)
	timFunctions = 10
	timWSASlots  = 10
	timFileName  = "%s.TIM"

	// a command: its length in words, its delay, its opcode, then its parameters
	timCmdLength = 0
	timCmdDelay  = 1
	timCmdOp     = 2
	timCmdParams = 3

	noTimFunction = -1
	noTimStop     = -1
	formHeader    = 12 // "FORM", its size and the form type
	chunkHeader   = 8
)

// IFF chunk tags (C: ovr_2de9_002a's ids).
const (
	tagForm = "FORM"
	tagAVTL = "AVTL"
	tagText = "TEXT"
)

// timScript is a loaded TIM (C: struct TimHdr with its function and WSA areas).
type timScript struct {
	stop    int // the function waiting on a dialogue answer, or noTimStop (C: stop)
	choices int // the dialogue's number of choices (C: numStart)
	funcs   [timFunctions]timFunc
	wsa     [timWSASlots]timWSA
	avtl    []int // the commands as words; offsets count words (C: g_4bc0)
	text    []byte
}

// timFunc is one running function. Word indexes into avtl stand for the original's
// pointers; 0 (the function table's first word) is the null pointer.
type timFunc struct {
	ip                 int
	lastTime, nextTime int
	loopIP             int // where a loop restarts (C: loopIp)
	start              int // the function's first command (C: avtl)
}

// timWSA is an animation a TIM opened (C: struct WsaSlot).
type timWSA struct {
	open    bool
	x, y    int
	unknown int // (C: w8)
	flags   int
}

// readIFFChunk finds chunk tag in an IFF file; the FORM tag gives the form's size
// (C: ovr_2de9_002a, ovr_2de9_002f).
func readIFFChunk(file []byte, tag string) []byte {
	be := binary.BigEndian
	if len(file) >= chunkHeader && string(file[:4]) == tag {
		return file[chunkHeader:min(chunkHeader+int(be.Uint32(file[4:])), len(file))]
	}
	for at := formHeader; at+chunkHeader <= len(file); {
		if be.Uint32(file[at:]) == 0 { // a pad word before the id
			at += 4
			continue
		}
		id := string(file[at : at+4])
		size := int(be.Uint32(file[at+4:]))
		if id == tag {
			return file[at+chunkHeader : min(at+chunkHeader+size, len(file))]
		}
		at += chunkHeader + (size+1)&^1
	}
	return nil
}

// loadTIM reads a TIM file (C: fn_0638_0008).
func (g *Game) loadTIM(name string) (*timScript, error) {
	file, err := g.Files.Read(name)
	if err != nil {
		return nil, err
	}
	const minForm = 20
	if len(readIFFChunk(file, tagForm))+4 < minForm {
		return nil, nil
	}
	t := &timScript{stop: noTimStop}
	avtl := readIFFChunk(file, tagAVTL)
	t.avtl = make([]int, len(avtl)/2)
	for i := range t.avtl {
		t.avtl[i] = int(int16(binary.LittleEndian.Uint16(avtl[2*i:])))
	}
	t.text = append([]byte(nil), readIFFChunk(file, tagText)...)
	if len(t.avtl) > 0 {
		t.funcs[0].start = t.avtl[0]
	}
	return t, nil
}

// timString is string n of the TIM's TEXT chunk (C: g_4bc4 + g_4bc4[n]).
func (t *timScript) timString(n int) string {
	le := binary.LittleEndian
	if 2*n+2 > len(t.text) {
		return ""
	}
	at := int(le.Uint16(t.text[2*n:]))
	end := at
	for end < len(t.text) && t.text[end] != 0 {
		end++
	}
	return string(t.text[min(at, end):end])
}

func (t *timScript) word(i int) int {
	if i < 0 || i >= len(t.avtl) {
		return 0
	}
	return t.avtl[i]
}

// advance moves function f to its next command and schedules it (C: fn_0638_01a6).
func (g *Game) advanceTIM(t *timScript, f *timFunc) {
	f.ip += t.word(f.ip + timCmdLength)
	f.lastTime = f.nextTime
	f.nextTime += t.word(f.ip+timCmdDelay) + g.speechDelay
	g.speechDelay = 0
}

// TIM opcode results.
const (
	timStopAll    = -1
	timWaitAnswer = -3
	timYield      = -2
	timLoopEnded  = 22
)

// runTIM runs a TIM's functions; with loop it keeps going until the script stops.
// It returns the dialogue's answer (C: fn_0638_01e5).
func (g *Game) runTIM(t *timScript, loop bool) int {
	if t == nil {
		return 0
	}
	g.tim = t
	f0 := &t.funcs[0]
	if f0.ip == 0 {
		f0.ip = f0.start
		f0.lastTime, f0.nextTime = clock.Ticks(), clock.Ticks()
	}
	for {
		g.idleWork()
		for g.timFunction = 0; g.timFunction < timFunctions; g.timFunction++ {
			cur := &t.funcs[g.timFunction]
			if t.stop != noTimStop {
				g.timCheckAnswer()
			}
			g.idleWork()
			if t.choices > 1 && cur.loopIP != 0 && g.speechVolume != 0 && g.updateSpeech() != speechPlaying {
				cur.loopIP = 0
				g.timDialogueFunction = g.timFunction
				g.timSkipTo(timOpContinueAfterSpeech)
				g.timDialogueFunction = noTimFunction
			}
			if cur.ip == 0 {
				continue
			}
			run := true
			for n := 0; cur.nextTime <= clock.Ticks() && run; n++ {
				if n > 0 {
					if t.stop != noTimStop {
						g.timCheckAnswer()
					}
					g.idleWork()
				}
				op := t.word(cur.ip+timCmdOp) & lowByteMask
				params := cur.ip + timCmdParams
				switch g.timOp(t, op, params) {
				case timStopAll:
					loop = false
					run = false
					g.timFunction = timFunctions
				case timWaitAnswer:
					t.stop = g.timFunction
					g.timDialogueFunction = noTimFunction
				case timYield:
					run = false
				case timLoopEnded:
					cur.loopIP = 0
				}
				g.advanceTIM(t, cur)
			}
			if g.timFunction >= timFunctions {
				break
			}
		}
		if !loop {
			break
		}
		host.Yield()
	}
	return g.timResult
}

const (
	lowByteMask   = 1<<8 - 1
	speechPlaying = 2
	timOpStop     = 1
	// the command a dialogue's speech skips to (C: fn_057f_07aa(0x15))
	timOpContinueAfterSpeech = 21
)

// timParam is parameter i of the command whose parameters start at p.
func (t *timScript) param(p, i int) int { return t.word(p + i) }

// timOp runs TIM opcode op (C: g_1dcc).
func (g *Game) timOp(t *timScript, op, p int) int {
	arg := func(i int) int { return t.param(p, i) }
	switch op {
	case 0: // restart (C: fn_057f_000a)
		for i := 1; i < timWSASlots; i++ {
			t.wsa[i] = timWSA{}
		}
		t.funcs[0].ip = t.funcs[0].start
		t.funcs[0].lastTime = clock.Ticks()
		return 1
	case timOpStop: // wait for the answer, then stop (C: fn_057f_0053)
		for g.timDialogueFunction == noTimFunction && g.timResult == 0 {
			g.idleWork()
			g.timResult = g.dialogueAnswer()
			host.Yield()
		}
		for i := range t.funcs {
			t.funcs[i].ip = 0
		}
		return timStopAll
	case 2: // open an animation (C: fn_057f_02b6)
		n := arg(0)
		w := &t.wsa[n]
		w.x, w.y, w.unknown, w.flags = arg(2), arg(3), arg(4), arg(5)
		const wsaOffset = 10
		w.open = g.openSceneAnimation(n, t.timString(arg(1)), w.x, w.y, wsaOffset, w.flags)
		return 1
	case 3: // close an animation (C: fn_057f_034f)
		n := arg(0)
		g.closeSceneAnimation(n)
		t.wsa[n] = timWSA{}
		return 1
	case 4: // start a function (C: fn_057f_0384)
		f := &t.funcs[arg(0)]
		if f.start == 0 {
			f.start = t.word(arg(0))
		}
		f.ip = f.start
		return 1
	case 5: // stop a function (C: fn_057f_03f7)
		t.funcs[arg(0)].ip = 0
		return 1
	case 6: // show an animation frame (C: fn_057f_0426)
		g.showAnimationFrame(arg(0), arg(1))
		return 1
	case 14: // (C: fn_057f_0501)
		notPorted("fn_178d_0091 (TIM opcode 14)")
		return 1
	case 15: // (C: fn_057f_0515)
		notPorted("fn_178d_015e (TIM opcode 15)")
		return 1
	case 18: // (C: fn_057f_0532)
		notPorted("fn_17d6_0006 (TIM opcode 18)")
		return 1
	case 19: // (C: fn_057f_0546)
		notPorted("fn_17d6_0123 (TIM opcode 19)")
		return 1
	case 20: // mark the loop start, unless speech is to be waited for (C: fn_057f_0553)
		if g.speechVolume != 0 && g.updateSpeech() != speechPlaying {
			g.timSkipTo(timOpContinueAfterSpeech)
			return 1
		}
		f := &t.funcs[g.timFunction]
		f.loopIP = f.ip
		return 1
	case timOpContinueAfterSpeech: // loop back, a random time later (C: fn_057f_059b)
		r := arg(0)
		f := &t.funcs[g.timFunction]
		if f.loopIP != 0 {
			f.ip = f.loopIP
			if r != 0 && g.updateSpeech() != speechPlaying {
				const randScale = 1 << 15
				f.nextTime += random.Rand() * r / randScale
			}
		}
		return timYield
	case 22: // end a function's loop (C: fn_057f_060e)
		t.funcs[arg(0)].loopIP = 0
		return 1
	case 23: // run all functions now (C: fn_057f_065c)
		for i := range t.funcs {
			if t.funcs[i].ip != 0 {
				t.funcs[i].nextTime = clock.Ticks() + 1
			}
		}
		return 1
	case 25: // a sub-opcode (C: fn_057f_0631, whose result was not returned)
		g.timSubOp(t, arg(0), p+1)
		return 0
	case 28: // check the dialogue (C: fn_057f_0730)
		return g.timCheckAnswer()
	case 29: // start a dialogue (C: fn_057f_06a1)
		t.choices = arg(0)
		g.timResult = 0
		var choices []string
		for i := 1; i <= 3; i++ {
			if arg(i) != noString {
				choices = append(choices, g.text(arg(i)))
			} else {
				choices = append(choices, "")
			}
		}
		g.showDialogueChoices(choices)
		g.flushInput()
		g.GUI.MarkChanged()
		return timWaitAnswer
	case 7, 8, 9, 10, 11, 12, 13, 16, 17, 24, 30:
		return 1
	}
	// fix: opcodes 26 and 27 were null pointers in the original's table
	return 1
}

// timCheckAnswer takes the dialogue's answer once given (C: fn_057f_0730).
func (g *Game) timCheckAnswer() int {
	t := g.tim
	r := g.dialogueAnswer()
	if r != 0 && t.choices != 0 {
		g.speechKey(0)
		t.funcs[t.stop].loopIP = 0
		g.timDialogueFunction = t.stop
		t.stop = noTimStop
		g.timResult = r
		if t.choices != 0 {
			g.timSkipTo(timOpContinueAfterSpeech)
		}
	}
	return r
}

// timSkipTo moves the dialogue function on to its next command with op; at the
// script's stop command it gives that the previous command's length (C: fn_057f_07aa).
func (g *Game) timSkipTo(op int) {
	t := g.tim
	f := &t.funcs[g.timDialogueFunction]
	prev := t.word(f.ip)
	for t.word(f.ip+timCmdOp)&lowByteMask != op {
		if t.word(f.ip+timCmdOp)&lowByteMask == timOpStop {
			if f.ip > 0 && f.ip < len(t.avtl) {
				t.avtl[f.ip] = prev
			}
			break
		}
		prev = t.word(f.ip)
		next := f.ip + t.word(f.ip)
		if next == f.ip { // fix: a zero length looped for ever
			break
		}
		f.ip = next
	}
	f.nextTime = clock.Ticks()
}

// timSubOp runs sub-opcode op (C: g_1e48).
func (g *Game) timSubOp(t *timScript, op, p int) int {
	arg := func(i int) int { return t.param(p, i) }
	switch op {
	case 0: // (C: fn_057f_081f)
		notPorted("ovr_2d5f_00fc (TIM sub-opcode 0)")
	case 1: // (C: fn_057f_0833)
		notPorted("ovr_2d5f_0101 (TIM sub-opcode 1)")
	case 2:
	case 3: // give an item (C: fn_057f_084f)
		notPorted("fn_07bc_012d (TIM: give an item)")
	case 4: // set the party's block or direction (C: fn_057f_0887)
		switch arg(0) {
		case 0:
			g.currentBlock = arg(1)
			g.partyX, g.partyY = blockPosition(g.currentBlock, blockCentre, blockCentre)
		case 1:
			g.direction = arg(1)
		}
	case 5: // palette (C: fn_057f_08cd)
		notPorted(fmt.Sprintf("fn_057f_08cd (TIM palette %d)", arg(0)))
	case 6: // copy a rectangle (C: fn_057f_099a)
		g.Screen.HideMouse()
		g.Screen.CopyRect(arg(0), arg(1), arg(2), arg(3), arg(4), arg(5), arg(6), arg(7))
		g.Screen.ShowMouse()
	case 7: // print a text (C: fn_057f_09d6)
		notPorted("fn_130a_03cd (TIM: print a text)")
	case 8: // draw the scene (C: fn_057f_0a0a)
		n := arg(0)
		g.drawScene(n)
		if g.viewBackPage != pageBuild && n == pageBuild {
			// the original gave the column copy pixel values, so it copied nothing
			g.Screen.CopyRectColumns(viewLeft, 0, viewLeft, 0, viewColumns*vcnPixels, viewFrameH, g.viewBackPage, pageBuild)
		}
	case 9: // (C: fn_057f_0a47)
		g.idleWork()
	case 10: // (C: fn_057f_0a54)
		notPorted("fn_057f_0a54 (TIM: clear the text box)")
	case 11: // (C: fn_057f_0ac1)
		g.queueTrack(arg(0))
	case 12: // (C: fn_057f_0ad5)
		g.playMusic(arg(0))
	case 13: // speak and show a text (C: fn_057f_0ae9)
		notPorted("fn_057f_0ae9 (TIM: speech)")
	case 14: // (C: fn_057f_0b4a)
		g.playSound(arg(0), 0)
	case 15: // (C: fn_057f_0b62)
		notPorted("fn_130a_0c28 (TIM sub-opcode 15)")
	case 16: // (C: fn_057f_0b7c)
		notPorted("fn_130a_0cf6 (TIM sub-opcode 16)")
	}
	return 1
}

// TIM opcodes of the EMC scripts.

// opLoadTIM loads a TIM into a slot unless one is there (C: fn_0fbe_1910).
func (g *Game) opLoadTIM(s *emc.State) int {
	i := s.Arg(0)
	if g.tims[i] == nil {
		t, err := g.loadTIM(fmt.Sprintf(timFileName, s.StringArg(1)))
		if err != nil && g.scriptErr == nil {
			g.scriptErr = err
		}
		g.tims[i] = t
	}
	return 1
}

// opRunTIM (C: fn_0fbe_19ad).
func (g *Game) opRunTIM(s *emc.State) int { return g.runTIM(g.tims[s.Arg(0)], s.Arg(1) == 1) }

// opFreeTIM (C: fn_0fbe_19e1).
func (g *Game) opFreeTIM(s *emc.State) int {
	g.tims[s.Arg(0)] = nil
	return 1
}

// opStopTIM stops all of a TIM's functions (C: fn_0fbe_1ad1).
func (g *Game) opStopTIM(s *emc.State) int {
	t := g.tims[s.Arg(0)]
	for i := range t.funcs {
		t.funcs[i].ip = 0
	}
	return 1
}

// opStartTIMDialogue gets a TIM ready for a dialogue (C: fn_0fbe_28cc).
func (g *Game) opStartTIMDialogue(s *emc.State) int {
	t := g.tims[s.Arg(0)]
	t.stop = 0
	g.timResult = 0
	t.choices = g.dialogueChoices
	if t.choices == 0 {
		t.choices = 1
	}
	g.timDialogueFunction = noTimFunction
	return 1
}

// Not ported yet.

func (g *Game) openSceneAnimation(n int, name string, x, y, offset, flags int) bool {
	notPorted("ovr_2d5f_0084 (open a scene animation)")
	return false
}
func (g *Game) closeSceneAnimation(n int) { notPorted("ovr_2d5f_008e (close a scene animation)") }
func (g *Game) showAnimationFrame(n, frame int) {
	notPorted("fn_057f_0426 (show a level animation frame)")
}
func (g *Game) dialogueAnswer() int {
	notPorted("fn_13e8_06a1 (dialogue answer)")
	return 0
}
func (g *Game) showDialogueChoices(choices []string) {
	notPorted("fn_13e8_0903 (dialogue choices)")
}
