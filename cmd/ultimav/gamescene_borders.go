package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

type gameBorders struct {
	outsideBorder                 *ebiten.Image
	outsideBorderOp               *ebiten.DrawImageOptions
	surfaceAperture               image.Rectangle
	skyBounds                     image.Rectangle
	skyBackingBounds              image.Rectangle
	skyBacking                    *ebiten.Image
	rightSideBorder               *ebiten.Image
	rightSideBorderOp             *ebiten.DrawImageOptions
	topRightCharacterBorder       *ebiten.Image
	topRightCharacterBorderOp     *ebiten.DrawImageOptions
	activeEffectsBorder           *ebiten.Image
	activeEffectsBorderOp         *ebiten.DrawImageOptions
	midRightTextOutputBorder      *ebiten.Image
	midRightTextOutputBorderOp    *ebiten.DrawImageOptions
	bottomRightProvisionsBorder   *ebiten.Image
	bottomRightProvisionsBorderOp *ebiten.DrawImageOptions
}

func (g *GameScene) initializeBorders() {
	width, height := display.GetManager().GetCurrentSize()
	bounds := image.Rect(0, 0, width, height)

	mainBorder := sprites.NewBorderSprites()
	g.borders.outsideBorder, g.borders.outsideBorderOp = mainBorder.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(borderWidthScaling, gameScreenPercents)
	_, g.borders.surfaceAperture = surfaceMapBorderGeometry()
	g.initializeSkyBorder()

	rightSideBorder := sprites.NewBorderSprites()
	g.borders.rightSideBorder, g.borders.rightSideBorderOp = rightSideBorder.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(
		borderWidthScaling, rectanglePlacement(bounds, g.rightRail.railBorder))

	topRightCharacterBorder := sprites.NewBorderSprites()
	g.borders.topRightCharacterBorder, g.borders.topRightCharacterBorderOp = topRightCharacterBorder.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(
		borderWidthScaling, rectanglePlacement(bounds, g.rightRail.characterBorder))

	activeEffectsBorder := sprites.NewBorderSprites()
	g.borders.activeEffectsBorder, g.borders.activeEffectsBorderOp = activeEffectsBorder.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(
		borderWidthScaling, rectanglePlacement(bounds, g.rightRail.activeBorder))

	midRightTextOutputBorder := sprites.NewBorderSprites()
	g.borders.midRightTextOutputBorder, g.borders.midRightTextOutputBorderOp = midRightTextOutputBorder.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(
		borderWidthScaling, rectanglePlacement(bounds, g.rightRail.outputBorder))

	bottomRightTextOutputBorder := sprites.NewBorderSprites()
	g.borders.bottomRightProvisionsBorder, g.borders.bottomRightProvisionsBorderOp = bottomRightTextOutputBorder.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(
		borderWidthScaling, rectanglePlacement(bounds, g.rightRail.provisionBorder))
}

// Measure the actual rasterized frame, including its asymmetric slice rounding.
// The center scan stops at the inner antialiased edge, not the transparent
// exterior corners. The opaque rounded corners are composed over this rectangle.
func surfaceMapBorderGeometry() (image.Rectangle, image.Rectangle) {
	border := sprites.NewBorderSprites()
	frame, _ := border.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(borderWidthScaling, gameScreenPercents)
	outer := frame.Bounds()
	pixels := make([]byte, outer.Dx()*outer.Dy()*4)
	frame.ReadPixels(pixels)
	alpha := func(x, y int) byte { return pixels[(y*outer.Dx()+x)*4+3] }
	x, y := outer.Dx()/2, outer.Dy()/2
	aperture := image.Rect(x, y, x, y)
	for aperture.Min.X > 0 && alpha(aperture.Min.X-1, y) < 255 {
		aperture.Min.X--
	}
	for aperture.Max.X < outer.Max.X && alpha(aperture.Max.X, y) < 255 {
		aperture.Max.X++
	}
	for aperture.Min.Y > 0 && alpha(x, aperture.Min.Y-1) < 255 {
		aperture.Min.Y--
	}
	for aperture.Max.Y < outer.Max.Y && alpha(x, aperture.Max.Y) < 255 {
		aperture.Max.Y++
	}

	return outer, aperture
}
