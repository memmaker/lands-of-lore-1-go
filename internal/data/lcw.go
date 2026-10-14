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

// LCW encoder limits (C: fn_20f7_0006).
const (
	lcwRunCheck       = 64 // a fill is only tried when the byte this far ahead is the same
	lcwMinFill        = 65 // shorter runs are left to the copies
	lcwMaxShortCopy   = 10 // the longest copy of the relative form
	lcwMaxShortBack   = 1<<12 - 1
	lcwMaxMediumCopy  = lcwCountMask + lcwMinRun
	lcwMaxLiteralByte = lcwLiteralFlag | lcwCountMask
)

// EncodeLCW packs src as LCW the way the original's compressor does, byte for byte
// (C: fn_20f7_0006). It searches the whole input before each position for the longest
// match, the last one of equal length winning; copies use absolute offsets but for
// short and near ones.
func EncodeLCW(src []byte) []byte {
	end := len(src)
	if end == 0 {
		return []byte{lcwLiteralFlag | 1, 0, lcwEnd} // sic: one byte is always copied
	}
	out := []byte{lcwLiteralFlag | 1, src[0]}
	literalAt := 0      // the count byte of the open literal run (C: cs:1a8)
	literalOpen := true // the last command was a literal (C: cs:1a2)
	si := 1
	for si < end {
		best, bestAt := 1, 0 // (C: cs:1ae, cs:1b0)
		di := 0
		for {
			b := src[si]
			if si+lcwRunCheck < end && src[si+lcwRunCheck] == b {
				n := 0
				for si+n < end && src[si+n] == b {
					n++
				}
				if si+n == end {
					n-- // sic: a run up to the end leaves its last byte out
				}
				if n >= lcwMinFill {
					literalOpen = false
					si += n
					out = append(out, lcwFill, byte(n), byte(n>>8), b)
					continue // sic: the fill keeps the last search's best match
				}
			}
			// find b again in the input from di on, then measure the match there
			found := false
			for di < si {
				pos := di
				di++
				if src[pos] != b {
					continue
				}
				// sic: after a fill best is the one from before it and may reach past
				// the end, where the original compares whatever lies behind the input
				if si+best-1 >= end || src[si+best-1] != src[pos+best-1] {
					continue
				}
				found = true
				n := 0
				for si+n < end && src[si+n] == src[pos+n] {
					n++
				}
				if n >= best {
					best, bestAt = n, pos
				}
				break
			}
			if !found {
				break
			}
		}
		switch {
		case best <= 2:
			if !literalOpen || out[literalAt] == lcwMaxLiteralByte {
				literalAt = len(out)
				out = append(out, lcwLiteralFlag)
			}
			out[literalAt]++
			out = append(out, src[si])
			si++
			literalOpen = true
			continue
		case best <= lcwMaxShortCopy && si-bestAt <= lcwMaxShortBack:
			back := si - bestAt
			out = append(out, byte((best-lcwMinRun)<<nibbleBits|back>>8), byte(back))
		case best <= lcwMaxMediumCopy:
			out = append(out, byte(lcwLiteralFlag|lcwAbsoluteBit|(best-lcwMinRun)), byte(bestAt), byte(bestAt>>8))
		default:
			out = append(out, lcwLongCopy, byte(best), byte(best>>8), byte(bestAt), byte(bestAt>>8))
		}
		si += best
		literalOpen = false
	}
	return append(out, lcwEnd)
}
