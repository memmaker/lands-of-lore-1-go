// Command lol runs Lands of Lore: The Throne of Chaos (MAIN.EXE of the DOS CD version 1.02) from
// the original data files, ported to Go.
//
//	lol [-data DIR]            play with the game files in DIR (default $GOPLAY_DATA, else ./game)
//	lol -import SRC -data DIR  copy the game files out of SRC (folder, zip, ISO/BIN/CUE, GOG
//	                           installer, loose files) into DIR and exit
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/game"
	"github.com/memmaker/lands-of-lore-1-go/internal/gameimport"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
)

// markerFiles prove a folder holds Lands of Lore.
var markerFiles = []string{"GENERAL.PAK", "STARTUP.PAK"}

// Permissions of the directories and files written.
const (
	dirMode  = 0o755
	fileMode = 0o644
)

// gameFile says whether name is one of the game's data files (the installer's and CD's
// programs, drivers, documents and pictures are not).
func gameFile(name string) bool {
	switch strings.ToUpper(filepath.Ext(name)) {
	case ".PAK", ".TLK", ".ADL", ".FDT", ".FNT", ".TAB", ".CFG", ".RST", ".TDS", ".LM":
		return true
	}
	return false
}

func main() {
	defaultDir := os.Getenv("GOPLAY_DATA")
	if defaultDir == "" {
		defaultDir = "game"
	}
	dir := flag.String("data", defaultDir, "directory with the original game files")
	source := flag.String("import", "", "import the game files from this folder/zip/ISO/installer into -data and exit")
	scale := flag.Int("scale", 3, "window scale")
	flag.Parse()

	if *source != "" {
		if err := importGame(*source, *dir); err != nil {
			fmt.Fprintln(os.Stderr, "import:", err)
			os.Exit(1)
		}
		return
	}
	if _, err := os.Stat(filepath.Join(*dir, markerFiles[0])); err != nil {
		fmt.Fprintf(os.Stderr, "no game files in %s; use -import SRC or -data DIR\n", *dir)
		os.Exit(1)
	}
	var gameErr error
	err := host.Run("Lands of Lore", *scale, func() { gameErr = run(*dir) })
	if err == nil {
		err = gameErr
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// importGame copies the game files found in source into dir.
func importGame(source, dir string) error {
	files, err := gameimport.Find(source, gameFile)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for name := range files {
		have[strings.ToUpper(filepath.Base(name))] = true
	}
	for _, marker := range markerFiles {
		if !have[marker] {
			return fmt.Errorf("no Lands of Lore files (%s) found in %s", marker, source)
		}
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}
	for name, contents := range files {
		// The CD keeps the files shared by all languages in DATA and each language's own
		// versions, some under the same names (STARTUP.PAK is the intro's there), in
		// DATA/ENG, FRE and GER; the language folders stay apart.
		target := dir
		if lang := strings.ToUpper(filepath.Base(filepath.Dir(name))); data.IsLanguageFolder(lang) {
			target = filepath.Join(dir, lang)
		}
		if err := os.MkdirAll(target, dirMode); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, strings.ToUpper(filepath.Base(name))), contents, fileMode); err != nil {
			return err
		}
	}
	return nil
}

// run is the game.
func run(dir string) error {
	g, err := game.New(dir)
	if err != nil {
		return err
	}
	return g.Run()
}
