package game

import (
	"fmt"

	"github.com/memmaker/lands-of-lore-1-go/internal/emc"
)

// opcodeCount is the size of the game's opcode table (C: g_2494).
const opcodeCount = 192

// Opcodes with known numbers.
const (
	opcodeSetGameFlag    = 7
	opcodeTestGameFlag   = 8
	opcodeAllocItemTypes = 13
	opcodeDefineItemType = 14
	opcodeMakeItem       = 15
	opcodeGetItemField   = 18
	opcodeGetCharField   = 19
	opcodeSetCharField   = 20
	opcodeNothing        = 23
	opcodeCharJoins      = 75
	opcodeSetItemField   = 171
	opcodeNothingToo     = 191

	opcodeDice                   = 3
	opcodeLoadGraphics           = 9
	opcodeLoadMaze               = 10
	opcodeLoadMonsterShapes      = 11
	opcodeOpenDecorations        = 21
	opcodeCloseDecorations       = 22
	opcodeLoadDoorShapes         = 24
	opcodeSetLevelVar            = 46
	opcodeGetEngineVar           = 47
	opcodeSetEngineVar           = 48
	opcodeAssignDecoration       = 53
	opcodeDoorDecoration         = 54
	opcodePlaceMonster           = 56
	opcodeMonsterType            = 63
	opcodeGiveMonsterItem        = 85
	opcodeLevelStrings           = 86
	opcodeQueueTrack             = 99
	opcodeBrightness             = 107
	opcodePendingBlockScript     = 124
	opcodeLoadTIM                = 78
	opcodeOpenAnimation          = 25
	opcodeCloseAnimation         = 27
	opcodeRunTIM                 = 79
	opcodeFreeTIM                = 80
	opcodeStopTIM                = 89
	opcodeStartTIMDialogue       = 135
	opcodeLoadSoundFile          = 148
	opcodeSceneOn                = 154
	opcodeMoveMonster            = 68
	opcodeSay                    = 122
	opcodeSetWall                = 0
	opcodeDrawScene              = 2
	opcodeMoveParty              = 4
	opcodePrintMessage           = 5
	opcodeWait                   = 6
	opcodeTakeHeldItem           = 12
	opcodeToggleWall             = 52
	opcodeGas                    = 182
	opcodeDarkLite               = 177
	opcodeGameEnd                = 174
	opcodeDrawLabel              = 98
	opcodeBezelHeal              = 170
	opcodeFreeMonsterShapes      = 93
	opcodeSoundLoaded            = 149
	opcodeSetSceneOverlay        = 164
	opcodeCastSpell              = 173
	opcodeFlashPalette           = 176
	opcodeBusyCursor             = 186
	opcodeChangeLevel            = 60
	opcodeSpellTooCostly         = 84
	opcodeLearnSpell             = 120
	opcodeSaveSparePage          = 138
	opcodeRestoreSparePage       = 139
	opcodeOpenSceneBox           = 140
	opcodeCloseSceneBox          = 141
	opcodeOpenSceneArea          = 142
	opcodeCloseSceneArea         = 143
	opcodeHideControls           = 144
	opcodeShowControls           = 145
	opcodeBeginFullScene         = 146
	opcodeEndFullScene           = 147
	opcodeSceneIsOn              = 156
	opcodeRaiseLevel             = 175
	opcodeSetPoolStart           = 178
	opcodeLockControls           = 179
	opcodeUnlockControls         = 180
	opcodeShakeScreen            = 181
	opcodeBlockInDirection2      = 183
	opcodeDissolveScene          = 184
	opcodeCopyViewFrame          = 185
	opcodeSpeak                  = 187
	opcodeQueueSpeech            = 188
	opcodeRoundPrice             = 189
	opcodeLanguage               = 190
	opcodeEnableClock            = 161
	opcodeDisableClock           = 162
	opcodeLastSequence           = 163
	opcodeWhoCarries             = 165
	opcodeRestoreLevelColours    = 166
	opcodeBlockInDirection       = 167
	opcodeSelectedCharacter      = 168
	opcodeSetHeldItem            = 169
	opcodeTakeFromInventory      = 172
	opcodeNextMonsterOn          = 150
	opcodeMonsterOn              = 151
	opcodeDissolve               = 152
	opcodeTurnInBlock            = 153
	opcodeNextCharacter          = 157
	opcodeAfflict                = 158
	opcodeDrawPortraits          = 159
	opcodeRemoveItemOfType       = 160
	opcodeTrue                   = 125
	opcodeQuitGame               = 126
	opcodeRemoveMonster          = 127
	opcodeSetScriptValue         = 128
	opcodeKeyPressed             = 129
	opcodePrintInWindow          = 130
	opcodeCountMonstersOfTypes   = 131
	opcodeAnimateWall            = 132
	opcodeHasItemType            = 133
	opcodeSetDoorHandled         = 134
	opcodeFindItem               = 136
	opcodeTakeItemFromBlock      = 137
	opcodeCountLiveMonsters      = 104
	opcodeFinale                 = 105
	opcodeStopTalk               = 106
	opcodeAttackDamage           = 108
	opcodeScaledDamage           = 109
	opcodeInParty                = 110
	opcodeMessage                = 111
	opcodeDestroyItem            = 112
	opcodeDamageOfKind           = 113
	opcodeLaunchObject           = 114
	opcodeClearStatus            = 115
	opcodeInventoryFull          = 116
	opcodeMoveBlockContents      = 117
	opcodeLamp                   = 118
	opcodeDrawGameScreen         = 119
	opcodePrintString            = 121
	opcodeHeldItem               = 83
	opcodePlaySound              = 87
	opcodeDialogueAnswer         = 88
	opcodeSetMonsterField        = 91
	opcodeGetMonsterField        = 92
	opcodeCharacterSays          = 94
	opcodeIdle                   = 95
	opcodePlaySoundAt            = 96
	opcodeHeal                   = 97
	opcodePlayMusic              = 100
	opcodeKillMonstersOn         = 101
	opcodeCountItemsOn           = 102
	opcodeSkillTest              = 103
	opcodeCopyRect               = 55
	opcodeFadeOutView            = 57
	opcodeFadeInLevel            = 58
	opcodeRedrawAfterScene       = 59
	opcodeMonsterInFront         = 61
	opcodeNothingZero            = 62
	opcodeAttackHits             = 64
	opcodeDamage                 = 65
	opcodeDialogueChoices        = 69
	opcodeCredits                = 70
	opcodeGiveHeldItem           = 73
	opcodeItemSound              = 74
	opcodeGiveInventoryItem      = 76
	opcodeAddExperience          = 77
	opcodePlaceItem              = 16
	opcodeCreateItem             = 17
	opcodePlayAnimationFrames    = 26
	opcodePartySound             = 29
	opcodeMouseIn                = 33
	opcodeSetAnimationSequence   = 35
	opcodeStartAnimationSequence = 36
	opcodeFadeToScenePalette     = 40
	opcodeLoadPicture            = 41
	opcodeStopAnimation          = 42
	opcodeAskAnimationStop       = 43
	opcodePlayAnimationOnce      = 44
	opcodeToggleDoor             = 49
	opcodePartyHasWeapon         = 50
	opcodeMarkAutomap            = 51
	opcodeAnyMonsterAwake        = 123
	opcodeClearDialogue          = 34
	opcodeSceneOff               = 155
	opcodeStartScene             = 81
	opcodeEndScene               = 82
	opcodeTextBox                = 66
	opcodeUnrollTextBox          = 67
	opcodeSceneRegion            = 31
	opcodeResetSceneRegion       = 32
	opcodeGetWall                = 1
	opcodeGetDirection           = 28
	opcodeHideMouse              = 37
	opcodeShowMouse              = 38
	opcodeFadeToBlack            = 39
	opcodeGetLevelVar            = 45
	opcodeHasCredits             = 71
	opcodeSetLevelTimer          = 72
	opcodeGetWallFlags           = 90
	opcodeSetLevelMusic          = 30
)

