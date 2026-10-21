package main

import (
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func newDaylightOriginalTestScene(t *testing.T) *GameScene {
	t.Helper()
	directory := originaldata.Directory(t, files.DataOvl, files.KeepDat)
	configuration, err := config.LoadUltimaVConfiguration("", directory)
	require.NoError(t, err)
	data, err := references.NewDataOvl(configuration)
	require.NoError(t, err)
	locations, err := references.NewSmallMapReferences(configuration, data)
	require.NoError(t, err)
	player := references.Position{X: 10, Y: 25}
	scene := newTerrainDitherTestScene(t, references.SerpentsHold, 0, player)
	state := scene.gameState
	state.GameReferences.LocationReferences = locations
	state.MapState.LayeredMaps.ResetAndCreateSmallMap(locations.GetLocationReference(references.SerpentsHold),
		state.GameReferences.TileReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	layer := state.GetLayeredMapByCurrentLocation()
	state.MapState.Lighting = map_state.NewLighting(layer)
	layer.RecalculateVisibleTilesWithMoongates(player, &state.MapState.Lighting, nil, map_state.PlayerLightNone)
	return scene
}

// Freeze the scene's pre-daylight composition around the existing source helper.
// The map-state tests independently verify that helper's legacy arithmetic.
func assertDaylightLegacyScale(t *testing.T, scene *GameScene, pos references.Position, elapsed int64) {
	t.Helper()
	state := scene.gameState
	darkness := state.DateTime.GetAmbientTintDarkness()
	opts := ebiten.DrawImageOptions{}
	opts.ColorScale.Scale(0.8, 0.6, 0.4, 0.7)
	want := opts
	applied := false
	if tint := state.LightSourceTintingState.GetTintScaleFactors(pos, elapsed, darkness); tint != nil {
		want.ColorScale.Scale(tint.R, tint.G, tint.B, 1)
		applied = true
	} else if darkness > 0 {
		factor := 1 + (map_state.UnlitDarkeningFactor-1)*darkness
		want.ColorScale.Scale(factor, factor, factor, 1)
		applied = true
	}
	assert.Equal(t, applied, scene.applyLightTinting(pos, &opts, elapsed), "position=%v elapsed=%d", pos, elapsed)
	assert.Equal(t, want.ColorScale, opts.ColorScale, "exact legacy RGBA at %v elapsed=%d", pos, elapsed)
}

func TestDaylightSceneOriginalUnlitRGB(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	defaults := map_state.DefaultDaylightOptions()
	room := references.Position{X: 8, Y: 25}
	require.Equal(t, map_state.ExposureIndoor, layer.ExposureAt(room))
	require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(references.Position{X: 13, Y: 25}))
	type daylightSample struct {
		name   string
		pos    references.Position
		target float32
	}
	samples := []daylightSample{
		{"away_from_openings", room, defaults.IndoorBrightness},
		{"outside", references.Position{X: 13, Y: 25}, defaults.OutdoorBrightness},
		{"window", references.Position{X: 12, Y: 25}, defaults.OutdoorBrightness},
	}
	for distance, weight := range defaults.OpeningFalloff {
		samples = append(samples, daylightSample{
			fmt.Sprintf("window_band_%d", distance+1), references.Position{X: 11 - references.Coordinate(distance), Y: 25},
			defaults.IndoorBrightness + (defaults.OutdoorBrightness-defaults.IndoorBrightness)*weight,
		})
	}
	for _, clock := range []struct{ hour, minute byte }{{9, 0}, {12, 0}, {15, 0}, {5, 45}, {19, 45}, {0, 0}} {
		t.Run(fmt.Sprintf("%02d:%02d", clock.hour, clock.minute), func(t *testing.T) {
			state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
			state.UpdateLightSourceTinting(layer)
			// Keep the public field preparation, but isolate RGB from source collection/player light.
			state.LightSourceTintingState.BuildLitTilesMap(nil, nil)
			darkness := state.DateTime.GetAmbientTintDarkness()
			for _, sample := range samples {
				t.Run(sample.name, func(t *testing.T) {
					target, found := state.DaylightTintTarget(sample.pos)
					if darkness == 1 {
						assert.False(t, found)
						assertDaylightLegacyScale(t, scene, sample.pos, 0)
						return
					}
					require.True(t, found)
					require.Equal(t, sample.target, target)
					factor := sample.target + (map_state.UnlitDarkeningFactor-sample.target)*darkness
					opts, want := ebiten.DrawImageOptions{}, ebiten.DrawImageOptions{}
					want.ColorScale.Scale(factor, factor, factor, 1)
					assert.Equal(t, factor != 1, scene.applyLightTinting(sample.pos, &opts, 0))
					assert.Equal(t, want.ColorScale, opts.ColorScale, "interpolate ambient before composing RGB")
				})
			}
		})
	}
}

func TestDaylightSceneClockBoundaryContinuity(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	pos := references.Position{X: 8, Y: 25}
	outside := references.Position{X: 13, Y: 25}
	state.MapState.PlayerLocation.Position = pos
	require.False(t, state.MapState.Lighting.HasAnyLight(), "no carried torch or magic light")
	require.Nil(t, state.LighthouseState)
	layer.RecalculateVisibleTilesWithMoongates(pos, &state.MapState.Lighting, nil, map_state.PlayerLightNone)
	state.UpdateLightSourceTinting(layer)
	target, found := state.DaylightTintTarget(pos)
	require.True(t, found)
	require.Equal(t, map_state.ExposureIndoor, layer.ExposureAt(pos))
	require.Equal(t, map_state.DefaultDaylightOptions().IndoorBrightness, target, "no opening lifts the indoor baseline")
	require.False(t, state.LightSourceTintingState.IsTileLit(pos), "no collected source reaches this fixture at noon")
	sources := slices.Clone(layer.GetLastComputedLightSources())
	props := map_state.GetLightSourceProperties(0)
	require.Equal(t, map_state.TintNightOnly, props.TintMode)
	require.Equal(t, map_state.LightTintColor{R: 1, G: 1, B: 1}, props.TintColor, "ambient marker has no color delta")

	// Retain the actual player-source history. In particular, dusk injects the
	// ambient marker while dawn keeps it present but filters it at zero darkness.
	for _, hour := range []byte{19, 5} {
		var previous, previousDarkness float32
		for minute := byte(28); minute <= 32; minute++ {
			t.Run(fmt.Sprintf("%02d:%02d", hour, minute), func(t *testing.T) {
				state.DateTime.Hour, state.DateTime.Minute = hour, minute
				state.UpdateLightSourceTinting(layer)
				assert.Equal(t, sources, layer.GetLastComputedLightSources())
				prepared, active := state.DaylightTintTarget(pos)
				require.True(t, active)
				require.Equal(t, target, prepared)
				darkness := state.DateTime.GetAmbientTintDarkness()
				if (hour == 19 && minute <= 30) || (hour == 5 && minute >= 30) {
					require.Zero(t, darkness)
				}
				if (hour == 19 && minute == 31) || (hour == 5 && minute == 29) {
					require.InDelta(t, 1.0/30, darkness, 0.0000001)
				}
				require.Equal(t, darkness > 0 || state.GetPlayerLightType() == map_state.PlayerLightAmbient,
					state.LightSourceTintingState.IsTileLit(pos), "exercise automatic ambient injection, not a replacement source map")
				if darkness == 0 {
					require.Nil(t, state.LightSourceTintingState.GetTintScaleFactorsWithDaylight(pos, state.ElapsedMs, darkness, target),
						"dim interiors must not enable night-only tinting during full day")
				}
				factor := float32(float64(target) + (map_state.CenterBrightness-float64(target))*float64(darkness))
				opts, want := ebiten.DrawImageOptions{}, ebiten.DrawImageOptions{}
				want.ColorScale.Scale(factor, factor, factor, 1)
				require.True(t, scene.applyLightTinting(pos, &opts, state.ElapsedMs))
				assert.Equal(t, want.ColorScale, opts.ColorScale, "blend the indoor baseline toward the neutral source center")
				assert.Less(t, opts.ColorScale.R(), float32(1), "the first ambient-light minute must not jump above full brightness")
				if minute > 28 {
					expectedStep := (map_state.CenterBrightness - float64(target)) * float64(darkness-previousDarkness)
					assert.InDelta(t, expectedStep, float64(opts.ColorScale.R()-previous), 0.0000001,
						"adjacent clock minutes follow interpolation without a night-only activation jump")
				}
				previous = opts.ColorScale.R()
				previousDarkness = darkness
				outdoorTarget, active := state.DaylightTintTarget(outside)
				require.True(t, active)
				require.Equal(t, map_state.DefaultDaylightOptions().OutdoorBrightness, outdoorTarget)
				assertDaylightLegacyScale(t, scene, outside, state.ElapsedMs)
			})
		}
	}
	state.DateTime.Hour, state.DateTime.Minute = 0, 0
	state.UpdateLightSourceTinting(layer)
	_, found = state.DaylightTintTarget(pos)
	require.False(t, found)
	assertDaylightLegacyScale(t, scene, pos, state.ElapsedMs)
}

func TestDaylightSceneSourceComposition(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	pos := references.Position{X: 8, Y: 25}
	target := map_state.DefaultDaylightOptions().IndoorBrightness
	const radius = 3
	for _, source := range []struct {
		name  string
		index indexes.SpriteIndex
		mode  map_state.TintMode
	}{
		{"night_only", 0, map_state.TintNightOnly},
		{"half_day", indexes.CookStove, map_state.TintHalfDuringDay},
		{"always", indexes.BlueFlame, map_state.TintAlways},
	} {
		for _, distance := range []references.Coordinate{0, 1, radius} {
			t.Run(fmt.Sprintf("%s/distance_%d", source.name, distance), func(t *testing.T) {
				props := map_state.GetLightSourceProperties(source.index)
				require.Equal(t, source.mode, props.TintMode)
				sourcePos := references.Position{X: pos.X - distance, Y: pos.Y}
				// Blue flame is synchronized; phase zero isolates its undoubled color delta.
				elapsed := -int64(sourcePos.AnimationHash()) * 17
				for _, clock := range []struct{ hour, minute byte }{{12, 0}, {5, 45}, {19, 45}} {
					state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
					state.UpdateLightSourceTinting(layer)
					state.LightSourceTintingState.BuildLitTilesMap(map_state.LightSources{{
						Tile: &references.Tile{Index: source.index, LightEmission: radius}, Pos: sourcePos,
					}}, nil)
					prepared, found := state.DaylightTintTarget(pos)
					require.True(t, found)
					require.Equal(t, target, prepared)
					darkness := float64(state.DateTime.GetAmbientTintDarkness())
					night := map_state.CenterBrightness + (map_state.BaseBrightness-map_state.CenterBrightness)*float64(distance)/(radius+0.5)
					day := max(float64(target), min(1.0, night))
					multiplier := 1.0
					switch source.mode {
					case map_state.TintNightOnly:
						day = float64(target)
						multiplier = darkness
						if darkness == 0 {
							require.Nil(t, state.LightSourceTintingState.GetTintScaleFactors(pos, elapsed, 0),
								"indoor dimness must not enable a night-only source")
						}
					case map_state.TintHalfDuringDay:
						multiplier = 0.5 + 0.5*darkness
					case map_state.TintAlways:
					}
					brightness := day + (night-day)*darkness
					var rgb [3]float32
					for channel, tint := range []float32{props.TintColor.R, props.TintColor.G, props.TintColor.B} {
						rgb[channel] = float32(math.Max(0.3, math.Min(1.5, brightness+(float64(tint)-1)*multiplier*map_state.TintStrength)))
					}
					opts := ebiten.DrawImageOptions{}
					opts.ColorScale.Scale(0.8, 0.6, 0.4, 0.7)
					want := opts
					want.ColorScale.Scale(rgb[0], rgb[1], rgb[2], 1)
					assert.True(t, scene.applyLightTinting(pos, &opts, elapsed))
					assert.Equal(t, want.ColorScale, opts.ColorScale, "clock=%02d:%02d; add hue once after scalar composition", clock.hour, clock.minute)
				}
			})
		}
	}
}

func TestDaylightSceneLegacyContexts(t *testing.T) {
	pos := references.Position{X: 16, Y: 16}
	type sourceFixture struct {
		name   string
		lights map_state.LightSources
		player map_state.PlayerLightType
	}
	fixtures := []sourceFixture{{name: "unlit"}}
	for _, index := range append([]indexes.SpriteIndex{0, 1}, slices.Sorted(maps.Keys(map_state.LightSourcePropertiesMap))...) {
		fixtures = append(fixtures, sourceFixture{name: fmt.Sprintf("source_%04x", index), lights: map_state.LightSources{{
			Tile: &references.Tile{Index: index, LightEmission: 3}, Pos: pos,
		}}})
	}
	for _, player := range []map_state.PlayerLightType{
		map_state.PlayerLightAmbient, map_state.PlayerLightTorch, map_state.PlayerLightInLor, map_state.PlayerLightVasLor,
	} {
		fixtures = append(fixtures, sourceFixture{name: fmt.Sprintf("player_%d", player), player: player})
	}
	fixtures = append(fixtures, sourceFixture{name: "overlapping", lights: map_state.LightSources{
		{Tile: &references.Tile{Index: indexes.RightSconce, LightEmission: 3}, Pos: pos},
		{Tile: &references.Tile{Index: indexes.BlueFlame, LightEmission: 3}, Pos: references.Position{X: 17, Y: 16}},
		{Tile: &references.Tile{Index: map_state.MoongateMarkerIndex, LightEmission: 4}, Pos: references.Position{X: 16, Y: 17}},
		{Tile: &references.Tile{Index: 0, LightEmission: 2}, Pos: pos},
	}})
	for _, context := range []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
		hours    []byte
	}{
		{"surface_small_midnight", references.SerpentsHold, 0, []byte{0}},
		{"surface_large_midnight", references.BritanniaUnderworld, 0, []byte{0}},
		{"basement", references.SerpentsHold, -1, []byte{12, 0}},
	} {
		t.Run(context.name, func(t *testing.T) {
			scene := newTerrainDitherTestScene(t, context.location, context.floor, pos)
			state := scene.gameState
			layer := state.GetLayeredMapByCurrentLocation()
			state.UpdateLightSourceTinting(layer)
			for _, hour := range context.hours {
				t.Run(fmt.Sprintf("hour_%d", hour), func(t *testing.T) {
					state.DateTime.Hour = hour
					state.UpdateLightSourceTinting(layer)
					_, found := state.DaylightTintTarget(pos)
					require.False(t, found, "no local daylight, but retain the global clock rather than forcing night")
					for _, fixture := range fixtures {
						t.Run(fixture.name, func(t *testing.T) {
							state.LightSourceTintingState.Clear()
							state.LightSourceTintingState.BuildLitTilesMap(fixture.lights, nil)
							state.LightSourceTintingState.UpdatePlayerLight(pos, fixture.player)
							for _, offset := range []references.Coordinate{0, 1, 3, 9} {
								for _, elapsed := range []int64{0, 562, 1125, 2250} {
									assertDaylightLegacyScale(t, scene, references.Position{X: pos.X + offset, Y: pos.Y}, elapsed)
								}
							}
						})
					}
				})
			}
		})
	}
}

