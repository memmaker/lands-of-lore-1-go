package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// The WSA player (C: o2dfd). An animation either keeps its frame in a buffer of its own
// and draws that, or changes the page it is drawn on directly.
const (
	wsaDirect   = 1 << 1 // no buffer: frames are applied onto the page (C: user_flags 2)
	wsaNoBuffer = 1 << 0 // the original's smaller buffer; no difference here
)

type wsaPlayer struct {
	anim  *data.Animation
	frame []byte // nil when direct
}

// openWSA opens an animation; its palette, if it has one, goes into palette
// (C: ovr_2dfd_0020).
func (g *Game) openWSA(name string, flags int, palette []byte) *wsaPlayer {
	file, err := g.Files.Read(name)
	if err != nil {
		return nil
	}
	return newWSAPlayer(file, flags, palette)
}

// newWSAPlayer opens an animation from its file's contents.
func newWSAPlayer(file []byte, flags int, palette []byte) *wsaPlayer {
	a, err := data.DecodeWSA(file)
	if err != nil {
		return nil
	}
	if a.Palette != nil && palette != nil {
		copy(palette, a.Palette)
	}
	w := &wsaPlayer{anim: a}
	if flags&wsaDirect == 0 {
		w.frame = make([]byte, a.Width*a.Height)
	}
	return w
}

// width and height are the animation's size, 0 for none (C: ovr_2cef_0020, ovr_2cef_0025).
func (w *wsaPlayer) width() int {
	if w == nil {
		return 0
	}
	return w.anim.Width
}

func (w *wsaPlayer) height() int {
	if w == nil {
		return 0
	}
	return w.anim.Height
}

func (w *wsaPlayer) frames() int {
	if w == nil {
		return 0
	}
	return w.anim.Frames
}

// drawWSA brings the animation to frame and draws it at (x, y) on page: a buffered one
// through DrawBuffer with the mode in bits 12-15 of flags and its tables
// (C: ovr_2dfd_002a).
const (
	wsaModeShift = 12
	wsaModeMask  = 1<<4 - 1
)

func (g *Game) drawWSA(w *wsaPlayer, frame, x, y, page, mode int, levels, tables []byte) bool {
	if w == nil || frame >= w.anim.Frames {
		return false
	}
	x += w.anim.X
	y += w.anim.Y
	if w.frame == nil {
		dst := g.Screen.Page(page)
		at := y*gfx.Width + x
		if at < 0 || at >= len(dst) {
			return false
		}
		return w.anim.Seek(frame, dst[at:], gfx.Width, true)
	}
	if !w.anim.Seek(frame, w.frame, w.anim.Width, false) {
		return false
	}
	mode = mode >> wsaModeShift & wsaModeMask
	if levels == nil {
		mode &^= gfx.BufferRemapped // fix: the original read its tables from beyond its arguments
	}
	old := g.Screen.SetPage(page)
	g.Screen.DrawBuffer(x, y, w.anim.Width, w.anim.Height, 0, mode, w.frame, levels, tables)
	g.Screen.SetPage(old)
	return true
}

// Scene animations: six slots of WSAs the scripts and TIMs open (C: struct Anim of
// g_4ee0, o2d5f).
const (
	sceneAnimations   = 6
	animSequences     = 10
	sceneAnimFile     = "%s.%s" // (C: g_121e with g_1224)
	sceneAnimExt      = "WSA"
	noSequence        = -1
	animFadeOut       = 1 << 0 // fade the view out and take the animation's palette
	animFadeBlack     = 1 << 1 // fade to black first
	animShowFirst     = 1<<3 - 1
	animSmallBuffer   = 1 << 3  // (C: open flag 1)
	animDirect        = 1 << 4  // drawn straight onto the page
	animRemapped      = 1 << 12 // drawn through the level's TLC tables
	animOnBuildPage   = 1 << 14
	animPoolPage3     = 1 << 5 // may be kept in page 3's spare memory
	animPoolPage7     = 1 << 6
	animPoolSpare     = 1 << 7                   // in the spare page's: restoring that page closes it
	animationPools    = 8                        // indexed by the pool's page
	animDrawFlagsMask = (1<<16 - 1) &^ (15 << 8) // the flags without bits 8-11 (C: 0xf0ff)
	fadeTicks         = 10
)

type sceneAnimation struct {
	wsa          *wsaPlayer
	x, y         int
	next         int // tick of the next frame
	playing      bool
	stopAsked    bool
	delay        int
	sequence     int
	frame        int
	loops        int
	flags        int
	lastSequence int  // the last sequence that ended (C: c14)
	inSparePage  bool // kept in the spare page's pool (C: flags 0x80)
	sequences    [animSequences][8]int
}

