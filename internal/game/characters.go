package game

// The party (C: f_2803_362d[4]).
const (
	partySize      = 4
	equipmentSlots = 11
	skillCount     = 3
	updateSlots    = 5
	itemStatSlots  = 8
)

// Character flags.
const (
	charPresent = 1 << 0
	charDead    = 1 << 3
	charMute    = 1 << 12
)

// Character is a party member (C: struct Char, 130 bytes; the names follow ScummVM's
// LoLCharacter, which has the same layout).
type Character struct {
	Flags        int
	Name         string
	RaceClassSex int
	ID           int // negative for the four heroes one can start with
	Face         int // current face frame
	FaceDefault  int
	Scream       int // sound effect when hit
	Modifiers    int // index into characterModifiers (C: a far pointer)

	ItemsMight              [itemStatSlots]int
	ProtectionAgainstItems  [itemStatSlots]int
	ItemProtection          int
	HP, MaxHP               int
	MP, MaxMP               int
	Field41                 int
	DamageSuffered          int
	WeaponHit               int
	TotalMightModifier      int
	TotalProtectionModifier int
	Might, Protection       int
	AnimCountdown           int // ticks to the next face animation
	Items                   [equipmentSlots]int
	SkillLevels             [skillCount]int // fighter, rogue, mage
	SkillModifiers          [skillCount]int
	Experience              [skillCount]int
	UpdateEvents            [updateSlots]int // status effects that run out (C: timer)
	UpdateDelays            [updateSlots]int // (C: timerType)
}

// characterModifiers are the default modifier sets the templates point to
// (C: f_27b1_01c2, 5 sets of 9 words; ScummVM: _charDefsMan and the others).
var characterModifiers = [][9]int{
	{256, 256, 256, 50, 256, 256, 256, 256, 4},
	{256, 256, 256, 50, 256, 256, 256, 256, 4},
	{384, 128, 192, 50, 512, 166, 256, 320, 6},
	{256, 256, 256, 50, 256, 256, 256, 256, 4},
	{192, 192, 320, 50, 256, 384, 384, 256, 4},
}

