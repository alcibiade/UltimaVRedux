package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestArchForegroundComposesAboveActorsWithoutChangingTerrain(t *testing.T) {
	scene := newSurfaceCameraTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.xRayVisionActive = true
	scene.mapImage = ebiten.NewImage(xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	position := scene.gameState.MapState.PlayerLocation.Position
	layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.BrickWallArchway)
	rng := scene.gameState.GetRNG().Calls()
	w, h := ebiten.WindowSize()
	screen := newPresentationRaster(w, h)
	scene.drawSurfaceArchForeground(screen)
	pixels := make([]byte, w*h*4)
	screen.ReadPixels(pixels)
	nonzero := 0
	for _, v := range pixels {
		if v != 0 {
			nonzero++
		}
	}
	require.Positive(t, nonzero, "arch must be drawn in foreground")
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), layer.GetTileTopMapOnlyTile(&position).Index)
	assert.Equal(t, rng, scene.gameState.GetRNG().Calls())
	layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Grass)
	screen.Clear()
	scene.drawSurfaceArchForeground(screen)
	screen.ReadPixels(pixels)
	// This fixture's surrounding tiles are ordinary terrain, so no foreground remains.
	assert.Equal(t, make([]byte, len(pixels)), pixels)
}
