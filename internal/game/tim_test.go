package game

import (
	"strings"
	"testing"
)

// TestLoadAllTIMs loads every TIM of the game and checks its function table.
func TestLoadAllTIMs(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, name := range g.Files.Names() {
		if !strings.HasSuffix(strings.ToUpper(name), ".TIM") {
			continue
		}
		tim, err := g.loadTIM(name)
		if err != nil || tim == nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if s := tim.funcs[0].start; s <= 0 || s >= len(tim.avtl) {
			t.Errorf("%s: function 0 at %d of %d words", name, s, len(tim.avtl))
		}
		n++
	}
	if n == 0 {
		t.Fatal("no TIM files")
	}
	t.Logf("%d TIM files", n)
}
