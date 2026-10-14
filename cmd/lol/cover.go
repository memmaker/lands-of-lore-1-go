package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/memmaker/lands-of-lore-1-go/internal/data"
)

// The cover goplay shows on the floppy label: the title picture.
const (
	coverFile    = "COVER.PNG"
	coverPicture = "TITLE.CPS"
	coverWidth   = 320
	coverHeight  = 200
	sixBitScale  = 255.0 / 63.0
)

// writeCover renders the title picture of the game in dir to dir/COVER.PNG.
func writeCover(dir string) error {
	files, err := data.Open(dir)
	if err != nil {
		return err
	}
	b, err := files.Read(coverPicture)
	if err != nil {
		return err
	}
	pic, err := data.DecodeCPS(b)
	if err != nil {
		return err
	}
	pal := make(color.Palette, len(pic.Palette)/3)
	for i := range pal {
		c := func(k int) uint8 { return uint8(float64(pic.Palette[3*i+k]&63)*sixBitScale + 0.5) }
		pal[i] = color.RGBA{c(0), c(1), c(2), 255}
	}
	img := image.NewPaletted(image.Rect(0, 0, coverWidth, coverHeight), pal)
	copy(img.Pix, pic.Pixels)
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, coverFile), out.Bytes(), fileMode)
}
