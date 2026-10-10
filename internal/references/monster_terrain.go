package references

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Terrain-based monster pools and selection logic.
// Terrain gates and weighted encounter pools follow ref:ENEMY-0004.

// RandomProvider is an interface for deterministic random number generation.
// This allows GameState.RNG to be used without importing game_state package.
type RandomProvider interface {
	Intn(n int) int
}

// SurfaceLandMonsters are surface land monsters ordered by frequency of occurrence.
// Ordered Britannia land candidates; ref:ENEMY-0004.
//
//nolint:gochecknoglobals // Immutable encounter pool; ref:ENEMY-0004.
var SurfaceLandMonsters = [12]MonsterType{
	MonsterOrc,      // Most frequent (60)
	MonsterSnake,    // (50)
	MonsterRat,      // (40)
	MonsterSpider,   // (30)
	MonsterInsect,   // (20)
	MonsterSkeleton, // (15)
	MonsterHeadless, // (15)
	MonsterTroll,    // (10)
	MonsterEttin,    // (10)
	MonsterWisp,     // (3)
	MonsterDragon,   // (2)
	MonsterDevil,    // Least frequent (1)
}

// SurfaceLandFrequencies are surface land frequencies where higher values = more common.
// Britannia land weights sum to 256; ref:ENEMY-0004.
// Used for weighted random selection.
//
//nolint:gochecknoglobals // Immutable encounter weights; ref:ENEMY-0004.
var SurfaceLandFrequencies = [12]int{
	60, 50, 40, 30, 20, 15, 15, 10, 10, 3, 2, 1,
}

// SurfaceWaterMonsters are surface water monsters.
// Ordered Britannia water candidates; ref:ENEMY-0004.
// Pirate Ships (MonsterStocks) are the fifth water candidate; ref:ENEMY-0004.
//
//nolint:gochecknoglobals // Immutable encounter pool; ref:ENEMY-0004.
var SurfaceWaterMonsters = [5]MonsterType{
	MonsterShark,
	MonsterSquid,
	MonsterSeaSerpent,
	MonsterSeaHorse,
	MonsterStocks, // Pirate Ships
}

// SurfaceWaterFrequencies are surface water frequencies.
// Britannia water weights sum to 256; ref:ENEMY-0004.
//
//nolint:gochecknoglobals // Immutable encounter weights; ref:ENEMY-0004.
var SurfaceWaterFrequencies = [5]int{
	72, 72, 40, 38, 34,
}

// UnderworldLandMonsters are underworld land monsters.
// Ordered Underworld land candidates; ref:ENEMY-0004.
//
//nolint:gochecknoglobals // Immutable encounter pool; ref:ENEMY-0004.
var UnderworldLandMonsters = [7]MonsterType{
	MonsterBat,
	MonsterRat,
	MonsterSpider,
	MonsterMongbat,
	MonsterCorpser,
	MonsterDevil,
	MonsterDragon,
}

// UnderworldLandFrequencies are underworld land frequencies.
// Underworld land weights sum to 256; ref:ENEMY-0004.
//
//nolint:gochecknoglobals // Immutable encounter weights; ref:ENEMY-0004.
var UnderworldLandFrequencies = [7]int{
	64, 56, 56, 32, 32, 8, 8,
}

// UnderworldWaterMonsters are underworld water monsters.
// Ordered Underworld water candidates; ref:ENEMY-0004.
//
//nolint:gochecknoglobals // Immutable encounter pool; ref:ENEMY-0004.
var UnderworldWaterMonsters = [2]MonsterType{
	MonsterSquid,
	MonsterSeaSerpent,
}

// UnderworldWaterFrequencies are underworld water frequencies.
// Underworld water weights sum to 256; ref:ENEMY-0004.
//
//nolint:gochecknoglobals // Immutable encounter weights; ref:ENEMY-0004.
var UnderworldWaterFrequencies = [2]int{
	128, 128,
}

