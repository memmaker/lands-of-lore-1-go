package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/gfx"
)

// Character flags the portrait shows (C: Char.flags, as fn_0a21_022b tests them).
const (
	charSpellLevels = 1 << 4 // the spell levels show in place of the hand icons
	charFaceOverlay = 1 << 6 // game shape 21 is drawn over the face
	charFaceTinted  = 1 << 7 // the face is drawn through the tint table
	charEyesShut    = charDead | 1<<8 | charMute
	charDisabled    = 1<<2 | charDead | 1<<6 | 1<<8 | 1<<12 | 1<<13 // dimmed portrait (C: 0x314c)
	charHitOnly     = 1 << 2                                        // dimmed only while it shows no damage dealt
)

// Portrait layout: a 66 x 34 box at the bottom, built on page 6 and copied to the
// screen (C: fn_0a21_022b).
const (
	portraitPage       = 6
	portraitW          = 66
	portraitH          = 34
	portraitY          = 143
	portraitFrame      = 1
	magicBarX          = 33
	hitBarX            = 39
	barY               = 32
	barW               = 5
	barH               = 32
	magicBarColour     = 162
	hitBarColour       = 154
	barTextY           = 1
	magicLabelColour   = 160
	hitLabelColour     = 152
	stringMagicLabel   = 16979 // "M" (C: 0x4253)
	stringHitLabel     = 16980 // "P" (C: 0x4254)
	handIconX          = 44
	handIconW          = 22
	secondHandY        = 17
	secondHandH        = 15
	spellLevelRowShift = 3
	dimColour          = 1
	damageTextX        = 57
	damageTextY        = 7
	sufferedTextX      = 17
	sufferedTextY      = 28
	damageColour       = 254
	selectedColour     = 212
	normalFrameColour  = 1

	gameShapeSpellLevels = 73
	gameShapeSecondHand  = 72 // plus Field41
	gameShapeDamage      = 34
	gameShapeFaceOverlay = 21
	gameShapeFirstWeapon = 68 // plus the race (C: 0x44)
	noWeaponShape        = 90 // (C: 0x5a)
	hurtFaces            = 14 // faces at half hit points or less follow the others
	lastAnimatedFace     = 7
)

// portraitWeapons pairs each item shape with the game shape of its hand icon
// (C: f_27b1_004d, read at shape*2).
var portraitWeapons = [...]byte{
	90, 5, 55, 9, 56, 0, 54, 0, 54, 0, 54, 0,
	60, 38, 61, 5, 55, 7, 57, 24, 64, 13, 62, 6,
	66, 6, 63, 1, 90, 1, 90, 10, 90, 10, 90, 10,
	90, 4, 90, 4, 90, 45, 90, 45, 90, 11, 59, 36,
	90, 3, 58, 0, 60, 30, 62, 30, 62, 27, 90, 20,
	90, 41, 90, 2, 90, 12, 90, 34, 90, 33, 65, 13,
	62, 37, 90, 26, 65, 4, 90, 48, 90, 49, 90, 50,
	90, 50, 90, 0, 54, 0, 54, 0, 54, 0, 54, 23,
	90, 23, 90, 14, 90, 14, 90, 14, 90, 38, 61, 44,
	90, 33, 65, 7, 57, 39, 90, 27, 90, 27, 90, 27,
	90, 27, 90, 27, 90, 27, 90, 27, 90, 27, 90, 34,
	90, 34, 90, 22, 90, 45, 90, 59, 90, 59, 90, 58,
	90, 23, 90, 27, 90, 27, 90, 23, 90, 10, 90, 10,
	90, 10, 90, 31, 90, 23, 90, 39, 90, 27, 90, 30,
	62, 42, 90, 47, 90, 47, 90, 47, 90, 47, 90, 47,
	90, 47, 90, 44, 90, 44, 90, 44, 90, 44, 90, 52,
	90, 15, 90, 53, 67, 27, 90, 27, 90, 27, 90, 27,
	90, 43, 90, 29, 90, 29, 90, 40, 90, 40, 90, 40,
	90, 40, 90, 1, 90, 20, 90, 20, 90, 20, 90, 20,
	90, 27, 90, 39, 90, 27, 90, 27, 90, 27, 90, 27,
	90, 54, 90, 55, 90, 56, 90, 57, 90, 27, 90, 27,
	90, 27, 90, 27, 90, 39, 90, 46, 90, 46, 90, 46,
	90, 46, 90, 23, 90, 1, 90, 28, 90, 28, 90,
}