func TestDaylightScenePresentationExclusions(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	state.EventQueueManager = event_queue.NewEventQueueManager(func() int64 { return state.ElapsedMs }, nil)
	pos := references.Position{X: 8, Y: 25}
	for _, mode := range []string{"minimap", "finale", "combat", "uninitialized"} {
		t.Run(mode, func(t *testing.T) {
			for _, hour := range []byte{12, 0} {
				state.MapState.PlayerLocation.Location = references.SerpentsHold
				state.DateTime.Hour = hour
				scene.miniMapScene, scene.finalePresentation = nil, nil
				state.UpdateLightSourceTinting(state.GetLayeredMapByCurrentLocation())
				if hour == 12 {
					target, found := state.DaylightTintTarget(pos)
					require.True(t, found, "prepare an indoor field before excluding the presentation")
					require.Less(t, target, float32(1))
				}
				for _, lit := range []bool{false, true} {
					state.LightSourceTintingState = map_state.NewLightSourceTintingState()
					if lit {
						state.LightSourceTintingState.UpdatePlayerLight(pos, map_state.PlayerLightTorch)
					}
					switch mode {
					case "minimap":
						scene.miniMapScene = &minimap.Scene{}
					case "finale":
						scene.finalePresentation = &lordBritishFinalePresentation{mode: finaleRescueMap}
					case "combat":
						state.MapState.PlayerLocation.Location = references.CombatLocation
					case "uninitialized":
						state.LightSourceTintingState = nil
					}
					if mode == "combat" || mode == "uninitialized" {
						opts := ebiten.DrawImageOptions{}
						opts.ColorScale.Scale(0.8, 0.6, 0.4, 0.7)
						before := opts
						assert.False(t, scene.applyLightTinting(pos, &opts, 562))
						assert.Equal(t, before, opts)
					} else {
						assert.False(t, scene.usesSurfaceMapFrame())
						assertDaylightLegacyScale(t, scene, pos, 562)
					}
				}
			}
		})
	}
}

