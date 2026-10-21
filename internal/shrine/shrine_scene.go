package shrine

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Walk animation constants
const (
	WalkDelayMs = references.StandardWalkDelayMs // ~330ms, 6 ticks per step (original 18Hz timing)
	EntranceY   = 10                             // Y position at shrine entrance
	AltarY      = 5                              // Y position at altar (center)
	CenterX     = 5                              // X position (always centered)
)

// ShrineScene manages the shrine visual experience and state
type ShrineScene struct {
	// Shrine identification
	ShrineIndex int

	// Scene state
	IsActive bool

	// Original player location (to return to after shrine visit)
	OriginalPlayerLocation references.PlayerLocation

	// Shrine map reference
	ShrineMap *ShrineMap
}

// NewShrineSceneWithMap creates a shrine scene with a specific map reference
func NewShrineSceneWithMap(shrineIndex int, originalPlayerLocation references.PlayerLocation, mapData *MiniMapData) *ShrineScene {
	scene := &ShrineScene{
		ShrineIndex:            shrineIndex,
		IsActive:               true,
		OriginalPlayerLocation: originalPlayerLocation,
	}

	// Create the shrine map with loaded data
	scene.ShrineMap = NewShrineMapWithData(mapData)

	logging.Debug(logging.CategoryGameState, "Created shrine scene for shrine %d with loaded map data", shrineIndex)

	return scene
}

// GetOriginalPlayerLocation returns the player location to return to after shrine
func (ss *ShrineScene) GetOriginalPlayerLocation() references.PlayerLocation {
	return ss.OriginalPlayerLocation
}