// characterTemplates are the characters who can join (C: f_2656_0000).
var characterTemplates = [...]Character{
	{
		Flags: 1, Name: "Michael", RaceClassSex: 0, ID: -1,
		Face: 0, FaceDefault: 0, Scream: 17, Modifiers: 0,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 20, MaxHP: 20, MP: 6, MaxMP: 6,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 384, TotalProtectionModifier: 320, Might: 5, Protection: 2,
		AnimCountdown: 0, Items: [equipmentSlots]int{44, 0, 0, 0, 0, 138, 0, 0, 162, 0, 0},
		SkillLevels: [skillCount]int{1, 1, 1}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{1, 1, 1},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Timothy", RaceClassSex: 0, ID: 2,
		Face: 0, FaceDefault: 0, Scream: 22, Modifiers: 0,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 27, MaxHP: 27, MP: 5, MaxMP: 5,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 256, TotalProtectionModifier: 256, Might: 0, Protection: 0,
		AnimCountdown: 0, Items: [equipmentSlots]int{93, 131, 0, 0, 0, 153, 0, 0, 162, 0, 0},
		SkillLevels: [skillCount]int{3, 2, 1}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{2500, 1000, 250},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Baccata", RaceClassSex: 3, ID: 3,
		Face: 0, FaceDefault: 0, Scream: 23, Modifiers: 0,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 22, MaxHP: 22, MP: 18, MaxMP: 18,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 256, TotalProtectionModifier: 256, Might: 0, Protection: 0,
		AnimCountdown: 0, Items: [equipmentSlots]int{108, 149, 0, 0, 145, 153, 0, 0, 129, 0, 0},
		SkillLevels: [skillCount]int{1, 1, 3}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{1, 1, 4000},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Lora", RaceClassSex: 1, ID: 4,
		Face: 0, FaceDefault: 0, Scream: 24, Modifiers: 1,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 25, MaxHP: 48, MP: 5, MaxMP: 5,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 256, TotalProtectionModifier: 256, Might: 0, Protection: 0,
		AnimCountdown: 0, Items: [equipmentSlots]int{0, 0, 0, 0, 0, 137, 0, 0, 162, 0, 0},
		SkillLevels: [skillCount]int{1, 4, 1}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{1, 15001, 1},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Conrad", RaceClassSex: 0, ID: -5,
		Face: 0, FaceDefault: 0, Scream: 26, Modifiers: 0,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 23, MaxHP: 23, MP: 10, MaxMP: 10,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 366, TotalProtectionModifier: 358, Might: 2, Protection: 3,
		AnimCountdown: 0, Items: [equipmentSlots]int{44, 0, 0, 0, 0, 138, 0, 0, 162, 0, 0},
		SkillLevels: [skillCount]int{1, 1, 1}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{1, 1, 1},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Paulson", RaceClassSex: 0, ID: 6,
		Face: 0, FaceDefault: 0, Scream: 25, Modifiers: 0,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 105, MaxHP: 105, MP: 45, MaxMP: 45,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 256, TotalProtectionModifier: 256, Might: 0, Protection: 0,
		AnimCountdown: 0, Items: [equipmentSlots]int{0, 0, 0, 0, 0, 138, 0, 0, 129, 0, 0},
		SkillLevels: [skillCount]int{5, 2, 3}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{40000, 501, 3500},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Dawn", RaceClassSex: 1, ID: 7,
		Face: 0, FaceDefault: 0, Scream: 24, Modifiers: 1,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 50, MaxHP: 50, MP: 80, MaxMP: 80,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 256, TotalProtectionModifier: 256, Might: 0, Protection: 0,
		AnimCountdown: 0, Items: [equipmentSlots]int{0, 0, 0, 0, 0, 138, 0, 0, 129, 0, 0},
		SkillLevels: [skillCount]int{1, 2, 10}, SkillModifiers: [skillCount]int{1, -2, -79},
		Experience:   [skillCount]int{1, 501, 1},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Kieran", RaceClassSex: 2, ID: -8,
		Face: 0, FaceDefault: 0, Scream: 21, Modifiers: 2,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 23, MaxHP: 23, MP: 8, MaxMP: 8,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 422, TotalProtectionModifier: 332, Might: 5, Protection: 1,
		AnimCountdown: 0, Items: [equipmentSlots]int{44, 0, 0, 0, 0, 138, 0, 0, 162, 0, 0},
		SkillLevels: [skillCount]int{1, 1, 1}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{1, 1, 1},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
	{
		Flags: 1, Name: "Ak'shel", RaceClassSex: 4, ID: -9,
		Face: 0, FaceDefault: 0, Scream: 20, Modifiers: 4,
		ItemsMight:             [8]int{0, 0, 0, 0, 0, 0, 0, 0},
		ProtectionAgainstItems: [8]int{256, 256, 256, 256, 256, 256, 256, 256},
		ItemProtection:         0, HP: 28, MaxHP: 28, MP: 32, MaxMP: 32,
		Field41: 0, DamageSuffered: 0, WeaponHit: 0,
		TotalMightModifier: 320, TotalProtectionModifier: 256, Might: 1, Protection: 1,
		AnimCountdown: 0, Items: [equipmentSlots]int{44, 0, 0, 0, 0, 138, 0, 0, 162, 0, 0},
		SkillLevels: [skillCount]int{1, 1, 1}, SkillModifiers: [skillCount]int{0, 0, 0},
		Experience:   [skillCount]int{1, 1, 1},
		UpdateEvents: [updateSlots]int{0, 0, 0, 0, 0}, UpdateDelays: [updateSlots]int{0, 0, 0, 0, 0},
	},
}