// partyCount is the number of characters present (C: ovr_2d74_0025).
func (g *Game) partyCount() int {
	n := 0
	for i := range g.party {
		if g.party[i].Flags&charPresent != 0 {
			n++
		}
	}
	return n
}

// drawFrame draws a box's edges, c1 left and bottom, c2 top and right, filled unless
// fill is noFill (C: fn_0845_0008).
func (g *Game) drawFrame(x, y, w, h, c1, c2, fill int) {
	s := g.Screen
	w--
	h--
	if fill != noFill {
		s.FillRect(x+1, y+1, x+w-1, y+h-1, fill)
	}
	s.DrawLine(x+1, y, x+w, y, c2)
	s.DrawLine(x+w, y, x+w, y+h-1, c2)
	s.DrawLine(x, y, x, y+h, c1)
	s.DrawLine(x, y+h, x+w, y+h, c1)
}

const noFill = -1

// drawPortrait draws character n's portrait box with face, bars and hand icons
// (C: fn_0a21_022b).
func (g *Game) drawPortrait(n int) {
	c := &g.party[n]
	if c.Flags&charPresent == 0 || g.updateFlags&updateNoPortraits != 0 {
		return
	}
	s := g.Screen
	font := g.Text.SetFont(g.SmallFont)
	page := s.SetPage(portraitPage)
	x := g.portraitX[n]
	g.drawFrame(0, 0, portraitW, portraitH, portraitFrame, portraitFrame, noFill)
	g.drawFace(n, 0, 1, s.Current)
	g.drawBar(magicBarX, barY, c.MP, 0, c.MaxMP, barW, barH, magicBarColour, 1, false)
	g.drawBar(hitBarX, barY, c.HP, 0, c.MaxHP, barW, barH, hitBarColour, 1, true)
	s.DrawText(g.text(stringMagicLabel), magicBarX, barTextY, magicLabelColour, 0)
	s.DrawText(g.text(stringHitLabel), hitBarX, barTextY, hitLabelColour, 0)
	levels := g.castableLevels(c, g.availableSpells[g.selectedSpell])
	if c.Flags&charSpellLevels != 0 {
		s.DrawShape(s.Current, g.gameShapes[gameShapeSpellLevels], handIconX, 0, 0, 0, gfx.ShapeOptions{})
		if levels < spellLevels {
			s.DitherRect(handIconX, levels<<spellLevelRowShift+1, handIconW, portraitH-2-levels<<spellLevelRowShift, dimColour)
		}
	} else {
		s.DrawShape(s.Current, g.gameShapes[g.weaponShape(n)], handIconX, 0, 0, 0, gfx.ShapeOptions{})
		s.DrawShape(s.Current, g.gameShapes[gameShapeSecondHand+c.Field41], handIconX, secondHandY, 0, 0, gfx.ShapeOptions{})
		if levels == 0 {
			s.DitherRect(handIconX, secondHandY, handIconW, secondHandH, dimColour)
		}
	}
	if d := c.Flags & charDisabled; (d != 0 && (d != charHitOnly || c.WeaponHit == 0)) || g.screenFrozen {
		s.DitherRect(handIconX, 0, handIconW, portraitH, dimColour)
	}
	if c.WeaponHit != 0 {
		s.DrawShape(s.Current, g.gameShapes[gameShapeDamage], handIconX, 0, 0, 0, gfx.ShapeOptions{})
		g.printAt(fmt.Sprintf("%d", c.WeaponHit), damageTextX, damageTextY, damageColour, 0, printCentred)
	}
	if int16(c.DamageSuffered) > 0 {
		g.printAt(fmt.Sprintf("%d", c.DamageSuffered), sufferedTextX, sufferedTextY, damageColour, 0, printCentred)
	}
	if page == 0 {
		s.HideMouse()
	}
	frame := normalFrameColour
	if g.selectedCharacter == n && g.partyCount() != 1 {
		frame = selectedColour
	}
	s.DrawRectOutline(0, 0, portraitW-1, portraitH-1, frame)
	s.CopyRect(0, 0, x, portraitY, portraitW, portraitH, s.Current, page)
	if page == 0 {
		s.ShowMouse()
	}
	g.Text.SetFont(font)
	s.SetPage(page)
}

