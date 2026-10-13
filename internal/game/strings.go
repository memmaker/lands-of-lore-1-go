package game

import "github.com/memmaker/lands-of-lore-1-go/internal/data"

// String ids carry their table in a flag bit: set for the language file (LANDS.ENG and
// its siblings), clear for the current level's strings.
const (
	stringLanguageFile = 1 << 14
	stringIndexMask    = stringLanguageFile - 1
	noString           = -1
)

// languageFileBase and the language extensions name the language file (C: g_10b6,
// g_0fb4, g_0f8a).
const languageFileBase = "LANDS"

var languageExtensions = []string{"ENG", "FRE", "GER"}

// loadLanguageStrings reads the language's string file (C: ovr_2d50_0020, which opened
// all three to size the buffer for the largest).
func (g *Game) loadLanguageStrings() error {
	b, err := g.Files.Read(languageFileBase + "." + languageExtensions[g.Config.Language])
	if err != nil {
		return err
	}
	g.strings = b
	return nil
}

// text returns string id unpacked and ready for the printer (C: fn_0845_06f8).
func (g *Game) text(id int) string {
	if id == noString {
		return ""
	}
	table := g.levelStrings
	if id&stringLanguageFile != 0 {
		table = g.strings
	}
	return string(data.Unescape(data.Unpack(data.TableString(table, id&stringIndexMask))))
}
