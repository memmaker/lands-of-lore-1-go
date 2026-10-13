package text

import (
	"os"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

func newPrinter(t *testing.T) *Printer {
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
	b, err := f.Read("FONT9P.FNT")
	if err != nil {
		t.Fatal(err)
	}
	s := gfx.NewScreen()
	s.SetPage(2) // off screen: no cursor bookkeeping
	p := New(s)
	p.SetFont(gfx.NewFont(b))
	s.SetWindow(1, 1, 10, 10, 50)
	s.Windows[1].Foreground, s.Windows[1].Background = 15, 0
	p.SelectAndClear(1)
	return p
}

func TestWrap(t *testing.T) {
	p := newPrinter(t)
	p.Printf("%s and %d more words that wrap", "Some text", 42)
	if p.Row == 0 {
		t.Fatalf("no wrap: row %d column %d", p.Row, p.Column)
	}
	if p.Column > p.PixelWidth {
		t.Fatalf("column %d past width %d", p.Column, p.PixelWidth)
	}
	if p.Screen.Windows[1].Row != p.Row {
		t.Fatal("window row not stored")
	}
}

func TestScroll(t *testing.T) {
	p := newPrinter(t)
	for range 20 {
		p.Printf("line\r")
	}
	if p.Row != p.Rows {
		t.Fatalf("row %d, rows %d", p.Row, p.Rows)
	}
}
