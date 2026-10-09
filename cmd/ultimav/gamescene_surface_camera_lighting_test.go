package main

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSurfaceCameraDaylightCornerPixels(t *testing.T) {
	for _, name := range []string{"town", "town edge", "overworld seam"} {
		t.Run(name, func(t *testing.T) {
			scene := newSurfaceCameraTestScene(t)
			if name == "town edge" {
				scene.gameState.MapState.PlayerLocation.Position = references.Position{X: 1, Y: 1}
			} else if name == "overworld seam" {
				scene = newLargeMapCameraTestScene(t, 0, references.Position{})
			}
			scene.spriteSheet = sharedPresentationTestSpriteSheet()
			scene.gameState.DateTime.Hour = 12
			layer := scene.gameState.GetLayeredMapByCurrentLocation()
			player := scene.gameState.MapState.PlayerLocation.Position
			for x := references.Coordinate(-12); x <= 12; x++ {
				for y := references.Coordinate(-12); y <= 12; y++ {
					pos := references.Position{X: player.X + x, Y: player.Y + y}
					if layer.IsWrappedMap() {
						pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
					}
					layer.SetTileByLayer(map_state.MapOverrideLayer, &pos, indexes.Grass)
				}
			}
			scene.surfaceCamera = surfaceCameraProjection{active: true}
			scene.refreshAllMapLayerTiles()
			defer scene.unscaledMapImage.Deallocate()
			defer scene.surfaceTerrainImage.Deallocate()
			corner := references.Position{X: player.X - 10, Y: player.Y - 6}
			if layer.IsWrappedMap() {
				corner = *corner.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}
			require.True(t, layer.GetVisibilityFlags()[corner.X][corner.Y], "fixture must have clear LOS")
			require.Equal(t, 11, layer.GetPrimaryDistanceMap()[corner])
			require.False(t, layer.IsPositionVisible(&corner, scene.gameState.DateTime, false), "authority retains its settled cutoff")

			width, height := xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize
			screen := newPresentationRaster(width, height)
			defer screen.Deallocate()
			got, want := make([]byte, width*height*4), make([]byte, width*height*4)
			for _, delta := range []references.Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
				for _, offset := range []float64{16, 8, 0.001, 0} {
					op := &ebiten.DrawImageOptions{}
					margin := float64(scene.surfaceOverscanTiles()) * sprites.TileSize
					op.GeoM.Translate(-margin+float64(delta.X)*offset, -margin+float64(delta.Y)*offset)
					for _, xray := range []bool{false, true} {
						scene.xRayVisionActive = xray
						scene.refreshAllMapLayerTiles()
						screen.Clear()
						screen.DrawImage(scene.surfaceTerrainImage, op)
						if xray {
							screen.ReadPixels(want)
						} else {
							screen.ReadPixels(got)
						}
					}
					assert.True(t, bytes.Equal(want, got), "clear daylight terrain must not acquire black corners: delta=%+v offset=%v", delta, offset)
				}
			}
			scene.xRayVisionActive = false
			for _, hour := range []byte{0, 5, 12, 20} {
				for _, basement := range []bool{false, true} {
					scene.gameState.DateTime.Hour = hour
					scene.gameState.DateTime.Minute = 30
					scene.gameState.MapState.PlayerLocation.Floor = 0
					if basement {
						scene.gameState.MapState.PlayerLocation.Floor = -1
					}
					wantVisible := layer.IsPositionVisible(&corner, scene.gameState.DateTime, basement)
					if hour != 12 || basement {
						assert.Equal(t, wantVisible, scene.isPositionVisible(layer, &corner), "darkness must retain authoritative range")
					}
					for x := -xCenter; x <= xCenter; x++ {
						for y := -yCenter; y <= yCenter; y++ {
							pos := references.Position{X: player.X + x, Y: player.Y + y}
							if layer.IsWrappedMap() {
								pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
							}
							assert.Equal(t, layer.IsPositionVisible(&pos, scene.gameState.DateTime, basement),
								scene.isPositionVisible(layer, &pos), "the settled viewport must retain authoritative visibility")
						}
					}
					scene.surfaceCamera.active = false
					assert.Equal(t, wantVisible, scene.isPositionVisible(layer, &corner), "settled rendering must be unchanged")
					scene.surfaceCamera.active = true
				}
			}
			scene.gameState.DateTime.Hour = 12
			scene.gameState.MapState.PlayerLocation.Floor = 0
			assert.False(t, layer.IsPositionVisible(&corner, scene.gameState.DateTime, false), "rendering must not widen authority")
			layer.SetVisible(false, &corner)
			assert.False(t, scene.isPositionVisible(layer, &corner), "overscan must not bypass occlusion")
		})
	}
}
