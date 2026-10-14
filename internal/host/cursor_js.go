package host

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"syscall/js"
)

// hardwareCursor: in a browser the game's pointer is the page's own cursor (a CSS cursor
// image), so it moves with the hand at the browser's speed instead of the game's tick.
const hardwareCursor = true

// the cursor set last: its CSS and the picture and hot spot it was made from
var (
	cursorCSS string
	cursorKey string
)

// opaqueBox cuts img to the box around its visible pixels and the hot spot: transparent
// rows and columns still count as cursor to the browser, so a cursor image larger than its
// shape is taken for the default one as soon as the empty part reaches the viewport's edge.
func opaqueBox(img image.Image, hotX, hotY int) (image.Image, int, int) {
	b := img.Bounds()
	box := image.Rect(hotX, hotY, hotX+1, hotY+1).Add(b.Min)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	box = box.Intersect(b)
	if box == b {
		return img, hotX, hotY
	}
	out := image.NewRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
	draw.Draw(out, out.Bounds(), img, box.Min, draw.Src)
	return out, hotX - (box.Min.X - b.Min.X), hotY - (box.Min.Y - b.Min.Y)
}

// setCursor makes img, with its hot spot at (hotX, hotY), the cursor over the canvas; a nil
// img hides the cursor.
func setCursor(img image.Image, hotX, hotY int) {
	css := "none"
	if img != nil {
		img, hotX, hotY = opaqueBox(img, hotX, hotY)
		rgba, ok := img.(*image.RGBA)
		if !ok {
			rgba = image.NewRGBA(img.Bounds())
			draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
		}
		key := fmt.Sprint(rgba.Bounds(), hotX, hotY) + string(rgba.Pix)
		if key == cursorKey {
			return
		}
		cursorKey = key
		var b bytes.Buffer
		if png.Encode(&b, rgba) != nil {
			return
		}
		css = fmt.Sprintf("url(data:image/png;base64,%s) %d %d, default",
			base64.StdEncoding.EncodeToString(b.Bytes()), hotX, hotY)
	} else {
		cursorKey = ""
	}
	if css == cursorCSS {
		return
	}
	cursorCSS = css
	js.Global().Get("document").Get("body").Get("style").Set("cursor", css)
	if c := js.Global().Get("document").Call("querySelector", "canvas"); c.Truthy() {
		c.Get("style").Set("cursor", css)
	}
}
