package game

import (
	"strings"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

// FINALE.EXE is the language folder's third program: the ending. MAIN.EXE replaces
// itself with it as "FINALE.EXE <hero> HOMIE <difficulty> <folder>" (the folder must start with a drive); it plays LOLFINAL.TIM
// with LANDS.EXE's AV engine, rolls the credits, shows the hero's picture and starts
// over until Escape is pressed at it, then ends the program (C: FINALE.EXE _Main_Program,
// _Finale). Unlike LANDS.EXE it draws straight onto the screen.

// Files and settings (C: FINALE.EXE g_142, g_14c, g_159, g_71a).
const (
	finaleTextFile = "LOLFINAL.DIP"
	finaleFont     = "NEW6P.FNT" // the startup font, which the credits use (C: g_218)
	finaleScript   = "LOLFINAL.TIM"

	finaleFadeTicks  = 5 // the animations' fades (C: _LOL_Open_Animation)
	finaleTextH      = 40
	finaleClearText  = 255     // Finale_Print's column that only takes the text away
	finaleForceText  = 1 << 15 // shown even when the voices play
	finaleTextColour = 242     // the AV text opcode's (C: Default_Display_Text, 0xf2)
	finaleWonColour  = 220     // (C: _Finale, 0xdc)
	finaleWonText    = 0
	finaleHardest    = 2 // the difficulty that earns the extra line (C: [0xd2] == 2)
	finaleSampleHero = 2 // the sample slot that holds the hero's thanks
	finaleEscape     = 27
	blendSteps       = 3
	blendStepTicks   = 10
	blendTableSize   = 1 << 16
)

// The archives FINALE.EXE searches (C: g_71a).
var finaleArchives = []string{"FINALE1.PAK", "FINALE2.PAK", "DRIVERS.PAK", "FINALE.PAK"}

// finaleWindows is FINALE.EXE's window table (C: _WindowList, g_468).
var finaleWindows = []gfx.Window{
	{X: 0, Y: 0, W: 40, H: 200, Foreground: 220, Background: 253},
	{X: 8, Y: 72, W: 24, H: 56, Foreground: 220, Background: 253},
	{X: 1, Y: 32, W: 38, H: 128, Foreground: 220, Background: 253},
	{X: 9, Y: 41, W: 8, H: 44},
	{X: 25, Y: 41, W: 8, H: 44},
	{X: 1, Y: 2, W: 38, H: 20, Background: 15, Column: 14},
}

// The intro font's colours in the ending (C: g_94).
var finaleFontPalette = []byte{0, 160, 161, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// The heroes' pictures, voices and thanks, by the hero MAIN.EXE passes: Kieran,
// Ak'shel, Michael, Conrad (C: _Finale's and _Custom_Select_Character's switches,
// Default_Sample_Load's g_f3c).
var (
	finalePictures = [...]string{"KIERAN.CPS", "AK'SHEL.CPS", "MICHAEL.CPS", "CONRAD.CPS"}
	finaleVoices   = [...]string{"KINGCAT.VOC", "KINGDRAC.VOC", "KINGMIKE.VOC", "KINGCONR.VOC"}
	finaleThanks   = [...]string{"CONGRATK.VOC", "CONGRATA.VOC", "CONGRATM.VOC", "CONGRATC.VOC"}
)

// finaleHeroModes maps the leader's id to FINALE.EXE's hero: Ak'shel, Michael, Conrad,
// anyone else Kieran (C: fn_0fbe_3282).
var finaleHeroModes = map[int]int{-9: 1, -1: 2, -5: 3}

// runFinale is FINALE.EXE started for the hero (C: _Main_Program, _Finale). The original
// then exits to DOS; the caller ends the program.
func (g *Game) runFinale(hero, difficulty int) error {
	l := &lands{files: data.OpenArchives(g.Files.LanguageDir(), finaleArchives), longPriority: noPriority,
		finale: true, hero: hero}
	g.lands = l
	defer func() { g.lands = nil }()
	s := g.Screen
	windows := s.Windows
	copy(s.Windows[:], finaleWindows)
	font := s.Font
	defer func() {
		s.Windows = windows
		s.SetFont(font)
		s.CharSpacing, s.LineSpacing = 0, 0
	}()
	s.HideMouse()
	defer s.ShowMouse()

	var err error
	if l.font8, err = l.loadFont(finaleFont); err != nil {
		return err
	}
	if l.introFont, err = l.loadFont(introFontFile); err != nil {
		return err
	}
	g.mt32Init(false)
	input.Flush()
	l.bumpRate = 0
	s.SetPalette(l.black[:])
	s.ClearPage(pageScreen)
	if l.rawText, err = l.files.Read(finaleTextFile); err != nil {
		return err
	}
	s.SetFont(l.introFont)
	s.SetFontPalette(finaleFontPalette)
	for {
		input.Flush()
		s.SetPalette(l.black[:])
		l.avEnd = false
		av, err := g.loadAV(finaleScript)
		if err != nil {
			return err
		}
		l.av = av
		for !l.avEnd {
			g.longSampleStatus()
			g.processAV(av)
			g.bumpPalette()
			host.Yield()
		}
		l.av = nil
		l.anims = [landsAnimations]landsAnim{}
		g.audio.Fade(sound.Music)
		g.stopLongSample()
		s.FadePalette(l.black[:], landsFadeOut, nil)
		if err := g.finaleCredits(); err != nil {
			return err
		}
		if hero >= 0 && hero < len(finalePictures) {
			if err := g.loadPictureLands(finalePictures[hero], landsLoadPage, l.palette[:]); err != nil {
				return err
			}
		}
		s.CopyPage(landsPicPage, pageScreen)
		if difficulty == finaleHardest {
			g.finalePrint(finaleWonText|finaleForceText, 0, finaleWonColour)
		}
		s.FadePalette(l.palette[:], landsFadeOut, nil)
		input.Flush()
		if input.WaitChar() == finaleEscape {
			return nil
		}
	}
}

// finalePrint shows text n of LOLFINAL.DIP at the top of the screen, centred unless col
// is negative; col 0 first saves what it covers, a positive col first puts that back and
// finaleClearText only does that. Unless forced, nothing shows while the voices play
// (C: _Finale_Print).
func (g *Game) finalePrint(n, col, colour int) {
	l := g.lands
	if g.Config.DigitalSound && n&finaleForceText == 0 {
		return
	}
	text := g.introText(n &^ finaleForceText)
	s := g.Screen
	if col > 0 {
		s.RestoreRect(0, 0, gfx.Width, finaleTextH, l.textRows, pageScreen)
	}
	if col == finaleClearText {
		return
	}
	s.SetFont(l.introFont)
	s.SetFontPalette(finaleFontPalette)
	s.CharSpacing, s.LineSpacing = 0, -4
	if col == 0 {
		l.textRows = s.SaveRect(0, 0, gfx.Width, finaleTextH, pageScreen)
	}
	old := s.SetPage(pageScreen)
	y := 0
	for len(text) > 0 {
		line, rest, _ := strings.Cut(text, "\r")
		x := 0
		if col >= 0 {
			x = (gfx.Width - s.TextWidth(line)) / 2
		}
		s.DrawText(line, x, y, colour, 0)
		y += s.Font.Height() + s.LineSpacing
		text = rest
	}
	s.SetPage(old)
}

// finaleOpenAnimation opens animation n, or shows the CPS of that name when there is no
// WSA, on the screen (C: FINALE.EXE _LOL_Open_Animation).
func (g *Game) finaleOpenAnimation(n int, name string, x, y, delay, p4, flags int) {
	l := g.lands
	s := g.Screen
	an := &l.anims[n]
	*an = landsAnim{x: x, y: y, delay: delay, flags: flags}
	open := 0
	if flags&landsAnimDirect != 0 {
		open = wsaDirect
	}
	// p4 == 2 and landsAnimPicPage kept the frame buffer in page 2's memory
	var pal []byte
	if n == 1 {
		pal = l.palette[:]
	}
	if file, err := l.files.Read(name + ".WSA"); err == nil {
		an.wsa = newWSAPlayer(file, open, pal)
	}
	showPicture := func() {
		if l.files.Exists(name + ".CPS") {
			g.loadPictureLands(name+".CPS", landsLoadPage, l.palette[:])
			s.CopyPage(landsPicPage, pageScreen)
		}
	}
	fadeOut := func() {
		if flags&landsAnimFade != 0 {
			s.FadePalette(l.black[:], finaleFadeTicks, nil)
			s.ClearPage(pageScreen)
		}
	}
	if an.wsa == nil {
		fadeOut()
		showPicture()
		if flags&landsAnimFade == 0 {
			s.SetPalette(l.palette[:])
		} else {
			s.FadePalette(l.palette[:], finaleFadeTicks, nil)
		}
		return
	}
	if an.x == -1 {
		an.x = (gfx.Width - an.wsa.width()) / 2
	}
	if an.y == -1 {
		an.y = (gfx.Height - an.wsa.height()) / 2
	}
	fadeOut()
	if flags&landsAnimShow != 0 {
		showPicture()
		g.drawWSA(an.wsa, 0, an.x, an.y, pageScreen, 0, nil, nil)
	}
	if flags&landsAnimFade != 0 {
		s.FadePalette(l.palette[:], finaleFadeTicks, nil)
	}
}

// finaleSample loads sample slot n; the hero's thanks take the place of the slot's file
// (C: FINALE.EXE Default_Sample_Load).
func (g *Game) finaleSample(name string, n int) *sound.Sample {
	if n == finaleSampleHero && g.lands.hero >= 0 && g.lands.hero < len(finaleThanks) {
		name = finaleThanks[g.lands.hero]
	}
	return g.loadQuickSample(name)
}

// finaleUser runs one of FINALE.EXE's custom functions (C: the User opcode's table g_efc).
func (g *Game) finaleUser(a *avScript, p int) int {
	l := g.lands
	s := g.Screen
	arg := func(i int) int { return a.word(p + 1 + i) }
	switch a.word(p) {
	case 0: // C: _Custom_Start_Fade
		g.startBump(arg(0))
	case 1:
		return avStopAll // sic: a null entry
	case 2: // C: _Custom_Load_Palette
		if b, err := l.files.Read(a.str(arg(0))); err == nil {
			copy(l.palette[:], b)
		}
	case 3: // C: _Custom_Fade_To_Black
		l.palette = l.black
		g.startBump(arg(0))
	case 4: // C: _Custom_Scale_Blit
		g.finaleScaleBlit(arg(0), arg(1), arg(2), arg(3), a.str(arg(4)))
	case 5: // C: _Custom_Blend
		g.finaleBlend(a.str(arg(0)), a.str(arg(1)))
	case 6, 7: // C: _Custom_Make_File_Resident, _Custom_Free_Resident_File
	case 8: // C: _Custom_Fade_Palette_To
		var pal [data.PaletteSize]byte
		g.loadPictureLands(a.str(arg(0)), landsLoadPage, pal[:])
		s.FadePalette(pal[:], arg(1), nil)
	case 9: // C: _Custom_Smart_Display_Frame
		an := &l.anims[arg(0)]
		if arg(1) == 0 {
			s.CopyPage(pageScreen, landsPicPage)
			break
		}
		g.drawWSA(an.wsa, arg(1), an.x, an.y, landsPicPage, 0, nil, nil)
		s.CopyPage(landsPicPage, pageScreen)
	case 10: // C: _Custom_Select_Character
		if l.hero >= 0 && l.hero < len(finaleVoices) {
			l.samples[arg(0)] = g.loadQuickSample(finaleVoices[l.hero])
		}
	case 11: // C: _Custom_Fade_Score
		g.audio.Fade(sound.Music)
	case 12: // C: _Custom_Stupid_Display_Frame
		an := &l.anims[arg(0)]
		if arg(1) == -1 {
			s.CopyPage(pageScreen, landsPicPage)
			break
		}
		g.drawWSA(an.wsa, arg(1), an.x, an.y, landsPicPage, 0, nil, nil)
		if an.wsa != nil {
			x, y := an.x+an.wsa.anim.X, an.y+an.wsa.anim.Y
			s.CopyRect(x, y, x, y, an.wsa.width(), an.wsa.height(), landsPicPage, pageScreen)
		}
	case 13: // C: _Custom_Delay_Text
		if !g.Config.DigitalSound {
			clock.WaitUntil(clock.Ticks()+arg(0), nil)
		}
	case 14: // C: Custom_Display_Text
		g.finalePrint(arg(0), arg(1), arg(2))
	case 15: // C: Default_AV_Error_Trap
		return avStopAll
	default:
		return avStopAll
	}
	return 0
}

// Pages of the blend and the scaling (C: _Custom_Blend, _Custom_Scale_Blit).
const (
	finaleNewPage  = 4
	finaleOldPage  = landsWorkPage // FINALE.EXE's page 6; ours is smaller
	finaleScaleAll = 0xf0          // the flags that end a scaling
	finaleScaleNew = 0x0f          // the flags that start one
)

// finaleBlend blends the screen into picture name through a table of 256 rows of 256
// colours, one for each new colour, in three steps (C: _Custom_Blend, _Blend_Blit).
func (g *Game) finaleBlend(name, tableName string) {
	l := g.lands
	s := g.Screen
	table, _ := l.files.Read(tableName)
	if len(table) < blendTableSize {
		table = append(table, make([]byte, blendTableSize-len(table))...)
	}
	s.CopyPage(pageScreen, landsPicPage)
	g.loadPictureLands(name+".CPS", finaleNewPage, l.palette[:])
	old := s.Page(landsPicPage)[:gfx.Width*gfx.Height]
	next := s.Page(finaleNewPage)[:gfx.Width*gfx.Height]
	for range blendSteps {
		end := clock.Ticks() + blendStepTicks
		for i, c := range next {
			old[i] = table[int(c)<<8|int(old[i])]
		}
		s.CopyPage(landsPicPage, pageScreen)
		clock.WaitUntil(end, nil)
	}
	s.CopyPage(finaleNewPage, pageScreen)
}

// finaleScaleBlit zooms the old screen, kept on a page of its own, to pct percent around
// (x, y) over picture name: the first call (flags 0x0f) keeps the screen and loads the
// picture, the last (flags 0xf0) ends it (C: _Custom_Scale_Blit). LOLFINAL.TIM does not
// use it.
func (g *Game) finaleScaleBlit(x, y, pct, flags int, name string) {
	l := g.lands
	s := g.Screen
	pct = max(pct, 0)
	if flags&finaleScaleAll != 0 {
		s.CopyPage(finaleNewPage, pageScreen)
		return
	}
	if flags&finaleScaleNew != 0 {
		s.CopyPage(pageScreen, finaleOldPage)
		g.loadPictureLands(name+".CPS", finaleNewPage, l.palette[:])
	}
	w, h := gfx.Width*pct/percent, pct*2
	s.CopyPage(finaleNewPage, landsPicPage)
	s.ScaleBlit(0, 0, x-w/2, y-h/2, gfx.Width, gfx.Height, w, h, finaleOldPage, landsPicPage, 0)
	s.CopyPage(landsPicPage, pageScreen)
}

// The credits (C: _Finale_Sequence, _Write_Credits).
const (
	finCreditsScore      = "lorefinl"
	finCreditsSong       = 4
	finCreditsPicture    = "room.cps"
	finCreditsText       = "CREDITS.TXT"
	finCreditsDoor       = "door.shp"
	monsterPalette       = "MONSTERS.PAL"
	monsterColours       = 40 // each monster's colours in MONSTERS.PAL
	monsterFirst         = 88 // where they go in the palette
	finCreditsWindow     = 2
	leftDoorWindow       = 3
	rightDoorWindow      = 4
	finCreditsPage       = 4             // the window is drawn here, then copied to the screen
	finCreditsKeepPage   = landsWorkPage // the room with the monster, kept for the frames between (C: page 6, which is smaller here)
	finCreditsTicks      = 5
	finCreditsAfterTicks = 120
	finCreditsLines      = 35
	finCreditsColour     = 220
	finCreditsGone       = -10 // a line this far above the window is dropped
	finCreditsRight      = 300 // where right-aligned lines end
	finCreditsStepEvery  = 3   // frames per step of the monster
	finCreditsSteps      = 24
	monsterFrames        = 12
	monsterKinds         = 28

	finCreditsSameLine   = 5 // the next segment stays on this line
	finCreditsAlignLeft  = 3
	finCreditsAlignRight = 4
	finCreditsFont1      = 1
	finCreditsFont2      = 2
	doorOpenSteps        = 8 // the door is in front while it opens
	monsterTurnStep      = 20
	monsterFlags         = gfx.ShapeScaled | gfx.ShapeFaded
)

// The credits font's colours (C: g_e5e).
var finCreditsFontPalette = []byte{0, 0, 12, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// The monsters that come through the doors, a SHP each (C: g_6aa).
var monsterNames = [monsterKinds]string{
	"AMAZON", "ARCHRSLG", "AVIANWRM", "BANDIT", "BOAR", "CABAL", "GUARD", "HAG", "HORNET",
	"HURZELL", "IRONGRZR", "KNOWLES", "LIZARD", "MANTHA", "MINOTAUR", "MORIBUND", "ORC",
	"ORCLDR", "PENTROG", "RATMAN", "ROCKLING", "SCAVNGR", "STARK", "SWAMPCIT", "SWAMPMON",
	"THUG", "VIPER", "XEOB",
}

// The shapes of a monster file used, in the order the steps name them (C: _Write_Credits'
// loading loops: shapes 0-7 and 12-15).
var monsterShapes = [monsterFrames]int{0, 1, 2, 3, 4, 5, 6, 7, 12, 13, 14, 15}

type finPoint struct{ x, y int }

// Per step: where the monster stands on the right and the left, where the left door is
// (the right one is creditsDoorRight further), the monster's scale and frame
// (C: g_b98, g_bf8, g_c58, g_cb8, g_da8, g_dd8, g_e08).
var (
	monsterRight = [finCreditsSteps]finPoint{{205, 55}, {205, 55}, {205, 55}, {205, 55}, {205, 56},
		{207, 57}, {208, 58}, {210, 59}, {213, 60}, {216, 61}, {220, 61}, {225, 61}, {230, 61},
		{235, 61}, {240, 61}, {240, 61}, {240, 61}, {240, 61}, {240, 61}, {240, 61}, {240, 61},
		{265, 61}, {290, 61}, {315, 61}}
	monsterLeft = [finCreditsSteps]finPoint{{92, 55}, {92, 55}, {92, 55}, {92, 55}, {92, 56},
		{90, 57}, {85, 58}, {77, 59}, {67, 60}, {57, 61}, {47, 61}, {35, 61}, {35, 61},
		{35, 61}, {35, 61}, {35, 61}, {35, 61}, {35, 61}, {35, 61}, {35, 61}, {35, 61},
		{10, 61}, {-20, 61}, {-45, 61}}
	doorY       = [finCreditsSteps]int{41, 29, 17, 5, -7, -7, -7, -7, 5, 17, 29, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41}
	monsterSclX = [finCreditsSteps]int{80, 80, 80, 80, 80, 93, 112, 133, 160, 192, 226, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256}
	monsterSclY = [finCreditsSteps]int{76, 76, 76, 76, 76, 89, 107, 128, 153, 184, 217, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256, 256}
	monsterStep = [finCreditsSteps]int{0, 0, 0, 0, 0, 1, 2, 3, 0, 1, 2, 3, 8, 9, 10, 11, 8, 9, 10, 11, 4, 5, 6, 7}
)

const (
	leftDoorX  = 72
	rightDoorX = 200
)

// finaleCredits plays the credits music, rolls CREDITS.TXT over the room with the
// monsters coming through its doors and fades out (C: _Finale_Sequence).
func (g *Game) finaleCredits() error {
	l := g.lands
	s := g.Screen
	l.samples = [avSamples]*sound.Sample{}
	g.loadScore(finCreditsScore)
	g.playScore(finCreditsSong)
	monsterPal, err := l.files.Read(monsterPalette)
	if err != nil {
		return err
	}
	s.SetFont(l.font8) // both credits fonts are the startup font (C: g_39fd into g_2d54, g_2d58)
	s.SetFontPalette(finCreditsFontPalette)
	s.CharSpacing = 0
	if err := g.loadPictureLands(finCreditsPicture, landsPicPage, l.palette[:]); err != nil {
		return err
	}
	clear(l.palette[764:767]) // sic: the last two colours' bytes from colour 254's blue
	s.FadePalette(l.black[:], landsFadeOut, nil)
	s.CopyPage(landsPicPage, pageScreen)
	input.Flush()
	text, err := l.files.Read(finCreditsText)
	if err != nil {
		return err
	}
	if err := g.writeCredits(text, monsterPal); err != nil {
		return err
	}
	end := clock.Ticks() + finCreditsAfterTicks
	for clock.Ticks() < end && input.Check() == 0 {
		host.Yield()
	}
	s.FadePalette(l.black[:], landsFadeOut, nil)
	g.audio.Fade(sound.Music)
	s.ClearPage(pageScreen)
	return nil
}

// finCreditsLine is one segment of the credits on its way up (C: _Write_Credits' 11-byte
// records).
type finCreditsLine struct {
	text     string
	x, y, h  int
	sameLine bool // the next segment shares the line
}

// writeCredits scrolls the text up the credits window a pixel every finCreditsTicks while
// a monster at a time opens a door, walks into the room and out at the side
// (C: _Write_Credits).
func (g *Game) writeCredits(text []byte, monsterPal []byte) error {
	l := g.lands
	s := g.Screen
	win := s.Windows[finCreditsWindow]
	winX, winW := win.X*gfx.ColumnWidth, win.W*gfx.ColumnWidth
	s.ClearPage(finCreditsPage)
	s.ClearPage(finCreditsKeepPage)
	doors, err := g.landsShapes(finCreditsDoor)
	if err != nil {
		return err
	}
	door := doors[0]
	s.DrawShape(pageScreen, door, 0, 0, leftDoorWindow, gfx.ShapeWindowRelative, gfx.ShapeOptions{})
	s.DrawShape(pageScreen, door, 0, 0, rightDoorWindow, gfx.ShapeWindowRelative|gfx.ShapeFlipX, gfx.ShapeOptions{})
	fade := make([]byte, 256)
	for i := range 255 {
		fade[i] = byte(i)
	}
	monster, step := 0, 0
	var frames []([]byte)
	load := func(n int) {
		frames = frames[:0]
		all, _ := g.landsShapes(monsterNames[n] + ".SHP")
		for _, i := range monsterShapes {
			var f []byte
			if i < len(all) {
				f = all[i]
			}
			frames = append(frames, f)
		}
		copy(l.palette[monsterFirst*3:], monsterPal[n*monsterColours*3:][:monsterColours*3])
	}
	load(monster)
	monster++
	s.FadePalette(l.palette[:], landsFadeOut, nil)

	var lines []finCreditsLine
	timer := clock.Ticks()
	count, first := 0, true
	for {
		clock.WaitUntil(timer, nil)
		timer = clock.Ticks() + finCreditsTicks
		for len(lines) < finCreditsLines && len(text) > 0 {
			c := finCreditsLine{h: s.Font.Height()}
			if n := len(lines); n == 0 {
				c.y = win.H
			} else {
				p := lines[n-1]
				c.y = p.y + p.h + p.h>>3
				if p.sameLine {
					c.y -= p.h + p.h>>3
				}
			}
			seg := text
			if i := bytesIndexAny(text, finCreditsSameLine, '\r'); i >= 0 {
				c.sameLine = text[i] == finCreditsSameLine
				seg, text = text[:i], text[i+1:]
			} else {
				text = nil
			}
			align := 0
			for len(seg) > 0 && seg[0] <= finCreditsAlignRight {
				switch seg[0] {
				case finCreditsAlignLeft, finCreditsAlignRight:
					align = int(seg[0])
				}
				seg = seg[1:]
			}
			c.text = string(seg)
			wd := s.TextWidth(c.text)
			switch align {
			case finCreditsAlignLeft:
				c.x = 0
			case finCreditsAlignRight:
				c.x = finCreditsRight - wd
			default:
				c.x = int(uint16(winW-wd)>>1) + 1
			}
			lines = append(lines, c)
		}
		newMonster := false
		if count%finCreditsStepEvery == 0 {
			count++
			if step == 0 && first {
				s.CopyPage(landsPicPage, finCreditsPage)
				s.DrawShape(finCreditsPage, door, 0, 0, leftDoorWindow, gfx.ShapeWindowRelative, gfx.ShapeOptions{})
				s.DrawShape(finCreditsPage, door, 0, 0, rightDoorWindow, gfx.ShapeWindowRelative|gfx.ShapeFlipX, gfx.ShapeOptions{})
				count--
				first = false
			} else {
				if step == 0 {
					s.SetPalette(l.palette[:])
				}
				s.CopyPage(landsPicPage, finCreditsPage)
				right := (monster-1)&1 != 0
				dw, dx, mp, other, otherFlags, dflip := leftDoorWindow, leftDoorX, monsterLeft[step], rightDoorWindow, gfx.ShapeFlipX, 0
				if right {
					dw, dx, mp, other, otherFlags, dflip = rightDoorWindow, rightDoorX, monsterRight[step], leftDoorWindow, 0, gfx.ShapeFlipX
				}
				s.DrawShape(finCreditsPage, door, 0, 0, other, gfx.ShapeWindowRelative|otherFlags, gfx.ShapeOptions{})
				drawDoor := func() { s.DrawShape(finCreditsPage, door, dx, doorY[step], dw, dflip, gfx.ShapeOptions{}) }
				if step >= doorOpenSteps {
					drawDoor()
				}
				flags := monsterFlags
				if right && step >= monsterTurnStep {
					flags |= gfx.ShapeFlipX
				}
				s.DrawShape(finCreditsPage, frames[monsterStep[step]], mp.x, mp.y, 0, flags,
					gfx.ShapeOptions{Fade: fade, FadeSteps: 1, ScaleX: monsterSclX[step], ScaleY: monsterSclY[step]})
				if step < doorOpenSteps {
					drawDoor()
				}
				first = true
				s.CopyPage(finCreditsPage, finCreditsKeepPage)
				step = (step + 1) % finCreditsSteps
				newMonster = step == 0
			}
		} else {
			count++
			s.CopyPage(finCreditsKeepPage, finCreditsPage)
		}
		old := s.SetPage(finCreditsPage)
		for i := range lines {
			c := &lines[i]
			if c.y < win.H {
				s.DrawText(c.text, c.x+winX, c.y+win.Y, finCreditsColour, 0)
			}
			c.y--
		}
		s.SetPage(old)
		s.CopyRect(0, win.Y, 0, win.Y, gfx.Width, win.H, finCreditsPage, pageScreen)
		if len(lines) > 0 && lines[0].y < finCreditsGone {
			lines = lines[1:]
		}
		if newMonster {
			// sic: the first increment skips ARCHRSLG, which the first monster's ++ meant
			monster++
			if monster == 16 { // ORC and ORCLDR
				monster += 2
			}
			if monster == 6 { // GUARD and HAG
				monster += 2
			}
			monster %= monsterKinds
			load(monster)
			s.SetPalette(l.palette[:])
		}
		if len(lines) == 0 || input.Check() != 0 {
			return nil
		}
	}
}

// bytesIndexAny is the first index in b of one of the delimiters, -1 if none (C: strcspn).
func bytesIndexAny(b []byte, delims ...byte) int {
	for i, c := range b {
		for _, d := range delims {
			if c == d {
				return i
			}
		}
	}
	return -1
}