// opcodes is the table level, item and start-up scripts call into (C: g_2494). Entries
// still nil are ported along with the systems they drive; a script reaching one stops
// with an error.
func (g *Game) opcodes() []emc.Opcode {
	ops := make([]emc.Opcode, opcodeCount)
	ops[opcodeAllocItemTypes] = g.opAllocItemTypes
	ops[opcodeDefineItemType] = g.opDefineItemType
	ops[opcodeSetGameFlag] = g.opSetGameFlag
	ops[opcodeTestGameFlag] = g.opTestGameFlag
	ops[opcodeMakeItem] = g.opMakeItem
	ops[opcodeGetItemField] = g.opGetItemField
	ops[opcodeSetItemField] = g.opSetItemField
	ops[opcodeGetCharField] = g.opGetCharacterField
	ops[opcodeSetCharField] = g.opSetCharacterField
	ops[opcodeCharJoins] = g.opCharacterJoinsOrLeaves
	ops[opcodeNothing] = opNothing
	ops[opcodeNothingToo] = opNothing
	ops[opcodeDice] = g.opDice
	ops[opcodeLoadGraphics] = g.opLoadGraphics
	ops[opcodeLoadMaze] = g.opLoadMaze
	ops[opcodeLoadMonsterShapes] = g.opLoadMonsterShapes
	ops[opcodeOpenDecorations] = g.opOpenDecorations
	ops[opcodeCloseDecorations] = g.opCloseDecorations
	ops[opcodeLoadDoorShapes] = g.opLoadDoorShapes
	ops[opcodeSetLevelVar] = g.opSetLevelVar
	ops[opcodeGetEngineVar] = g.opGetEngineVar
	ops[opcodeSetEngineVar] = g.opSetEngineVar
	ops[opcodeAssignDecoration] = g.opAssignDecoration
	ops[opcodeDoorDecoration] = g.opSetDoorDecoration
	ops[opcodePlaceMonster] = g.opPlaceMonster
	ops[opcodeMonsterType] = g.opDefineMonsterType
	ops[opcodeGiveMonsterItem] = g.opGiveMonsterItem
	ops[opcodeLevelStrings] = g.opLoadLevelStrings
	ops[opcodeQueueTrack] = g.opQueueTrack
	ops[opcodeBrightness] = g.opSetBrightness
	ops[opcodePendingBlockScript] = g.opSetPendingBlockScript
	ops[opcodeLoadSoundFile] = g.opLoadSoundFile
	ops[opcodeSceneOn] = g.opSceneOn
	ops[opcodeMoveMonster] = g.opMoveMonster
	ops[opcodeSay] = g.opSay
	ops[opcodeSetWall] = g.opSetWall
	ops[opcodeDrawScene] = g.opDrawScene
	ops[opcodeMoveParty] = g.opMoveParty
	ops[opcodePrintMessage] = g.opPrintMessage
	ops[opcodeWait] = g.opWait
	ops[opcodeTakeHeldItem] = g.opTakeHeldItem
	ops[opcodeToggleWall] = g.opToggleWall
	ops[opcodeGas] = g.opGas
	ops[opcodeDarkLite] = g.opDarkLite
	ops[opcodeGameEnd] = g.opGameEnd
	ops[opcodeDrawLabel] = g.opDrawLabel
	ops[opcodeBezelHeal] = g.opBezelHeal
	ops[opcodeFreeMonsterShapes] = g.opFreeMonsterShapes
	ops[opcodeSoundLoaded] = g.opSoundLoaded
	ops[opcodeSetSceneOverlay] = g.opSetSceneOverlay
	ops[opcodeCastSpell] = g.opCastSpell
	ops[opcodeFlashPalette] = g.opFlashPalette
	ops[opcodeBusyCursor] = g.opBusyCursor
	ops[opcodeChangeLevel] = g.opChangeLevel
	ops[opcodeSpellTooCostly] = g.opSpellTooCostly
	ops[opcodeLearnSpell] = g.opLearnSpell
	ops[opcodeSaveSparePage] = g.opSaveSparePage
	ops[opcodeRestoreSparePage] = g.opRestoreSparePage
	ops[opcodeOpenSceneBox] = g.opOpenSceneBox
	ops[opcodeCloseSceneBox] = g.opCloseSceneBox
	ops[opcodeOpenSceneArea] = g.opOpenSceneArea
	ops[opcodeCloseSceneArea] = g.opCloseSceneArea
	ops[opcodeHideControls] = g.opHideControls
	ops[opcodeShowControls] = g.opShowControls
	ops[opcodeBeginFullScene] = g.opBeginFullScene
	ops[opcodeEndFullScene] = g.opEndFullScene
	ops[opcodeSceneIsOn] = g.opSceneIsOn
	ops[opcodeRaiseLevel] = g.opRaiseLevel
	ops[opcodeSetPoolStart] = g.opSetPoolStart
	ops[opcodeLockControls] = g.opLockControls
	ops[opcodeUnlockControls] = g.opUnlockControls
	ops[opcodeShakeScreen] = g.opShakeScreen
	ops[opcodeBlockInDirection2] = g.opBlockInDirection // C: fn_0fbe_3218, the same as 167
	ops[opcodeDissolveScene] = g.opDissolveScene
	ops[opcodeCopyViewFrame] = g.opCopyViewFrame
	ops[opcodeSpeak] = g.opSpeak
	ops[opcodeQueueSpeech] = g.opQueueSpeech
	ops[opcodeRoundPrice] = g.opRoundPrice
	ops[opcodeLanguage] = g.opLanguage
	ops[opcodeEnableClock] = g.opEnableClock
	ops[opcodeDisableClock] = g.opDisableClock
	ops[opcodeLastSequence] = g.opLastSequence
	ops[opcodeWhoCarries] = g.opWhoCarries
	ops[opcodeRestoreLevelColours] = g.opRestoreLevelColours
	ops[opcodeBlockInDirection] = g.opBlockInDirection
	ops[opcodeSelectedCharacter] = g.opSelectedCharacter
	ops[opcodeSetHeldItem] = g.opSetHeldItem
	ops[opcodeTakeFromInventory] = g.opTakeFromInventory
	ops[opcodeNextMonsterOn] = g.opNextMonsterOn
	ops[opcodeMonsterOn] = g.opMonsterOn
	ops[opcodeDissolve] = g.opDissolve
	ops[opcodeTurnInBlock] = g.opTurnInBlock
	ops[opcodeNextCharacter] = g.opNextCharacter
	ops[opcodeAfflict] = g.opAfflict
	ops[opcodeDrawPortraits] = g.opDrawPortraits
	ops[opcodeRemoveItemOfType] = g.opRemoveItemOfType
	ops[opcodeTrue] = g.opTrue
	ops[opcodeQuitGame] = g.opQuitGame
	ops[opcodeRemoveMonster] = g.opRemoveMonster
	ops[opcodeSetScriptValue] = g.opSetScriptValue
	ops[opcodeKeyPressed] = g.opKeyPressed
	ops[opcodePrintInWindow] = g.opPrintInWindow
	ops[opcodeCountMonstersOfTypes] = g.opCountMonstersOfTypes
	ops[opcodeAnimateWall] = g.opAnimateWall
	ops[opcodeHasItemType] = g.opHasItemType
	ops[opcodeSetDoorHandled] = g.opSetDoorHandled
	ops[opcodeFindItem] = g.opFindItem
	ops[opcodeTakeItemFromBlock] = g.opTakeItemFromBlock
	ops[opcodeCountLiveMonsters] = g.opCountLiveMonsters
	ops[opcodeFinale] = g.opFinale
	ops[opcodeStopTalk] = g.opStopTalk
	ops[opcodeAttackDamage] = g.opAttackDamage
	ops[opcodeScaledDamage] = g.opScaledDamage
	ops[opcodeInParty] = g.opInParty
	ops[opcodeMessage] = g.opMessage
	ops[opcodeDestroyItem] = g.opDestroyItem
	ops[opcodeDamageOfKind] = g.opDamageOfKind
	ops[opcodeLaunchObject] = g.opLaunchObject
	ops[opcodeClearStatus] = g.opClearStatus
	ops[opcodeInventoryFull] = g.opInventoryFull
	ops[opcodeMoveBlockContents] = g.opMoveBlockContents
	ops[opcodeLamp] = g.opLamp
	ops[opcodeDrawGameScreen] = g.opDrawGameScreen
	ops[opcodePrintString] = g.opPrintString
	ops[opcodeHeldItem] = g.opHeldItem
	ops[opcodePlaySound] = g.opPlaySound
	ops[opcodeDialogueAnswer] = g.opDialogueAnswer
	ops[opcodeSetMonsterField] = g.opSetMonsterField
	ops[opcodeGetMonsterField] = g.opGetMonsterField
	ops[opcodeCharacterSays] = g.opCharacterSays
	ops[opcodeIdle] = g.opIdle
	ops[opcodePlaySoundAt] = g.opPlaySoundAt
	ops[opcodeHeal] = g.opHeal
	ops[opcodePlayMusic] = g.opPlayMusic
	ops[opcodeKillMonstersOn] = g.opKillMonstersOn
	ops[opcodeCountItemsOn] = g.opCountItemsOn
	ops[opcodeSkillTest] = g.opSkillTest
	ops[opcodeCopyRect] = g.opCopyRect
	ops[opcodeFadeOutView] = g.opFadeOutView
	ops[opcodeFadeInLevel] = g.opFadeInLevel
	ops[opcodeRedrawAfterScene] = g.opRedrawAfterScene
	ops[opcodeMonsterInFront] = g.opMonsterInFront
	ops[opcodeNothingZero] = g.opNothingZero
	ops[opcodeAttackHits] = g.opAttackHits
	ops[opcodeDamage] = g.opDamage
	ops[opcodeDialogueChoices] = g.opDialogueChoices
	ops[opcodeCredits] = g.opCredits
	ops[opcodeGiveHeldItem] = g.opGiveHeldItem
	ops[opcodeItemSound] = g.opItemSound
	ops[opcodeGiveInventoryItem] = g.opGiveInventoryItem
	ops[opcodeAddExperience] = g.opAddExperience
	ops[opcodePlaceItem] = g.opPlaceItem
	ops[opcodeCreateItem] = g.opCreateItem
	ops[opcodePlayAnimationFrames] = g.opPlayAnimationFrames
	ops[opcodePartySound] = g.opPartySound
	ops[opcodeMouseIn] = g.opMouseIn
	ops[opcodeSetAnimationSequence] = g.opSetAnimationSequence
	ops[opcodeStartAnimationSequence] = g.opStartAnimationSequence
	ops[opcodeFadeToScenePalette] = g.opFadeToScenePalette
	ops[opcodeLoadPicture] = g.opLoadPicture
	ops[opcodeStopAnimation] = g.opStopAnimation
	ops[opcodeAskAnimationStop] = g.opAskAnimationStop
	ops[opcodePlayAnimationOnce] = g.opPlayAnimationOnce
	ops[opcodeToggleDoor] = g.opToggleDoor
	ops[opcodePartyHasWeapon] = g.opPartyHasWeapon
	ops[opcodeMarkAutomap] = g.opMarkAutomap
	ops[opcodeAnyMonsterAwake] = g.opAnyMonsterAwake
	ops[opcodeClearDialogue] = g.opClearDialogue
	ops[opcodeSceneOff] = g.opSceneOff
	ops[opcodeStartScene] = g.opStartScene
	ops[opcodeEndScene] = g.opEndScene
	ops[opcodeTextBox] = g.opTextBox
	ops[opcodeUnrollTextBox] = g.opUnrollTextBox
	ops[opcodeSceneRegion] = g.opSceneRegion
	ops[opcodeResetSceneRegion] = g.opResetSceneRegion
	ops[opcodeGetWall] = g.opGetWall
	ops[opcodeGetDirection] = g.opGetDirection
	ops[opcodeHideMouse] = g.opHideMouse
	ops[opcodeShowMouse] = g.opShowMouse
	ops[opcodeFadeToBlack] = g.opFadeToBlack
	ops[opcodeGetLevelVar] = g.opGetLevelVar
	ops[opcodeHasCredits] = g.opHasCredits
	ops[opcodeSetLevelTimer] = g.opSetLevelTimer
	ops[opcodeGetWallFlags] = g.opGetWallFlags
	ops[opcodeSetLevelMusic] = g.opSetLevelMusic
	ops[opcodeLoadTIM] = g.opLoadTIM
	ops[opcodeOpenAnimation] = g.opOpenAnimation
	ops[opcodeCloseAnimation] = g.opCloseAnimation
	ops[opcodeRunTIM] = g.opRunTIM
	ops[opcodeFreeTIM] = g.opFreeTIM
	ops[opcodeStopTIM] = g.opStopTIM
	ops[opcodeStartTIMDialogue] = g.opStartTIMDialogue
	return ops
}

