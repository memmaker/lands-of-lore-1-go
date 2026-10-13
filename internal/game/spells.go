package game

// spell is a spell's definition (C: 14-word rows of f_27dd_0010, read by fn_0845_122b
// through f_27dd_0000 eight words earlier).
type spell struct {
	Name    int // string id
	MPCost  [spellLevels]int
	Field5  int
	Field6  int
	HPCost  [spellLevels]int // hit points a level needs more than (C: words 7-10)
	Field11 int
	Field12 int
	Flags   int // (C: word 13)
}

const spellLevels = 4

// spells is the spell table; the original sized it for 17 rows and filled 10
// (C: f_27dd_0010).
var spells = [...]spell{
	{Name: 16436, MPCost: [spellLevels]int{5, 10, 15, 25}, Flags: 256},
	{Name: 16437, MPCost: [spellLevels]int{5, 10, 25, 60}, Flags: 1},
	{Name: 16438, MPCost: [spellLevels]int{8, 15, 30, 80}, Flags: 258},
	{Name: 16440, MPCost: [spellLevels]int{20, 35, 65, 100}, Flags: 259},
	{Name: 16441, MPCost: [spellLevels]int{30, 60, 90, 120}, Flags: 260},
	{Name: 16442, MPCost: [spellLevels]int{30, 60, 90, 150}, Flags: 261},
	{Name: 16439, MPCost: [spellLevels]int{25, 30, 50, 100}, Flags: 262},
	{Name: 16443, MPCost: [spellLevels]int{3, 6, 12, 24}, Flags: 263},
	{Name: 16975, MPCost: [spellLevels]int{15, 15, 15, 15}, Flags: 264},
	{Name: 16975, MPCost: [spellLevels]int{15, 15, 15, 15}, Field5: 15, Flags: 264},
	{}, {}, {}, {}, {}, {}, {},
}

// castableLevels is how many of a spell's levels character c has the points for
// (C: fn_0845_122b).
func (g *Game) castableLevels(c *Character, sp int) int {
	if sp == noSpell { // fix: the original read the row before the table
		return 0
	}
	n := 0
	for i := range spellLevels {
		if spells[sp].MPCost[i] <= c.MP && spells[sp].HPCost[i] < c.HP {
			n++
		}
	}
	return n
}
