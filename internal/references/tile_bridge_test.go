package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestIsTrollBridgeExcludesRedBridge(t *testing.T) {
	tiles := NewTileReferences()

	assert.False(t, tiles.GetTile(indexes.Bridge).IsTrollBridge())
	assert.True(t, tiles.GetTile(indexes.TrollBridgeHoriz).IsTrollBridge())
	assert.True(t, tiles.GetTile(indexes.TrollBridgeVert).IsTrollBridge())
	assert.True(t, tiles.GetTile(indexes.Bridge).IsBridge())
}

func TestSkiffsPassUnderRiverBridgesAlongTheWaterway(t *testing.T) {
	tiles := NewTileReferences()
	for _, bridge := range []indexes.SpriteIndex{indexes.TrollBridgeHoriz, indexes.TrollBridgeVert} {
		tile := tiles.GetTile(bridge)
		assert.True(t, tile.IsPassable(SkiffVehicle))
		assert.True(t, tile.IsPassable(NoPartyVehicle))
		assert.False(t, tile.IsPassable(FrigateVehicle))
		for _, direction := range []Direction{Up, Down, Left, Right, NoneDirection} {
			allowed := bridge == indexes.TrollBridgeHoriz && (direction == Up || direction == Down) ||
				bridge == indexes.TrollBridgeVert && (direction == Left || direction == Right)
			assert.Equal(t, allowed, tile.IsSkiffPassableInDirection(direction), "bridge=%d direction=%d", bridge, direction)
		}
	}
	for _, tile := range []indexes.SpriteIndex{indexes.Bridge, indexes.LeftWoodFloor2, indexes.RightWoodFloor2, indexes.Grass} {
		assert.False(t, tiles.GetTile(tile).IsSkiffPassable())
	}
}
