package map_state

import (
	"fmt"
	"maps"
	"math"
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

func TestDaylightOriginalSource(t *testing.T) {
	t.Parallel()
	defaults := DefaultDaylightOptions()
	indoor, outdoor := defaults.IndoorBrightness, defaults.OutdoorBrightness
	directory := originaldata.Directory(t, files.DATA_OVL, files.CASTLE_DAT, files.KEEP_DAT, files.TOWNE_DAT)
	configuration, err := config.LoadUltimaVConfiguration("", directory)
	require.NoError(t, err)
	dataOvl, err := references.NewDataOvl(configuration)
	require.NoError(t, err)
	locations, err := references.NewSmallMapReferences(configuration, dataOvl)
	require.NoError(t, err)

	for _, location := range []references.Location{
		references.SerpentsHold,
		references.LordBritishsCastle,
		references.Yew,
		references.Minoc,
	} {
		t.Run(fmt.Sprint(location), func(t *testing.T) {
			source := locations.GetLocationReference(location)
			layeredMaps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
			layeredMaps.ResetAndCreateSmallMap(source, nil, 19, 13)
			field := NewDaylightField(DefaultDaylightOptions())
			for _, floor := range source.ListOfFloors {
				t.Run(fmt.Sprint(floor), func(t *testing.T) {
					layer := layeredMaps.GetLayeredMap(references.SmallMapType, floor)
					openings := daylightOriginalOpenings(source, floor, layer.exposure)
					assert.Equal(t, openings, layer.daylightOpenings, "collect exactly eligible immutable source openings")
					field.PrepareSmallMap(layer, daylightTestClosedDoor)
					buffer := &field.values[0]
					for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
						for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
							pos := references.Position{X: x, Y: y}
							assertDaylightTarget(t, field, pos, daylightOriginalTarget(layer.exposure, openings, pos, false, defaults), true)
						}
					}
					if location == references.SerpentsHold && floor == 0 {
						window := references.Position{X: 12, Y: 25}
						require.Equal(t, indexes.SpriteIndex(indexes.StoneCrossWindow), source.GetTileNumber(window, floor))
						assert.Contains(t, openings, daylightOpening{position: window, inward: references.Position{X: -1, Y: 0}})
						assert.Equal(t, ExposureOutdoor, layer.ExposureAt(references.Position{X: 13, Y: 25}))
						assert.Equal(t, ExposureIndoor, layer.ExposureAt(references.Position{X: 11, Y: 25}))
						for index, target := range []float32{outdoor, indoor + (outdoor-indoor)*0.6, indoor + (outdoor-indoor)*0.25} {
							pos := references.Position{X: 11 - references.Coordinate(index), Y: 25}
							assertDaylightTarget(t, field, pos, target, true)
						}
						for _, receiver := range []struct {
							pos    references.Position
							weight float32
						}{
							{references.Position{X: 12, Y: 24}, 1}, // Wall between windows.
							{references.Position{X: 12, Y: 22}, 1},
							{references.Position{X: 12, Y: 28}, 1},
							{references.Position{X: 11, Y: 22}, 0.6},
							{references.Position{X: 10, Y: 22}, 0.25},
						} {
							assert.Equal(t, ExposureBoundary, layer.ExposureAt(receiver.pos))
							assert.False(t, slices.ContainsFunc(openings, func(opening daylightOpening) bool {
								return opening.position == receiver.pos
							}), "receiving wall is not an opening at %v", receiver.pos)
							want := indoor + (outdoor-indoor)*receiver.weight
							assertDaylightTarget(t, field, receiver.pos, want, true)
						}
						door := references.Position{X: 5, Y: 25}
						require.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), source.GetTileNumber(door, floor))
						assert.Equal(t, ExposureIndoor, layer.ExposureAt(references.Position{X: 4, Y: 25}), "source classification, not presumed exterior")
						assert.Equal(t, ExposureIndoor, layer.ExposureAt(references.Position{X: 6, Y: 25}))
						assert.False(t, slices.ContainsFunc(openings, func(opening daylightOpening) bool {
							return opening.position == door
						}), "an internal door must never relay daylight")
					}
					if location == references.LordBritishsCastle && floor == 1 {
						for _, seed := range smallMapOutdoorSeeds(location, floor) {
							pos := references.Position{X: references.Coordinate(seed.X), Y: references.Coordinate(seed.Y)}
							assert.Equal(t, ExposureOutdoor, layer.ExposureAt(pos))
							assertDaylightTarget(t, field, pos, outdoor, true)
						}
					}
					if location == references.Yew && floor == 0 {
						reportedWindow := references.Position{X: 14, Y: 10}
						assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), source.GetTileNumber(reportedWindow, floor),
							"the reported coordinate is source floor, not an opening")
						assert.Equal(t, ExposureOutdoor, layer.ExposureAt(reportedWindow))

						window := references.Position{X: 2, Y: 8}
						require.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), source.GetTileNumber(window, floor))
						assert.Equal(t, ExposureBoundary, layer.ExposureAt(window))
						index := slices.IndexFunc(openings, func(opening daylightOpening) bool {
							return opening.position == window
						})
						require.NotEqual(t, -1, index)
						opening := openings[index]
						outside := references.Position{X: window.X - opening.inward.X, Y: window.Y - opening.inward.Y}
						assert.Equal(t, ExposureOutdoor, layer.ExposureAt(outside))
						for distance, target := range []float32{outdoor, outdoor, indoor + (outdoor-indoor)*0.6, indoor + (outdoor-indoor)*0.25} {
							pos := references.Position{
								X: window.X + references.Coordinate(distance)*opening.inward.X,
								Y: window.Y + references.Coordinate(distance)*opening.inward.Y,
							}
							assertDaylightTarget(t, field, pos, target, true)
						}
					}
					// Check live door activation against all sources, including overlapping windows.
					field.PrepareSmallMap(layer, func(*references.Position) bool { return true })
					for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
						for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
							pos := references.Position{X: x, Y: y}
							assertDaylightTarget(t, field, pos, daylightOriginalTarget(layer.exposure, openings, pos, true, defaults), true)
						}
					}
					assert.Same(t, buffer, &field.values[0])
					field.PrepareOutdoor()
					layeredMaps.ReloadSmallMapFloor(source, nil, floor, 19, 13)
					reloaded := layeredMaps.GetLayeredMap(references.SmallMapType, floor)
					assert.NotSame(t, layer, reloaded)
					assert.Equal(t, openings, reloaded.daylightOpenings)
					field.PrepareSmallMap(reloaded, daylightTestClosedDoor)
					assert.Same(t, buffer, &field.values[0])
				})
			}
		})
	}

	t.Run("runtime and authority stay independent", func(t *testing.T) {
		source := locations.GetLocationReference(references.SerpentsHold)
		layeredMaps := &LayeredMaps{layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap)}
		layeredMaps.ResetAndCreateSmallMap(source, nil, 19, 13)
		layer := layeredMaps.GetLayeredMap(references.SmallMapType, 0)
		var sourceSprites [references.YSmallMapTiles][references.XSmallMapTiles]indexes.SpriteIndex
		for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
			for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
				sourceSprites[y][x] = source.GetTileNumber(references.Position{X: x, Y: y}, 0)
			}
		}
		window, room := references.Position{X: 12, Y: 25}, references.Position{X: 10, Y: 25}
		layer.SetTileByLayer(MapLayer, &window, indexes.BrickFloor)
		layer.SetTileByLayer(MapOverrideLayer, &window, indexes.StoneBrickWall)
		layer.SetTileByLayer(MapUnitLayer, &room, indexes.StoneBrickWall)
		layer.SetVisible(true, &room)
		layer.primaryDistanceMaskMap = DistanceMap{room: 5}
		layer.lightSourcesDistanceMap = DistanceMap{window: 1}
		lightTile := &references.Tile{Index: indexes.Fireplace, LightEmission: 3}
		layer.cachedLightSources = LightSources{{Tile: lightTile, Pos: room}}
		layer.lightSourceCacheValid = true
		exposure, openings := slices.Clone(layer.exposure.cells), slices.Clone(layer.daylightOpenings)
		visible, pending := make(VisibilityCoords), make(VisibilityCoords)
		for row := range layer.visibleFlags {
			visible[row] = maps.Clone(layer.visibleFlags[row])
			pending[row] = maps.Clone(layer.testForVisibilityMap[row])
		}
		var layers [totalLayers]Layer
		for index, rows := range layer.layers {
			layers[index] = make(Layer, len(rows))
			for y, row := range rows {
				layers[index][y] = slices.Clone(row)
			}
		}
		field := NewDaylightField(DefaultDaylightOptions())
		for hour := byte(0); hour < datetime.HoursPerDay; hour++ {
			date := datetime.UltimaDate{Year: 139, Month: 1, Day: 1, Hour: hour}
			wasVisible := layer.IsPositionVisible(&room, date, false)
			field.PrepareSmallMap(layer, daylightTestClosedDoor)
			assertDaylightTarget(t, field, room, indoor+(outdoor-indoor)*0.6, true)
			assert.Equal(t, wasVisible, layer.IsPositionVisible(&room, date, false))
			field.PrepareOutdoor()
			field.Clear()
		}
		assert.Equal(t, layers, layer.layers)
		assert.Equal(t, visible, layer.visibleFlags)
		assert.Equal(t, pending, layer.testForVisibilityMap)
		assert.Equal(t, DistanceMap{room: 5}, layer.primaryDistanceMaskMap)
		assert.Equal(t, DistanceMap{window: 1}, layer.lightSourcesDistanceMap)
		assert.Equal(t, LightSources{{Tile: lightTile, Pos: room}}, layer.cachedLightSources)
		assert.True(t, layer.lightSourceCacheValid)
		assert.Equal(t, 3, lightTile.LightEmission)
		assert.Equal(t, exposure, layer.exposure.cells)
		assert.Equal(t, openings, layer.daylightOpenings)
		for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
			for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
				pos := references.Position{X: x, Y: y}
				assert.Equal(t, sourceSprites[y][x], source.GetTileNumber(pos, 0), "source unchanged at %v", pos)
			}
		}
	})
}

