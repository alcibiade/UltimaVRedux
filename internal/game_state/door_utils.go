package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// CanAutoOpenDoor checks if the tile at the given position is an unlocked door that can be automatically opened
// This is used for the auto-open door feature which deviates from original Ultima V behavior
func (g *GameState) CanAutoOpenDoor(pos *references.Position) bool {
	// Only works in small maps (buildings/dungeons)
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		return false
	}

	// Get the tile at the position
	theMap := g.GetLayeredMapByCurrentLocation()
	topTile := theMap.GetTileTopMapOnlyTile(pos)

	// Check if it's an unlocked door
	return topTile.Is(indexes.RegularDoor) || topTile.Is(indexes.RegularDoorView)
}