// withStubs fills the opcodes not ported yet with ones that fail loudly with their
// original's name.
func withStubs(ops []emc.Opcode) []emc.Opcode {
	for i, op := range ops {
		if op == nil && opcodeNames[i] != "" {
			name := opcodeNames[i]
			ops[i] = func(*emc.State) int {
				notPorted(name + " (opcode)")
				return 0
			}
		}
	}
	return ops
}

// opNothing is an empty opcode (C: fn_0fbe_004c).
func opNothing(*emc.State) int { return 0 }

// loadScript reads an EMC file with the game's opcodes.
func (g *Game) loadScript(name string) (*emc.Script, error) {
	file, err := g.Files.Read(name)
	if err != nil {
		return nil, err
	}
	s, err := emc.Load(file, withStubs(g.opcodes()))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return s, nil
}

// runFunction runs a function of a script to its end (C: fn_0fbe_000c on the level
// script slot).
func runFunction(s *emc.Script, function int) error {
	var st emc.State
	st.Init(s)
	st.Start(function)
	st.Run()
	return st.Err
}

// runScriptOnce loads a script into the level script slot, runs its function 0 and then
// function, if not 0, and drops it again (C: ovr_2d2c_007f).
func (g *Game) runScriptOnce(name string, function int) error {
	g.runningOnce = true
	defer func() { g.runningOnce = false }()
	s, err := g.loadScript(name)
	if err != nil {
		return err
	}
	g.levelScript = s
	defer func() { g.levelScript = nil }()
	if err := runFunction(s, 0); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if function != 0 {
		if err := runFunction(s, function); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
