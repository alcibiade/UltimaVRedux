package map_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetAndCreateDungeonMapMaterializesEveryWrappedFloor(t *testing.T) {
	layeredMaps := &LayeredMaps{
		layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
	}
	var cells references.DungeonCells
	cells.SetCell(2, references.Position{X: 4, Y: 6}, references.DungeonCell(0xd8))

	layeredMaps.ResetAndCreateDungeonMap(cells, nil, 19, 13)

	for floor := references.FloorNumber(0); floor < references.DungeonFloorCount; floor++ {
		projection := layeredMaps.GetLayeredMap(references.DungeonMapType, floor)
		require.NotNil(t, projection)
		assert.Equal(t, references.XDungeonMapTiles, projection.XMaxTilesPerMap)
		assert.Equal(t, references.YDungeonMapTiles, projection.YMaxTilesPerMap)
		assert.True(t, projection.IsWrappedMap())
	}
	projection := layeredMaps.GetLayeredMap(references.DungeonMapType, 2)
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWallSecret),
		projection.SpriteIndexByLayer(MapLayer, references.Position{X: 4, Y: 6}))
}

func TestRebuildDungeonFloorReplacesDerivedState(t *testing.T) {
	layeredMaps := &LayeredMaps{
		layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
	}
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	cells.SetCell(0, position, references.DungeonCell(0xc4))
	layeredMaps.ResetAndCreateDungeonMap(cells, nil, 19, 13)

	projection := layeredMaps.GetLayeredMap(references.DungeonMapType, 0)
	assert.Equal(t, indexes.SpriteIndex(indexes.LargeRockWall), projection.SpriteIndexByLayer(MapLayer, position))

	cells.SetCell(0, position, references.DungeonCell(0xb0))
	layeredMaps.RebuildDungeonFloor(cells, 0)
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), projection.SpriteIndexByLayer(MapLayer, position))
}

func TestDungeonProjectionMapsEverySemanticKind(t *testing.T) {
	for kind := references.DungeonPassage; kind <= references.DungeonLiveRoom; kind++ {
		assert.NotPanics(t, func() {
			dungeonProjectionSprite(references.DungeonCell(byte(kind) << 4))
		})
	}
}