func TestDaylightScenePreservesSmallMapEdgePixels(t *testing.T) {
	scene := newTerrainDitherTestScene(t, references.Yew, 0, references.Position{X: 0, Y: 16})
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.unscaledMapImage = ebiten.NewImage(sprites.ExpandedTileSize, sprites.ExpandedTileSize)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	// Admit both edge samples at midnight too; tint sources are isolated below.
	layer.RecalculateVisibleTilesWithMoongates(state.MapState.PlayerLocation.Position,
		&state.MapState.Lighting, nil, map_state.PlayerLightTorch)
	for _, clock := range []struct{ hour, minute byte }{{12, 0}, {19, 45}, {0, 0}} {
		state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
		for _, x := range []references.Coordinate{-1, -3} {
			pos := references.Position{X: x, Y: 16}
			terrainDitherTestVisibility(scene, pos)
			require.True(t, scene.isPositionVisible(layer, &pos), "edge sample %v clock=%02d:%02d", pos, clock.hour, clock.minute)
			state.ClearLightSourceTinting()
			state.LightSourceTintingState = map_state.NewLightSourceTintingState()
			render := func() []byte {
				scene.unscaledMapImage.Clear()
				opts := ebiten.DrawImageOptions{}
				scene.refreshStaticMapTiles(&pos, layer, &opts)
				assert.Equal(t, ebiten.ColorScale{}, opts.ColorScale, "per-tile edge/tint scale must reset")
				pixels := make([]byte, sprites.ExpandedTileSize*sprites.ExpandedTileSize*4)
				scene.unscaledMapImage.ReadPixels(pixels)
				return pixels
			}
			before := render()
			require.NotZero(t, slices.Max(before), "edge fixture must actually draw")
			state.UpdateLightSourceTinting(layer)
			state.LightSourceTintingState.BuildLitTilesMap(nil, nil)
			_, found := state.DaylightTintTarget(pos)
			require.False(t, found)
			assertDaylightLegacyScale(t, scene, pos, 0)
			assert.Equal(t, before, render(), "edge RGB changed at %v, clock=%02d:%02d", pos, clock.hour, clock.minute)
		}
	}
}

