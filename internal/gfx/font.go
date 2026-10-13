package gfx

import "encoding/binary"

// A font file (.FNT) starts with a table of offsets into itself:
//
//	+4 info block (byte 4: height, byte 5: widest glyph)
//	+6 one word per character: where its glyph starts, 0 if it has none
//	+8 one byte per character: its width
//	+12 two bytes per character: blank rows above the glyph, rows of the glyph
//
// Glyph rows hold two pixels a byte, low nibble first; each nibble is a colour index
// through the colour map, where 0 is the background.
const (
	fontInfoOffset    = 4
	fontGlyphsOffset  = 6
	fontWidthsOffset  = 8
	fontHeightsOffset = 12
	fontHeaderSize    = 14
	fontInfoHeight    = 4
	fontInfoWidth     = 5
	fontColours       = 16
	nibbleBits        = 4
	nibbleMask        = fontColours - 1
)

// carriageReturn ends a line in printed strings.
const carriageReturn = '\r'

// Font is a loaded font file.
type Font struct {
	data                          []byte
	info, glyphs, widths, heights int
}

// NewFont wraps a font file's contents; nil if it is too short to be one.
func NewFont(b []byte) *Font {
	if len(b) < fontHeaderSize {
		return nil
	}
	word := func(at int) int { return int(binary.LittleEndian.Uint16(b[at:])) }
	return &Font{
		data:    b,
		info:    word(fontInfoOffset),
		glyphs:  word(fontGlyphsOffset),
		widths:  word(fontWidthsOffset),
		heights: word(fontHeightsOffset),
	}
}

func (f *Font) byteAt(at int) int {
	if at < 0 || at >= len(f.data) {
		return 0
	}
	return int(f.data[at])
}

// Height is the font's line height in pixels.
func (f *Font) Height() int { return f.byteAt(f.info + fontInfoHeight) }

// MaxWidth is the width of its widest glyph.
func (f *Font) MaxWidth() int { return f.byteAt(f.info + fontInfoWidth) }

// GlyphWidth is a character's width without spacing.
func (f *Font) GlyphWidth(c byte) int { return f.byteAt(f.widths + int(c)) }

// SetFont makes f the current font and returns the previous one (C: fn_1f2d_0002, whose
// text-window bookkeeping lives in the text layer).
func (s *Screen) SetFont(f *Font) *Font {
	old := s.Font
	if f != nil {
		s.Font = f
	}
	return old
}

// SetFontColours puts colours into the font colour map from entry first to last
// (C: fn_1e07_0255).
func (s *Screen) SetFontColours(colours []byte, first, last int) {
	first &= nibbleMask
	last &= nibbleMask
	for i := first; i <= last && i-first < len(colours); i++ {
		s.fontColours[i] = colours[i-first]
	}
}

// CharWidth is how far a character advances the pen (C: fn_1e64_0007).
func (s *Screen) CharWidth(c byte) int { return s.Font.GlyphWidth(c) + s.CharSpacing }

// TextWidth is the width of the widest line of a string (C: fn_1e64_002b).
func (s *Screen) TextWidth(text string) int {
	widest, line := 0, 0
	for i := 0; i < len(text); i++ {
		if text[i] == carriageReturn {
			widest = max(widest, line)
			line = 0
			continue
		}
		line += s.CharWidth(text[i])
	}
	return max(widest, line)
}

// DrawChar draws one character with its top left corner at (x, y) on the current page;
// it is left out if any of it would be off screen (C: fn_1e07_00ff).
func (s *Screen) DrawChar(c byte, x, y int) {
	f := s.Font
	if f == nil {
		return
	}
	glyph := f.byteAt(f.glyphs+2*int(c)) | f.byteAt(f.glyphs+2*int(c)+1)<<8
	if glyph == 0 || x >= Width {
		return
	}
	w := f.GlyphWidth(c)
	if x+w > Width || y >= Height {
		return
	}
	height := f.Height()
	if y+height > Height {
		return
	}
	above := f.byteAt(f.heights + 2*int(c))
	rows := f.byteAt(f.heights + 2*int(c) + 1)
	below := height - above - rows

	page := s.Page(s.Current)
	at := y*Width + x
	background := s.fontColours[0]
	fill := func(n int) {
		for ; n > 0; n-- {
			if background != 0 {
				row := page[at : at+w]
				for i := range row {
					row[i] = background
				}
			}
			at += Width
		}
	}
	fill(above)
	in := glyph
	for ; rows > 0; rows-- {
		for i := 0; i < w; i++ {
			b := f.byteAt(in + i/2)
			nibble := b & nibbleMask
			if i%2 == 1 {
				nibble = b >> nibbleBits
			}
			if colour := s.fontColours[nibble]; colour != 0 {
				page[at+i] = colour
			}
		}
		in += (w + 1) / 2
		at += Width
	}
	fill(below)
}

// DrawText prints a string at (x, y) on the current page in the current font with the
// given foreground and background colours (background 0 leaves it transparent). Lines
// break at carriage returns and where a character would pass the right edge
// (C: fn_1f6c_0006).
func (s *Screen) DrawText(text string, x, y, foreground, background int) {
	f := s.Font
	if f == nil {
		return
	}
	x, y = max(x, 0), max(y, 0)
	if x >= Width || y >= Height {
		return
	}
	s.SetFontColours([]byte{byte(background), byte(foreground)}, 0, 1)
	left := x
	lineHeight := f.Height()
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == 0 {
			return
		}
		if c == carriageReturn {
			x = left
			y += lineHeight + s.LineSpacing
			continue
		}
		w := s.CharWidth(c)
		if x+w > Width {
			x = left
			y += lineHeight + s.LineSpacing
		}
		if y >= Height {
			return
		}
		s.DrawChar(c, x, y)
		x += w
	}
}
