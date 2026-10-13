package host

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ScanEvent is a key going down or up as the PC keyboard reported it to its interrupt:
// a set 1 scan code, Extended for the keys the keyboard prefixed with 0xE0.
type ScanEvent struct {
	Scan     byte
	Extended bool
	Released bool
}

// ScanEvents holds key events the game has not taken yet.
var ScanEvents []ScanEvent

// TakeScanEvents returns the queued key events and empties the queue.
func TakeScanEvents() []ScanEvent {
	e := ScanEvents
	ScanEvents = nil
	return e
}

// PushScan queues a key event (for automation and tests).
func PushScan(e ScanEvent) { ScanEvents = append(ScanEvents, e) }

type scanCode struct {
	scan     byte
	extended bool
}

// scanCodes are the set 1 scan codes of the host's keys.
var scanCodes = map[ebiten.Key]scanCode{
	ebiten.KeyEscape: {1, false}, ebiten.KeyDigit1: {2, false}, ebiten.KeyDigit2: {3, false},
	ebiten.KeyDigit3: {4, false}, ebiten.KeyDigit4: {5, false}, ebiten.KeyDigit5: {6, false},
	ebiten.KeyDigit6: {7, false}, ebiten.KeyDigit7: {8, false}, ebiten.KeyDigit8: {9, false},
	ebiten.KeyDigit9: {10, false}, ebiten.KeyDigit0: {11, false}, ebiten.KeyMinus: {12, false},
	ebiten.KeyEqual: {13, false}, ebiten.KeyBackspace: {14, false}, ebiten.KeyTab: {15, false},
	ebiten.KeyQ: {16, false}, ebiten.KeyW: {17, false}, ebiten.KeyE: {18, false},
	ebiten.KeyR: {19, false}, ebiten.KeyT: {20, false}, ebiten.KeyY: {21, false},
	ebiten.KeyU: {22, false}, ebiten.KeyI: {23, false}, ebiten.KeyO: {24, false},
	ebiten.KeyP: {25, false}, ebiten.KeyBracketLeft: {26, false}, ebiten.KeyBracketRight: {27, false},
	ebiten.KeyEnter: {28, false}, ebiten.KeyControlLeft: {29, false}, ebiten.KeyA: {30, false},
	ebiten.KeyS: {31, false}, ebiten.KeyD: {32, false}, ebiten.KeyF: {33, false},
	ebiten.KeyG: {34, false}, ebiten.KeyH: {35, false}, ebiten.KeyJ: {36, false},
	ebiten.KeyK: {37, false}, ebiten.KeyL: {38, false}, ebiten.KeySemicolon: {39, false},
	ebiten.KeyQuote: {40, false}, ebiten.KeyBackquote: {41, false}, ebiten.KeyShiftLeft: {42, false},
	ebiten.KeyBackslash: {43, false}, ebiten.KeyZ: {44, false}, ebiten.KeyX: {45, false},
	ebiten.KeyC: {46, false}, ebiten.KeyV: {47, false}, ebiten.KeyB: {48, false},
	ebiten.KeyN: {49, false}, ebiten.KeyM: {50, false}, ebiten.KeyComma: {51, false},
	ebiten.KeyPeriod: {52, false}, ebiten.KeySlash: {53, false}, ebiten.KeyShiftRight: {54, false},
	ebiten.KeyNumpadMultiply: {55, false}, ebiten.KeyAltLeft: {56, false}, ebiten.KeySpace: {57, false},
	ebiten.KeyCapsLock: {58, false}, ebiten.KeyF1: {59, false}, ebiten.KeyF2: {60, false},
	ebiten.KeyF3: {61, false}, ebiten.KeyF4: {62, false}, ebiten.KeyF5: {63, false},
	ebiten.KeyF6: {64, false}, ebiten.KeyF7: {65, false}, ebiten.KeyF8: {66, false},
	ebiten.KeyF9: {67, false}, ebiten.KeyF10: {68, false}, ebiten.KeyNumLock: {69, false},
	ebiten.KeyScrollLock: {70, false}, ebiten.KeyNumpad7: {71, false}, ebiten.KeyNumpad8: {72, false},
	ebiten.KeyNumpad9: {73, false}, ebiten.KeyNumpadSubtract: {74, false}, ebiten.KeyNumpad4: {75, false},
	ebiten.KeyNumpad5: {76, false}, ebiten.KeyNumpad6: {77, false}, ebiten.KeyNumpadAdd: {78, false},
	ebiten.KeyNumpad1: {79, false}, ebiten.KeyNumpad2: {80, false}, ebiten.KeyNumpad3: {81, false},
	ebiten.KeyNumpad0: {82, false}, ebiten.KeyNumpadDecimal: {83, false}, ebiten.KeyF11: {87, false},
	ebiten.KeyF12: {88, false},
	ebiten.KeyNumpadEnter: {28, true}, ebiten.KeyControlRight: {29, true},
	ebiten.KeyNumpadDivide: {53, true}, ebiten.KeyAltRight: {56, true},
	ebiten.KeyHome: {71, true}, ebiten.KeyArrowUp: {72, true}, ebiten.KeyPageUp: {73, true},
	ebiten.KeyArrowLeft: {75, true}, ebiten.KeyArrowRight: {77, true}, ebiten.KeyEnd: {79, true},
	ebiten.KeyArrowDown: {80, true}, ebiten.KeyPageDown: {81, true}, ebiten.KeyInsert: {82, true},
	ebiten.KeyDelete: {83, true},
}

// maxScanEvents bounds the queue if the game stops reading.
const maxScanEvents = 64

var pressedScratch, releasedScratch []ebiten.Key

func pollScanEvents() {
	pressedScratch = inpututil.AppendJustPressedKeys(pressedScratch[:0])
	releasedScratch = inpututil.AppendJustReleasedKeys(releasedScratch[:0])
	for _, k := range releasedScratch {
		if c, ok := scanCodes[k]; ok {
			ScanEvents = append(ScanEvents, ScanEvent{c.scan, c.extended, true})
		}
	}
	for _, k := range pressedScratch {
		if c, ok := scanCodes[k]; ok {
			ScanEvents = append(ScanEvents, ScanEvent{c.scan, c.extended, false})
		}
	}
	if len(ScanEvents) > maxScanEvents {
		ScanEvents = ScanEvents[len(ScanEvents)-maxScanEvents:]
	}
}
