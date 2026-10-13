package data

import (
	"strings"
	"testing"
)

func TestDecodeCPS(t *testing.T) {
	f, err := Open(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.Read("PLAYFLD.CPS")
	if err != nil {
		t.Fatal(err)
	}
	img, err := DecodeCPS(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(img.Pixels) != 320*200 || len(img.Palette) != PaletteSize {
		t.Fatalf("%d pixels, %d palette bytes", len(img.Pixels), len(img.Palette))
	}
	used := map[byte]bool{}
	for _, p := range img.Pixels {
		used[p] = true
	}
	if len(used) < 16 {
		t.Fatalf("only %d colours: not decoded", len(used))
	}
}

func TestApplyDelta(t *testing.T) {
	dst := make([]byte, 12) // 3 wide inside rows of 4
	delta := []byte{
		2, 1, 2, // XOR two bytes
		129,     // skip one: end of the first row
		0, 3, 7, // XOR 7 onto three bytes, wrapping to the third row
		128, 0, 0, // end
	}
	ApplyDelta(dst, delta, 3, 4, false)
	want := []byte{1, 2, 0, 0, 7, 7, 7, 0, 0, 0, 0, 0}
	if string(dst) != string(want) {
		t.Fatalf("got %v, want %v", dst, want)
	}
}

func TestDecodeWSA(t *testing.T) {
	f, err := Open(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	tested := 0
	for _, e := range f.entries {
		if !strings.HasSuffix(e.name, ".WSA") || e.size == 0 {
			continue
		}
		b, err := f.Read(e.name)
		if err != nil {
			continue
		}
		a, err := DecodeWSA(b)
		if err != nil {
			t.Fatalf("%s: %v", e.name, err)
		}
		buf := make([]byte, a.Width*a.Height)
		a.Seek(0, buf, a.Width, false)
		first := string(buf)
		for i := range a.Frames {
			if !a.Seek(i, buf, a.Width, false) {
				t.Fatalf("%s: frame %d", e.name, i)
			}
		}
		if a.Frames > 1 && !a.Seek(0, buf, a.Width, false) {
			t.Fatalf("%s: back to frame 0", e.name)
		}
		if a.loops && string(buf) != first {
			t.Errorf("%s: frame 0 differs after a full loop", e.name)
		}
		tested++
	}
	if tested == 0 {
		t.Skip("no WSA files")
	}
	t.Logf("%d animations played", tested)
}
