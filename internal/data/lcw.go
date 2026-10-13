package data

// LCW (Westwood's "format 80") commands, by their first byte:
//
//	0ccc pppp pppppppp         copy ccc+3 bytes from pppp pppppppp back in the output
//	10cc cccc                  copy cccccc literal bytes (10000000 ends the data)
//	11cc cccc pppp pppp(x2)    copy cccccc+3 bytes from output offset p
//	11111110 cccc(x2) b        fill cccc bytes with b
//	11111111 cccc(x2) pppp(x2) copy cccc bytes from output offset p
const (
	lcwLiteralFlag = 0x80
	lcwAbsoluteBit = 0x40
	lcwEnd         = 0x80
	lcwFill        = 0xfe
	lcwLongCopy    = 0xff
	lcwCountMask   = 0x3f
	lcwMinRun      = 3
	nibbleBits     = 4
	lowNibble      = 0x0f
)

// DecodeLCW unpacks LCW data into dst, at most len(dst) bytes, and returns how many it
// wrote (C: fn_232c_000c). Like the original it stops at the end marker or when dst is
// full; copies that reach outside the written output read zeros instead of failing.
func DecodeLCW(src, dst []byte) int {
	in, out := 0, 0
	next := func() byte {
		if in >= len(src) {
			return 0
		}
		in++
		return src[in-1]
	}
	word := func() int { lo := int(next()); return lo | int(next())<<8 }
	copyFrom := func(from, count int) {
		for ; count > 0 && out < len(dst); count-- {
			if from >= 0 && from < len(dst) {
				dst[out] = dst[from]
			} else {
				dst[out] = 0
			}
			from++
			out++
		}
	}
	for out < len(dst) {
		cmd := next()
		switch {
		case cmd&lcwLiteralFlag == 0:
			count := int(cmd>>nibbleBits) + lcwMinRun
			back := int(cmd&lowNibble)<<8 | int(next())
			copyFrom(out-back, count)
		case cmd&lcwAbsoluteBit == 0:
			if cmd == lcwEnd {
				return out
			}
			for count := int(cmd & lcwCountMask); count > 0 && out < len(dst); count-- {
				dst[out] = next()
				out++
			}
		case cmd == lcwFill:
			count := word()
			b := next()
			for ; count > 0 && out < len(dst); count-- {
				dst[out] = b
				out++
			}
		case cmd == lcwLongCopy:
			count := word()
			copyFrom(word(), count)
		default:
			count := int(cmd&lcwCountMask) + lcwMinRun
			copyFrom(word(), count)
		}
	}
	return out
}

// DecodeRLE unpacks Westwood RLE into dst until it is full (C: fn_232c_0111): a positive
// count byte n copies n bytes, a negative one repeats the next byte -n times, and 0 is
// followed by a big-endian word count of the repeated byte.
func DecodeRLE(src, dst []byte) {
	in, out := 0, 0
	next := func() byte {
		if in >= len(src) {
			return 0
		}
		in++
		return src[in-1]
	}
	for out < len(dst) {
		n := int(int8(next()))
		literal := n > 0
		if n == 0 {
			n = int(next())<<8 | int(next())
		} else if n < 0 {
			n = -n
		}
		n = min(n, len(dst)-out)
		if literal {
			for ; n > 0; n-- {
				dst[out] = next()
				out++
			}
			continue
		}
		b := next()
		for ; n > 0; n-- {
			dst[out] = b
			out++
		}
	}
}
