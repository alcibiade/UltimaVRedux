package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestDaylightSceneExteriorWallMatchesSunlight(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	defaults := map_state.DefaultDaylightOptions()
	unlitWall := references.Position{X: 11, Y: 19}
	require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(unlitWall))
	require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(references.Position{X: 12, Y: 19}))
	for _, test := range []struct {
		name string
		view references.Position
		wall float32
	}{
		{"inside", references.Position{X: 10, Y: 24}, defaults.IndoorBrightness},
		{"outside", references.Position{X: 15, Y: 23}, defaults.OutdoorBrightness},
		{"diagonal_only_boundary_view", references.Position{X: 12, Y: 22}, defaults.IndoorBrightness},
	} {
		t.Run(test.name, func(t *testing.T) {
			state.MapState.PlayerLocation.Position = test.view
			state.UpdateLightSourceTinting(layer)
			state.LightSourceTintingState.BuildLitTilesMap(nil, nil)
			for _, sample := range []struct {
				position references.Position
				want     float32
			}{
				{unlitWall, test.wall},
				{references.Position{X: 12, Y: 24}, defaults.OutdoorBrightness},
				{references.Position{X: 12, Y: 25}, defaults.OutdoorBrightness},
				{references.Position{X: 15, Y: 25}, defaults.OutdoorBrightness},
				{references.Position{X: 8, Y: 25}, defaults.IndoorBrightness},
			} {
				target, found := state.DaylightTintTarget(sample.position)
				require.True(t, found)
				assert.Equal(t, sample.want, target, "position %v", sample.position)
				require.False(t, state.LightSourceTintingState.IsTileLit(sample.position))
				op := ebiten.DrawImageOptions{}
				op.ColorScale.Scale(0.8, 0.6, 0.4, 0.7)
				want := op.ColorScale
				want.Scale(sample.want, sample.want, sample.want, 1)
				assert.Equal(t, sample.want != 1, scene.applyLightTinting(sample.position, &op, 0))
				assert.Equal(t, want, op.ColorScale, "position %v", sample.position)
				assert.Equal(t, want.A(), op.ColorScale.A(), "daylight leaves alpha unchanged")
			}
		})
	}
}

func TestDaylightSceneExteriorCorners(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	source := state.GameReferences.LocationReferences.GetLocationReference(references.SerpentsHold)
	defaults := map_state.DefaultDaylightOptions()
	thirdBand := defaults.IndoorBrightness + (defaults.OutdoorBrightness-defaults.IndoorBrightness)*defaults.OpeningFalloff[2]
	for _, test := range []struct {
		name              string
		position, outside references.Position
		indoorTarget      float32
	}{
		{"west_courtyard", references.Position{X: 11, Y: 21}, references.Position{X: 12, Y: 20}, thirdBand},
		{"east_courtyard", references.Position{X: 19, Y: 21}, references.Position{X: 18, Y: 20}, thirdBand},
		{"west_window", references.Position{X: 12, Y: 22}, references.Position{X: 13, Y: 23}, defaults.OutdoorBrightness},
		{"east_window", references.Position{X: 18, Y: 22}, references.Position{X: 17, Y: 23}, defaults.OutdoorBrightness},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), source.GetTileNumber(test.position, 0))
			require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(test.position))
			require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(test.outside))
			for _, step := range []references.Position{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}} {
				neighbor := references.Position{X: test.position.X + step.X, Y: test.position.Y + step.Y}
				require.NotEqual(t, map_state.ExposureOutdoor, layer.ExposureAt(neighbor), "outdoor contact must be diagonal only")
			}
			for _, view := range []struct {
				position references.Position
				want     float32
			}{
				{references.Position{X: 15, Y: 23}, defaults.OutdoorBrightness},
				{references.Position{X: 10, Y: 24}, test.indoorTarget},
				{test.position, test.indoorTarget},
			} {
				state.MapState.PlayerLocation.Position = view.position
				state.UpdateLightSourceTinting(layer)
				state.LightSourceTintingState.BuildLitTilesMap(nil, nil)
				target, found := state.DaylightTintTarget(test.position)
				require.True(t, found)
				assert.Equal(t, view.want, target, "viewer %v; diagonal contact lights targets, not boundary viewpoints", view.position)
				op := ebiten.DrawImageOptions{}
				op.ColorScale.Scale(0.8, 0.6, 0.4, 0.7)
				want := op.ColorScale
				want.Scale(view.want, view.want, view.want, 1)
				assert.Equal(t, view.want != 1, scene.applyLightTinting(test.position, &op, 0))
				assert.Equal(t, want, op.ColorScale, "viewer %v", view.position)
				assert.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(test.position))
				if view.position == test.position {
					unlit, active := state.DaylightTintTarget(references.Position{X: 11, Y: 19})
					require.True(t, active)
					assert.Equal(t, defaults.IndoorBrightness, unlit, "viewer thresholds still use four neighbors")
				}
			}
		})
	}
}
