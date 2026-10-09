package ai

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
)

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

const MaxTileDistanceBeforeCleanup = 22
