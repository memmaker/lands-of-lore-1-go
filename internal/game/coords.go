package game

// The level is a 32 x 32 grid of blocks; positions are 256 units per block, a block
// number holding the row in its upper and the column in its lower 5 bits (C: s040).
const (
	mapSize        = 32
	blockCount     = mapSize * mapSize
	blockMask      = blockCount - 1
	blockUnits     = 256
	blockCentre    = blockUnits / 2
	columnMask     = mapSize - 1
	byteShift      = 8
	positionMask   = 1<<13 - 1
	fixedOne       = 1 << byteShift // 1.0 in the 8.8 scale factors
	directionCount = 4
)

// blockPosition is the map position of offset (dx, dy) inside a block
// (C: fn_1aff_030e).
func blockPosition(block, dx, dy int) (x, y int) {
	x = (block&columnMask)<<byteShift | dx
	y = (block&^columnMask)<<3 | dy
	return x, y
}

// positionBlock is the block a map position lies in (C: fn_1aff_039a).
func positionBlock(x, y int) int {
	return (y>>byteShift)<<5 | (x>>byteShift)&columnMask
}

// scale multiplies by an 8.8 factor, unsigned, keeping 16 bits (C: fn_1aff_0420).
func scale(v, factor int) int {
	return int(uint32(uint16(v))*uint32(uint16(factor))>>byteShift) & (1<<16 - 1)
}

// scaleSigned is scale with a signed factor (C: fn_1aff_03fd).
func scaleSigned(v, factor int) int {
	r := int(int16(uint32(uint16(v)) * uint32(uint16(abs(factor))) >> byteShift))
	if factor < 0 {
		r = -r
	}
	return int(int16(r))
}

// ratio is a / b as an 8.8 factor (C: fn_1aff_0434). The original's division faulted on 0.
func ratio(a, b int) int {
	if b == 0 { // fix: a division by zero ended the program
		return 0
	}
	return int(uint32(uint16(a))<<byteShift/uint32(uint16(b))) & (1<<16 - 1)
}

// blockDistance is the rough distance between two blocks, the longer axis plus half the
// shorter (C: fn_1aff_02d2).
func blockDistance(a, b int) int {
	ax, ay := a&columnMask, a>>5&columnMask
	bx, by := b&columnMask, b>>5&columnMask
	dx, dy := abs(bx-ax), abs(by-ay)
	if dx >= dy {
		dx, dy = dy, dx
	}
	return dx>>1 + dy
}

// directions8 is the direction from the bits of directionTo: target below, target to
// the left, mostly vertical, clearly more along one axis (C: s039's table).
var directions8 = [16]int{1, 2, 1, 0, 7, 6, 7, 0, 3, 2, 3, 4, 5, 6, 5, 4}

// directionTo is the 8-way direction (0 north, clockwise) from (x1, y1) to (x2, y2)
// (C: fn_1afa_0010).
func directionTo(x1, y1, x2, y2 int) int {
	i := 0
	dy := int(int16(y1 - y2))
	if dy < 0 {
		i, dy = 1, -dy
	}
	i <<= 1
	dx := int(int16(x2 - x1))
	if dx < 0 {
		i, dx = i+1, -dx
	}
	minor, major := dy, dx
	vertical := uint16(dx) < uint16(dy)
	if vertical {
		minor, major = dx, dy
	}
	i <<= 1
	if vertical {
		i++
	}
	i <<= 1
	if uint16(minor) < uint16((major+1)>>1) {
		i++
	}
	return directions8[i]
}
