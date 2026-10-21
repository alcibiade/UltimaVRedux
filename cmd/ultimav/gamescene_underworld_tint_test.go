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
				for _, clock := range []struct{ hour, minute byte }{{0, 0}, {5, 0}, {5, 59}, {6, 0}, {12, 0}, {19, 30}, {19, 59}, {20, 0}} {
					state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
					date := state.DateTime
					layer.RecalculateVisibleTilesWithMoongates(pos, &state.MapState.Lighting, nil, state.GetPlayerLightType())
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
						assert.Equal(t, map_state.PlayerLightAmbient, state.GetPlayerLightType())
						assert.True(t, state.LightSourceTintingState.IsTileLit(pos), "ambient source exists at noon")
					}
				}
			}
		})
	}
}

func TestUnderworldVisibleTilesIndependentOfSurfaceClock(t *testing.T) {
	pos := references.Position{X: 16, Y: 16}
	for radius := 1; radius <= 5; radius++ {
		t.Run(fmt.Sprintf("radius_%d", radius), func(t *testing.T) {
			scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, -1, pos)
			state := scene.gameState
			state.MapState.Lighting.SetAmbientLightRadius(radius)
			layer := state.GetLayeredMapByCurrentLocation()
			var baseline []references.Position
			for _, clock := range []struct{ hour, minute byte }{{0, 0}, {5, 59}, {6, 0}, {12, 0}, {19, 59}, {20, 0}} {
				state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
				layer.RecalculateVisibleTilesWithMoongates(pos, &state.MapState.Lighting, nil, state.GetPlayerLightType())
				state.UpdateLightSourceTinting(layer)
				var visible []references.Position
				for y := references.Coordinate(8); y <= 24; y++ {
					for x := references.Coordinate(8); x <= 24; x++ {
						sample := references.Position{X: x, Y: y}
						if scene.isPositionVisible(layer, &sample) {
							visible = append(visible, sample)
						}
					}
				}
				if baseline == nil {
					baseline = visible
				}
				require.Equal(t, baseline, visible, "clock %02d:%02d", clock.hour, clock.minute)
				require.Contains(t, visible, references.Position{X: pos.X + references.Coordinate(radius), Y: pos.Y})
				require.NotContains(t, visible, references.Position{X: pos.X + references.Coordinate(radius+1), Y: pos.Y})
			}
		})
	}
}

