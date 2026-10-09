package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestTerrainDitherAllFloorsIndoor(t *testing.T) {
	scene := newDaylightOriginalTestScene(t)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	player := scene.gameState.MapState.PlayerLocation.Position
	hiddenRoom := references.Position{X: 11, Y: 25}
	wall := references.Position{X: 10, Y: 22}
	require.Equal(t, map_state.ExposureIndoor, layer.ExposureAt(hiddenRoom))
	require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(wall))
	terrainDitherTestVisibility(scene, player, wall)
	layer.GetPrimaryDistanceMap()[wall] = 0 // Keep the wall lit on every tested floor.
	beforeDate, beforeCalls := scene.gameState.DateTime, scene.gameState.GetRNG().Calls()
	// Hold source geometry and LOS fixed to verify consistent rendering across
	// floor transitions, including returning from a basement to the surface.
	for _, floor := range []references.FloorNumber{0, -1, 1, 0} {
		t.Run(fmt.Sprintf("floor_%d", floor), func(t *testing.T) {
			scene.gameState.MapState.PlayerLocation.Floor = floor
			scene.prepareTerrainDither(layer)
			frame := &scene.terrainDither
			cell, found := frame.cellAt(hiddenRoom)
			require.True(t, found)
			assert.Equal(t, terrainDitherCell{bonus: true}, cell)
			assert.False(t, scene.isPositionVisible(layer, &hiddenRoom))
			assert.False(t, scene.ditherAllowsWorldEffect(hiddenRoom))
			pixels := terrainDitherTestPixels(t, frame)
			left, top := int(wall.X-frame.origin.X)*sprites.TileSize, int(wall.Y-frame.origin.Y)*sprites.TileSize
			for y := range sprites.TileSize {
				for x := range sprites.TileSize {
					require.Zero(t, pixels[((top+y)*frame.width*sprites.TileSize+left+x)*4+3],
						"visible structural wall retains whole-tile coverage")
				}
			}
			assert.Equal(t, beforeDate, scene.gameState.DateTime)
			assert.Equal(t, beforeCalls, scene.gameState.GetRNG().Calls())
		})
	}
}

func TestTerrainDitherStructuralWholeTiles(t *testing.T) {
	frame := terrainDitherFrame{
		width: 1, height: 1, cells: make([]terrainDitherCell, 1), samples: make([]terrainDitherSample, 9),
		masks: sprites.NewQuarterMaskSet(sprites.EdgeDitherOptions{
			TileSize: sprites.TileSize, OutwardPixels: terrainDitherOutwardPixels,
			InwardPixels: terrainDitherInwardPixels, CornerRadius: terrainDitherCornerRadius,
		}),
	}
	defer frame.image.Dispose()
	bits := [...]uint16{
		sprites.EdgeDitherNorthWest, sprites.EdgeDitherNorth, sprites.EdgeDitherNorthEast,
		sprites.EdgeDitherWest, sprites.EdgeDitherCenter, sprites.EdgeDitherEast,
		sprites.EdgeDitherSouthWest, sprites.EdgeDitherSouth, sprites.EdgeDitherSouthEast,
	}
	for state := uint16(0); state < 512; state++ {
		for i, bit := range bits {
			frame.samples[i] = terrainDitherSample{visible: state&bit != 0, trusted: true, hardEdge: i == 4}
		}
		frame.buildMask()
		visible := state&sprites.EdgeDitherCenter != 0
		require.Equal(t, terrainDitherCell{visible: visible}, frame.cells[0], "state %09b", state)
		pixels := terrainDitherTestPixels(t, &frame)
		wantAlpha := byte(255)
		if visible {
			wantAlpha = 0
		}
		for pixel := 0; pixel < len(pixels); pixel += 4 {
			if pixels[pixel+3] != wantAlpha {
				t.Fatalf("indoor state %09b pixel %d: alpha=%d, want %d", state, pixel/4, pixels[pixel+3], wantAlpha)
			}
		}
	}
}

