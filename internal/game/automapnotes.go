package game

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
)

// The player's notes on the automap, an addition of the port: a left click on a cell
// marks it with a letter and asks for a label (or edits the label already there), a
// right click removes the note. The cells show their letters, the legend the labels.
// The notes are kept in a file beside each saved game, which itself stays as the
// original wrote it.

// automapNote is a note on block of level.
type automapNote struct {
	level, block int
	label        string
}

const (
	maxNotesPerLevel = 26 // A to Z
	noteLabelLength  = 30
	noteFileExt      = ".not"
	noteMapRight     = automapSymbolX - 2 // clicks right of this are on the legend
	noteLegendBottom = 172                // the legend's rows end above the buttons
	noteLegendWidth  = gfx.Width - automapLegendX - 1

	// the label prompt over the bottom of the map
	notePromptX, notePromptY = 6, 178
	notePromptW, notePromptH = 222, 14
	notePromptText           = "Label: "
)

// The note colours, the nearest of the parchment's palette to these.
var (
	noteMarkRGB   = []byte{56, 6, 4}
	notePromptRGB = []byte{14, 9, 4}
	noteTextRGB   = []byte{63, 58, 44}
)

func (g *Game) noteColour(rgb []byte) int {
	return gfx.NearestColour(rgb, g.automap.palette[:], fadeTableSize, false)
}

// levelNotes are the notes of a level, in the order they were made.
func (g *Game) levelNotes(level int) []*automapNote {
	var ns []*automapNote
	for i := range g.notes {
		if g.notes[i].level == level {
			ns = append(ns, &g.notes[i])
		}
	}
	return ns
}

// noteAt is the note on a block of a level and its letter, nil if none.
func (g *Game) noteAt(level, block int) (*automapNote, byte) {
	for i, n := range g.levelNotes(level) {
		if n.block == block {
			return n, byte('A' + i)
		}
	}
	return nil, 0
}

// removeNote deletes the note on a block of a level.
func (g *Game) removeNote(level, block int) bool {
	for i, n := range g.notes {
		if n.level == level && n.block == block {
			g.notes = append(g.notes[:i], g.notes[i+1:]...)
			return true
		}
	}
	return false
}

// automapCell is the block under the screen position, false off the map.
func (g *Game) automapCell(mx, my int) (int, bool) {
	a := &g.automap
	x0, y0 := g.automapLeft(), g.automapTop()
	if mx < a.left || my < a.top || mx >= noteMapRight {
		return 0, false
	}
	x := (mx-a.left)/automapCellWidth + x0
	y := (my-a.top)/automapCellHeight + y0
	if x >= mapSize || y >= mapSize {
		return 0, false
	}
	return y*mapSize + x, true
}

// automapNoteClick handles a click on the map: left adds or edits the cell's note,
// right removes it. It returns false when the click was not on the map.
func (g *Game) automapNoteClick() bool {
	b := input.Buttons()
	if b == 0 {
		return false
	}
	block, ok := g.automapCell(g.Screen.Mouse())
	if !ok {
		return false
	}
	for input.Buttons() != 0 && !g.quit {
		host.Yield()
	}
	a := &g.automap
	switch {
	case b&2 != 0:
		if g.removeNote(a.level, block) {
			a.redraw = true
		}
	case b&1 != 0:
		n, _ := g.noteAt(a.level, block)
		if n == nil && len(g.levelNotes(a.level)) >= maxNotesPerLevel {
			return true
		}
		label := ""
		if n != nil {
			label = n.label
		}
		label, ok := g.noteLabelInput(label)
		switch {
		case !ok:
		case label == "" && n != nil:
			g.removeNote(a.level, block)
		case label == "":
		case n != nil:
			n.label = label
		default:
			g.notes = append(g.notes, automapNote{level: a.level, block: block, label: label})
		}
		a.redraw = true
	}
	return true
}