func TestDaylightSceneWindowWallReceivers(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	source := state.GameReferences.LocationReferences.GetLocationReference(references.SerpentsHold)
	defaults := map_state.DefaultDaylightOptions()
	viewer, window := references.Position{X: 10, Y: 24}, references.Position{X: 12, Y: 23}
	state.MapState.PlayerLocation.Position = viewer
	layer.RecalculateVisibleTilesWithMoongates(viewer, &state.MapState.Lighting, nil, map_state.PlayerLightNone)
	require.Equal(t, indexes.SpriteIndex(indexes.StoneCrossWindow), source.GetTileNumber(window, 0))
	require.False(t, state.MapState.Lighting.HasAnyLight())
	samples := []struct {
		position references.Position
		band     int
	}{
		{references.Position{X: 11, Y: 22}, 2},
		{references.Position{X: 10, Y: 22}, 3},
	}
	for _, visible := range []bool{false, true} {
		t.Run(fmt.Sprintf("visible_%t", visible), func(t *testing.T) {
			state.ClearLightSourceTinting()
			terrainDitherTestVisibility(scene, viewer, window)
			for _, sample := range samples {
				require.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), source.GetTileNumber(sample.position, 0))
				require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(sample.position))
				layer.SetVisible(visible, &sample.position)
			}
			scene.prepareTerrainDither(layer)
			frame := &scene.terrainDither
			require.True(t, frame.enabled)
			before := terrainDitherTestPixels(t, frame)
			cells, samplesBefore := slices.Clone(frame.cells), slices.Clone(frame.samples)
			state.UpdateLightSourceTinting(layer)
			state.LightSourceTintingState.BuildLitTilesMap(nil, nil)
			for _, sample := range samples {
				target, found := state.DaylightTintTarget(sample.position)
				require.True(t, found)
				wantTarget := defaults.IndoorBrightness +
					(defaults.OutdoorBrightness-defaults.IndoorBrightness)*defaults.OpeningFalloff[sample.band-1]
				require.Equal(t, wantTarget, target, "Euclidean window band at %v", sample.position)
				require.False(t, state.LightSourceTintingState.IsTileLit(sample.position), "wall spill is not an artificial source")
				op := ebiten.DrawImageOptions{}
				op.ColorScale.Scale(0.8, 0.6, 0.4, 0.7)
				want := op.ColorScale
				want.Scale(wantTarget, wantTarget, wantTarget, 1)
				assert.True(t, scene.applyLightTinting(sample.position, &op, 0))
				assert.Equal(t, want, op.ColorScale, "gradual brick brightness at %v", sample.position)
				assert.Equal(t, want.A(), op.ColorScale.A(), "wall daylight changes RGB only")
				assert.Equal(t, visible, scene.isPositionVisible(layer, &sample.position), "wall receivers cannot admit visibility")
			}
			scene.prepareTerrainDither(layer)
			pixels := terrainDitherTestPixels(t, frame)
			assert.Equal(t, before, pixels, "wall daylight must not change mask pixels")
			assert.Equal(t, cells, frame.cells)
			assert.Equal(t, samplesBefore, frame.samples)
			for _, sample := range samples {
				cell, found := frame.cellAt(sample.position)
				require.True(t, found)
				assert.Equal(t, terrainDitherCell{visible: visible}, cell, "structural tiles never gain a bonus fringe")
				alpha := byte(255)
				if visible {
					alpha = 0
				}
				x := int(sample.position.X-frame.origin.X) * sprites.TileSize
				y := int(sample.position.Y-frame.origin.Y) * sprites.TileSize
				for dy := range sprites.TileSize {
					for dx := range sprites.TileSize {
						index := ((y+dy)*frame.width*sprites.TileSize + x + dx) * 4
						require.Equal(t, alpha, pixels[index+3], "whole-tile wall mask at %v pixel (%d,%d)", sample.position, dx, dy)
					}
				}
			}
		})
	}
}

