package game

import (
	"bytes"
	"fmt"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"image"
	"image/color"
	"image/png"
	"os"
	"slices"
	"strings"
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
	if f.PointerShown { // the cursor, 8 times enlarged, as the host shows it
		const k = 8
		c := image.NewRGBA(image.Rect(0, 0, f.PointerWidth*k, f.PointerHeight*k))
		for i, v := range f.Pointer {
			if v == 0 {
				continue
			}
			p := f.Palette[int(v)*3:]
			for j := range k * k {
				c.Set(i%f.PointerWidth*k+j%k, i/f.PointerWidth*k+j/k, color.RGBA{p[0] << 2, p[1] << 2, p[2] << 2, 255})
			}
		}
		if cf, err := os.Create(dir + "/" + strings.TrimSuffix(name, ".png") + ".cursor.png"); err == nil {
			png.Encode(cf, c)
			cf.Close()
		}
	}
	// the palette indices too, to compare with the original's video memory
	os.WriteFile(dir+"/"+strings.TrimSuffix(name, ".png")+".raw", f.Pixels[:320*200], 0o644)
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
	scanShiftL   = 42
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
	g.SkipCharGen = true
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
	if g.party[0].MaxHP == 0 {
		t.Fatalf("no game started (ended with %v)", runErr)
	}
	savePNG(t, host.ShowFrame(), "game.png")
	// Walk: Shift and the arrow keys strafe (the original's plain Left and Right, as in the
	// DOSBox dumps), Home turns, Up and Down step.
	for _, k := range []struct {
		scan  byte
		shift bool
		name  string
	}{{scanLeft, true, "strafe.png"}, {scanUp, false, "bump.png"}, {scanRight, true, "strafe2.png"}, {scanHome, false, "turn.png"}, {scanUp, false, "walk.png"}, {scanDown, false, "back.png"}} {
		block, dir := g.currentBlock, g.direction
		if k.shift {
			host.PushScan(host.ScanEvent{Scan: scanShiftL})
		}
		host.PushScan(host.ScanEvent{Scan: k.scan, Extended: true})
		host.PushScan(host.ScanEvent{Scan: k.scan, Extended: true, Released: true})
		if k.shift {
			host.PushScan(host.ScanEvent{Scan: scanShiftL, Released: true})
		}
		steps(100)
		t.Logf("%s: block %d dir %d -> block %d dir %d; %v", k.name, block, dir, g.currentBlock, g.direction, runErr)
		if turned := g.direction != dir; turned != (k.scan == scanHome) {
			t.Errorf("%s: direction %d -> %d", k.name, dir, g.direction)
		}
		savePNG(t, host.ShowFrame(), k.name)
	}
	// Plain Left and Right turn (Right first: facing left, the guard starts talking).
	for _, k := range []struct {
		scan byte
		turn int
	}{{scanRight, 1}, {scanLeft, -1}} {
		block, dir := g.currentBlock, g.direction
		host.PushScan(host.ScanEvent{Scan: k.scan, Extended: true})
		host.PushScan(host.ScanEvent{Scan: k.scan, Extended: true, Released: true})
		steps(100)
		if g.currentBlock != block || g.direction != (dir+k.turn)&(directionCount-1) {
			t.Errorf("scan %d: block %d dir %d -> block %d dir %d", k.scan, block, dir, g.currentBlock, g.direction)
		}
	}
	// The port's layout: E and Q turn, D and A strafe back where the party stood; M opens
	// the automap and closes it again.
	for _, k := range []struct {
		scan    byte
		turn    int
		strafes bool
	}{{18, 1, false}, {16, -1, false}, {32, 0, true}, {30, 0, true}} { // E Q D A
		block, dir := g.currentBlock, g.direction
		host.PushScan(host.ScanEvent{Scan: k.scan})
		host.PushScan(host.ScanEvent{Scan: k.scan, Released: true})
		steps(100)
		if g.direction != (dir+k.turn)&(directionCount-1) || (k.turn != 0 && g.currentBlock != block) {
			t.Errorf("scan %d: block %d dir %d -> block %d dir %d", k.scan, block, dir, g.currentBlock, g.direction)
		}
		t.Logf("scan %d: block %d -> %d", k.scan, block, g.currentBlock)
	}
	const scanM = 50
	hadMap, decorations := g.flag(flagHasAutomap), len(g.lvl.decorations)
	g.setFlag(flagHasAutomap, true)
	host.PushScan(host.ScanEvent{Scan: scanM})
	host.PushScan(host.ScanEvent{Scan: scanM, Released: true})
	steps(100)
	savePNG(t, host.ShowFrame(), "automap_m.png")
	if !g.automap.open {
		t.Errorf("M did not open the automap")
	}
	host.PushScan(host.ScanEvent{Scan: scanM})
	host.PushScan(host.ScanEvent{Scan: scanM, Released: true})
	steps(100)
	if g.automap.open {
		t.Errorf("M did not close the automap")
	}
	if len(g.lvl.decorations) != decorations { // the automap once emptied them (the fountain went)
		t.Errorf("decorations %d after the automap, %d before", len(g.lvl.decorations), decorations)
	}
	g.setFlag(flagHasAutomap, hadMap)
	// A tooltip names the spell the pointer rests on in the list.
	hadMagic := g.flag(flagHasMagic)
	g.setFlag(flagHasMagic, true)
	g.drawGameScreen()
	g.buildButtons()
	if g.availableSpells[0] != noSpell {
		host.MouseX, host.MouseY = 34, 19
		steps(60)
		savePNG(t, host.ShowFrame(), "tooltip.png")
		if want := g.text(spells[g.availableSpells[0]].Name); !g.tooltip.shown || !strings.HasPrefix(g.tooltip.text, want) {
			t.Errorf("tooltip %q shown %v, want %q", g.tooltip.text, g.tooltip.shown, want)
		}
		host.MouseX, host.MouseY = 160, 60
		steps(5)
		if g.tooltip.shown {
			t.Errorf("tooltip %q still shown", g.tooltip.text)
		}
	}
	// One on the first inventory slot names its item.
	if it := g.inventory[g.inventoryStart]; it != 0 {
		host.MouseX, host.MouseY = inventorySlotX[0]+10, inventoryY+10
		steps(60)
		savePNG(t, host.ShowFrame(), "tooltip_item.png")
		if want := g.text(g.itemTypes[g.items[it].Type].Name); !g.tooltip.shown || !strings.HasPrefix(g.tooltip.text, want) {
			t.Errorf("item tooltip %q shown %v, want %q", g.tooltip.text, g.tooltip.shown, want)
		}
		host.MouseX, host.MouseY = 160, 60
		steps(5)
	}
	g.setFlag(flagHasMagic, hadMagic)
	g.drawGameScreen()
	g.buildButtons()
	checkJoin(t, g)
	checkRemovedMonsterRestore(t, g)
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
		checkAutomapNotes(t, g, click)
		click(295, 185, "automap_exit.png")
		steps(100)
		if g.scriptErr != nil || g.automap.done != true {
			t.Errorf("automap: done %v, %v", g.automap.done, g.scriptErr)
		}
		savePNG(t, host.ShowFrame(), "automap_closed.png")
		checkAutoWalk(t, g)
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
	notes := append([]automapNote(nil), g.notes...)
	g.notes = nil
	g.loadNotes("_SAVE000.DAT")
	if len(notes) == 0 || fmt.Sprint(g.notes) != fmt.Sprint(notes) {
		t.Errorf("notes saved %+v, read %+v", notes, g.notes)
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

// checkAutomapNotes adds a note on the party's cell, edits it and removes it again with
// a right click, then leaves a second one for the save check.
func checkAutomapNotes(t *testing.T, g *Game, click func(x, y int, name string)) {
	a := &g.automap
	x := a.left + (g.currentBlock%mapSize-g.automapLeft())*automapCellWidth + 3
	y := a.top + (g.currentBlock/mapSize-g.automapTop())*automapCellHeight + 3
	typeKeys := func(scans ...byte) {
		for _, sc := range scans {
			host.PushScan(host.ScanEvent{Scan: sc})
			host.PushScan(host.ScanEvent{Scan: sc, Released: true})
			steps(3)
		}
	}
	const scanW, scanE, scanL, scanBackspace = 17, 18, 38, 14
	click(x, y, "automap_prompt.png")
	typeKeys(scanW, scanE, scanL, scanL)
	savePNG(t, host.ShowFrame(), "automap_typing.png")
	typeKeys(scanReturn)
	steps(20)
	savePNG(t, host.ShowFrame(), "automap_note.png")
	if len(g.notes) != 1 || g.notes[0].label != "well" || g.notes[0].block != g.currentBlock {
		t.Fatalf("notes after adding: %+v", g.notes)
	}
	click(x, y, "automap_edit.png")
	typeKeys(scanBackspace, scanBackspace, scanReturn)
	steps(20)
	if len(g.notes) != 1 || g.notes[0].label != "we" {
		t.Fatalf("notes after editing: %+v", g.notes)
	}
	click(x, y, "automap_edit2.png") // an empty label removes the note
	typeKeys(scanBackspace, scanBackspace, scanReturn)
	steps(20)
	if len(g.notes) != 0 {
		t.Fatalf("notes after removing: %+v", g.notes)
	}
	click(x, y, "automap_prompt2.png")
	typeKeys(scanE, scanReturn)
	steps(20)
	if len(g.notes) != 1 {
		t.Fatalf("notes after the second: %+v", g.notes)
	}
}

// checkAutoWalk opens the automap with M, right clicks the farthest block the party can
// reach (without facing west, where the guard stops it to talk) and says yes: the map closes and the party walks there.
func checkAutoWalk(t *testing.T, g *Game) {
	from, to, far := g.currentBlock, -1, 0
	for b := range g.lvl.blocks {
		if p := g.findPath(from, b); len(p) > far && len(p) <= 4 && !slices.Contains(p, 3) { // facing west the guard talks
			to, far = b, len(p)
		}
	}
	if to < 0 {
		t.Log("no block to walk to")
		return
	}
	press := func(scan byte) {
		host.PushScan(host.ScanEvent{Scan: scan})
		host.PushScan(host.ScanEvent{Scan: scan, Released: true})
		steps(5)
	}
	const scanM, scanY = 50, 21
	press(scanM)
	steps(100)
	a := &g.automap
	host.MouseX = a.left + (to%mapSize-g.automapLeft())*automapCellWidth + 3
	host.MouseY = a.top + (to/mapSize-g.automapTop())*automapCellHeight + 3
	steps(5)
	host.MouseRight = true
	steps(10)
	host.MouseRight = false
	steps(20)
	savePNG(t, host.ShowFrame(), "automap_walk.png")
	press(scanY)
	steps(400)
	savePNG(t, host.ShowFrame(), "automap_walked.png")
	if a.open || g.currentBlock != to {
		t.Errorf("walk from %d to %d (%d steps): at %d, map open %v", from, to, far, g.currentBlock, a.open)
	}
}

// checkJoin lets Baccata join and leave: each time the portrait row must look as a fresh
// draw of the game screen has it, not keep the old layout's portraits.
func checkJoin(t *testing.T, g *Game) {
	const baccata, rowTop, rowBottom, portraitBoxW = 3, 143, 177, 66
	for _, id := range []int{baccata, -baccata} {
		st := &emc.State{}
		st.Stack[0] = id
		g.opCharacterJoinsOrLeaves(st)
		steps(5)
		got := slices.Clone(host.ShowFrame().Pixels[rowTop*320 : rowBottom*320])
		savePNG(t, host.ShowFrame(), fmt.Sprintf("join%d.png", id))
		g.drawGameScreen()
		steps(5)
		want := host.ShowFrame().Pixels[rowTop*320 : rowBottom*320]
		for i := range got { // the faces themselves blink and greet; the rest must match
			x, inPortrait := i%320, false
			for c := range g.partyCount() {
				inPortrait = inPortrait || x >= g.portraitX[c] && x < g.portraitX[c]+portraitBoxW
			}
			if !inPortrait && got[i] != want[i] {
				t.Errorf("character %d: the portrait row differs from a fresh draw at %d,%d", id, x, rowTop+i/320)
				break
			}
		}
	}
}

// checkRemovedMonsterRestore saves and restores the level with a removed monster in a
// free slot: the monster keeps thinking off the map, so it needs its type back.
func checkRemovedMonsterRestore(t *testing.T, g *Game) {
	i := len(g.monsters) - 1
	saved := g.monsters[i]
	g.monsters[i] = Monster{ID: i, Mode: monsterModeRemove}
	if err := g.saveLevelState(); err != nil {
		t.Fatal(err)
	}
	g.restoreLevelState(g.level)
	if g.monsters[i].Type == nil {
		t.Errorf("removed monster restored without a type")
	}
	g.moveMonsters(i & 1)
	g.monsters[i] = saved
}
