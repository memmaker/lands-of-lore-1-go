package game

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

func gameDir(t *testing.T) string {
	dir := os.Getenv("LOLDATA")
	if dir == "" {
		dir = "../../game"
	}
	if _, err := os.Stat(dir + "/FILEDATA.FDT"); err != nil {
		t.Skip("no game files; set LOLDATA")
	}
	return dir
}

// savePNG writes the frame to LOLSHOT if set, to look at.
func savePNG(t *testing.T, f *host.Frame, name string) {
	dir := os.Getenv("LOLSHOT")
	if dir == "" {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for i, c := range f.Pixels[:320*200] {
		p := f.Palette[int(c)*3:]
		img.Set(i%320, i/320, color.RGBA{p[0] << 2, p[1] << 2, p[2] << 2, 255})
	}
	out, err := os.Create(dir + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	png.Encode(out, img)
}

const (
	scanReturn = 28
	scanUp     = 72
)

func steps(n int) bool {
	for range n {
		if !host.Step() {
			return false
		}
	}
	return true
}

func TestTitleAndMenu(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	var runErr error
	host.StartHeadless(func() { runErr = g.Run() })
	if !steps(300) {
		t.Fatalf("ended early: %v", runErr)
	}
	for _, id := range mainMenuChoices[0] {
		t.Logf("%#x %q", id, g.text(id))
	}
	if g.text(choiceExit) == "" {
		t.Fatal("no menu text")
	}
	savePNG(t, host.ShowFrame(), "menu.png")
	// Up wraps to the last entry, Exit; Return chooses it.
	host.PushScan(host.ScanEvent{Scan: scanUp, Extended: true})
	host.PushScan(host.ScanEvent{Scan: scanUp, Extended: true, Released: true})
	steps(5)
	host.PushScan(host.ScanEvent{Scan: scanReturn})
	host.PushScan(host.ScanEvent{Scan: scanReturn, Released: true})
	if steps(200) {
		t.Fatal("still running after choosing Exit")
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func TestLoadGameData(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.loadLanguageStrings(); err != nil {
		t.Fatal(err)
	}
	if err := g.loadGameData(); err != nil {
		t.Fatal(err)
	}
	if len(g.itemTypes) == 0 || g.items[1].Flags&itemFree == 0 {
		t.Fatalf("%d item types", len(g.itemTypes))
	}
	t.Logf("%d item types, %d icons, %d item shapes, %d game shapes; type 1 %q %+v",
		len(g.itemTypes), len(g.cursors), len(g.itemShapes), len(g.gameShapes), g.text(g.itemTypes[1].Name), g.itemTypes[1])
}
