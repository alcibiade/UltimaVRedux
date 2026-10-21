package sprites

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
)

// PercentXBasedPlacement places a sprite using screen-width percentages.
// Note: there is no end percent Y as aspect ratio will always be maintained
type PercentXBasedPlacement struct {
	StartPercentX float64
	EndPercentX   float64
	StartPercentY float64
}

// PercentBasedPlacement places a sprite within a percentage-based screen rectangle.
// Note: this will not maintain an aspect ratio and there is a bottom Y value
type PercentBasedPlacement struct {
	StartPercentX float64
	EndPercentX   float64
	StartPercentY float64
	EndPercentY   float64
}

// GetRectangleFromPercents resolves a placement against the current logical screen.
func GetRectangleFromPercents(placement PercentBasedPlacement) *image.Rectangle {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	// get the x start and end values based on the percent
	xLeft := float64(screenWidth) * placement.StartPercentX
	xRight := float64(screenWidth) * placement.EndPercentX
	yTop := float64(screenHeight) * placement.StartPercentY
	yBottom := float64(screenHeight) * placement.EndPercentY

	return &image.Rectangle{
		Min: image.Point{X: int(xLeft), Y: int(yTop)},
		Max: image.Point{X: int(xRight), Y: int(yBottom)},
	}
}

// GetDrawOptionsFromPercentsForWholeScreen creates a transform for a screen placement.
func GetDrawOptionsFromPercentsForWholeScreen(
	origImage *ebiten.Image, placement PercentBasedPlacement,
) *ebiten.DrawImageOptions {
	options := &ebiten.DrawImageOptions{}
	ApplyDrawOptionsFromPercentsForWholeScreen(options, origImage, placement)

	return options
}

// ApplyDrawOptionsFromPercentsForWholeScreen updates an existing DrawImageOptions instead of allocating a new one.
// This reduces GC pressure in hot paths like the render loop.
func ApplyDrawOptionsFromPercentsForWholeScreen(
	options *ebiten.DrawImageOptions, origImage *ebiten.Image, placement PercentBasedPlacement,
) {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	// get the x start and end values based on the percent
	xLeft := float64(screenWidth) * placement.StartPercentX
	xRight := float64(screenWidth) * placement.EndPercentX
	yTop := float64(screenHeight) * placement.StartPercentY
	yBottom := float64(screenHeight) * placement.EndPercentY

	targetWidth := xRight - xLeft
	originalImgWidth := origImage.Bounds().Dx()

	targetHeight := yBottom - yTop
	originalImgHeight := origImage.Bounds().Dy()

	scaleX := targetWidth / float64(originalImgWidth)
	scaleY := targetHeight / float64(originalImgHeight)

	// GeoM should already be reset before calling this
	options.GeoM.Scale(scaleX, scaleY)
	options.GeoM.Translate(xLeft, yTop)
}

// GetXSpriteWithPercents scales uniformly to the selected width and centers horizontally.
func GetXSpriteWithPercents(rect image.Rectangle, placement PercentXBasedPlacement) *ebiten.DrawImageOptions {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	options := &ebiten.DrawImageOptions{}

	// get the x start and end values based on the percent
	xLeft := float64(screenWidth) * placement.StartPercentX
	xRight := float64(screenWidth) * placement.EndPercentX
	yTop := float64(screenHeight) * placement.StartPercentY

	targetWidth := xRight - xLeft
	originalImgWidth := rect.Bounds().Dx()

	// we scale to X value, and match the ratio on Y so the image doesn't stretch and history
	scaleX := targetWidth / float64(originalImgWidth)
	scaleY := scaleX
	options.GeoM.Scale(scaleX, scaleY)

	scaledWidth := scaleX * float64(originalImgWidth)

	const centerDivisor = 2
	topLeftX := (float64(screenWidth) - scaledWidth) / centerDivisor
	options.GeoM.Translate(topLeftX, yTop)

	return options
}
