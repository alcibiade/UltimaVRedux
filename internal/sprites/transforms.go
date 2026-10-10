package sprites

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
)

// PercentXBasedPlacement
// Structure used for placing sprite on screen based on x and y positions based on percentage
// Note: there is no end percent Y as aspect ratio will always be maintained
type PercentXBasedPlacement struct {
	StartPercentX float64
	EndPercentX   float64
	StartPercentY float64
}

type PercentYBasedPlacement struct {
	StartPercentY float64
	EndPercentY   float64
}

// PercentBasedPlacement
// Structure used for placing sprite on screen based on x and y positions based on percentage
// Note: this will not maintain an aspect ratio and there is a bottom Y value
type PercentBasedPlacement struct {
	StartPercentX float64
	EndPercentX   float64
	StartPercentY float64
	EndPercentY   float64
}

func (p *PercentBasedPlacement) GetCenterPoint() *PercentBasedCenterPoint {
	return &PercentBasedCenterPoint{
		X: ((p.EndPercentX - p.StartPercentX) / 2) + p.StartPercentX,
		Y: ((p.EndPercentY - p.StartPercentY) / 2) + p.StartPercentY,
	}
}

type PercentBasedCenterPoint struct {
	X, Y float64
}

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

func GetDrawOptionsFromPercentsForWholeScreen(origImage *ebiten.Image, placement PercentBasedPlacement) *ebiten.DrawImageOptions {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	op := &ebiten.DrawImageOptions{}

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
	op.GeoM.Scale(scaleX, scaleY)

	op.GeoM.Translate(xLeft, yTop)

	return op
}

// ApplyDrawOptionsFromPercentsForWholeScreen updates an existing DrawImageOptions instead of allocating a new one.
// This reduces GC pressure in hot paths like the render loop.
func ApplyDrawOptionsFromPercentsForWholeScreen(op *ebiten.DrawImageOptions, origImage *ebiten.Image, placement PercentBasedPlacement) {
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
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(xLeft, yTop)
}

func GetXSpriteWithPercents(rect image.Rectangle, placement PercentXBasedPlacement) *ebiten.DrawImageOptions {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	op := &ebiten.DrawImageOptions{}

	// get the x start and end values based on the percent
	xLeft := float64(screenWidth) * placement.StartPercentX
	xRight := float64(screenWidth) * placement.EndPercentX
	yTop := float64(screenHeight) * placement.StartPercentY

	targetWidth := xRight - xLeft
	originalImgWidth := rect.Bounds().Dx()

	// we scale to X value, and match the ratio on Y so the image doesn't stretch and history
	scaleX := targetWidth / float64(originalImgWidth)
	scaleY := scaleX
	op.GeoM.Scale(scaleX, scaleY)

	scaledWidth := scaleX * float64(originalImgWidth)

	topLeftX := (float64(screenWidth) - scaledWidth) / 2
	op.GeoM.Translate(topLeftX, yTop)

	return op
}

// GetYSpriteWithPercents
// Scales to the preferred Y % positions. It will center the X coordinate to the screen.
func GetYSpriteWithPercents(rect image.Rectangle, placement PercentYBasedPlacement) (*ebiten.DrawImageOptions, image.Point) {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	op := &ebiten.DrawImageOptions{}

	// get the x start and end values based on the percent
	yTop := float64(screenHeight) * placement.StartPercentY
	yBottom := float64(screenHeight) * placement.EndPercentY

	targetHeight := yBottom - yTop
	originalImgHeight := rect.Bounds().Dy()

	// we scale to X value, and match the ratio on Y so the image doesn't stretch and history
	scaleY := targetHeight / float64(originalImgHeight)
	scaleX := scaleY
	op.GeoM.Scale(scaleX, scaleY)

	topLeftX := float64(screenWidth) / 2

	op.GeoM.Translate(topLeftX, yTop)

	return op, image.Point{
		X: int(topLeftX),
		Y: int(yTop),
	}
}

// GetTranslateXYByPercent
// Use to get an X, Y transform based on the percentage of the screen. This is not for scale.
// Used often for top left of fonts which should be scaled via point, not GeoM.Scale
func GetTranslateXYByPercent(pbcp PercentBasedCenterPoint) (float64, float64) {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	// get the x start and end values based on the percent
	xLeft := float64(screenWidth) * pbcp.X
	yTop := float64(screenHeight) * pbcp.Y

	return xLeft, yTop
}