// Independently inspect opposite axis pairs in source data, not production descriptors.
func daylightOriginalOpenings(source *references.SmallLocationReference, floor references.FloorNumber, exposure MapExposure) []daylightOpening {
	var result []daylightOpening
	for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			pos := references.Position{X: x, Y: y}
			sprite := source.GetTileNumber(pos, floor)
			if sprite != indexes.StoneCrossWindow && sprite != indexes.StoneGlassWindow && !sprite.IsDoor() {
				continue
			}
			var directions []references.Position
			for _, axis := range []references.Position{{X: 1, Y: 0}, {X: 0, Y: 1}} {
				minus := exposure.At(int(x-axis.X), int(y-axis.Y))
				plus := exposure.At(int(x+axis.X), int(y+axis.Y))
				if minus == ExposureOutdoor && plus == ExposureIndoor {
					directions = append(directions, axis)
				}
				if minus == ExposureIndoor && plus == ExposureOutdoor {
					directions = append(directions, references.Position{X: -axis.X, Y: -axis.Y})
				}
			}
			if len(directions) == 1 {
				opaqueDoor := sprite == indexes.RegularDoor || sprite == indexes.LockedDoor || sprite == indexes.MagicLockDoor
				result = append(result, daylightOpening{position: pos, inward: directions[0], requiresOpen: opaqueDoor})
			}
		}
	}
	return result
}

