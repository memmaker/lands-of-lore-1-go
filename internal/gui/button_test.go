package gui

import (
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

func TestClickFires(t *testing.T) {
	s := gfx.NewScreen()
	s.SetPage(2)
	g := New(s)
	fired := 0
	b := &Button{ID: 7, X: 150, Y: 90, W: 20, H: 20, Flags: EventLeftRelease | FlagNeedsInside,
		Func: func(*Button) bool { fired++; return false }}
	list := g.Append(nil, b)
	input.Init(s)
	g.Process(list, 0)
	// a press then a release over the button (the cursor starts at 160,100)
	input.Put(input.KeyMouseLeft)
	if r := g.ProcessKeys(list); r != 0 {
		t.Fatalf("press returned %d", r)
	}
	input.Put(input.KeyMouseLeft | input.Released)
	if r := g.ProcessKeys(list); r != 7|keyFromCharacter {
		t.Fatalf("release returned %d", r)
	}
	if fired != 1 {
		t.Fatalf("fired %d times", fired)
	}
}

func TestKeyFires(t *testing.T) {
	s := gfx.NewScreen()
	s.SetPage(2)
	g := New(s)
	b := &Button{ID: 3, Key1: 'x', X: 0, Y: 0, W: 5, H: 5, Flags: EventLeftPress}
	list := g.Append(nil, b)
	if r := g.Process(list, 'x'); r != 3|keyFromCharacter {
		t.Fatalf("key returned %d", r)
	}
	if r := g.Process(list, 'y'); r != 'y' {
		t.Fatalf("other key returned %d", r)
	}
}
