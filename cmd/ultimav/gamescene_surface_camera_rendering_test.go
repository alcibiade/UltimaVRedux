package main

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSurfaceCameraRenderingDisabledMoongatePreservesTransform(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	const width, height = 400, 208
	manager.PublishLogicalSize(width, height)
	scene := newSurfaceCameraTestScene(t)
	scene.gameConfig.GameplayConfiguration.SmoothSurfaceMovement = false
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.initMoongateSprite()
	scene.mapImage = ebiten.NewImage(xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize)
	screen := newPresentationRaster(width, height)
	defer screen.Deallocate()
	defer scene.mapImage.Deallocate()
	pos := scene.gameState.MapState.PlayerLocation.Position
	scene.drawAnimatedMoongate(screen, pos, 1, sprites.TileSize)
	want := make([]byte, width*height*4)
	screen.ReadPixels(want)
	require.False(t, bytes.Equal(make([]byte, len(want)), want), "fixture must render a visible moongate")
	screen.Clear()
	scene.screenShakeOffsetX, scene.screenShakeOffsetY = 7, -3
	scene.drawAnimatedMoongate(screen, pos, 1, sprites.TileSize)
	got := make([]byte, len(want))
	screen.ReadPixels(got)
	assert.True(t, bytes.Equal(want, got), "disabled camera must preserve the layout-only moongate transform, including glow")

	screen.Clear()
	scene.surfaceCamera = surfaceCameraProjection{active: true, x: 8}
	scene.drawAnimatedMoongate(screen, pos, 1, sprites.TileSize)
	screen.ReadPixels(got)
	assert.False(t, bytes.Equal(want, got), "active camera must still move the moongate")
}

func TestSurfaceCameraRenderingTerrainEndpointParity(t *testing.T) {
	scene := newSurfaceCameraTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.xRayVisionActive = true
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	player := scene.gameState.MapState.PlayerLocation.Position
	for x := references.Coordinate(-2); x < xTilesVisibleOnGameScreen+2; x++ {
		for y := references.Coordinate(-2); y < yTilesVisibleOnGameScreen+2; y++ {
			pos := references.Position{X: player.X - xCenter + x, Y: player.Y - yCenter + y}
			tile := indexes.SpriteIndex(indexes.Grass)
			if x < 0 || x >= xTilesVisibleOnGameScreen || y < 0 || y >= yTilesVisibleOnGameScreen {
				tile = indexes.BrickFloor
			}
			layer.SetTileByLayer(map_state.MapOverrideLayer, &pos, tile)
		}
	}
	width, height := xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize
	screen := newPresentationRaster(width, height)
	defer screen.Deallocate()
	scene.refreshAllMapLayerTiles()
	defer scene.unscaledMapImage.Deallocate()
	scene.drawMap(screen)
	want := make([]byte, width*height*4)
	screen.ReadPixels(want)

	scene.surfaceCamera = surfaceCameraProjection{active: true}
	scene.refreshAllMapLayerTiles()
	defer scene.surfaceTerrainImage.Deallocate()
	got := make([]byte, len(want))
	for _, offset := range []struct{ x, y float64 }{{0, 0}, {.001, 0}, {-.001, 0}, {0, .001}, {0, -.001}} {
		screen.Clear()
		op := &ebiten.DrawImageOptions{}
		margin := float64(scene.surfaceOverscanTiles()) * sprites.TileSize
		op.GeoM.Translate(-margin+offset.x, -margin+offset.y)
		screen.DrawImage(scene.surfaceTerrainImage, op)
		screen.ReadPixels(got)
		assert.True(t, bytes.Equal(want, got), "terrain must converge to settled pixels at offset %+v", offset)
	}
}

func TestSurfaceCameraRenderingMoongateSeams(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	const width, height = 400, 208
	manager.PublishLogicalSize(width, height)
	scene := newLargeMapCameraTestScene(t, 0, references.Position{X: 50, Y: 50})
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.initMoongateSprite()
	scene.mapImage = ebiten.NewImage(xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize)
	defer scene.mapImage.Deallocate()
	screen := newPresentationRaster(width, height)
	defer screen.Deallocate()
	want, got := make([]byte, width*height*4), make([]byte, width*height*4)
	for _, delta := range []references.Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		scene.surfaceCamera = surfaceCameraProjection{active: true, x: float64(delta.X) * 8, y: float64(delta.Y) * 8}
		scene.gameState.MapState.PlayerLocation.Position = references.Position{X: 50, Y: 50}
		screen.Clear()
		scene.drawAnimatedMoongate(screen, references.Position{X: 50 + delta.X, Y: 50 + delta.Y}, 1, sprites.TileSize)
		screen.ReadPixels(want)
		require.False(t, bytes.Equal(make([]byte, len(want)), want))
		player := references.Position{}
		if delta.X > 0 {
			player.X = 255
		}
		if delta.Y > 0 {
			player.Y = 255
		}
		world := references.Position{X: player.X + delta.X, Y: player.Y + delta.Y}
		scene.gameState.MapState.PlayerLocation.Position = player
		screen.Clear()
		scene.drawAnimatedMoongate(screen, *world.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles), 1, sprites.TileSize)
		screen.ReadPixels(got)
		assert.Equal(t, want, got, "moongate and glow must use the same nearest-copy projection at every seam")

		// Hold visual time fixed while settling so the glow remains directly comparable.
		scene.surfaceCamera.x, scene.surfaceCamera.y = 0, 0
		screen.Clear()
		scene.drawAnimatedMoongate(screen, *world.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles), 1, sprites.TileSize)
		screen.ReadPixels(want)
		require.False(t, bytes.Equal(make([]byte, len(want)), want), "active endpoint must show the moongate and glow")
		scene.surfaceCamera.sample(scene.surfaceCamera.startedMs + scene.surfaceCamera.durationMs)
		require.False(t, scene.surfaceCamera.active)
		screen.Clear()
		scene.drawAnimatedMoongate(screen, *world.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles), 1, sprites.TileSize)
		screen.ReadPixels(got)
		assert.Equal(t, want, got, "settling must preserve the visible moongate and glow pixels")

		scene.gameConfig.GameplayConfiguration.SmoothSurfaceMovement = false
		screen.Clear()
		scene.drawAnimatedMoongate(screen, *world.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles), 1, sprites.TileSize)
		screen.ReadPixels(got)
		assert.Equal(t, make([]byte, len(got)), got, "disabled preference must retain legacy unwrapped seam culling")
		scene.gameConfig.GameplayConfiguration.SmoothSurfaceMovement = true
	}
}