func TestDaylightScenePreservesDitherMask(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	room, wall := references.Position{X: 10, Y: 25}, references.Position{X: 10, Y: 22}
	hall, fringe := references.Position{X: 15, Y: 25}, references.Position{X: 15, Y: 26}
	terrainDitherTestVisibility(scene, room, wall, hall)
	for _, overscan := range []bool{false, true} {
		scene.surfaceCamera.active = overscan
		state.ClearLightSourceTinting()
		scene.prepareTerrainDither(layer)
		frame := &scene.terrainDither
		require.True(t, frame.enabled)
		before := *frame
		before.samples, before.cells = slices.Clone(frame.samples), slices.Clone(frame.cells)
		pixels := terrainDitherTestPixels(t, frame)
		visibility := make(map_state.VisibilityCoords)
		for row, columns := range layer.GetVisibleFlags() {
			visibility[row] = maps.Clone(columns)
		}
		distances := maps.Clone(layer.GetPrimaryDistanceMap())
		date, calls := state.DateTime, state.GetRNG().Calls()
		state.UpdateLightSourceTinting(layer)
		target, found := state.DaylightTintTarget(room)
		require.True(t, found)
		require.Less(t, target, float32(1), "field generation must not be a no-op")
		scene.prepareTerrainDither(layer)
		assert.Equal(t, before.samples, frame.samples)
		assert.Equal(t, before.cells, frame.cells)
		assert.Same(t, before.masks, frame.masks, "same dither options and mask set")
		assert.Equal(t, before.origin, frame.origin)
		assert.Equal(t, pixels, terrainDitherTestPixels(t, frame), "daylight must not alter mask pixels")
		cell, found := frame.cellAt(fringe)
		require.True(t, found)
		assert.Equal(t, terrainDitherCell{bonus: true}, cell)
		assert.Equal(t, visibility, layer.GetVisibleFlags())
		assert.Equal(t, distances, layer.GetPrimaryDistanceMap())
		assert.Equal(t, date, state.DateTime)
		assert.Equal(t, calls, state.GetRNG().Calls())
	}
}

func TestDaylightSceneSharedRenderCallers(t *testing.T) {
	// These wiring assertions complement the RGB tests without constructing a full
	// Draw lifecycle. All three renderers must use the same world-position helper.
	for _, caller := range []struct {
		file, call string
	}{
		{"gamescene_tiles.go", "g.applyLightTinting(*pos, drawOptions, animationTime)"},
		{"gamescene_draw_entities.go", "g.applyLightTinting(avatarPos, g.avatarDrawOptions, g.getElapsedMs())"},
		{"gamescene_draw_map.go", "g.applyLightTinting(*worldPos, &npcDo, animationTime)"},
	} {
		t.Run(caller.file, func(t *testing.T) {
			source, err := os.ReadFile(caller.file)
			require.NoError(t, err)
			assert.Contains(t, string(source), caller.call)
			if caller.file == "gamescene_draw_entities.go" {
				assert.Contains(t, string(source), "avatarPos := g.gameState.MapState.PlayerLocation.Position")
			}
		})
	}
}
