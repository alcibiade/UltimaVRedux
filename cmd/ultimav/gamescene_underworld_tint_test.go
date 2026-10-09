package main

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestUnderworldTintMatchesMidnightAcrossSurfaceClock(t *testing.T) {
	pos := references.Position{X: 16, Y: 16}
	for _, fixture := range []string{"ambient", "torch", "magic", "blue_flame", "stove"} {
		t.Run(fixture, func(t *testing.T) {
			scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, -1, pos)
			state := scene.gameState
			layer := state.GetLayeredMapByCurrentLocation()
			sourcePos := references.Position{X: 21, Y: 16}
			switch fixture {
			case "torch":
				state.MapState.Lighting.LightTorch()
			case "magic":
				state.MapState.Lighting.SetMagicLight(true)
			case "blue_flame":
				layer.SetTileByLayer(map_state.MapLayer, &sourcePos, indexes.BlueFlame)
			case "stove":
				layer.SetTileByLayer(map_state.MapLayer, &sourcePos, indexes.CookStove)
			}
			state.DateTime.Hour = 0
			layer.InvalidateLightSourceCache()
			layer.RecalculateVisibleTilesWithMoongates(pos, &state.MapState.Lighting, nil, state.GetPlayerLightType())
			if fixture == "blue_flame" || fixture == "stove" {
				require.NotEmpty(t, layer.GetLastComputedLightSources())
			}
			sources := slices.Clone(layer.GetLastComputedLightSources())
			visibility := make(map_state.VisibilityCoords)
			for y, row := range layer.GetVisibleFlags() {
				visibility[y] = maps.Clone(row)
			}
			lighting, calls := state.MapState.Lighting, state.GetRNG().Calls()
			state.UpdateLightSourceTinting(layer)
			samples := []references.Position{pos, {X: 17, Y: 16}, sourcePos, {X: 22, Y: 16}, {X: 24, Y: 16}, {X: 28, Y: 16}}
			require.False(t, state.LightSourceTintingState.IsTileLit(samples[len(samples)-1]), "exercise genuinely unlit fallback")
			var firstSource ebiten.ColorScale
			for _, elapsed := range []int64{0, 562, 1125, 2250} {
				midnight := make([]ebiten.ColorScale, len(samples))
				for i, sample := range samples {
					// Freeze the existing midnight composition as the independent target.
					opts := ebiten.DrawImageOptions{}
					if tint := state.LightSourceTintingState.GetTintScaleFactors(sample, elapsed, 1); tint != nil {
						opts.ColorScale.Scale(tint.R, tint.G, tint.B, 1)
					} else {
						opts.ColorScale.Scale(map_state.UnlitDarkeningFactor, map_state.UnlitDarkeningFactor, map_state.UnlitDarkeningFactor, 1)
					}
					midnight[i] = opts.ColorScale
				}
				require.NotEqual(t, midnight[0], midnight[len(midnight)-1], "source falloff must remain visible")
				if fixture == "torch" {
					assert.Greater(t, midnight[0].R(), midnight[0].B(), "torch keeps its warm hue")
				}
				if fixture == "blue_flame" {
					assert.Greater(t, midnight[2].B(), midnight[2].R(), "static blue source is actually composed")
					if elapsed == 0 {
						firstSource = midnight[2]
					} else if elapsed == 1125 {
						assert.NotEqual(t, firstSource, midnight[2], "source breathing remains active")
					}
				}
				for _, clock := range []struct{ hour, minute byte }{{0, 0}, {5, 0}, {5, 15}, {5, 30}, {12, 0}, {19, 30}, {19, 45}, {20, 0}} {
					state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
					date := state.DateTime
					state.UpdateLightSourceTinting(layer)
					for i, sample := range samples {
						opts := ebiten.DrawImageOptions{}
						require.True(t, scene.applyLightTinting(sample, &opts, elapsed))
						assert.Equal(t, midnight[i], opts.ColorScale, "sample %v clock %02d:%02d animation %d", sample, clock.hour, clock.minute, elapsed)
					}
					assert.Equal(t, date, state.DateTime)
					assert.Equal(t, sources, layer.GetLastComputedLightSources())
					assert.Equal(t, visibility, layer.GetVisibleFlags())
					assert.True(t, lighting == state.MapState.Lighting, "fuel and gameplay lighting unchanged")
					assert.Equal(t, calls, state.GetRNG().Calls())
					if fixture == "ambient" && clock.hour == 12 {
						assert.Equal(t, map_state.PlayerLightNone, state.GetPlayerLightType(), "visibility keeps its old schedule")
						assert.True(t, state.LightSourceTintingState.IsTileLit(pos), "tint-only ambient source exists at noon")
					}
				}
			}
		})
	}
}

func TestUnderworldTintLayerTransitions(t *testing.T) {
	pos := references.Position{X: 16, Y: 16}
	for _, hour := range []byte{12, 0} {
		t.Run(fmt.Sprintf("hour_%d", hour), func(t *testing.T) {
			scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, pos)
			state := scene.gameState
			state.DateTime.Hour = hour
			for _, floor := range []references.FloorNumber{0, -1, 0, -1} {
				state.MapState.PlayerLocation.Floor = floor
				if floor == -1 {
					_, active := state.DaylightTintTarget(pos)
					assert.False(t, active, "no stale surface target even before refresh")
					assert.Equal(t, float32(1), state.AmbientTintDarkness())
				}
				state.UpdateLightSourceTinting(state.GetLayeredMapByCurrentLocation())
				// An unlit sample isolates the fallback from carried-light history.
				opts := ebiten.DrawImageOptions{}
				scene.applyLightTinting(references.Position{X: 28, Y: 16}, &opts, 0)
				want := float32(map_state.UnlitDarkeningFactor)
				if floor == 0 && hour == 12 {
					want = 1
				}
				assert.InDelta(t, want, opts.ColorScale.R(), 0.000001)
				assert.InDelta(t, want, opts.ColorScale.G(), 0.000001)
				assert.InDelta(t, want, opts.ColorScale.B(), 0.000001)
			}
		})
	}
}
