package gfx

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
	if _, err := os.Stat(dir + "/FILEDATA.FDT"); err != nil {
		t.Skip("no game files; set LOLDATA")
	}
	f, err := data.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func read(t *testing.T, f *data.Files, name string) []byte {
	b, err := f.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func countSet(page []byte) int {
	n := 0
	for _, c := range page {
		if c != 0 {
			n++
		}
	}
	return n
}

func TestDrawText(t *testing.T) {
	f := gameFiles(t)
	s := NewScreen()
	for _, name := range []string{"FONT9P.FNT", "NEW6P.FNT", "FONT6PN.FNT"} {
		font := NewFont(read(t, f, name))
		if font == nil || font.Height() == 0 {
			t.Fatalf("%s: no font", name)
		}
		s.SetFont(font)
		s.ClearPage(0)
		s.DrawText("Lands of Lore", 10, 10, 15, 0)
		if countSet(s.Page(0)) == 0 {
			t.Fatalf("%s: nothing drawn", name)
		}
		if w := s.TextWidth("Lands of Lore"); w <= 0 || w > Width {
			t.Fatalf("%s: width %d", name, w)
		}
	}
}

func TestDrawShapes(t *testing.T) {
	f := gameFiles(t)
	s := NewScreen()
	// shape files come packed like CPS pictures
	img, err := data.DecodeCPS(read(t, f, "AUTOBUT.SHP"))
	if err != nil {
		t.Fatal(err)
	}
	file := img.Pixels
	n := ShapeCount(file)
	if n == 0 {
		t.Fatal("no shapes")
	}
	drawn := 0
	for i := range n {
		shape := ShapeFrame(file, i)
		if shape == nil {
			continue
		}
		s.ClearPage(0)
		s.DrawShape(0, shape, 0, 0, 0, 0, ShapeOptions{})
		if countSet(s.Page(0)) > 0 {
			drawn++
		}
		px, w, h := shapePixels(shape)
		if len(px) != w*h || w != ShapeWidth(shape) {
			t.Fatalf("shape %d: %d pixels for %dx%d", i, len(px), w, h)
		}
	}
	if drawn < n/2 {
		t.Fatalf("only %d of %d shapes drew anything", drawn, n)
	}
}

func TestMouseHiding(t *testing.T) {
	s := NewScreen()
	if s.Frame().PointerShown {
		t.Fatal("cursor shown at start")
	}
	s.SetMouseShape(0, 0, []byte{shapeRaw, 0, 1, 1, 0, 1, 1, 0, 1, 0, 7})
	s.ShowMouse()
	if !s.Frame().PointerShown {
		t.Fatal("not shown after ShowMouse")
	}
	s.HideMouseIn(150, 90, 170, 110)
	if s.Frame().PointerShown {
		t.Fatal("shown inside the drawing region")
	}
	s.ShowMouseIn()
	if !s.Frame().PointerShown {
		t.Fatal("not shown after the region ended")
	}
	s.MouseMoved(0, 0)
	s.HideMouseIn(150, 90, 170, 110)
	if !s.Frame().PointerShown {
		t.Fatal("hidden outside the region")
	}
	s.MouseMoved(160, 100)
	if s.Frame().PointerShown {
		t.Fatal("shown after moving into the region")
	}
	s.ShowMouseIn()
	if !s.Frame().PointerShown || s.Frame().Pointer[0] != 7 {
		t.Fatal("cursor not restored")
	}
}