// Encounter attempt thresholds, distinct from candidate gates; ref:ENEMY-0004.
// These determine the base probability of a monster encounter based on terrain.
const (
	// Road tiles have lowest encounter probability (p=0)
	RoadEncounterProbability = 0
	// Standard terrain has medium encounter probability (p=1)
	StandardEncounterProbability = 1
	// Swamp, forest, mountains have higher encounter probability (p=2)
	DangerousEncounterProbability = 2
	// Nighttime adds +3 to encounter probability
	NighttimeProbabilityBonus = 3
)

// IsOriginalMonsterSpawnNightBonusHour reports whether the original attempt rule
// applies its nighttime bonus. Its hour >= 0x20 branch is unreachable because
// the original clock stores hours from 0 through 23.
func IsOriginalMonsterSpawnNightBonusHour(hour uint8) bool {
	return hour < 5
}

// PickMonsterByTerrain selects a monster type based on terrain tile.
// Applies terrain-specific candidate gates before weighted selection; ref:ENEMY-0004.
//
// Returns MonsterNone if no monster should appear on this terrain.
func PickMonsterByTerrain(
	tile *Tile,
	isUnderworld bool,
	rng RandomProvider,
) MonsterType {
	if tile == nil {
		return MonsterNone
	}

	// Water tiles - 75% less frequent, special handling
	if isWaterTerrain(tile) {
		return pickWaterMonster(tile, isUnderworld, rng)
	}

	// Desert: 25% chance of Sandtrap; ref:ENEMY-0004.
	if tile.Is(indexes.Desert1) {
		//nolint:mnd // Authentic 1/4 probability from original
		if rng.Intn(4) == 0 {
			return MonsterSandtrap
		}

		return MonsterNone
	}

	// Underworld swamp: Rot Worms; ref:ENEMY-0004.
	if tile.IsSwamp() && isUnderworld {
		return MonsterRotWorm
	}

	// Mountains and peaks: No monsters; ref:ENEMY-0004.
	if isImpassableMountain(tile) {
		return MonsterNone
	}

	// Standard land terrain
	if isLandTerrain(tile) {
		if isUnderworld {
			return PickByFrequency(UnderworldLandMonsters[:], UnderworldLandFrequencies[:], rng)
		}

		return PickByFrequency(SurfaceLandMonsters[:], SurfaceLandFrequencies[:], rng)
	}

	return MonsterNone
}

// pickWaterMonster handles water-based monster selection.
// Water candidates have an extra rarity gate and deep-water whirlpool choice; ref:ENEMY-0004.
//
//nolint:mnd // Original candidate-gate bounds; ref:ENEMY-0004.
func pickWaterMonster(tile *Tile, isUnderworld bool, rng RandomProvider) MonsterType {
	// Sea monsters pass only the low-result portion of the rarity draw; ref:ENEMY-0004.
	if rng.Intn(65) >= 16 {
		return MonsterNone
	}

	// Deep water (Water1) has 1/8 chance of whirlpool; ref:ENEMY-0004.
	if !isUnderworld && tile.Is(indexes.Water1) && rng.Intn(8) == 7 {
		return MonsterWhirlpool
	}

	// Select from appropriate water monster pool
	if isUnderworld {
		return PickByFrequency(UnderworldWaterMonsters[:], UnderworldWaterFrequencies[:], rng)
	}

	return PickByFrequency(SurfaceWaterMonsters[:], SurfaceWaterFrequencies[:], rng)
}

// PickByFrequency performs weighted random selection from monster pool.
// Subtracts ordered weights from an inclusive 0..255 draw; ref:ENEMY-0004.
//
// The algorithm uses cumulative frequency subtraction:
//  1. Generate random number 0-255
//  2. Subtract frequencies until random number goes negative
//  3. Return the monster at that index
//
//nolint:mnd // Original frequency range is 0..255 inclusive; ref:ENEMY-0004.
func PickByFrequency(monsters []MonsterType, frequencies []int, rng RandomProvider) MonsterType {
	if len(monsters) == 0 || len(frequencies) == 0 {
		return MonsterNone
	}

	// Generate a value in 0..255 inclusive; ref:ENEMY-0004.
	randVal := rng.Intn(256)

	// Find which monster this random value maps to
	for i, freq := range frequencies {
		if i >= len(monsters) {
			break
		}
		if randVal < freq {
			return monsters[i]
		}
		randVal -= freq
	}

	// Fallback to last monster if we somehow went past all frequencies
	return monsters[len(monsters)-1]
}

