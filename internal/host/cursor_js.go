package host

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
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

// setCursor makes img, with its hot spot at (hotX, hotY), the cursor over the canvas; a nil
// img hides the cursor.
func setCursor(img image.Image, hotX, hotY int) {
	css := "none"
	if img != nil {
		key := fmt.Sprint(img.Bounds(), hotX, hotY) + string(img.(*image.RGBA).Pix)
		if key == cursorKey {
			return
		}
		cursorKey = key
		var b bytes.Buffer
		if png.Encode(&b, img) != nil {
			return
		}
		css = fmt.Sprintf("url(data:image/png;base64,%s) %d %d, default",
			base64.StdEncoding.EncodeToString(b.Bytes()), hotX, hotY)
	}
	if img == nil {
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
