package game

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
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
	newGameSteps = 20000
	returnEvery  = 300
	scanReturn   = 28
	scanUp       = 72
	scanLeft     = 75
	scanDown     = 80
	scanRight    = 77
	scanHome     = 71
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

func TestNewGame(t *testing.T) {
	g, err := New(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	var runErr error
	host.StartHeadless(func() { runErr = g.Run() })
	steps(300)
	host.PushScan(host.ScanEvent{Scan: scanReturn})
	host.PushScan(host.ScanEvent{Scan: scanReturn, Released: true})
	// Return now and then gets past the <MORE> prompts; the game may run on for good.
	for i := 0; i < newGameSteps && steps(1); i++ {
		if i%returnEvery == 0 {
			host.PushScan(host.ScanEvent{Scan: scanReturn})
			host.PushScan(host.ScanEvent{Scan: scanReturn, Released: true})
		}
	}
	t.Logf("party: %q, items %v; ended with %v", g.party[0].Name, g.party[0].Items, runErr)
	savePNG(t, host.ShowFrame(), "game.png")
	// Walk: the arrow keys turn and step.
	for _, k := range []struct {
		scan byte
		name string
	}{{scanLeft, "strafe.png"}, {scanUp, "bump.png"}, {scanRight, "strafe2.png"}, {scanHome, "turn.png"}, {scanUp, "walk.png"}, {scanDown, "back.png"}} {
		block, dir := g.currentBlock, g.direction
		host.PushScan(host.ScanEvent{Scan: k.scan, Extended: true})
		host.PushScan(host.ScanEvent{Scan: k.scan, Extended: true, Released: true})
		steps(100)
		t.Logf("%s: block %d dir %d -> block %d dir %d; %v", k.name, block, dir, g.currentBlock, g.direction, runErr)
		savePNG(t, host.ShowFrame(), k.name)
	}
	// The character screen: a click on the first portrait opens it, its exit closes it.
	click := func(x, y int, name string) {
		host.MouseX, host.MouseY = x, y
		steps(5)
		host.MouseLeft = true
		steps(10)
		host.MouseLeft = false
		steps(60)
		t.Logf("%s: frozen %v, selected %d; %v", name, g.screenFrozen, g.selectedCharacter, runErr)
		savePNG(t, host.ShowFrame(), name)
	}
	click(g.portraitX[0]+30, 160, "character.png")
	click(viewLeft+160+20, 103, "character_exit.png")
	if g.party[0].MaxHP > 0 {
		if !g.audio.Loaded(sound.Music) || g.snd.file < 0 {
			t.Errorf("no music file loaded (file %d, track %d)", g.snd.file, g.musicTrack)
		}
		g.party[0].HP--
		click(43, 189, "rest.png")
		steps(300)
		if g.party[0].HP != g.party[0].MaxHP {
			t.Errorf("rested to %d of %d hit points", g.party[0].HP, g.party[0].MaxHP)
		}
		savePNG(t, host.ShowFrame(), "rested.png")
		g.setFlag(flagHasAutomap, true)
		g.revealAround(g.currentBlock)
		click(300, 40, "automap.png")
		steps(200)
		savePNG(t, host.ShowFrame(), "automap_arrow.png")
		click(295, 185, "automap_exit.png")
		steps(100)
		if g.scriptErr != nil || g.automap.done != true {
			t.Errorf("automap: done %v, %v", g.automap.done, g.scriptErr)
		}
		savePNG(t, host.ShowFrame(), "automap_closed.png")
	}
	checkGameMenu(t, g, click)
	checkSaveRoundTrip(t, g)
}

// checkSaveRoundTrip saves the running game, reads the save's records back into a
// fresh game and saves again: both must be the same bytes. The level kept on saving
// must restore to the same walls.
func checkSaveRoundTrip(t *testing.T, g *Game) {
	if g.level == 0 {
		t.Log("no game running: save not checked")
		return
	}
	store := MemoryStore{}
	g.Saves = store
	blocks := g.lvl.blocks
	if err := g.saveGame("_SAVE000.DAT", "test"); err != nil {
		t.Fatal(err)
	}
	b, _ := store.Read("_SAVE000.DAT")
	want := saveDescriptionSize + saveMagicSize + saveVersionSize + partyRecordSize + globalsSize +
		itemTableSize + savedLevels*(levelLengthSize+levelRecordSize)
	if len(b) != want {
		t.Fatalf("save is %d bytes, want %d", len(b), want)
	}
	var h Game
	r := reader{b: b, at: saveDescriptionSize + saveMagicSize + saveVersionSize}
	for i := range h.party {
		h.party[i] = readCharacter(&r)
	}
	h.readGlobals(&r)
	for i := range h.items {
		h.items[i] = readItem(&r)
	}
	var w record
	w.b = append(w.b, b[:saveDescriptionSize+saveMagicSize+saveVersionSize]...)
	for i := range h.party {
		writeCharacter(&w, &h.party[i])
	}
	h.writeGlobals(&w)
	for i := range h.items {
		writeItem(&w, &h.items[i])
	}
	if n := len(w.b); !bytes.Equal(w.b, b[:n]) {
		for i := range w.b {
			if w.b[i] != b[i] {
				t.Fatalf("records differ at byte %d", i)
			}
		}
	}
	if h.level != g.level || h.currentBlock != g.currentBlock || h.party[0].Name != g.party[0].Name {
		t.Fatalf("read back level %d block %d %q", h.level, h.currentBlock, h.party[0].Name)
	}
	g.restoreLevelState(g.level)
	for i := range blocks {
		if blocks[i].Walls != g.lvl.blocks[i].Walls || blocks[i].Flags != g.lvl.blocks[i].Flags {
			t.Fatalf("block %d: %v %x, restored %v %x", i, blocks[i].Walls, blocks[i].Flags, g.lvl.blocks[i].Walls, g.lvl.blocks[i].Flags)
		}
	}
	t.Logf("save: %d bytes, level file %d bytes", len(b), len(g.levelFiles[g.level]))
}

// checkGameMenu opens the game menu, saves under a typed name and loads that save.
func checkGameMenu(t *testing.T, g *Game, click func(x, y int, name string)) {
	if g.level == 0 {
		return
	}
	store := MemoryStore{}
	g.Saves = store
	click(20, 188, "menu.png")
	click(160, 72, "menu_save.png")
	click(150, 76, "menu_name.png")
	for _, scan := range []byte{20, 18, 31, 20} { // t e s t
		host.PushScan(host.ScanEvent{Scan: scan})
		host.PushScan(host.ScanEvent{Scan: scan, Released: true})
		steps(2)
	}
	steps(30)
	savePNG(t, host.ShowFrame(), "menu_typed.png")
	host.PushScan(host.ScanEvent{Scan: scanReturn})
	host.PushScan(host.ScanEvent{Scan: scanReturn, Released: true})
	steps(60)
	b, err := store.Read("_SAVE000.DAT")
	if err != nil {
		t.Fatalf("no save written; menu flags %d", g.updateFlags)
	}
	if desc := (&reader{b: b}).str(saveDescriptionSize); desc != "test" {
		t.Fatalf("save named %q", desc)
	}
	savePNG(t, host.ShowFrame(), "menu_saved.png")
	block := g.currentBlock
	click(20, 188, "menu2.png")
	click(160, 55, "menu_load.png")
	click(150, 76, "menu_loaded.png")
	if g.currentBlock != block || g.updateFlags&updateGameMenu != 0 {
		t.Fatalf("after loading: block %d (was %d), menu flags %d", g.currentBlock, block, g.updateFlags)
	}
}
