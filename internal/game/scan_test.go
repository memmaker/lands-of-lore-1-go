package game

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
)

// TestScanOpcodes lists the opcodes the level scripts use and which are still missing
// (set LOLSCAN=1).
func TestScanOpcodes(t *testing.T) {
	if os.Getenv("LOLSCAN") == "" {
		t.Skip("set LOLSCAN=1")
	}
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	ops := g.opcodes()
	missing := map[int][]string{}
	last := 29
	if os.Getenv("LOLSCAN") == "1" {
		last = 1
	}
	for level := 1; level <= last; level++ {
		for _, ext := range []string{"ini", "inf"} {
			name := fmt.Sprintf("level%d.%s", level, ext)
			file, err := g.Files.Read(name)
			if err != nil {
				continue
			}
			s, err := emc.Load(file, ops)
			if err != nil {
				t.Fatal(name, err)
			}
			for _, op := range s.CalledOpcodes() {
				if ops[op] == nil {
					missing[op] = append(missing[op], name)
				}
			}
		}
	}
	keys := make([]int, 0, len(missing))
	for k := range missing {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		t.Logf("opcode %d: %d files, first %s", k, len(missing[k]), missing[k][0])
	}
}
