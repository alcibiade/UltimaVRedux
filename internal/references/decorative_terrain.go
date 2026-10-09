package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

type decorativeTerrainPlacement struct {
	location   Location
	floor      FloorNumber
	position   Position
	decoration indexes.SpriteIndex
}

// Presentation metadata, deliberately separate from authoritative map overrides.
// The decoration guard prevents an opened/replaced tile inheriting stale ground.
var decorativeTerrainBases = map[decorativeTerrainPlacement]indexes.SpriteIndex{ //nolint:gochecknoglobals // Immutable authored presentation placements.
	{Cove, MainFloor, Position{X: 14, Y: 17}, indexes.LampPost}: indexes.BrickFloor,
	{Cove, MainFloor, Position{X: 16, Y: 17}, indexes.LampPost}: indexes.BrickFloor,
}

// DecorativeTerrainBase selects a location-specific visual underlay, if authored.
func DecorativeTerrainBase(location Location, floor FloorNumber, position Position, decoration indexes.SpriteIndex) (indexes.SpriteIndex, bool) {
	base, found := decorativeTerrainBases[decorativeTerrainPlacement{location, floor, position, decoration}]

	return base, found
}
