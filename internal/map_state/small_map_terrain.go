package map_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// GetSmallMapTerrainTile is the live player/render terrain contract. Door
// occupancy is derived on every read, independently of the cached sprite layer.
func (l *LayeredMap) GetSmallMapTerrainTile(position *references.Position, hour byte) *references.Tile {
	return l.resolveOccupiedDoor(position, l.GetSmallMapPersistentTerrainTile(position, hour))
}

// GetSmallMapPersistentTerrainTile preserves NPC traversal and door lock rules;
// another actor holding a locked doorway open grants no new traversal permission.
func (l *LayeredMap) GetSmallMapPersistentTerrainTile(position *references.Position, hour byte) *references.Tile {
	if override := l.GetTileByLayer(MapOverrideLayer, position); override.Index > 0 {
		return override
	}
	terrain := l.GetTileByLayer(MapLayer, position)
	if position.X < 0 || position.X >= l.XMaxTilesPerMap || position.Y <= 0 || position.Y >= l.YMaxTilesPerMap ||
		(terrain.Index != indexes.BrickFloor && terrain.Index != indexes.Portcullis) {
		return terrain
	}
	arch := position.Up()
	if l.GetTileByLayer(MapLayer, &arch).Index != indexes.BrickWallArchway {
		return terrain
	}
	// Scheduled gates leave the arch intact and change only its southern
	// FLOOR/PORTCULLIS square. Ignore malformed pairs rather than XORing furniture.
	// REDUX DEVIATION: Original inn sleep can miss/repeat gate toggles (ref:TILE-0006). Derive
	// the approved schedule on every read, including skips/debug edits/native load.
	const openHour, closeHour = 5, 20
	if hour >= openHour && hour < closeHour {
		return l.tileRefs.GetTile(indexes.BrickFloor)
	}

	return l.tileRefs.GetTile(indexes.Portcullis)
}
