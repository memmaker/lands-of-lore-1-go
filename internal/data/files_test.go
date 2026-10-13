package data

import (
	"os"
	"testing"
)

// gameDir is where the tests find the original files (LOLDATA, else ../../game).
func gameDir(t *testing.T) string {
	dir := os.Getenv("LOLDATA")
	if dir == "" {
		dir = "../../game"
	}
	if _, err := os.Stat(dir + "/" + indexFile); err != nil {
		t.Skip("no game files; set LOLDATA")
	}
	return dir
}

func TestReadFromArchive(t *testing.T) {
	f, err := Open(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.Read("litepal2.col")
	if err != nil || len(b) != 768 {
		t.Fatalf("LITEPAL2.COL: %d bytes, %v", len(b), err)
	}
	cps, err := f.Read("PLAYFLD.CPS")
	if err != nil {
		t.Fatal(err)
	}
	if !f.Exists("PLAYFLD.CPS") || f.Exists("NOSUCH.CPS") {
		t.Fatal("Exists")
	}
	t.Logf("PLAYFLD.CPS %d bytes, header % x", len(cps), cps[:10])
}

// The shared STARTUP.PAK holds the game's start-up files; the English folder's one of the
// same name is the intro's, and its own files are found there.
func TestSharedAndLanguageFiles(t *testing.T) {
	f, err := Open(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"GAMESHP.SHP", "TITLE.CPS", "ITEM.INF", "HISTORY.PAK"} {
		if b, err := f.Read(name); err != nil || len(b) == 0 {
			t.Errorf("%s: %d bytes, %v", name, len(b), err)
		}
	}
}
