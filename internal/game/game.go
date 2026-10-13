// Package game is Lands of Lore itself: start-up, menus and play, on top of the library
// packages (data, gfx, text, input, clock).
package game

import (
	"errors"
	"fmt"
	"time"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/config"
	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
	"github.com/memmaker/lands-of-lore-1-go/internal/text"
)

// Game is the state the original kept in globals.
type Game struct {
	Files   *data.Files
	Screen  *gfx.Screen
	Text    *text.Printer
	Config  config.Config
	Palette gfx.Palette // the working palette (C: g_adea)

	SystemFont *gfx.Font // FONT9P.FNT (C: g_ae4f, then g_7fb8)
	SmallFont  *gfx.Font // FONT6P.FNT (C: g_7fbc)

	strings      []byte // the language's LANDS.ENG, FRE or GER (C: g_4d6c)
	levelStrings []byte // the current level's strings (C: g_4d68)
	menu         menuState

	itemIcons []byte   // ITEMICN.SHP (C: in page 3)
	cursors   [][]byte // item icons, which are also the cursors (C: g_7d82)

	items     [maxItems]Item // (C: f_2803_1cfd)
	itemTypes []ItemType     // (C: g_7fb2)

	levelScript *emc.Script // (C: f_2803_3611)
	itemScript  *emc.Script // ITEM.INF (C: f_2803_35fd)
	runningOnce bool        // a one-off script is running (C: g_4bf6)

	itemShapes   [][]byte // (C: g_7c92)
	gameShapes   [][]byte // (C: f_2803_00a6)
	thrownShapes [][]byte // (C: g_4e54)
	iceShapes    [][]byte // (C: f_27b1_0000)
	fireShapes   [][]byte // (C: g_088e)
	shadowTable  []byte   // (C: f_2803_030a)
	tintTable    []byte   // (C: f_2803_020a)
	textReady    bool     // (C: g_20d9)

	heldItem int  // the item on the cursor, 0 for none (C: g_8048)
	busy     bool // the busy cursor is up (C: g_806a bit 9)
	busyIcon int  // which icon it shows (C: g_ab1a)
}

// errSetup is main's message when LANDS.CFG is missing or damaged (C: g_18fc, whose
// French and German lines follow the English one).
var errSetup = errors.New("The setup program must be run first.")

// New opens the game folder and reads the configuration (C: main up to the video set-up;
// the memory, disk space and EMS checks have nothing to check here).
func New(dir string) (*Game, error) {
	files, err := data.Open(dir)
	if err != nil {
		return nil, err
	}
	g := &Game{Files: files, Screen: gfx.NewScreen(), menu: menuState{mouseHovers: true}}
	raw, err := files.Read(config.FileName)
	if err != nil {
		return nil, errSetup
	}
	cfg, ok := config.Decode(raw)
	if !ok {
		return nil, errSetup
	}
	if !cfg.VideoSupported() {
		return nil, fmt.Errorf("video mode %d is not supported", cfg.Video)
	}
	g.Config = cfg
	files.SetLanguage(cfg.Language)
	g.Text = text.New(g.Screen)
	return g, nil
}

// Run is the program from the video set-up on.
func (g *Game) Run() error {
	host.ShowFrame = g.Screen.Frame
	clock.Reset()
	input.Init(g.Screen)
	if err := g.setUpVideo(); err != nil {
		return err
	}
	input.SetNoMouse(false) // C: main's g_2f43 = 0
	return g.start()
}

// systemFont is the font the video set-up loads (C: main's ovr_2df7_0020 argument).
const systemFont = "FONT9P.FNT"

// setUpVideo clears the screen, loads the system font and sets an all-black palette
// (C: ovr_2df7_0020 in MCGA mode). The original also sampled the retrace bit's polarity
// (ovr_2de6_0020) for its retrace waits, which here are host ticks.
func (g *Game) setUpVideo() error {
	g.Screen.ClearPage(g.Screen.Current)
	font, err := g.loadFont(systemFont)
	if err != nil {
		return err
	}
	g.SystemFont = font
	g.Text.SetFont(font)
	g.Palette = gfx.Palette{}
	g.Screen.SetPalette(g.Palette[:])
	random.Seed(int(time.Now().Unix())) // C: srand(time(0))
	g.Text.SelectWindow(0)
	return nil
}

// A font file starts with its length and then this version word (00 05).
const (
	fontVersionAt = 2
	fontVersion   = 5
)

// loadFont reads a font file and checks its version (C: ovr_2df1_0020).
func (g *Game) loadFont(name string) (*gfx.Font, error) {
	b, err := g.Files.Read(name)
	if err != nil || len(b) <= fontVersionAt+1 || b[fontVersionAt] != 0 || b[fontVersionAt+1] != fontVersion {
		return nil, fmt.Errorf("unable to load font %s", name)
	}
	return gfx.NewFont(b), nil
}
