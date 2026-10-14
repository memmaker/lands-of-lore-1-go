package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
	"github.com/memmaker/lands-of-lore-1-go/internal/host"
	"github.com/memmaker/lands-of-lore-1-go/internal/input"
	"github.com/memmaker/lands-of-lore-1-go/internal/random"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

// LANDS.EXE's character generation: King Richard asks for a champion, the four
// candidates blink in their frames, and a click on one has them introduce themselves
// until the player says yes or no (C: LANDS.EXE _Char_Gen and the functions after it).

// Files (C: g_728, g_733, g_73f, g_748, g_755, g_764, g_76e..g_7b0).
const (
	charGenFont       = "font9p.FNT"
	charGenIcons      = "ITEMICN.SHP"
	charGenPicture    = "char.cps"
	charGenBackground = "backgrnd.cps"
	charGenAnimation  = "chargen.wsa"
	charGenVoice      = "000%c%d.VOC" // a candidate's lines, numbered from 0
	kingWelcome       = "KING01.VOC"
	kingWaiting       = "KING02.VOC"
	kingChosen        = "KING03.VOC"
)

// Pages: the picture with the candidates, the background with their faces' frames.
const (
	charPicPage  = 2
	charFacePage = 4
)

// charHero is a candidate of the table at 0x3c8: name, frame position and the three
// stats under the frame.
type charHero struct {
	name     string
	x, y     int
	stats    [3]int
	face     string // the talking face's shapes (C: Process_Char's switch)
	voice    byte   // the letter of its lines
	textBase int    // the first of its five lines of description in LOLINTRO.DIP (C: g_400)
}

var charHeroes = [4]charHero{
	{"Ak'shel", 96, 127, [3]int{15, 8, 5}, "face09.shp", 'A', 29},
	{"Michael", 154, 127, [3]int{6, 10, 15}, "face01.shp", 'M', 34},
	{"Kieran", 212, 127, [3]int{8, 6, 8}, "face08.shp", 'K', 39},
	{"Conrad", 271, 127, [3]int{10, 12, 10}, "face05.shp", 'C', 44},
}

// Strings of LOLINTRO.DIP.
const (
	charHelpText    = 51 // the three lines at the bottom (C: 0x33, 0x35, 0x37)
	kingWelcomeText = 57 // five lines (C: 0x39)
	kingWaitingText = 62 // two lines (C: 0x3e)
	kingChosenText  = 64 // five lines (C: 0x40)
	charConfirmText = 69 // (C: 0x45)
	heroLines       = 5
)

// Where things go.
const (
	heroFrame       = 32 // a candidate's face
	heroBorder      = 3
	heroBigFrame    = 38 // the face with its border, copied to the left
	heroBigX        = 8
	heroBigY        = 127
	kingX           = 113 // the king's animation
	kingRight       = 290
	kingBottom      = 120
	kingTextColumns = 14 // the text box left of the king, in columns
	cgTextBoxX      = 8
	cgTextBoxY      = 38
	lineStep        = 10
	descX           = 50
	descY           = 127
	confirmX        = 100
	confirmY        = 168
	chosenX         = 3
	chosenY         = 28
	helpX           = 36
	helpY           = 173
	helpStep        = 8
	nameDX          = 16
	nameDY          = 36
	statDX          = 21
	statDY          = 48
	statStep        = 8
	faceX           = 11 // the talking face
	cgFaceY         = 130
	lowerPanelY     = 122 // the lower part with the candidates (C: Show_Chars)
	lowerPanelRows  = 78
	panelX          = 6 // the left panel, in columns
	panelY          = 127
	panelColumns    = 34
	panelRows       = 60
	panelClearY     = 160
	panelClearRows  = 35
	yesX1, yesX2    = 88, 128
	noX1, noX2      = 196, 236
	answerY1        = 180
	answerY2        = 194
	eyesX           = 111 // the open and closed eyes on the background page
	eyesClosedY     = 0
	eyesOpenY       = 32
)