// drawFace draws character n's face at (x, y) on a page (C: fn_0a21_00ec).
func (g *Game) drawFace(n, x, y, page int) {
	c := &g.party[n]
	if c.Face < lastAnimatedFace && c.FaceDefault != 0 {
		c.Face = c.FaceDefault
	}
	if c.FaceDefault == 0 && c.Face > faceBlinking && c.Face < lastAnimatedFace {
		c.Face = c.FaceDefault
	}
	f := c.Face
	if c.Flags&charEyesShut != 0 && c.Face < lastAnimatedFace {
		f = faceBlinking
	}
	if c.HP <= c.MaxHP>>1 {
		f += hurtFaces
	}
	s := g.Screen
	if page == 0 {
		s.HideMouse()
	}
	steps := 0
	if c.Flags&charFaceTinted != 0 {
		steps = 1
	}
	s.DrawShape(page, g.faces[n][f], x, y, 0, gfx.ShapeFaded, gfx.ShapeOptions{Fade: g.tintTable, FadeSteps: steps})
	if c.Flags&charFaceOverlay != 0 {
		s.DrawShape(page, g.gameShapes[gameShapeFaceOverlay], x, y, 0, 0, gfx.ShapeOptions{})
	}
	if page == 0 {
		s.ShowMouse()
	}
}

// weaponShape is the game shape of character n's hand icon: the right hand's item's,
// or a fist by race (C: fn_0a21_000b, whose loop looked at the right hand only).
func (g *Game) weaponShape(n int) int {
	c := &g.party[n]
	best, t := -1, 0
	if it := c.Items[0]; it != 0 && int(g.itemTypes[g.items[it].Type].Might) > best {
		t = g.items[it].Type
	}
	shape := int(portraitWeapons[int(g.itemTypes[t].Shape)*2])
	if shape == noWeaponShape {
		shape = max(c.RaceClassSex-1, 0) + gameShapeFirstWeapon
	}
	return shape
}

// drawPortraits draws every present character's portrait (C: fn_0a21_05ce).
func (g *Game) drawPortraits() {
	for i := range g.partyCount() {
		g.drawPortrait(i)
	}
}

// The inventory bar along the bottom: nine slots of the 48 (C: fn_0c59_0055).
const (
	inventoryShown    = 9
	inventoryY        = 179
	inventorySlotW    = 22
	inventorySlotH    = 22
	gameShapeSlot     = 4
	inventoryIconStep = 1
)

var inventorySlotX = [inventoryShown]int{106, 127, 148, 169, 190, 211, 232, 253, 274} // (C: f_27d3_0000)

// drawInventorySlot draws inventory bar slot n (C: fn_0c59_0055).
func (g *Game) drawInventorySlot(n int) {
	s := g.Screen
	x := inventorySlotX[n]
	i := (n + g.inventoryStart) % inventorySlots
	flip := 1
	if i&1 != 0 {
		flip = 0
	}
	s.HideMouseIn(x, inventoryY, x+inventorySlotW, inventoryY+inventorySlotH)
	s.DrawShape(s.Current, g.gameShapes[gameShapeSlot], x, inventoryY, 0, flip, gfx.ShapeOptions{})
	g.drawItemIcon(s.Current, g.inventory[i], x+inventoryIconStep, inventoryY+inventoryIconStep)
	s.ShowMouseIn()
}

// drawItemIcon draws an item's icon unless the slot is empty (C: fn_0c59_000d).
func (g *Game) drawItemIcon(page, item, x, y int) {
	if item != 0 {
		g.Screen.DrawShape(page, g.itemIcon(item), x, y, 0, 0, gfx.ShapeOptions{})
	}
}

// drawInventory draws the inventory bar unless a character's inventory is open
// (C: fn_0c59_0377).
func (g *Game) drawInventory() {
	if g.controlMode == 0 || !g.playing {
		for i := range inventoryShown {
			g.drawInventorySlot(i)
		}
	}
}
