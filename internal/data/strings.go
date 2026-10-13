package data

import (
	"bytes"
	"encoding/binary"
)

// Westwood's text compression packs a common letter and one of its eight most common
// followers into one byte: 1 kkkk jjj, k indexing commonChars and j its followers
// (C: g_38b0, g_38c0).
const commonChars = " etainosrlhcdupm"

var commonFollowers = [len(commonChars)]string{
	"tasio wb", " rnsdalm", "h ieoras", "nrtlc sy", "nstcloer", " dtgesio", "nr ufmsw", " tep.ica",
	"e oiadur", " laeiyod", "eia otru", "etoakhlr", " eiu,.oa", "nsrctlai", "leoiratp", "eaoip bm",
}

const (
	packedBit     = 1 << 7
	followerBits  = 3
	followerMask  = 1<<followerBits - 1
	sevenBitMask  = 1<<7 - 1
	escapeByte    = 27
	escapedOffset = 127
)

// Unpack expands Westwood-compressed text (C: fn_212a_00e5).
func Unpack(src []byte) []byte {
	out := make([]byte, 0, 2*len(src))
	for _, c := range src {
		if c == 0 {
			break
		}
		if c&packedBit != 0 {
			c &= sevenBitMask
			k := c >> followerBits
			out = append(out, commonChars[k])
			c = commonFollowers[k][c&followerMask]
		}
		out = append(out, c)
	}
	return out
}

// Unescape turns ESC c into c+127, the way text reaches the printer (C: fn_212a_0005).
func Unescape(src []byte) []byte {
	out := make([]byte, 0, len(src))
	for i := 0; i < len(src) && src[i] != 0; i++ {
		c := src[i]
		if c == escapeByte && i+1 < len(src) {
			i++
			c = src[i] + escapedOffset
		}
		out = append(out, c)
	}
	return out
}

// TableString returns string idx of a table that starts with a word offset per string,
// without its terminating zero; nil if there is none (C: fn_212a_014c).
func TableString(table []byte, idx int) []byte {
	if table == nil || idx < 0 || 2*idx+2 > len(table) {
		return nil
	}
	off := int(binary.LittleEndian.Uint16(table[2*idx:]))
	if off >= len(table) {
		return nil
	}
	s := table[off:]
	if end := bytes.IndexByte(s, 0); end >= 0 {
		s = s[:end]
	}
	return s
}
