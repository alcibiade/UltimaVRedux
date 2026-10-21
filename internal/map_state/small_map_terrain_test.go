package map_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortcullisTerrainBoundariesAndOverrides(t *testing.T) {
	for _, base := range []indexes.SpriteIndex{indexes.BrickFloor, indexes.Portcullis} {
		layer := newLayeredMap(6, 6, references.NewTileReferences(), 6, 6, false)
		arch, gate := references.Position{X: 2, Y: 2}, references.Position{X: 2, Y: 3}
		layer.SetTileByLayer(MapLayer, &arch, indexes.BrickWallArchway)
		layer.SetTileByLayer(MapLayer, &gate, base)
		layer.SetTileByLayer(MapUnitLayer, &gate, indexes.TownsPersonKeyIndex)
		for _, hour := range []byte{4, 5, 19, 20, 23, 0, 5, 20, 5} {
			want := indexes.SpriteIndex(indexes.Portcullis)
			if hour >= 5 && hour < 20 {
				want = indexes.BrickFloor
			}
			for range 2 {
				assert.Equal(t, want, layer.GetSmallMapTerrainTile(&gate, hour).Index)
				assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), layer.GetSmallMapTerrainTile(&arch, hour).Index)
			}
			assert.Equal(t, base, layer.SpriteIndexByLayer(MapLayer, gate), "queries never mutate/save gate state")
			assert.Equal(t, indexes.SpriteIndex(indexes.TownsPersonKeyIndex), layer.SpriteIndexByLayer(MapUnitLayer, gate), "no crushing or displacement")
		}
		for _, override := range []indexes.SpriteIndex{indexes.BrickFloor, indexes.Portcullis, indexes.MagicLockDoor, indexes.Chest} {
			layer.SetTileByLayer(MapOverrideLayer, &gate, override)
			for _, hour := range []byte{4, 5, 19, 20} {
				assert.Equal(t, override, layer.GetSmallMapTerrainTile(&gate, hour).Index, "explicit terrain overrides win")
			}
		}
		layer.UnSetTileByLayer(MapOverrideLayer, &gate)
		assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), layer.GetSmallMapTerrainTile(&gate, 20).Index)
	}
}

func TestPortcullisTerrainIgnoresStandaloneMalformedAndEdgePairs(t *testing.T) {
	layer := newLayeredMap(6, 6, references.NewTileReferences(), 6, 6, false)
	standalone := references.Position{X: 1, Y: 1}
	layer.SetTileByLayer(MapLayer, &standalone, indexes.Portcullis)
	arch, malformed := references.Position{X: 2, Y: 2}, references.Position{X: 2, Y: 3}
	layer.SetTileByLayer(MapLayer, &arch, indexes.BrickWallArchway)
	layer.SetTileByLayer(MapLayer, &malformed, indexes.ChairBackForward)
	for _, position := range []references.Position{{X: 0, Y: 0}, {X: 3, Y: 6}, {X: -1, Y: 2}, {X: 6, Y: 2}} {
		north := position.Up()
		layer.SetTileByLayer(MapLayer, &north, indexes.BrickWallArchway)
		layer.SetTileByLayer(MapLayer, &position, indexes.BrickFloor)
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetSmallMapTerrainTile(&position, 20).Index)
	}
	for _, hour := range []byte{5, 20} {
		assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), layer.GetSmallMapTerrainTile(&standalone, hour).Index)
		assert.Equal(t, indexes.SpriteIndex(indexes.ChairBackForward), layer.GetSmallMapTerrainTile(&malformed, hour).Index)
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), layer.GetSmallMapTerrainTile(&arch, hour).Index)
	}
	// An action-created arch is not an authored gate mechanism.
	layer.SetTileByLayer(MapOverrideLayer, &arch, indexes.BrickWallArchway)
	layer.SetTileByLayer(MapLayer, &arch, indexes.StoneBrickWall)
	layer.SetTileByLayer(MapLayer, &malformed, indexes.BrickFloor)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetSmallMapTerrainTile(&malformed, 20).Index)
}

func TestPortcullisOriginalGateLayouts(t *testing.T) {
	directory := originaldata.Directory(t, files.DataOvl, files.CastleDat, files.TowneDat, files.KeepDat, files.DwellingDat)
	configuration, err := config.LoadUltimaVConfiguration("", directory)
	require.NoError(t, err)
	data, err := references.NewDataOvl(configuration)
	require.NoError(t, err)
	locations, err := references.NewSmallMapReferences(configuration, data)
	require.NoError(t, err)
	tiles := references.NewTileReferences()
	count := 0
	for location := references.Moonglow; location <= references.SerpentsHold; location++ {
		source := locations.GetLocationReference(location)
		maps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
		maps.ResetAndCreateSmallMap(source, tiles, 11, 11)
		for _, floor := range source.ListOfFloors {
			layer := maps.GetLayeredMap(references.SmallMapType, floor)
			for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
				for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
					arch := references.Position{X: x, Y: y}
					if source.GetTileNumber(arch, floor) != indexes.BrickWallArchway {
						continue
					}
					count++
					require.Less(t, y+1, references.YSmallMapTiles, "%s floor %d arch %v", location, floor, arch)
					gate := arch.Down()
					require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), source.GetTileNumber(gate, floor), "%s floor %d south %v", location, floor, gate)
					assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetSmallMapTerrainTile(&gate, 5).Index)
					assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), layer.GetSmallMapTerrainTile(&gate, 20).Index)
					assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), layer.GetSmallMapTerrainTile(&arch, 20).Index)
				}
			}
		}
	}
	assert.Equal(t, 28, count, "verified authored small-map ARCH/FLOOR pairs")
}
