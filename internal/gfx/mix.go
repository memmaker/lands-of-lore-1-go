package gfx

// MixTable maps every colour to the palette entry nearest to it moved a share of the way
// towards colour target; amount is that share in 256ths (at most 255) (C: fn_1b43_000e).
// Colour 0 stays 0; only entries 1-254 are candidates, the colour itself excluded, and
// on equal distances the later entry wins. The 8-bit arithmetic is the original's.
func MixTable(pal []byte, target, amount int) []byte {
	return mixTable(pal, target, amount, colours-2)
}

// MixTableLow is MixTable choosing only among entries 1-127, the level's own colours
// (C: fn_1b43_0109).
func MixTableLow(pal []byte, target, amount int) []byte {
	return mixTable(pal, target, amount, colours/2-1)
}

const colours = 256

func mixTable(pal []byte, target, amount, lastChoice int) []byte {
	const (
		maxAmount    = colours - 1
		firstChoice  = 1
		farthest     = 1<<15 - 1
		byteBits     = 8
		componentsOf = 3
	)
	table := make([]byte, colours)
	if pal == nil {
		return table
	}
	if amount >= colours {
		amount = maxAmount
	}
	weight := int8(amount >> 1)
	toward := pal[target*componentsOf : target*componentsOf+componentsOf]
	for c := 1; c < colours; c++ {
		var mixed [componentsOf]byte
		for k := range componentsOf {
			v := pal[c*componentsOf+k]
			step := uint16(int16(int8(v-toward[k]))*int16(weight)) << 1
			mixed[k] = v - byte(step>>byteBits)
		}
		best := byte(target)
		nearest := uint16(farthest)
		for candidate := firstChoice; candidate <= lastChoice; candidate++ {
			if candidate == c {
				continue
			}
			var distance uint16
			for k := range componentsOf {
				d := int16(int8(pal[candidate*componentsOf+k] - mixed[k]))
				distance += uint16(d * d)
			}
			if distance == 0 {
				best = byte(candidate)
				break
			}
			if distance <= nearest {
				nearest = distance
				best = byte(candidate)
			}
		}
		table[c] = best
	}
	return table
}

// NearestColour is the entry among the first n of pal closest to rgb, the later one on
// equal distances; with skipUI colours 192-195 are left out (C: fn_19c9_12a2). The
// 8-bit arithmetic is the original's, and so is its answer when every distance passes
// 32767: entry 257, which a byte keeps as 1.
func NearestColour(rgb, pal []byte, n int, skipUI bool) int {
	const (
		farthest    = 1<<15 - 1
		noneFound   = 257
		firstUI     = 192
		lastUI      = 195
		componentsN = 3
	)
	best, nearest := noneFound, uint16(farthest)
	for c := range n {
		if skipUI && c >= firstUI && c <= lastUI {
			// fix: the original lost its place in both tables here; no caller reaches it.
			continue
		}
		var d uint16
		for k := range componentsN {
			v := int16(int8(rgb[k] - pal[c*componentsN+k]))
			d += uint16(v * v)
		}
		if d <= nearest {
			nearest, best = d, c
		}
	}
	return best
}
