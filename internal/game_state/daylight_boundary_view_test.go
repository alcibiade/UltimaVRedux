package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaylightBoundaryViewOriginal(t *testing.T) {
	state, callbacks := newDaylightTintingTestState(t)
	layer := state.GetLayeredMapByCurrentLocation()
	state.UpdateLightSourceTinting(layer)
	options := map_state.DefaultDaylightOptions()
	wall := references.Position{X: 11, Y: 19}
	partition := references.Position{X: 5, Y: 24}
	window := references.Position{X: 12, Y: 25}
	source := state.GetCurrentSmallLocationReference()
	require.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), source.GetTileNumber(wall, 0))
	require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(wall))
	require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(references.Position{X: 12, Y: 19}))
	require.Equal(t, map_state.ExposureIndoor, layer.ExposureAt(references.Position{X: 10, Y: 19}))
	require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(partition))
	_, exposedPartition := daylightOutdoorNeighbor(layer, partition)
	require.False(t, exposedPartition)
	betweenWindows := []references.Position{{X: 12, Y: 24}, {X: 12, Y: 26}}
	for _, pos := range betweenWindows {
		require.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), source.GetTileNumber(pos, 0))
		require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(pos))
	}
	for _, test := range []struct {
		name string
		view references.Position
		want float32
	}{
		{"room side", references.Position{X: 10, Y: 24}, options.IndoorBrightness},
		{"outdoor hall", references.Position{X: 15, Y: 25}, options.OutdoorBrightness},
		{"exterior threshold", references.Position{X: 11, Y: 14}, options.OutdoorBrightness},
		{"internal threshold", references.Position{X: 5, Y: 25}, options.IndoorBrightness},
		{"return indoors", references.Position{X: 10, Y: 25}, options.IndoorBrightness},
	} {
		t.Run(test.name, func(t *testing.T) {
			state.MapState.PlayerLocation.Position = test.view
			unchanged := checkDaylightLightingInputs(t, state, layer, callbacks)
			assertGameDaylightTarget(t, state, wall, test.want, true)
			assertGameDaylightTarget(t, state, partition, options.IndoorBrightness, true)
			assertGameDaylightTarget(t, state, window, options.OutdoorBrightness, true)
			for _, pos := range betweenWindows {
				assertGameDaylightTarget(t, state, pos, options.OutdoorBrightness, true)
				base, ok := state.daylightTinting.TargetAt(pos)
				require.True(t, ok)
				assert.Equal(t, options.OutdoorBrightness, base, "window spill lights the boundary in either view")
			}
			assertGameDaylightTarget(t, state, references.Position{X: 8, Y: 25}, options.IndoorBrightness, true)
			for band, weight := range options.OpeningFalloff {
				target := options.IndoorBrightness + (options.OutdoorBrightness-options.IndoorBrightness)*weight
				assertGameDaylightTarget(t, state, references.Position{X: 11 - references.Coordinate(band), Y: 25}, target, true)
			}
			unchanged()
			// View-dependent wall treatment never changes the shared exposure or field.
			assert.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(wall))
			base, ok := state.daylightTinting.TargetAt(wall)
			require.True(t, ok)
			assert.Equal(t, options.IndoorBrightness, base)
		})
	}
	state.DateTime.Hour = 0
	assertGameDaylightTarget(t, state, wall, 0, false)
}

func TestDaylightBoundaryViewOriginalDiagonalOnlyCorners(t *testing.T) {
	state, callbacks := newDaylightTintingTestState(t)
	layer := state.GetLayeredMapByCurrentLocation()
	source := state.GetCurrentSmallLocationReference()
	options := map_state.DefaultDaylightOptions()
	viewer := references.Position{X: 15, Y: 23}
	wall := references.Position{X: 11, Y: 19}
	corners := []references.Position{{X: 12, Y: 22}, {X: 18, Y: 22}, {X: 19, Y: 21}, {X: 11, Y: 21}}
	require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(viewer))
	require.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), source.GetTileNumber(wall, 0))
	require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(wall))
	require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(references.Position{X: 12, Y: 19}))
	require.Equal(t, map_state.ExposureIndoor, layer.ExposureAt(references.Position{X: 10, Y: 19}))
	for _, corner := range corners {
		require.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), source.GetTileNumber(corner, 0))
		require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(corner))
		for _, step := range []references.Position{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}} {
			neighbor := references.Position{X: corner.X + step.X, Y: corner.Y + step.Y}
			require.NotEqual(t, map_state.ExposureOutdoor, layer.ExposureAt(neighbor), "cardinal neighbor of %v", corner)
		}
		_, cardinalOutdoor := daylightOutdoorNeighbor(layer, corner)
		require.False(t, cardinalOutdoor, "diagonal exposure must not classify %v as an exterior threshold", corner)
		diagonalOutdoor := false
		for _, step := range []references.Position{{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1}} {
			neighbor := references.Position{X: corner.X + step.X, Y: corner.Y + step.Y}
			diagonalOutdoor = diagonalOutdoor || layer.ExposureAt(neighbor) == map_state.ExposureOutdoor
		}
		require.True(t, diagonalOutdoor, "diagonal outdoor neighbor of %v", corner)
	}
	for _, test := range []struct {
		name    string
		falloff []float32
	}{
		{"default openings", options.OpeningFalloff},
		{"no opening spill", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			options.OpeningFalloff = test.falloff
			state.daylightTinting = map_state.NewDaylightField(options)
			state.MapState.PlayerLocation.Position = viewer
			unchanged := checkDaylightLightingInputs(t, state, layer, callbacks)
			state.UpdateLightSourceTinting(layer)
			for _, corner := range corners {
				base, ok := state.daylightTinting.TargetAt(corner)
				require.True(t, ok)
				if len(test.falloff) == 0 {
					// Without opening spill, only the target's diagonal check can brighten this corner.
					assert.Equal(t, options.IndoorBrightness, base)
				}
				state.MapState.PlayerLocation.Position = viewer
				assertGameDaylightTarget(t, state, corner, options.OutdoorBrightness, true)
				assertGameDaylightTarget(t, state, wall, options.OutdoorBrightness, true)

				state.MapState.PlayerLocation.Position = corner
				assertGameDaylightTarget(t, state, corner, base, true)
				assertGameDaylightTarget(t, state, wall, options.IndoorBrightness, true)
				unchanged()
			}
		})
	}
}
