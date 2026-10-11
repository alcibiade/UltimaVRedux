package game_state

import (
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Words of Power constants for dungeon unsealing
const (
	WordFallax    = "FALLAX"    // → Deceit
	WordVilis     = "VILIS"     // → Despise
	WordInopia    = "INOPIA"    // → Destard
	WordMalum     = "MALUM"     // → Wrong
	WordAvidus    = "AVIDUS"    // → Covetous
	WordInfama    = "INFAMA"    // → Shame
	WordIgnavus   = "IGNAVUS"   // → Hythloth
	WordVeramocor = "VERAMOCOR" // → Doom
)

// DungeonIndex represents the index in the DungeonStates array
type DungeonIndex int

const (
	DeceitIndex   DungeonIndex = 0 // Deceit dungeon
	DespiseIndex  DungeonIndex = 1 // Despise dungeon
	DestardIndex  DungeonIndex = 2 // Destard dungeon
	WrongIndex    DungeonIndex = 3 // Wrong dungeon
	CovetousIndex DungeonIndex = 4 // Covetous dungeon
	ShameIndex    DungeonIndex = 5 // Shame dungeon
	HythlothIndex DungeonIndex = 6 // Hythloth dungeon
	DoomIndex     DungeonIndex = 7 // Doom dungeon
)

// WordToDungeonMap maps Words of Power to their corresponding dungeons
var WordToDungeonMap = map[string]references.Location{
	WordFallax:    references.Deceit,
	WordVilis:     references.Despise,
	WordInopia:    references.Destard,
	WordMalum:     references.Wrong,
	WordAvidus:    references.Covetous,
	WordInfama:    references.Shame,
	WordIgnavus:   references.Hythloth,
	WordVeramocor: references.Doom,
}

//nolint:gochecknoglobals // Canonical dungeon word rows are immutable after initialization.
var canonicalDungeonWords = [...]struct {
	word    string
	dungeon references.Location
	open    indexes.SpriteIndex
}{
	{WordFallax, references.Deceit, indexes.DoomEntrance},
	{WordVilis, references.Despise, indexes.CaveEntrance},
	{WordInopia, references.Destard, indexes.CaveEntrance},
	{WordMalum, references.Wrong, indexes.DoomEntrance},
	{WordAvidus, references.Covetous, indexes.DoomEntrance},
	{WordInfama, references.Shame, indexes.MineEntrance},
	{WordIgnavus, references.Hythloth, indexes.MineEntrance},
	{WordVeramocor, references.Doom, indexes.CaveEntrance},
}

// DungeonToIndexMap maps dungeon locations to their array indices
var DungeonToIndexMap = map[references.Location]DungeonIndex{
	references.Deceit:   DeceitIndex,
	references.Despise:  DespiseIndex,
	references.Destard:  DestardIndex,
	references.Wrong:    WrongIndex,
	references.Covetous: CovetousIndex,
	references.Shame:    ShameIndex,
	references.Hythloth: HythlothIndex,
	references.Doom:     DoomIndex,
}

// GetDungeonEntranceCoordinates returns the eight canonical exterior
// entrance coordinates used by outdoor Yell. ref:WORD-0001
func (gs *GameState) GetDungeonEntranceCoordinates() map[references.Location]references.Position {
	return map[references.Location]references.Position{
		references.Deceit:   {X: 0xf0, Y: 0x49}, // (240, 73)
		references.Despise:  {X: 0x5b, Y: 0x43}, // (91, 67)
		references.Destard:  {X: 0x48, Y: 0xa8}, // (72, 168)
		references.Wrong:    {X: 0x7e, Y: 0x14}, // (126, 20)
		references.Covetous: {X: 0x9c, Y: 0x1b}, // (156, 27)
		references.Shame:    {X: 0x3a, Y: 0x66}, // (58, 102)
		references.Hythloth: {X: 0xef, Y: 0xf0}, // (239, 240)
		references.Doom:     {X: 0x80, Y: 0x80}, // (128, 128), Underworld
	}
}

func dungeonEntranceFloor(dungeon references.Location) references.FloorNumber {
	if dungeon == references.Doom {
		return references.FloorNumber(references.UNDERWORLD)
	}

	return references.FloorNumber(references.OVERWORLD)
}

// DungeonEntranceExistsOnFloor reports the original large-map planes that can
// route into a dungeon.
func DungeonEntranceExistsOnFloor(dungeon references.Location, floor references.FloorNumber) bool {
	return floor == dungeonEntranceFloor(dungeon) ||
		dungeon != references.Doom && floor == references.FloorNumber(references.UNDERWORLD)
}

func dungeonOpenEntranceSprite(dungeon references.Location) indexes.SpriteIndex {
	for _, candidate := range canonicalDungeonWords {
		if candidate.dungeon == dungeon {
			return candidate.open
		}
	}

	return indexes.NoSprites
}

// IsDungeonOpen returns true if the specified dungeon is currently open (unsealed)
func (gs *GameState) IsDungeonOpen(dungeon references.Location) bool {
	index, exists := DungeonToIndexMap[dungeon]
	if !exists {
		return false // Not a valid dungeon
	}

	return gs.DungeonStates[index]
}

// SetDungeonState sets the open/closed state of a specific dungeon
func (gs *GameState) SetDungeonState(dungeon references.Location, isOpen bool) {
	index, exists := DungeonToIndexMap[dungeon]
	if !exists {
		return // Not a valid dungeon
	}
	gs.DungeonStates[index] = isOpen
}

// ToggleDungeon switches a dungeon between open and sealed states
func (gs *GameState) ToggleDungeon(dungeon references.Location) bool {
	index, exists := DungeonToIndexMap[dungeon]
	if !exists {
		return false // Not a valid dungeon
	}
	gs.DungeonStates[index] = !gs.DungeonStates[index]

	return gs.DungeonStates[index]
}

// GetDungeonForWord returns the dungeon location for a given Word of Power (case-insensitive)
func GetDungeonForWord(word string) (references.Location, bool) {
	upperWord := strings.ToUpper(strings.TrimSpace(word))
	dungeon, exists := WordToDungeonMap[upperWord]

	return dungeon, exists
}

// GetDungeonForResponse returns the first complete canonical Word contained in
// a normalized player response. Canonical order makes multi-word input stable.
func GetDungeonForResponse(response string) (references.Location, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(response))
	if normalized == "" {
		return references.Location(0), false
	}
	for _, candidate := range canonicalDungeonWords {
		if strings.Contains(normalized, candidate.word) {
			return candidate.dungeon, true
		}
	}

	return references.Location(0), false
}

