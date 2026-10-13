package data

// XOR delta (Westwood's "format 40") commands, by their first byte b:
//
//	b in 1..127                 XOR the next b bytes onto the output
//	0, count, value             XOR value onto count bytes
//	b in 129..255               skip b-128 bytes
//	128, word w (little endian):
//	  w = 0                     end
//	  w in 1..32767             skip w bytes
//	  w in 32768..49151         XOR the next w-32768 bytes onto the output
//	  w in 49152..65535         XOR the next byte onto w-49152 bytes
const (
	deltaSkipFlag   = 128
	deltaLongRun    = 1 << 15
	deltaLongFill   = 1 << 14
	deltaLongCounts = deltaLongFill - 1
)

// ApplyDelta lays XOR delta data onto dst, a block width pixels wide inside an image whose
// rows are stride bytes apart (C: fn_1e76_0213; with stride == width it is the linear
// fn_1e76_0029). With overwrite the bytes are stored instead of XORed (the original's
// variant for a first frame drawn straight onto the screen). Writes outside dst are dropped.
func ApplyDelta(dst, delta []byte, width, stride int, overwrite bool) {
	in := 0
	next := func() byte {
		if in >= len(delta) {
			return 0
		}
		in++
		return delta[in-1]
	}
	rowStart, column := 0, 0
	put := func(b byte) {
		if at := rowStart + column; at >= 0 && at < len(dst) {
			if overwrite {
				dst[at] = b
			} else {
				dst[at] ^= b
			}
		}
		column++
		if column == width {
			column = 0
			rowStart += stride
		}
	}
	skip := func(n int) {
		column += n
		for column >= width {
			column -= width
			rowStart += stride
		}
	}
	literal := func(n int) {
		for ; n > 0; n-- {
			put(next())
		}
	}
	fill := func(n int, b byte) {
		for ; n > 0; n-- {
			put(b)
		}
	}
	if width <= 0 {
		return
	}
	for in < len(delta) {
		b := next()
		switch {
		case b == 0:
			// fix: a count of 0 made the original's loop run 65536 times.
			n := int(next())
			fill(n, next())
		case b&deltaSkipFlag == 0:
			literal(int(b))
		case b != deltaSkipFlag:
			skip(int(b &^ deltaSkipFlag))
		default:
			w := int(next()) | int(next())<<8
			switch {
			case w == 0:
				return
			case w&deltaLongRun == 0:
				skip(w)
			case w&deltaLongFill != 0:
				fill(w&deltaLongCounts, next())
			default:
				literal(w & deltaLongCounts)
			}
		}
	}
}
