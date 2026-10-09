package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestReturnViewDrawOmitsMenuUnderUnrevealedMap(t *testing.T) {
	logo := ebiten.NewImage(8, 8)
	logo.Fill(color.White)
	flame := ebiten.NewImage(1, 1)
	scene := &IntroMenuScene{
		introSprites: &sprites.IntroSprites{
			Ultima16Logo:   logo,
			ReduxLogo:      ebiten.NewImage(1, 1),
			FlameAnimation: sprites.NewImageAnimation([]*ebiten.Image{flame}, 100),
		},
		presentationCanvas: ebiten.NewImage(introLogicalWidth, introLogicalHeight),
		returnView:         &returnViewPresentation{scene: minimap.NewMiniMapScene()},
	}
	// Menu-only resources are deliberately absent: the view must not draw menu controls.
	for _, height := range []int{720, 1080} {
		screen := newPresentationRaster(height*16/9, height)
		scene.Draw(screen)
		scale, offsetX, offsetY := introCanvasTransform(screen)
		assert.Equal(t, color.RGBA{R: 255, G: 255, B: 255, A: 255},
			color.RGBAModel.Convert(screen.At(int(offsetX+4*scale), int(offsetY+4*scale))))
		for _, logicalY := range []int{returnViewStripY - 1, returnViewStripY + 8} {
			assert.Equal(t, color.RGBA{A: 255}, color.RGBAModel.Convert(screen.At(
				int(offsetX+160*scale), int(offsetY+float64(logicalY)*scale))))
		}
	}
}

func TestReturnViewUsesOriginalIntroStripBounds(t *testing.T) {
	assert.Equal(t, 8, returnViewStripX)
	assert.Equal(t, 128, returnViewStripY)
	assert.Equal(t, 320-8, returnViewStripX+19*sprites.TileSize)
	assert.Equal(t, 200-8, returnViewStripY+4*sprites.TileSize)
}

func TestReturnViewCaptionUsesOriginalCharacterBounds(t *testing.T) {
	left, right := returnViewCaptionCells("The Summoning")

	assert.Equal(t, 12, left)
	assert.Equal(t, 27, right)
}

func TestReturnViewMoongateRowsRiseAndDropVertically(t *testing.T) {
	raising := &minimap.TileEffectState{Command: minimap.TileCommand{Type: minimap.TileCommandRaiseGate}, Progress: 0.5}
	dropping := &minimap.TileEffectState{Command: minimap.TileCommand{Type: minimap.TileCommandDropGate}, Progress: 0.5}

	assert.Equal(t, 8, returnViewMoongateRows(raising))
	assert.Equal(t, 8, returnViewMoongateRows(dropping))
	raising.Progress = 1
	dropping.Progress = 1
	assert.Equal(t, sprites.TileSize, returnViewMoongateRows(raising))
	assert.Zero(t, returnViewMoongateRows(dropping))
}