// Colours (C: the Lands_Print calls).
const (
	cgNameColour   = 192
	cgStatColour   = 152
	kingColour     = 50
	descColour     = 83
	cgShadowColour = 156
	descShadow     = 207
)

// Lands_Print's flags.
const (
	cgPrintCentred  = 1 << 8
	cgPrintRight    = 2 << 8
	printAlignMask  = 15 << 8
	printShadowed   = 2 << 4
	printEffectMask = 15 << 4
)

// Times in ticks.
const (
	waitSelect   = 900 // before the king asks again (C: 900)
	waitAfter    = 420 // after the choice (C: 0x1a4)
	kingFrame    = 7
	talkFrame    = 8
	blinkOpen    = 80 // open for 80 + 0-31
	blinkOpenAdd = 31
	blinkShut    = 10 // shut for 10 + 0-3
	blinkShutAdd = 3
	blinkDelay   = 3
	scoreVolume  = 200
)

// The king's frames while he talks, the faces' frames meanwhile, and the talking face's
// shapes (C: g_408, g_41a, g_43e, g_444, g_484, g_4de, g_51e.., g_5f6, g_606..).
var (
	kingTalk1    = [22]int{0, 1, 2, 3, 4, 5, 4, 3, 2, 1, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	kingTalk2    = [15]int{1, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	kingRest     = 16
	kingTalkFrom = 4
	mouthFrames  = [32]int{0, 7, 8, 9, 10, 11, 10, 9, 8, 7, 0, 0, 7, 8, 9, 10, 11, 10, 9, 8, 7, 0, 0, 7, 8, 9, 10, 11, 10, 9, 8, 7}
	mouthRest    = 0
	facePos      = [16][2]int{
		{111, 0}, {143, 0}, {175, 0}, {207, 0}, {239, 0},
		{111, 32}, {143, 32}, {175, 32}, {207, 32}, {239, 32},
		{111, 64}, {143, 64}, {175, 64}, {207, 64}, {239, 64}, {271, 0},
	}
	faceLook1 = [4][27]int{
		{0, 0, 5, 5, 5, 5, 5, 5, 5, 5, 5, 0, 0, 5, 5, 5, 5, 5, 5, 5, 0, 0, 5, 5, 5, 5, 5},
		{1, 1, 6, 6, 1, 1, 6, 6, 6, 6, 6, 6, 6, 1, 1, 6, 6, 6, 1, 1, 6, 6, 6, 6, 6, 6, 6},
		{2, 2, 7, 7, 7, 7, 2, 2, 7, 7, 7, 7, 7, 7, 7, 2, 2, 7, 7, 7, 7, 2, 2, 7, 7, 7, 7},
		{3, 3, 8, 8, 8, 8, 3, 3, 8, 8, 3, 3, 8, 8, 8, 8, 8, 8, 8, 8, 8, 3, 3, 8, 8, 8, 8},
	}
	faceBlink = [4][2]int{{0, 5}, {1, 6}, {2, 7}, {3, 8}}
	faceLook2 = [4][17]int{
		{4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
		{9, 9, 9, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6},
		{14, 14, 14, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7},
		{15, 15, 15, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8},
	}
)

const (
	noHero    = -1
	noAnswer  = -1
	answerNo  = 0
	answerOK  = 1
	resetKing = 100 // a choice that has the king speak again (C: 100)
)

// charGenState is the state of the character generation.
type charGenState struct {
	font      *gfx.Font
	king      *wsaPlayer
	faces     [][]byte
	selected  int    // (C: g_4da)
	confirmed int    // (C: _Confirmed, g_4dc)
	blinkAt   [4]int // (C: g_6f2)
	blinkOpen [4]int // (C: g_702)
}

// charGen runs the character generation and returns the chosen hero (C: _Char_Gen).
func (g *Game) charGen() (int, error) {
	l := g.lands
	s := g.Screen
	c := &charGenState{selected: noHero, confirmed: noAnswer, blinkOpen: [4]int{0, 1, 0, 1}}
	var err error
	if c.font, err = l.loadFont(charGenFont); err != nil {
		return 0, err
	}
	icons, err := g.landsShapes(charGenIcons)
	if err != nil {
		return 0, err
	}
	if err := g.loadPictureLands(charGenPicture, charPicPage, l.palette[:]); err != nil {
		return 0, err
	}
	if err := g.loadPictureLands(charGenBackground, charFacePage, l.palette[:]); err != nil {
		return 0, err
	}
	g.audio.SetMusicVolume(scoreVolume)
	file, err := l.files.Read(charGenAnimation)
	if err != nil {
		return 0, err
	}
	c.king = newWSAPlayer(file, 0, nil)
	g.drawWSA(c.king, 0, kingX, 0, charPicPage, 0, nil, nil)
	s.SetFont(c.font)
	old := s.SetPage(charPicPage)
	for _, h := range charHeroes {
		g.landsPrint(h.name, h.x+nameDX, h.y+nameDY, cgNameColour, 0, cgShadowColour, cgPrintCentred|printShadowed)
	}
	for _, h := range charHeroes {
		for k, v := range h.stats {
			g.landsPrint(fmt.Sprint(v), h.x+statDX, h.y+statDY+k*statStep, cgStatColour, 0, cgShadowColour, cgPrintRight|printShadowed)
		}
	}
	for k := range 3 {
		g.landsPrint(g.introText(charHelpText+2*k), helpX, helpY+k*helpStep, cgStatColour, 0, cgShadowColour, printShadowed)
	}
	s.CopyRectColumns(0, 0, 0, 0, gfx.Width/gfx.ColumnWidth, gfx.Height, charPicPage, pageScreen)
	s.SetPage(old)
	s.FadePalette(l.palette[:], fadeInTicks, nil)
	s.SetFont(c.font)
	if len(icons) > 0 {
		s.SetMouseShape(0, 0, icons[0])
	}
	s.ShowMouse()
	defer s.HideMouse()

	kingSpeaks := true
	for {
		if kingSpeaks {
			g.animateKing1(c)
		}
		if c.selected < 0 {
			g.waitForSelect(c)
		}
		if c.selected == resetKing {
			kingSpeaks = true
		} else {
			s.HideMouseIn(0, 0, kingTextColumns*gfx.ColumnWidth, kingBottom)
			s.CopyRectColumns(0, 0, 0, 0, kingTextColumns, kingBottom, charFacePage, pageScreen)
			s.ShowMouseIn()
			if n := g.processChar(c, c.selected); n != noHero {
				break
			}
			kingSpeaks = false
		}
		c.selected = noHero
	}
	until := clock.Ticks() + waitAfter
	for clock.Ticks() < until {
		if landsClicked() {
			break
		}
		host.Yield()
	}
	input.Flush()
	return c.selected, nil
}

// landsShapes reads the shapes of a shape file (C: _Load_Shape_File, _Get_Shape).
func (g *Game) landsShapes(name string) ([][]byte, error) {
	file, err := g.lands.files.Read(name)
	if err != nil {
		return nil, err
	}
	img, err := data.DecodeCPS(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	shapes := make([][]byte, gfx.ShapeCount(img.Pixels))
	for i := range shapes {
		shapes[i] = gfx.ShapeFrame(img.Pixels, i)
	}
	return shapes, nil
}

// landsPrint prints text centred on or right-aligned to x by the flags, with a shadow
// left of and under it (C: _Lands_Print).
func (g *Game) landsPrint(text string, x, y, fore, back, shadow, flags int) {
	s := g.Screen
	switch flags & printAlignMask {
	case cgPrintCentred:
		x -= s.TextWidth(text) / 2
	case cgPrintRight:
		x -= s.TextWidth(text)
	}
	if flags&printEffectMask == printShadowed {
		s.DrawText(text, x-1, y, shadow, back)
		s.DrawText(text, x, y+1, shadow, back)
	}
	s.DrawText(text, x, y, fore, back)
}

// landsClicked takes the next key and says whether it was a mouse button
// (C: Check_Key and Get_Key for 0xc6 and 0xc7).
func landsClicked() bool {
	if input.Peek() == 0 {
		return false
	}
	return isClick(input.Get())
}

func isClick(key int) bool {
	n := key & input.Number
	return key&input.Released == 0 && (n == input.KeyMouseLeft || n == input.KeyMouseRight)
}

// waitForSelect lets the candidates blink until one is clicked, the king asking again
// when none is for a while (C: _Wait_For_Select).
func (g *Game) waitForSelect(c *charGenState) {
	c.selected = noHero
	for c.selected < 0 {
		until := clock.Ticks() + waitSelect
		for {
			g.partyWait(c)
			c.selected = g.getChar()
			if clock.Ticks() >= until || c.selected >= 0 {
				break
			}
			host.Yield()
		}
		if c.selected < 0 {
			g.animateKing2(c)
		}
	}
}

// getChar is the candidate a click hit, or noHero (C: _Get_Char).
func (g *Game) getChar() int {
	if !landsClicked() {
		return noHero
	}
	for i, h := range charHeroes {
		if g.Screen.MouseIn(h.x, h.y, h.x+heroFrame-1, h.y+heroFrame-1) {
			return i
		}
	}
	return noHero
}

// drawCandidateFace copies one of the faces of the background page into a candidate's frame.
func (g *Game) drawCandidateFace(hero, face int) {
	s := g.Screen
	h := charHeroes[hero]
	s.HideMouseIn(h.x, h.y, h.x+heroFrame, h.y+heroFrame)
	p := facePos[face]
	s.CopyRect(p[0], p[1], h.x, h.y, heroFrame, heroFrame, charFacePage, pageScreen)
	s.ShowMouseIn()
}

// partyWait opens and shuts the candidates' eyes when it is time (C: _Party_Wait_Processing).
func (g *Game) partyWait(c *charGenState) {
	for i := range charHeroes {
		if clock.Ticks() <= c.blinkAt[i] {
			continue
		}
		g.drawCandidateFace(i, faceBlink[i][c.blinkOpen[i]])
		if c.blinkOpen[i] == 1 {
			c.blinkAt[i] = clock.Ticks() + blinkOpen + random.Byte()&blinkOpenAdd
		} else {
			c.blinkAt[i] = clock.Ticks() + blinkShut + random.Byte()&blinkShutAdd
		}
		c.blinkOpen[i] = (c.blinkOpen[i] + 1) % 2
	}
}

// kingText clears the text box and prints lines of the king's.
func (g *Game) kingText(first, n, y int) {
	s := g.Screen
	s.HideMouseIn(0, 0, kingX-1, kingBottom)
	s.CopyRectColumns(0, 0, 0, 0, kingTextColumns, kingBottom, charFacePage, pageScreen)
	for k := range n {
		g.landsPrint(g.introText(first+k), cgTextBoxX, y+k*lineStep, kingColour, 0, cgShadowColour, printShadowed)
	}
	s.ShowMouseIn()
}

// drawKing shows a frame of the king.
func (g *Game) drawKing(c *charGenState, frame int) {
	s := g.Screen
	s.HideMouseIn(kingX, 0, kingRight, kingBottom)
	g.drawWSA(c.king, frame, kingX, 0, pageScreen, 0, nil, nil)
	s.ShowMouseIn()
}

// waitPicking waits n ticks unless a candidate is clicked.
func (g *Game) waitPicking(c *charGenState, n int) {
	until := clock.Ticks() + n
	for clock.Ticks() < until {
		if c.selected = g.getChar(); c.selected >= 0 {
			return
		}
		host.Yield()
	}
}

// animateKing1 has the king ask for a champion while the candidates look at him
// (C: _Animate_King_1).
func (g *Game) animateKing1(c *charGenState) {
	g.kingText(kingWelcomeText, 5, cgTextBoxY)
	voice := g.loadQuickSample(kingWelcome)
	if voice != nil {
		g.playLongSample(voice, voicePriority, introVoiceVolume)
	}
	for i := kingTalkFrom; ; i = (i + 1) % len(kingTalk1) {
		if !g.longSampleStatus() || c.selected >= 0 {
			break
		}
		i = max(i, kingTalkFrom)
		g.drawKing(c, kingTalk1[i])
		for h := range charHeroes {
			g.drawCandidateFace(h, faceLook1[h][i])
		}
		g.waitPicking(c, kingFrame)
	}
	s := g.Screen
	s.HideMouse()
	g.drawWSA(c.king, kingRest, kingX, 0, pageScreen, 0, nil, nil)
	s.ShowMouse()
	g.stopLongSample()
}

// animateKing2 has the king urge the player on (C: _Animate_King_2).
func (g *Game) animateKing2(c *charGenState) {
	g.kingText(kingWaitingText, 2, cgTextBoxY+lineStep)
	voice := g.loadQuickSample(kingWaiting)
	if voice != nil {
		g.playLongSample(voice, voicePriority, introVoiceVolume)
	}
	for i := range kingTalk2 {
		if !g.longSampleStatus() || c.selected >= 0 {
			break
		}
		g.drawKing(c, kingTalk2[i])
		for h := range charHeroes {
			g.drawCandidateFace(h, faceLook2[h][i])
		}
		g.waitPicking(c, talkFrame)
	}
	g.stopLongSample()
}

// animateKing3 has the king welcome the champion (C: _Animate_King_3).
func (g *Game) animateKing3(c *charGenState) {
	input.Flush()
	voice := g.loadQuickSample(kingChosen)
	if voice != nil {
		g.playLongSample(voice, voicePriority, introVoiceVolume)
	}
	for i := kingTalkFrom; input.Check() == 0 && g.longSampleStatus(); i = (i + 1) % len(kingTalk1) {
		i = max(i, kingTalkFrom)
		g.drawKing(c, kingTalk1[i])
		until := clock.Ticks() + talkFrame
		for clock.Ticks() < until && input.Check() == 0 {
			host.Yield()
		}
	}
	g.stopLongSample()
	g.drawKing(c, kingRest)
}

// processChar has a candidate introduce itself and asks whether to take it; it returns
// the candidate if so, else noHero (C: _Process_Char).
func (g *Game) processChar(c *charGenState, n int) int {
	s := g.Screen
	h := charHeroes[n]
	var err error
	if c.faces, err = g.landsShapes(h.face); err != nil {
		c.faces = nil
	}
	s.HideMouse()
	s.CopyRectColumns(0, lowerPanelY, 0, lowerPanelY, gfx.Width/gfx.ColumnWidth, lowerPanelRows, charFacePage, pageScreen)
	s.CopyRect(h.x-heroBorder, h.y-heroBorder, heroBigX, heroBigY, heroBigFrame, heroBigFrame, charPicPage, pageScreen)
	old := s.SetPage(pageScreen)
	for k := range heroLines {
		g.landsPrint(g.introText(h.textBase+k), descX, descY+k*lineStep, descColour, 0, descShadow, printShadowed)
	}
	g.landsPrint(g.introText(charConfirmText), confirmX, confirmY, kingColour, 0, descShadow, printShadowed)
	s.SetPage(old)
	s.ShowMouse()
	g.animateCharacter(c, h.voice)
	if c.confirmed == noAnswer {
		if g.getConfirm() {
			c.confirmed = answerOK
		} else {
			c.confirmed = answerNo
		}
	}
	if c.confirmed != answerOK {
		c.confirmed = noAnswer
		g.showChars()
		input.Flush()
		return noHero
	}
	s.HideMouseIn(panelX, panelY, 40, 187)
	s.CopyRectColumns(panelX, panelY, panelX, panelY, panelColumns, panelRows, charFacePage, pageScreen)
	s.ShowMouseIn()
	s.HideMouse()
	s.CopyRectColumns(panelX, panelY, panelX, panelClearY, panelColumns, panelClearRows, charFacePage, pageScreen)
	s.CopyRectColumns(0, 0, 0, 0, kingTextColumns, kingBottom, charFacePage, pageScreen)
	old = s.SetPage(pageScreen)
	for k := range 5 {
		g.landsPrint(g.introText(kingChosenText+k), chosenX, chosenY+k*lineStep, kingColour, 0, cgShadowColour, printShadowed)
	}
	s.SetPage(old)
	s.ShowMouse()
	g.animateKing3(c)
	return n
}

// animateCharacter plays a candidate's lines, moving its mouth, until they run out or
// the player answers (C: _Animate_Character).
func (g *Game) animateCharacter(c *charGenState, letter byte) {
	for k := 0; ; k++ {
		var voice *sound.Sample
		if c.confirmed < 0 {
			voice = g.loadQuickSample(fmt.Sprintf(charGenVoice, letter, k))
		}
		if voice == nil {
			g.drawTalkingFace(c, mouthRest)
			return
		}
		g.playLongSample(voice, voicePriority, introVoiceVolume)
		for i := 0; g.longSampleStatus() && c.confirmed < 0; i = (i + 1) % len(mouthFrames) {
			g.drawTalkingFace(c, mouthFrames[i])
			until := clock.Ticks() + talkFrame
			for clock.Ticks() < until && c.confirmed < 0 {
				c.confirmed = g.quickConfirm()
				host.Yield()
			}
		}
		g.stopLongSample()
	}
}

func (g *Game) drawTalkingFace(c *charGenState, n int) {
	if n >= len(c.faces) || c.faces[n] == nil {
		return
	}
	s := g.Screen
	f := c.faces[n]
	s.HideMouseIn(faceX, cgFaceY, faceX+gfx.ShapeWidth(f), cgFaceY+gfx.ShapeHeight(f))
	s.DrawShape(pageScreen, f, faceX, cgFaceY, 0, 0, gfx.ShapeOptions{})
	s.ShowMouseIn()
}

// answer says which of Yes and No the cursor is on, or noAnswer.
func (g *Game) answer() int {
	switch {
	case g.Screen.MouseIn(yesX1, answerY1, yesX2, answerY2):
		return answerOK
	case g.Screen.MouseIn(noX1, answerY1, noX2, answerY2):
		return answerNo
	}
	return noAnswer
}

// quickConfirm is the answer under a held button, or noAnswer (C: _Quick_Confirm).
func (g *Game) quickConfirm() int {
	if input.Down(input.KeyMouseLeft) || input.Down(input.KeyMouseRight) {
		return g.answer()
	}
	return noAnswer
}

// getConfirm waits for a click on Yes or No (C: _Get_Confirm).
func (g *Game) getConfirm() bool {
	input.Flush()
	for {
		if isClick(input.Get()) {
			if a := g.answer(); a != noAnswer {
				return a == answerOK
			}
		}
		input.Flush()
	}
}

// showChars puts the candidates back (C: _Show_Chars).
func (g *Game) showChars() {
	s := g.Screen
	s.HideMouse()
	s.CopyRectColumns(0, lowerPanelY, 0, lowerPanelY, gfx.Width/gfx.ColumnWidth, lowerPanelRows, charPicPage, pageScreen)
	s.ShowMouse()
}