func TestUnderworldFullMapRefreshPreservesAdmissionAndRestoresSurfacePolicy(t *testing.T) {
	position := references.Position{X: 16, Y: 16}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, position)
	state := scene.gameState
	state.MapState.Lighting.SetAmbientLightRadius(2)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.moongateVisualSystem = NewMoongateVisualSystem(state)
	source := references.Position{X: 20, Y: 16}
	behindWall := references.Position{X: 16, Y: 14}
	ambientEdge := references.Position{X: 14, Y: 16}
	unlit := references.Position{X: 13, Y: 16}
	sourceEdge := references.Position{X: 21, Y: 16}
	// Identical bounded terrain on both floors isolates policy from map layout.
	// A northern enclosure hides a flame inside the ambient radius; the eastern
	// flame admits terrain beyond that radius without lighting the western edge.
	for _, floor := range []references.FloorNumber{0, -1} {
		state.MapState.PlayerLocation.Floor = floor
		layer := state.GetLayeredMapByCurrentLocation()
		for y := references.Coordinate(4); y <= 28; y++ {
			for x := references.Coordinate(4); x <= 28; x++ {
				sample := references.Position{X: x, Y: y}
				layer.SetTileByLayer(map_state.MapLayer, &sample, indexes.Grass)
			}
		}
		for y := behindWall.Y - 1; y <= behindWall.Y+1; y++ {
			for x := behindWall.X - 1; x <= behindWall.X+1; x++ {
				wall := references.Position{X: x, Y: y}
				layer.SetTileByLayer(map_state.MapLayer, &wall, indexes.StoneBrickWall)
			}
		}
		layer.SetTileByLayer(map_state.MapLayer, &behindWall, indexes.BlueFlame)
		layer.SetTileByLayer(map_state.MapLayer, &source, indexes.BlueFlame)
		layer.InvalidateLightSourceCache()
	}
	type refreshSnapshot struct {
		admitted []references.Position
		tints    map[references.Position]ebiten.ColorScale
	}
	clocks := []struct{ hour, minute byte }{{0, 0}, {5, 59}, {6, 0}, {12, 0}, {19, 30}, {19, 59}, {20, 0}}
	surface := make([]refreshSnapshot, len(clocks))
	var underground refreshSnapshot
	var undergroundSources map_state.LightSources
	lighting, elapsed, rngCalls := state.MapState.Lighting, state.ElapsedMs, state.GetRNG().Calls()
	require.False(t, scene.xRayVisionActive)
	for visit, floor := range []references.FloorNumber{0, -1, 0} {
		state.MapState.PlayerLocation.Floor = floor
		for clockIndex, clock := range clocks {
			state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
			date := state.DateTime
			// Exercise the renderer's actual source collection, visibility, tint
			// and terrain preparation pipeline rather than manually rebuilding LOS.
			scene.refreshAllMapLayerTiles()
			layer := state.GetLayeredMapByCurrentLocation()
			current := refreshSnapshot{tints: make(map[references.Position]ebiten.ColorScale)}
			for y := position.Y - yCenter; y <= position.Y+yCenter; y++ {
				for x := position.X - xCenter; x <= position.X+xCenter; x++ {
					sample := references.Position{X: x, Y: y}
					if scene.isPositionVisible(layer, &sample) {
						current.admitted = append(current.admitted, sample)
						opts := ebiten.DrawImageOptions{}
						scene.applyLightTinting(sample, &opts, elapsed)
						current.tints[sample] = opts.ColorScale
					}
				}
			}
			require.NotContains(t, current.admitted, behindWall, "LOS must still obstruct illuminated terrain")
			require.False(t, layer.GetVisibilityFlags()[behindWall.X][behindWall.Y])
			require.Contains(t, current.admitted, source)
			require.Contains(t, current.admitted, sourceEdge, "static light expands final admission")
			require.Equal(t, date, state.DateTime)
			require.Equal(t, elapsed, state.ElapsedMs)
			require.Equal(t, rngCalls, state.GetRNG().Calls())
			require.True(t, lighting == state.MapState.Lighting, "refresh cannot burn fuel or change gameplay lighting")
			if floor == -1 {
				require.Equal(t, map_state.PlayerLightAmbient, state.GetPlayerLightType())
				require.Contains(t, current.admitted, ambientEdge, "configured radius-2 baseline survives noon")
				require.NotContains(t, current.admitted, unlit, "no daylight expansion underground")
				sourceTint := current.tints[source]
				require.Greater(t, sourceTint.B(), sourceTint.R(), "source hue survives the complete refresh")
				foundSource := false
				for _, light := range layer.GetLastComputedLightSources() {
					require.NotEqual(t, behindWall, light.Pos, "occluded emitter cannot contribute light")
					if light.Pos == source && light.Tile.Index == indexes.BlueFlame {
						foundSource = true
					}
				}
				require.True(t, foundSource, "production collector must retain the terrain emitter")
				if clockIndex == 0 {
					underground = current
					undergroundSources = slices.Clone(layer.GetLastComputedLightSources())
				} else {
					require.Equal(t, underground, current, "complete refresh at %02d:%02d", clock.hour, clock.minute)
					require.ElementsMatch(t, undergroundSources, layer.GetLastComputedLightSources())
				}
			} else if visit == 0 {
				surface[clockIndex] = current
			} else {
				require.Equal(t, surface[clockIndex], current, "surface policy restored at %02d:%02d", clock.hour, clock.minute)
			}
		}
	}
	require.NotEqual(t, surface[0].admitted, surface[3].admitted, "surface retains its day/night visibility")
	require.NotContains(t, surface[0].admitted, unlit)
	require.Contains(t, surface[3].admitted, unlit)
	require.Equal(t, surface[3].admitted, surface[5].admitted, "surface dusk tint must not prematurely shrink sight")
	require.NotEqual(t, surface[4].tints[unlit], surface[5].tints[unlit], "surface dusk still changes cosmetic tint")
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