// noteLabelInput asks for a label in a box over the bottom of the map, starting with
// text; Return takes it, Escape cancels (false).
func (g *Game) noteLabelInput(text string) (string, bool) {
	a := &g.automap
	s := g.Screen
	font := s.SetFont(a.legendFont)
	defer s.SetFont(font)
	pg := s.SetPage(pageScreen)
	defer s.SetPage(pg)
	input.SetFlags(input.FlagNoRemap)
	input.ClearFlags(input.FlagKeypadButtons)
	defer func() {
		input.ClearFlags(input.FlagNoRemap)
		input.SetFlags(input.FlagKeypadButtons)
	}()
	back, fore := g.noteColour(notePromptRGB), g.noteColour(noteTextRGB)
	textX := notePromptX + 3 + s.TextWidth(notePromptText)
	maxW := notePromptX + notePromptW - 4 - textX
	buf := []byte(text)
	draw := func(cursor bool) {
		s.HideMouse()
		s.FillRect(notePromptX, notePromptY, notePromptX+notePromptW-1, notePromptY+notePromptH-1, fore)
		s.FillRect(notePromptX+1, notePromptY+1, notePromptX+notePromptW-2, notePromptY+notePromptH-2, back)
		s.DrawText(notePromptText, notePromptX+3, notePromptY+4, fore, 0)
		shown := string(buf)
		if cursor {
			shown += "_"
		}
		s.DrawText(shown, textX, notePromptY+4, fore, 0)
		s.ShowMouse()
	}
	blink, cursor := 0, true
	draw(cursor)
	for !g.quit {
		key := g.GUI.ProcessKeys(nil)
		switch key {
		case 0:
			if blink++; blink == cursorBlink {
				blink, cursor = 0, !cursor
				draw(cursor)
			}
			host.Yield()
			continue
		case keyEscape:
			return "", false
		case keyReturn, keyKeypadEnter:
			return strings.TrimSpace(string(buf)), true
		case keyBackspace:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		default:
			c := input.ASCII(key) & lowByteMask
			if c < ' ' || c > maxTyped || (c == ' ' && len(buf) == 0) || len(buf) >= noteLabelLength ||
				s.TextWidth(string(buf)+string(rune(c))+"_") > maxW {
				continue
			}
			buf = append(buf, byte(c))
		}
		draw(cursor)
	}
	return "", false
}

// drawAutomapNotes draws the letters of the shown level's notes on its cells and their
// labels under the legend's rows from row on.
func (g *Game) drawAutomapNotes(row int) {
	a := &g.automap
	s := g.Screen
	x0, y0 := g.automapLeft(), g.automapTop()
	mark, text := g.noteColour(noteMarkRGB), automapTextColour
	pg := s.SetPage(automapPage)
	font := s.SetFont(a.legendFont)
	for i, n := range g.levelNotes(a.level) {
		letter := string(rune('A' + i))
		x := (n.block%mapSize-x0)*automapCellWidth + a.left
		y := (n.block/mapSize-y0)*automapCellHeight + a.top
		s.DrawText(letter, x+(automapCellWidth-s.TextWidth(letter))/2, y, mark, 0)
		ly := row*automapLegendRow + automapLegendY
		if ly+automapLegendRow > noteLegendBottom {
			continue
		}
		s.DrawText(letter, automapSymbolX+1, ly, mark, 0)
		label := n.label
		for label != "" && s.TextWidth(label) > noteLegendWidth {
			label = label[:len(label)-1]
		}
		s.DrawText(label, automapLegendX, ly, text, 0)
		row++
	}
	s.SetFont(font)
	s.SetPage(pg)
}

// noteFile is the notes file of a saved game.
func noteFile(save string) string {
	return strings.TrimSuffix(save, path.Ext(save)) + noteFileExt
}

// saveNotes writes the notes beside a saved game: a line "level block label" each.
func (g *Game) saveNotes(save string) error {
	if len(g.notes) == 0 {
		if g.Saves.Exists(noteFile(save)) {
			return g.Saves.Remove(noteFile(save))
		}
		return nil
	}
	var b strings.Builder
	for _, n := range g.notes {
		fmt.Fprintf(&b, "%d %d %s\n", n.level, n.block, n.label)
	}
	return g.Saves.Write(noteFile(save), []byte(b.String()))
}

// loadNotes reads the notes of a saved game; without the file there are none.
func (g *Game) loadNotes(save string) {
	g.notes = nil
	b, err := g.Saves.Read(noteFile(save))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.SplitN(line, " ", 3)
		if len(f) != 3 {
			continue
		}
		level, err1 := strconv.Atoi(f[0])
		block, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil || block < 0 || block >= mapSize*mapSize {
			continue
		}
		g.notes = append(g.notes, automapNote{level: level, block: block, label: f[2]})
	}
}

// moveNotes renames a saved game's notes with it; removeNotes deletes them.
func (g *Game) moveNotes(from, to string) {
	if g.Saves.Exists(noteFile(from)) {
		g.Saves.Rename(noteFile(from), noteFile(to))
	} else if g.Saves.Exists(noteFile(to)) {
		g.Saves.Remove(noteFile(to))
	}
}

func (g *Game) removeNotes(save string) {
	if g.Saves.Exists(noteFile(save)) {
		g.Saves.Remove(noteFile(save))
	}
}
