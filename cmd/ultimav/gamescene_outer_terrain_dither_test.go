package main

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestOuterTerrainCoverageUsesPixelDistanceOnEveryEdge(t *testing.T) {
	bounds := image.Rect(0, 0, 512, 512)
	for _, pos := range []image.Point{{0, 0}, {511, 0}, {0, 511}, {511, 511}, {256, 256}} {
		assert.Equal(t, uint16(sprites.DitherCoverageFull), outerTerrainCoverage(pos.X, pos.Y, bounds))
	}
	for _, test := range []struct {
		distance int
		coverage uint16
	}{{0, 4053}, {15, 2773}, {16, 2688}, {31, 1408}, {32, 1322}, {47, 42}, {48, 0}, {80, 0}} {
		near, far := -1-test.distance, bounds.Max.X+test.distance
		for _, pos := range []image.Point{
			{near, 256}, {far, 256}, {256, near}, {256, far},
			{near, near}, {far, near}, {near, far}, {far, far},
		} {
			assert.Equal(t, test.coverage, outerTerrainCoverage(pos.X, pos.Y, bounds), "world pixel %v", pos)
		}
	}
}

func TestOuterTerrainDitherStaysWorldAnchoredAcrossCameraBuffers(t *testing.T) {
	scene := newTerrainDitherTestScene(t, references.Britain, 0, references.Position{X: 1, Y: 15})
	green := color.RGBA{G: 170, A: 255}
	var baseline []byte
	for _, test := range []struct {
		player references.Position
		active bool
	}{{references.Position{X: 1, Y: 15}, false}, {references.Position{X: 1, Y: 16}, true}, {references.Position{X: 1, Y: 16}, false}} {
		scene.gameState.MapState.PlayerLocation.Position = test.player
		scene.surfaceCamera = surfaceCameraProjection{active: test.active, y: 7.5}
		margin := int(scene.surfaceOverscanTiles())
		width := (xTilesVisibleOnGameScreen+2*margin)*sprites.TileSize + borderPixels
		height := (yTilesVisibleOnGameScreen+2*margin)*sprites.TileSize + borderPixels
		target := ebiten.NewImage(width, height)
		if test.active {
			scene.surfaceTerrainImage = target
		} else {
			if scene.unscaledMapImage != nil {
				scene.unscaledMapImage.Deallocate()
			}
			scene.unscaledMapImage = target
		}
		target.Fill(green)
		scene.drawOuterTerrainDither()
		// Compare exactly the same world patch, including native tile seams and
		// the inside boundary, despite camera movement and overscan changes.
		const patchWidth, patchHeight = 80, 16
		patch := make([]byte, patchWidth*patchHeight*4)
		originX := (int(test.player.X) - int(xCenter) - margin) * sprites.TileSize
		originY := (int(test.player.Y) - int(yCenter) - margin) * sprites.TileSize
		worldPatch := image.Rect(-64, 15*sprites.TileSize, 16, 16*sprites.TileSize)
		target.SubImage(worldPatch.Sub(image.Pt(originX, originY))).(*ebiten.Image).ReadPixels(patch)
		if baseline == nil {
			baseline = patch
		} else {
			assert.Equal(t, baseline, patch, "camera state must not change the world-space pattern")
		}
		first, second, third := 0, 0, 0
		for y := range patchHeight {
			for x := range patchWidth {
				pixel := patch[(y*patchWidth+x)*4 : (y*patchWidth+x+1)*4]
				kept := pixel[1] != 0
				if kept {
					require.Equal(t, []byte{0, 170, 0, 255}, pixel, "retained terrain must not be dimmed")
				} else {
					require.Equal(t, []byte{0, 0, 0, 255}, pixel)
				}
				switch {
				case x < 16:
					assert.False(t, kept, "beyond three tiles must be black")
				case x < 32:
					if kept {
						third++
					}
				case x < 48:
					if kept {
						second++
					}
				case x < 64:
					if kept {
						first++
					}
				default:
					assert.True(t, kept, "playable terrain must not be rejected")
				}
			}
		}
		assert.Greater(t, first, second)
		assert.Greater(t, second, third)
		assert.Positive(t, third)
		assert.Less(t, first, sprites.TileSize*sprites.TileSize)
	}
}

func TestOuterTerrainDitherReplacesDarkeningAndPreservesLighting(t *testing.T) {
	for _, hour := range []byte{0, 12} {
		scene := newSurfaceCameraTestScene(t)
		scene.spriteSheet = sharedPresentationTestSpriteSheet()
		scene.gameState.MapState.PlayerLocation.Position = references.Position{X: 1, Y: 15}
		scene.gameState.DateTime.Hour = hour
		scene.xRayVisionActive = true // Exercise the border independently of LOS-mask eligibility.
		scene.refreshAllMapLayerTiles()
		t.Cleanup(scene.unscaledMapImage.Deallocate)
		pos := references.Position{X: -1, Y: 15}
		plain := ebiten.NewImage(sprites.TileSize, sprites.TileSize)
		t.Cleanup(plain.Deallocate)
		plain.Fill(color.Black)
		op := &ebiten.DrawImageOptions{}
		x, y := sprites.ConvertPosition16to18(0, 0)
		op.GeoM.Translate(x, y)
		scene.applyLightTinting(pos, op, scene.getElapsedMs())
		plain.DrawImage(scene.spriteSheet.GetAnimatedSprite(scene.gameState.GetOuterTileForCurrentLocation(), pos.AnimationHash(), scene.getElapsedMs()), op)
		location := scene.gameState.GetCurrentSmallLocationReference()
		bounds := image.Rect(0, 0, (int(location.GetMaxX())+1)*sprites.TileSize, (int(location.GetMaxY())+1)*sprites.TileSize)
		retainedColour := false
		for pixelY := range sprites.TileSize {
			for pixelX := range sprites.TileSize {
				worldX, worldY := -sprites.TileSize+pixelX, int(pos.Y)*sprites.TileSize+pixelY
				want := color.RGBA{A: 255}
				if sprites.DitherKeeps(outerTerrainCoverage(worldX, worldY, bounds), worldX, worldY) {
					want = plain.At(pixelX, pixelY).(color.RGBA)
					retainedColour = retainedColour || want.R != 0 || want.G != 0 || want.B != 0
				}
				actual := scene.unscaledMapImage.At((int(xCenter)-2)*sprites.TileSize+pixelX, int(yCenter)*sprites.TileSize+pixelY)
				assert.Equal(t, want, actual, "hour=%d pixel=%d,%d", hour, pixelX, pixelY)
			}
		}
		require.True(t, retainedColour, "fixture must retain coloured terrain at hour %d", hour)
	}
}

func TestOuterTerrainDitherDoesNotAffectWrappedMaps(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, floor, references.Position{})
		require.False(t, scene.usesOuterTerrainDither())
		// No terrain target is needed: the small-map decoration must return before
		// trying to treat the wrapped world's seam as a physical map boundary.
		require.NotPanics(t, scene.drawOuterTerrainDither)
	}
}
