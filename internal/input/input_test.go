package input

import (
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

func press(scan byte)   { scanned(host.ScanEvent{Scan: scan}) }
func release(scan byte) { scanned(host.ScanEvent{Scan: scan, Released: true}) }

const (
	scanA      = 30
	scanShiftL = 42
	scanCtrlL  = 29
	scanF2     = 60
	scanEsc    = 1
)

func reset() { in = state{flags: initialFlags, screen: gfx.NewScreen()} }

func TestTyping(t *testing.T) {
	reset()
	press(scanA)
	release(scanA)
	press(scanShiftL)
	press(scanA)
	release(scanA)
	release(scanShiftL)
	press(scanEsc)
	for _, want := range []int{'a', 'A', escapeChar} {
		if c := Check(); c != want {
			t.Fatalf("Check = %d, want %d", c, want)
		}
		if c := WaitChar(); c != want {
			t.Fatalf("WaitChar = %d, want %d", c, want)
		}
	}
	if Peek() != 0 {
		t.Fatalf("queue not empty: %d", Peek())
	}
}

func TestModifiers(t *testing.T) {
	reset()
	press(scanCtrlL)
	press(scanA)
	if c := WaitChar(); c != 1 {
		t.Fatalf("Ctrl+A = %d", c)
	}
	if !DownWith(31 | Ctrl) {
		t.Fatal("Ctrl+A not held")
	}
	press(scanF2)
	if k := Get(); ASCII(k)&lowByte != functionCtrl-1 {
		t.Fatalf("Ctrl+F2 = %d", ASCII(k))
	}
}

func TestHeldKeyDoesNotRepeatWithoutFlag(t *testing.T) {
	reset()
	ClearFlags(FlagRepeat)
	press(scanA)
	press(scanA)
	Get()
	if Peek() != 0 {
		t.Fatal("held key queued twice")
	}
}

func TestMouseButtons(t *testing.T) {
	reset()
	mouseEvent(10, 20, ButtonLeft) // pressed at the old position
	mouseEvent(10, 20, 0)
	if k := Get(); k != KeyMouseLeft {
		t.Fatalf("got %d", k)
	}
	if x, y := Click(); x != 160 || y != 100 {
		t.Fatalf("click at %d,%d", x, y)
	}
	if k := Get(); k != KeyMouseLeft|Released {
		t.Fatalf("got %d", k)
	}
	if x, y := Click(); x != 10 || y != 20 {
		t.Fatalf("release at %d,%d", x, y)
	}
}
