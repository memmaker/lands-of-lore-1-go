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

// the pointer's place in the page (CSS pixels), followed from the page's own events
var (
	pointerX, pointerY int
	pointerFollowed    bool
)

func followPointer() {
	if pointerFollowed {
		return
	}
	pointerFollowed = true
	js.Global().Call("addEventListener", "pointermove", js.FuncOf(func(_ js.Value, a []js.Value) any {
		pointerX, pointerY = a[0].Get("clientX").Int(), a[0].Get("clientY").Int()
		return nil
	}), true)
}

// maxWholeCursor is the largest cursor browsers show across the viewport's edge; a larger
// one is replaced by the default cursor there (Chrome), so it is cut to what is inside.
const maxWholeCursor = 32

// insideViewport cuts img, its hot spot at the pointer, to the part inside the viewport.
func insideViewport(img image.Image, hotX, hotY int) (image.Image, int, int) {
	b := img.Bounds()
	if b.Dx() <= maxWholeCursor && b.Dy() <= maxWholeCursor {
		return img, hotX, hotY
	}
	w := js.Global().Get("innerWidth").Int()
	h := js.Global().Get("innerHeight").Int()
	at := image.Pt(pointerX-hotX, pointerY-hotY) // where the image's corner is in the page
	in := image.Rect(0, 0, w, h).Sub(at).Intersect(image.Rect(0, 0, b.Dx(), b.Dy()))
	if in.Empty() || !image.Pt(hotX, hotY).In(in) {
		return img, hotX, hotY
	}
	out := image.NewRGBA(image.Rect(0, 0, in.Dx(), in.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min.Add(in.Min), draw.Src)
	return out, hotX - in.Min.X, hotY - in.Min.Y
}

// setCursor makes img, with its hot spot at (hotX, hotY), the cursor over the canvas; a nil
// img hides the cursor.
func setCursor(img image.Image, hotX, hotY int) {
	followPointer()
	css := "none"
	if img != nil {
		img, hotX, hotY = insideViewport(img, hotX, hotY)
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
