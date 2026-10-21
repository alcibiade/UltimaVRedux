package ai

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
)

// NPCAIController is the map-specific NPC behaviour the game state drives each turn.
type NPCAIController interface {
	PopulateMapFirstLoad()
	AdvanceNextTurnCalcAndMoveNPCs()
	FreshenExistingNPCsOnMap()
	GetNpcs() *map_units.MapUnits
	RemoveAllEnemies()
	// HardResetNPCs resets all NPCs to their scheduled positions.
	// If preserveOverriddenAI is true, the overridden AI types (e.g., from Shadowlord effects) are preserved.
	HardResetNPCs(preserveOverriddenAI bool)
}

// maxTileDistanceBeforeCleanup is the wrapped tile distance beyond which hostile units are despawned.
const maxTileDistanceBeforeCleanup = 22