// GetWordForDungeon returns the Word of Power for a given dungeon
func GetWordForDungeon(dungeon references.Location) string {
	for word, location := range WordToDungeonMap {
		if location == dungeon {
			return word
		}
	}

	return ""
}

// IsPlayerNearDungeonEntrance checks if the player is within proximity of a dungeon entrance
// Returns the dungeon location and true if near, or zero location and false if not
func (gs *GameState) IsPlayerNearDungeonEntrance(proximityRange int) (references.Location, bool) {
	playerPos := gs.MapState.PlayerLocation.Position

	// Get current dungeon entrance coordinates from authoritative source
	dungeonCoords := gs.GetDungeonEntranceCoordinates()

	for dungeon, entrancePos := range dungeonCoords {
		if !DungeonEntranceExistsOnFloor(dungeon, gs.MapState.PlayerLocation.Floor) {
			continue
		}

		// Calculate distance using Manhattan distance
		dx := absInt(int(playerPos.X) - int(entrancePos.X))
		dy := absInt(int(playerPos.Y) - int(entrancePos.Y))

		if dx <= proximityRange && dy <= proximityRange {
			return dungeon, true
		}
	}

	return references.Location(0), false
}

// InitializeDungeonStatesFromSave initializes dungeon states from save file data
// Save file offset 0x032A contains 8 bytes, one for each dungeon
// Bit 0x80 set means dungeon is open (matches original game logic)
func (gs *GameState) InitializeDungeonStatesFromSave(saveData []byte) {
	const dungeonStatesOffset = 0x032A
	const numDungeons = 8

	// Initialize all dungeons as closed by default
	for i := range numDungeons {
		gs.DungeonStates[i] = false
	}

	// If save data is available and large enough, read dungeon states
	if len(saveData) >= dungeonStatesOffset+numDungeons {
		for i := range numDungeons {
			// Bit 0x80 set means dungeon is open
			gs.DungeonStates[i] = (saveData[dungeonStatesOffset+i] & 0x80) != 0
		}
	}
}

// absInt returns the absolute value of an integer
func absInt(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
