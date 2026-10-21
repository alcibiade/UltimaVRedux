package map_state

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapExposureOriginalSource(t *testing.T) {
	t.Parallel()
	directory := originaldata.Directory(t, files.DataOvl, files.CastleDat, files.KeepDat, files.TowneDat)
	configuration, err := config.LoadUltimaVConfiguration("", directory)
	require.NoError(t, err)
	dataOvl, err := references.NewDataOvl(configuration)
	require.NoError(t, err)
	locations, err := references.NewSmallMapReferences(configuration, dataOvl)
	require.NoError(t, err)

	t.Run("Britain wall fixtures do not expose enclosed rooms", func(t *testing.T) {
		source := locations.GetLocationReference(references.Britain)
		layeredMaps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
		layeredMaps.ResetAndCreateSmallMap(source, nil, 19, 13)
		for _, test := range []struct {
			floor   references.FloorNumber
			room    references.Position
			barrier references.Position
			sprite  indexes.SpriteIndex
		}{
			{0, references.Position{X: 22, Y: 1}, references.Position{X: 22, Y: 0}, indexes.Bookcase},
			{1, references.Position{X: 12, Y: 1}, references.Position{X: 11, Y: 0}, indexes.SignWarning},
		} {
			layer := layeredMaps.GetLayeredMap(references.SmallMapType, test.floor)
			require.Equal(t, test.sprite, source.GetTileNumber(test.barrier, test.floor))
			assert.Equal(t, ExposureBoundary, layer.ExposureAt(test.barrier))
			assert.Equal(t, ExposureIndoor, layer.ExposureAt(test.room))
			assert.Equal(t, ExposureOutdoor, layer.ExposureAt(references.Position{X: 15, Y: 31}))
			options := DefaultDaylightOptions()
			field := NewDaylightField(options)
			field.PrepareSmallMap(layer, func(*references.Position) bool { return false })
			target, ok := field.TargetAt(test.room)
			require.True(t, ok)
			assert.Equal(t, options.IndoorBrightness, target, "Britain floor %d room %v", test.floor, test.room)
		}
	})

	t.Run("SerpentsHold room hall and courtyard", func(t *testing.T) {
		source := locations.GetLocationReference(references.SerpentsHold)
		layeredMaps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
		layeredMaps.ResetAndCreateSmallMap(source, nil, 19, 13)
		ground := layeredMaps.GetLayeredMap(references.SmallMapType, 0)
		assert.Equal(t, ExposureIndoor, ground.ExposureAt(references.Position{X: 10, Y: 25}))
		roomCells := 0
		for y := references.Coordinate(23); y <= 27; y++ {
			for x := references.Coordinate(6); x <= 11; x++ {
				if assert.Equal(t, ExposureIndoor, ground.ExposureAt(references.Position{X: x, Y: y}), "room (%d,%d)", x, y) {
					roomCells++
				}
			}
		}
		assert.Equal(t, 30, roomCells)
		for floor := references.FloorNumber(0); floor <= 1; floor++ {
			layeredMap := layeredMaps.GetLayeredMap(references.SmallMapType, floor)
			assert.Equal(t, ExposureOutdoor, layeredMap.ExposureAt(references.Position{X: 15, Y: 25}), "hall floor %d", floor)
			assert.Equal(t, ExposureOutdoor, layeredMap.ExposureAt(references.Position{X: 14, Y: 19}), "courtyard floor %d", floor)
			assert.Empty(t, smallMapOutdoorSeeds(source.Location, floor), "courtyard must be border-connected")
		}
		fireplace := references.Position{X: 8, Y: 23}
		assert.Equal(t, indexes.SpriteIndex(indexes.Fireplace), source.GetTileNumber(fireplace, 1))
		assert.Equal(t, ExposureBoundary, layeredMaps.GetLayeredMap(references.SmallMapType, 1).ExposureAt(fireplace))
	})

	t.Run("Lord British upper wall walks", func(t *testing.T) {
		source := locations.GetLocationReference(references.LordBritishsCastle)
		layeredMaps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
		layeredMaps.ResetAndCreateSmallMap(source, nil, 19, 13)
		upper := layeredMaps.GetLayeredMap(references.SmallMapType, 1)
		boundaries := make([]bool, int(references.XSmallMapTiles*references.YSmallMapTiles))
		for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
			for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
				boundaries[int(y*references.XSmallMapTiles+x)] = isExposureBoundary(source.GetTileNumber(references.Position{X: x, Y: y}, 1))
			}
		}
		unseeded := ClassifyMapExposure(32, 32, boundaries, nil)
		for index, seed := range smallMapOutdoorSeeds(source.Location, 1) {
			assert.Equal(t, ExposureIndoor, unseeded.At(seed.X, seed.Y), "enclosed wall walk %v", seed)
			assert.Equal(t, ExposureOutdoor, upper.exposure.At(seed.X, seed.Y))
			seeded := ClassifyMapExposure(32, 32, boundaries, []image.Point{seed})
			changed := 0
			for cell, kind := range unseeded.cells {
				if kind != seeded.cells[cell] {
					assert.Equal(t, ExposureIndoor, kind)
					assert.Equal(t, ExposureOutdoor, seeded.cells[cell])
					assert.Equal(t, ExposureOutdoor, upper.exposure.cells[cell])
					changed++
				}
			}
			assert.Equal(t, []int{57, 51, 51}[index], changed, "wall walk %v", seed)
		}
		for _, room := range []references.Position{{X: 8, Y: 12}, {X: 20, Y: 12}, {X: 15, Y: 15}} {
			assert.Equal(t, ExposureIndoor, upper.ExposureAt(room), "neighboring room or central hall %v", room)
		}
		fireplace := references.Position{X: 13, Y: 8}
		assert.Equal(t, indexes.SpriteIndex(indexes.Fireplace), source.GetTileNumber(fireplace, 0))
		assert.Equal(t, ExposureBoundary, layeredMaps.GetLayeredMap(references.SmallMapType, 0).ExposureAt(fireplace))
	})

	for _, location := range []references.Location{references.Yew, references.Minoc} {
		t.Run(fmt.Sprintf("%s enclosed ground exterior", location), func(t *testing.T) {
			source := locations.GetLocationReference(location)
			layeredMaps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
			layeredMaps.ResetAndCreateSmallMap(source, nil, 19, 13)
			ground := layeredMaps.GetLayeredMap(references.SmallMapType, 0)
			seed := references.Position{X: 15, Y: 15}
			boundaries := make([]bool, int(references.XSmallMapTiles*references.YSmallMapTiles))
			for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
				for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
					boundaries[int(y*references.XSmallMapTiles+x)] = isExposureBoundary(source.GetTileNumber(references.Position{X: x, Y: y}, 0))
				}
			}
			unseeded := ClassifyMapExposure(32, 32, boundaries, nil)
			assert.Equal(t, ExposureIndoor, unseeded.At(int(seed.X), int(seed.Y)))
			assert.Equal(t, ExposureOutdoor, ground.ExposureAt(seed))
			room := references.Position{X: 21, Y: 1}
			if location == references.Minoc {
				room = references.Position{X: 6, Y: 5}
			}
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), source.GetTileNumber(room, 0))
			assert.Equal(t, ExposureIndoor, ground.ExposureAt(room), "enclosed room %v", room)
			if location == references.Yew {
				assert.Empty(t, smallMapOutdoorSeeds(location, -1), "Yew basement remains unseeded")
			} else {
				assert.Empty(t, smallMapOutdoorSeeds(location, 1), "Minoc upper floor remains unseeded")
			}
		})
	}

	t.Run("reload and runtime independence", func(t *testing.T) {
		source := locations.GetLocationReference(references.SerpentsHold)
		layeredMaps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
		layeredMaps.ResetAndCreateSmallMap(source, nil, 19, 13)
		ground := layeredMaps.GetLayeredMap(references.SmallMapType, 0)
		upper := layeredMaps.GetLayeredMap(references.SmallMapType, 1)
		before := slices.Clone(ground.exposure.cells)
		fresh := newLayeredMap(references.XSmallMapTiles, references.YSmallMapTiles, nil, 19, 13, false)
		assert.Equal(t, fresh.visibleFlags, ground.visibleFlags)
		assert.Equal(t, fresh.testForVisibilityMap, ground.testForVisibilityMap)
		assert.Nil(t, ground.primaryDistanceMaskMap)
		assert.Nil(t, ground.lightSourcesDistanceMap)
		door := references.Position{X: 5, Y: 25}
		room := references.Position{X: 10, Y: 25}
		require.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), source.GetTileNumber(door, 0))
		ground.SetTileByLayer(MapOverrideLayer, &door, indexes.BrickWallArchway)
		ground.SetTileByLayer(MapLayer, &door, indexes.BrickFloor)
		ground.SetTileByLayer(MapUnitLayer, &room, indexes.BrickFloor)
		ground.SetVisible(true, &room)
		ground.primaryDistanceMaskMap = DistanceMap{room: 5}
		ground.lightSourcesDistanceMap = DistanceMap{door: 1}
		for hour := byte(0); hour < datetime.HoursPerDay; hour++ {
			date := datetime.UltimaDate{Year: 139, Month: 1, Day: 1, Hour: hour}
			visible := ground.IsPositionVisible(&room, date, false)
			assert.Equal(t, ExposureIndoor, ground.ExposureAt(room))
			assert.Equal(t, ExposureBoundary, ground.ExposureAt(door))
			assert.Equal(t, visible, ground.IsPositionVisible(&room, date, false))
			assert.Equal(t, DistanceMap{room: 5}, ground.GetPrimaryDistanceMap())
			assert.Equal(t, DistanceMap{door: 1}, ground.GetLightSourcesDistanceMap())
			assert.True(t, ground.visibleFlags[room.X][room.Y])
		}
		assert.Equal(t, before, ground.exposure.cells)
		assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), source.GetTileNumber(door, 0), "source stays immutable")

		layeredMaps.ReloadSmallMapFloor(source, nil, 0, 19, 13)
		reloaded := layeredMaps.GetLayeredMap(references.SmallMapType, 0)
		assert.NotSame(t, ground, reloaded)
		assert.Same(t, upper, layeredMaps.GetLayeredMap(references.SmallMapType, 1))
		assert.Equal(t, before, reloaded.exposure.cells)
		assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), reloaded.SpriteIndexByLayer(MapLayer, door))
		assert.Equal(t, fresh.visibleFlags, reloaded.visibleFlags)
		assert.Equal(t, fresh.testForVisibilityMap, reloaded.testForVisibilityMap)
		assert.Nil(t, reloaded.primaryDistanceMaskMap)
		assert.Nil(t, reloaded.lightSourcesDistanceMap)
	})
}