func TestTerrainDitherIndoorBoundaryProtectsOnlyTouchingQuarters(t *testing.T) {
	frame := terrainDitherFrame{
		width: 1, height: 1, cells: make([]terrainDitherCell, 1), samples: make([]terrainDitherSample, 9),
		masks: sprites.NewQuarterMaskSet(sprites.EdgeDitherOptions{
			TileSize: sprites.TileSize, OutwardPixels: terrainDitherOutwardPixels,
			InwardPixels: terrainDitherInwardPixels, CornerRadius: terrainDitherCornerRadius,
		}),
	}
	defer frame.image.Dispose()
	for i := range frame.samples {
		frame.samples[i] = terrainDitherSample{visible: true, trusted: true}
	}
	frame.samples[4].visible = false
	frame.buildMask()
	baseline := terrainDitherTestPixels(t, &frame)
	require.True(t, frame.cells[0].bonus)
	// A visible indoor donor to the west must not bleed through that boundary;
	// the unrelated outdoor east edge retains exactly its previous pattern.
	frame.samples[3].hardEdge = true
	frame.buildMask()
	protected := terrainDitherTestPixels(t, &frame)
	for y := range sprites.TileSize {
		for x := range sprites.TileSize {
			index := (y*sprites.TileSize+x)*4 + 3
			if x < sprites.TileSize/2 {
				require.Equal(t, byte(255), protected[index])
			} else {
				require.Equal(t, baseline[index], protected[index], "unrelated outdoor quarter changed")
			}
		}
	}
	frame.samples[3].hardEdge = false
	frame.buildMask()
	assert.Equal(t, baseline, terrainDitherTestPixels(t, &frame), "no stale interior mask state")
}

func TestTerrainDitherIndoorSerpentsHoldOriginal(t *testing.T) {
	directory := originaldata.Directory(t, files.DATA_OVL, files.KEEP_DAT)
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
	wall := references.Position{X: 10, Y: 22}
	hiddenRoom := references.Position{X: 11, Y: 25}
	hall := references.Position{X: 15, Y: 25}
	outsideBonus := references.Position{X: 15, Y: 26}
	require.Equal(t, map_state.ExposureIndoor, layer.ExposureAt(player))
	require.Equal(t, map_state.ExposureBoundary, layer.ExposureAt(wall))
	require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(hall))
	require.Equal(t, map_state.ExposureOutdoor, layer.ExposureAt(outsideBonus))
	terrainDitherTestVisibility(scene, player, wall, hall)
	beforeDate, beforeCalls := state.DateTime, state.GetRNG().Calls()
	scene.prepareTerrainDither(layer)
	frame := &scene.terrainDither
	pixels := terrainDitherTestPixels(t, frame)
	for _, test := range []struct {
		pos     references.Position
		visible bool
	}{
		{wall, true},
	} {
		cell, found := frame.cellAt(test.pos)
		require.True(t, found)
		assert.Equal(t, terrainDitherCell{visible: test.visible}, cell)
		wantAlpha := byte(255)
		if test.visible {
			wantAlpha = 0
		}
		left, top := int(test.pos.X-frame.origin.X)*sprites.TileSize, int(test.pos.Y-frame.origin.Y)*sprites.TileSize
		for y := range sprites.TileSize {
			for x := range sprites.TileSize {
				require.Equal(t, wantAlpha, pixels[((top+y)*frame.width*sprites.TileSize+left+x)*4+3],
					"room/wall %v must retain whole-tile coverage", test.pos)
			}
		}
	}
	cell, found := frame.cellAt(hiddenRoom)
	require.True(t, found)
	assert.Equal(t, terrainDitherCell{bonus: true}, cell, "indoor floor fringe is enabled")
	cell, found = frame.cellAt(outsideBonus)
	require.True(t, found)
	assert.Equal(t, terrainDitherCell{bonus: true}, cell, "outdoor small-map fringe remains enabled")
	assert.False(t, scene.isPositionVisible(layer, &hiddenRoom))
	assert.False(t, scene.isPositionVisible(layer, &outsideBonus), "bonus remains presentation-only")
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeCalls, state.GetRNG().Calls())
}
