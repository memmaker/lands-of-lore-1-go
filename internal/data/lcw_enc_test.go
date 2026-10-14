package data

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestEncodeLCWRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for n := range 200 {
		src := make([]byte, 1+r.Intn(6000))
		for i := range src {
			switch r.Intn(4) {
			case 0:
				src[i] = byte(r.Intn(4))
			case 1:
				if i > 0 {
					src[i] = src[i-1]
				}
			default:
				src[i] = byte(r.Intn(256))
			}
		}
		if n%3 == 0 {
			clear(src[len(src)/3:])
		}
		enc := EncodeLCW(src)
		dst := make([]byte, len(src))
		if k := DecodeLCW(enc, dst); k != len(src) || !bytes.Equal(dst, src) {
			t.Fatalf("case %d: round trip failed (%d of %d bytes)", n, k, len(src))
		}
	}
}