// openSceneAnimation opens animation name into slot n and maybe shows its first frame
// (C: ovr_2d5f_0084). The original also found room for it in a spare page.
func (g *Game) openSceneAnimation(n int, name string, x, y, delay, flags int) bool {
	a := &g.animations[n]
	a.x, a.y, a.delay, a.flags = x, y, delay, flags
	a.playing = false
	a.lastSequence = noSequence
	// The original tried the pools in page 3, page 7 and the spare page in turn, as
	// far as each had room; the port only needs to know who lands in the spare page.
	a.inSparePage = flags&animPoolSpare != 0 && flags&(animPoolPage3|animPoolPage7) == 0
	open := 0
	if flags&animDirect != 0 {
		open |= wsaDirect
	}
	if flags&animSmallBuffer != 0 {
		open |= wsaNoBuffer
	}
	a.wsa = g.openWSA(fmt.Sprintf(sceneAnimFile, name, sceneAnimExt), open, g.scenePalette[:])
	if a.wsa == nil {
		return false
	}
	if flags&animFadeOut != 0 {
		if g.paletteState != paletteFadedOut {
			g.fadeOutView(fadeTicks)
		}
		copy(g.scenePalette[levelColours*3:], g.Palette[levelColours*3:])
	} else if flags&animFadeBlack != 0 {
		g.fadeToBlack(fadeTicks)
	}
	if flags&animShowFirst != 0 {
		g.Screen.HideMouseIn(x, y, x+a.wsa.width(), y+a.wsa.height())
		g.drawWSA(a.wsa, 0, x, y, pageScreen, 0, nil, nil)
		g.Screen.ShowMouseIn()
	}
	if flags&(animFadeOut|animFadeBlack) != 0 {
		g.keepInterfaceColours(g.scenePalette[:])
		g.Screen.FadePalette(g.scenePalette[:], fadeTicks, nil)
		g.paletteState = paletteNormal
	}
	return true
}

// closeSceneAnimation closes slot n (C: ovr_2d5f_008e).
func (g *Game) closeSceneAnimation(n int) bool {
	a := &g.animations[n]
	if a.wsa == nil {
		return false
	}
	*a = sceneAnimation{}
	return true
}

// showAnimationFrame shows frame of slot n on the screen (C: fn_057f_0426).
func (g *Game) showAnimationFrame(n, frame int) {
	a := &g.animations[n]
	page := pageScreen
	if a.flags&animOnBuildPage != 0 {
		page = pageBuild
	}
	g.Screen.HideMouseIn(a.x, a.y, a.x+a.wsa.width(), a.y+a.wsa.height())
	g.drawWSA(a.wsa, frame, a.x, a.y, page, a.flags&animDrawFlagsMask, nil, nil)
	g.Screen.ShowMouseIn()
}

// loadPicture reads CPS picture name onto page, its palette into pal if it has one
// (C: ovr_2ded_002f).
func (g *Game) loadPicture(name string, page int, pal []byte) error {
	file, err := g.Files.Read(name)
	if err != nil {
		return err
	}
	img, err := data.DecodeCPS(file)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	copy(g.Screen.Page(page), img.Pixels)
	if pal != nil && img.Palette != nil {
		copy(pal, img.Palette)
	}
	return nil
}

// Palette helpers.

// interfaceColours are the colours the screen keeps through a fade: the flashing ones
// of the interface (C: fn_0845_1430's 12 bytes at 0x240).
const (
	interfaceColourFirst = 192
	interfaceColourBytes = 12
)

// keepInterfaceColours copies the screen's interface colours into pal
// (C: fn_0845_1430).
func (g *Game) keepInterfaceColours(pal []byte) {
	at := interfaceColourFirst * 3
	copy(pal[at:at+interfaceColourBytes], g.Screen.Palette[at:at+interfaceColourBytes])
}

// fadeOutView fades the level's colours out and clears the view (C: fn_0845_03d4).
func (g *Game) fadeOutView(ticks int) {
	if g.paletteState == paletteFadedOut {
		return
	}
	pal := g.Palette
	clear(pal[:levelColours*3])
	g.keepInterfaceColours(pal[:])
	g.Screen.FadePalette(pal[:], ticks, nil)
	g.Screen.HideMouse()
	old := g.Screen.SetPage(pageScreen)
	g.Screen.FillRect(viewLeft, viewTop, viewLeft+viewColumns*vcnPixels, viewBottom, 0)
	g.Screen.SetPage(old)
	g.Screen.ShowMouse()
	g.paletteState = paletteFadedOut
}

// fadeInDark fades to the palette at the current brightness (C: fn_0845_090f).
func (g *Game) fadeInDark(ticks int) {
	g.keepInterfaceColours(g.darkPalette[:])
	g.Screen.FadePalette(g.darkPalette[:], ticks, nil)
	g.paletteState = paletteNormal
}

// opOpenAnimation opens animation name into a slot: name, slot, x, y, delay, flags
// (C: ovr_2d5f_00d9).
func (g *Game) opOpenAnimation(s *emc.State) int {
	return boolInt(g.openSceneAnimation(s.Arg(1), s.StringArg(0), s.Arg(2), s.Arg(3), s.Arg(4), s.Arg(5)))
}

// opCloseAnimation (C: ovr_2d5f_00de).
func (g *Game) opCloseAnimation(s *emc.State) int { return boolInt(g.closeSceneAnimation(s.Arg(0))) }

// boolInt is a script's truth value.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
