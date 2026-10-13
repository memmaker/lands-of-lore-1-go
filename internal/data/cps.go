package data

import (
	"encoding/binary"
	"fmt"
)

// A CPS file: a 2-byte length (of the rest of the file), the 8-byte header below, an
// optional palette, then the image data.
//
//	compression (2), unpacked size (4), palette bytes (2)
const (
	cpsLengthSize = 2
	cpsHeaderSize = 8
)

// CPS compression methods (the header's first field).
const (
	cpsRaw = 0
	cpsRLE = 3
	cpsLCW = 4
)

// PaletteSize is the size of a full VGA palette: 256 colours of red, green, blue (0-63).
const PaletteSize = 768

// Image is a decoded CPS picture.
type Image struct {
	Pixels  []byte // usually a full 320x200 screen
	Palette []byte // nil if the file has none
}

// DecodeCPS unpacks a CPS file (C: ovr_2ded_002f reading it, ovr_2ded_0034 unpacking).
// Unknown compression methods are copied raw, as the original's switch default does.
func DecodeCPS(file []byte) (*Image, error) {
	if len(file) < cpsLengthSize+cpsHeaderSize {
		return nil, fmt.Errorf("cps: %d bytes is too short", len(file))
	}
	length := int(binary.LittleEndian.Uint16(file))
	header := file[cpsLengthSize:]
	compression := header[0]
	size := int(binary.LittleEndian.Uint32(header[2:]))
	paletteSize := int(binary.LittleEndian.Uint16(header[6:]))

	end := min(cpsLengthSize+length, len(file))
	body := file[cpsLengthSize+cpsHeaderSize : end]
	if paletteSize > len(body) {
		return nil, fmt.Errorf("cps: palette of %d bytes beyond the end", paletteSize)
	}
	img := &Image{Pixels: make([]byte, size)}
	if paletteSize != 0 {
		img.Palette = append([]byte(nil), body[:paletteSize]...)
	}
	data := body[paletteSize:]
	switch compression {
	case cpsRLE:
		DecodeRLE(data, img.Pixels)
	case cpsLCW:
		DecodeLCW(data, img.Pixels)
	default:
		copy(img.Pixels, data)
	}
	return img, nil
}
