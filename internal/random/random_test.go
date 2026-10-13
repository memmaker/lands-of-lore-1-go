package random

import "testing"

func TestRand(t *testing.T) {
	Seed(1)
	// Borland's first results from seed 1
	want := []int{346, 130, 10982, 1090}
	for i, w := range want {
		if r := Rand(); r != w {
			t.Fatalf("rand %d = %d, want %d", i, r, w)
		}
	}
}

func TestRange(t *testing.T) {
	for range 1000 {
		if r := Range(9, 3); r < 3 || r > 9 {
			t.Fatalf("Range(9, 3) = %d", r)
		}
	}
}

func TestByteVaries(t *testing.T) {
	seen := map[int]bool{}
	for range 1000 {
		seen[Byte()] = true
	}
	if len(seen) < 200 {
		t.Fatalf("only %d distinct values", len(seen))
	}
}