// isWaterTerrain checks if the tile is water terrain.
// Spawn-water includes more terrain than movement-water; ref:ENEMY-0004.
func isWaterTerrain(tile *Tile) bool {
	if tile == nil {
		return false
	}

	// Candidate selection tests the original numeric range, not movement flags; ref:ENEMY-0004.
	if tile.Index >= 0 && tile.Index < indexes.Swamp {
		return true
	}

	// Check for river tiles (0x60-0x6f in original)
	// In this codebase, rivers are represented differently - check via tile properties
	tileIndex := tile.Index

	// River tiles in original: 0x60-0x6f (96-111)
	// These map to various water-related tiles in Go codebase
	if tileIndex >= 96 && tileIndex <= 111 {
		return true
	}

	// Waterfall tiles (0xd4-0xd7 in original, 212-215)
	if tile.IsWaterfall() {
		return true
	}

	// Corner water tiles (0xe4-0xe7 in original, 228-231)
	if tileIndex >= 228 && tileIndex <= 231 {
		return true
	}

	return false
}

// isImpassableMountain checks if the tile is mountains or peaks.
// No monsters spawn on mountains; ref:ENEMY-0004.
func isImpassableMountain(tile *Tile) bool {
	if tile == nil {
		return false
	}

	return tile.IsMountain() || tile.Is(indexes.TallMountains)
}

// isLandTerrain checks if the tile is valid land for monster spawning.
// Only low land tiles and beach families enter a land pool; ref:ENEMY-0004.
//
//nolint:mnd // Original candidate terrain ranges; ref:ENEMY-0004.
func isLandTerrain(tile *Tile) bool {
	if tile == nil {
		return false
	}

	tileIndex := tile.Index

	// Beach begins at raw tile 0x30 (ref:ENEMY-0004). Roads, entrances and
	// castle parapets (0x50-0x53) are not land candidates, even if passable.
	return tileIndex >= 0 && tileIndex <= 0xff && (tileIndex < 0x10 || tileIndex&0xfc == 0x30)
}

// CalculateEncounterProbability returns the probability of a monster encounter.
// Chooses the attempt threshold from party terrain, world, and hour; ref:ENEMY-0004.
//
// Returns a probability value where higher = more likely.
// Base thresholds:
//   - Road: 0 (lowest)
//   - Standard terrain: 1 (medium)
//   - Swamp and tiles 0x09-0x0f: 2 (high)
//   - Nighttime adds +3 to any terrain
//
//nolint:mnd // Original attempt thresholds; ref:ENEMY-0004.
func CalculateEncounterProbability(tile *Tile, isNighttime bool, isUnderworld bool) int {
	if isUnderworld {
		return 3 // Underworld has fixed probability
	}

	if tile == nil {
		return 1
	}

	var prob int
	tileIndex := tile.Index

	// Road tiles have lowest probability (0x20-0x26 in original).
	// Roads are tiles 0x20-0x26 (32-38); ref:ENEMY-0004.
	switch {
	case tileIndex >= 32 && tileIndex <= 38:
		prob = RoadEncounterProbability
	case tileIndex == indexes.Swamp ||
		(tileIndex >= indexes.Forest3 && tileIndex <= indexes.RightHills2):
		// Only tile 4 and tiles 9-15 receive threshold 2; ref:ENEMY-0004.
		prob = DangerousEncounterProbability
	default:
		// Standard terrain
		prob = StandardEncounterProbability
	}

	// Original night-hour bonus; ref:ENEMY-0004.
	if isNighttime {
		prob += NighttimeProbabilityBonus
	}

	return prob
}