func daylightOriginalTarget(exposure MapExposure, openings []daylightOpening, pos references.Position, doorsOpen bool, options DaylightOptions) float32 {
	indoor, outdoor := options.IndoorBrightness, options.OutdoorBrightness
	kind := exposure.At(int(pos.X), int(pos.Y))
	if kind == ExposureUnknown {
		return 0
	}
	if kind == ExposureOutdoor {
		return outdoor
	}
	target := indoor
	if len(options.OpeningFalloff) == 0 {
		return target
	}
	for _, opening := range openings {
		if opening.requiresOpen && !doorsOpen {
			continue
		}
		if pos == opening.position {
			return outdoor
		}
		dx, dy := int(pos.X)-int(opening.position.X), int(pos.Y)-int(opening.position.Y)
		forward := dx*int(opening.inward.X) + dy*int(opening.inward.Y)
		if forward < 0 || (kind == ExposureIndoor && forward == 0) {
			continue
		}
		// Use floating-point geometry independently of production's squared thresholds.
		distance := int(math.Ceil(math.Hypot(float64(dx), float64(dy))))
		if distance <= len(options.OpeningFalloff) {
			weight := options.OpeningFalloff[distance-1]
			target = max(target, indoor+(outdoor-indoor)*weight)
		}
	}
	return target
}
