package sprites

import (
	_ "embed"
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
)

// referenceBorderHalfDivisor splits the reference border at its midpoint into the
// corner and edge regions that are copied when the border is scaled.
const referenceBorderHalfDivisor = 2

//go:embed assets/VeryPixelatedRoundedBlueWhite.png
var veryPixelatedRoundedBlueWhite []byte

// The cache owns bounded reuse references, not callers' borrowed image lifetimes.
// Resolution sweeps and interactive resizing must not retain every prior size.
//
//nolint:gochecknoglobals // Bounded, synchronized reuse of immutable frames across scene-owned borrowers.
var globalBorderCache = newBorderImageCache(borderCacheMaxBytes, borderCacheMaxEntries)

// BorderSprites holds the border artwork used by the game's UI panels.
type BorderSprites struct {
	// the default border. It is
	VeryPixelatedRoundedBlueWhite ReferenceBorder
}

// ReferenceBorder is a border image that is re-rasterized at a requested size.
type ReferenceBorder struct {
	border                    *ebiten.Image
	referenceBorderDimensions *referenceBorderDimensions
	referenceBorderBits       *borderBits
}

type referenceBorderDimensions struct {
	topLeft, bottomLeft, topRight, bottomRight                 image.Rectangle
	horizCopyLeft, horizCopyRight, vertCopyLeft, vertCopyRight image.Rectangle
}

type borderBits struct {
	topLeft        *ebiten.Image
	bottomRight    *ebiten.Image
	topRight       *ebiten.Image
	bottomLeft     *ebiten.Image
	vertCopyLeft   *ebiten.Image
	horizCopyLeft  *ebiten.Image
	vertCopyRight  *ebiten.Image
	horizCopyRight *ebiten.Image
}

// NewBorderSprites decodes the embedded border artwork into a new BorderSprites.
func NewBorderSprites() *BorderSprites {
	return &BorderSprites{
		VeryPixelatedRoundedBlueWhite: ReferenceBorder{
			border:                    NewPngSprite(veryPixelatedRoundedBlueWhite),
			referenceBorderDimensions: nil,
			referenceBorderBits:       nil,
		},
	}
}

//nolint:funlen // Keep the four edge strips and four corners in one auditable raster composition.
func (sb *ReferenceBorder) createBorderImageForWidth(
	idealWidthForScaling, width, height, screenWidth int,
) *ebiten.Image {
	idealScaleX := float64(screenWidth) / float64(idealWidthForScaling)
	scaledDimensions := sb.referenceBorderDimensions.createScaledDimensions(idealScaleX, idealScaleX)

	// A cached frame is already a reusable standalone composition. Packing large,
	// differently sized frame targets into a shared atlas retains oversized backing
	// textures for unrelated small images after resizing.
	img := ebiten.NewImageWithOptions(image.Rect(0, 0, width, height), &ebiten.NewImageOptions{Unmanaged: true})

	// cornerHeight := scaledDimensions.topLeft.Max.Y
	bothCornerHeight := scaledDimensions.topLeft.Bounds().Dy() + scaledDimensions.bottomRight.Bounds().Dy()
	bothCornerWidth := scaledDimensions.topLeft.Bounds().Dx() + scaledDimensions.bottomRight.Bounds().Dx()

	// / START WITH SIDE AND TOP BORDERS, THEY WILL GET OVERWRITTEN IF NASTY
	// now fill in the blanks with the horizontal and vertical bits
	// sprite is horizontal
	// goes from LEFT SIDE, top to bottom
	horizScaledOpLeft := ebiten.GeoM{}
	horizScaledOpLeft.Scale(idealScaleX, 1)
	horizScaledOpLeft.Translate(
		0,
		float64(scaledDimensions.topRight.Bounds().Dy()))
	for range height - (bothCornerHeight) {
		img.DrawImage(sb.referenceBorderBits.horizCopyLeft, &ebiten.DrawImageOptions{GeoM: horizScaledOpLeft})
		horizScaledOpLeft.Translate(0, 1)
	}

	// RIGHT HORIZONTAL
	// sprite is horizontal
	// goes from RIGHT, top to bottom
	horizScaledOpRight := ebiten.GeoM{}
	horizScaledOpRight.Scale(idealScaleX, 1)
	horizScaledOpRight.Translate(
		float64(width-scaledDimensions.horizCopyRight.Dx()),
		float64(scaledDimensions.topRight.Dy()))
	for range height - bothCornerHeight {
		img.DrawImage(sb.referenceBorderBits.horizCopyRight, &ebiten.DrawImageOptions{GeoM: horizScaledOpRight})
		horizScaledOpRight.Translate(0, 1)
	}

	// LEFT VERTICAL
	// sprite is vertical
	// goes from TOP, left to right
	vertScaledOpLeft := ebiten.GeoM{}
	vertScaledOpLeft.Scale(1, idealScaleX)
	vertScaledOpLeft.Translate(float64(scaledDimensions.topLeft.Dx()), 0)
	for i := 0; i <= width-(bothCornerWidth); i++ {
		img.DrawImage(sb.referenceBorderBits.vertCopyLeft, &ebiten.DrawImageOptions{GeoM: vertScaledOpLeft})
		vertScaledOpLeft.Translate(1, 0)
	}

	// RIGHT VERTICAL
	// sprite is vertical
	// goes from BOTTOM, left to right
	vertScaledOpRight := ebiten.GeoM{}
	vertScaledOpRight.Scale(1, idealScaleX)
	vertScaledOpRight.Translate(
		float64(scaledDimensions.bottomLeft.Dx()),
		float64(height-scaledDimensions.vertCopyRight.Dy()),
	)
	for range width - bothCornerWidth {
		img.DrawImage(sb.referenceBorderBits.vertCopyRight, &ebiten.DrawImageOptions{GeoM: vertScaledOpRight})
		vertScaledOpRight.Translate(1, 0)
	}

	// / START CORNERS
	topLeftOp := ebiten.GeoM{}
	topLeftOp.Scale(idealScaleX, idealScaleX)
	img.DrawImage(sb.referenceBorderBits.topLeft, &ebiten.DrawImageOptions{
		GeoM: topLeftOp,
	})

	bottomLeftOp := ebiten.GeoM{}
	bottomLeftOp.Scale(idealScaleX, idealScaleX)
	bottomLeftOp.Translate(0,
		float64(height-scaledDimensions.bottomLeft.Dy()),
	)
	img.DrawImage(sb.referenceBorderBits.bottomLeft, &ebiten.DrawImageOptions{GeoM: bottomLeftOp})

	topRightOp := ebiten.GeoM{}
	topRightOp.Scale(idealScaleX, idealScaleX)
	topRightOp.Translate(
		float64(width-scaledDimensions.topRight.Dx()),
		0,
	)
	img.DrawImage(sb.referenceBorderBits.topRight, &ebiten.DrawImageOptions{GeoM: topRightOp})

	bottomRightOp := ebiten.GeoM{}
	bottomRightOp.Scale(idealScaleX, idealScaleX)
	bottomRightOp.Translate(
		float64(width-scaledDimensions.bottomRight.Dx()),
		float64(height-scaledDimensions.bottomRight.Dy()),
	)
	img.DrawImage(sb.referenceBorderBits.bottomRight, &ebiten.DrawImageOptions{GeoM: bottomRightOp})

	return img
}

func (sb *ReferenceBorder) createBorderBits() *borderBits {
	// make our corner and edge bits
	dims := sb.referenceBorderDimensions

	return &borderBits{
		topLeft:        ebiten.NewImageFromImage(sb.border.SubImage(dims.topLeft)),
		bottomLeft:     ebiten.NewImageFromImage(sb.border.SubImage(dims.bottomLeft)),
		topRight:       ebiten.NewImageFromImage(sb.border.SubImage(dims.topRight)),
		bottomRight:    ebiten.NewImageFromImage(sb.border.SubImage(dims.bottomRight)),
		horizCopyLeft:  ebiten.NewImageFromImage(sb.border.SubImage(dims.horizCopyLeft)),
		vertCopyLeft:   ebiten.NewImageFromImage(sb.border.SubImage(dims.vertCopyLeft)),
		horizCopyRight: ebiten.NewImageFromImage(sb.border.SubImage(dims.horizCopyRight)),
		vertCopyRight:  ebiten.NewImageFromImage(sb.border.SubImage(dims.vertCopyRight)),
	}
}

// CreateSizedAndScaledBorderSprite returns the border image for the current display size.
func (sb *ReferenceBorder) CreateSizedAndScaledBorderSprite(
	idealWidthForScaling int, placement PercentBasedPlacement,
) (*ebiten.Image, *ebiten.DrawImageOptions) {
	// THIS STUFF IS FOR PLACEMENT AFTER WE HAVE THE IMAGE
	// get screen sizes
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	return sb.CreateSizedAndScaledBorderSpriteForSize(idealWidthForScaling, placement, screenWidth, screenHeight)
}

// CreateSizedAndScaledBorderSpriteForSize uses captured logical dimensions, so a
// lazily rasterized Border preserves the geometry established at construction.
func (sb *ReferenceBorder) CreateSizedAndScaledBorderSpriteForSize(
	idealWidthForScaling int, placement PercentBasedPlacement, screenWidth, screenHeight int,
) (*ebiten.Image, *ebiten.DrawImageOptions) {
	// get the x start and end values based on the percent
	xLeft := float64(screenWidth) * placement.StartPercentX
	xRight := float64(screenWidth) * placement.EndPercentX
	yTop := float64(screenHeight) * placement.StartPercentY
	yBottom := float64(screenHeight) * placement.EndPercentY

	targetWidth := xRight - xLeft
	targetHeight := yBottom - yTop

	// Create cache key for this specific border configuration
	cacheKey := fmt.Sprintf("%d_%d_%d_%d", screenWidth, idealWidthForScaling, int(targetWidth), int(targetHeight))

	// Check if we already have this border cached (using read lock for performance)
	if cachedBorder := globalBorderCache.get(cacheKey); cachedBorder != nil {
		// Return cached border image with new draw options
		drawOptions := &ebiten.DrawImageOptions{}
		drawOptions.GeoM.Translate(xLeft, yTop)

		return cachedBorder, drawOptions
	}

	// Border not cached, create it
	// get the corners and copies before we scale it
	sb.referenceBorderDimensions = getCornersOfReferenceBorder(sb.border.Bounds().Dx(), sb.border.Bounds().Dy())

	drawOptions := &ebiten.DrawImageOptions{}
	drawOptions.GeoM.Translate(xLeft, yTop)

	sb.referenceBorderBits = sb.createBorderBits()
	goodBorder := sb.createBorderImageForWidth(idealWidthForScaling, int(targetWidth), int(targetHeight), screenWidth)

	globalBorderCache.put(cacheKey, goodBorder)

	return goodBorder, drawOptions
}

// scalePoint multiplies a point by the given scale factors, rounding to the nearest pixel.
func scalePoint(point *image.Point, scaleX, scaleY float64) image.Point {
	return image.Point{
		X: int(math.Round(float64(point.X) * scaleX)),
		Y: int(math.Round(float64(point.Y) * scaleY)),
	}
}

func (r *referenceBorderDimensions) createScaledDimensions(scaleX, scaleY float64) *referenceBorderDimensions {
	referenceBorderDimensions := referenceBorderDimensions{
		topLeft: image.Rectangle{
			Min: scalePoint(&r.topLeft.Min, scaleX, scaleY),
			Max: scalePoint(&r.topLeft.Max, scaleX, scaleY),
		},
		bottomLeft: image.Rectangle{
			Min: scalePoint(&r.bottomLeft.Min, scaleX, scaleY),
			Max: scalePoint(&r.bottomLeft.Max, scaleX, scaleY),
		},
		topRight: image.Rectangle{
			Min: scalePoint(&r.topRight.Min, scaleX, scaleY),
			Max: scalePoint(&r.topRight.Max, scaleX, scaleY),
		},
		bottomRight: image.Rectangle{
			Min: scalePoint(&r.bottomRight.Min, scaleX, scaleY),
			Max: scalePoint(&r.bottomRight.Max, scaleX, scaleY),
		},
		horizCopyLeft: image.Rectangle{
			Min: scalePoint(&r.horizCopyLeft.Min, scaleX, 1),
			Max: scalePoint(&r.horizCopyLeft.Max, scaleX, 1),
		},
		vertCopyLeft: image.Rectangle{
			Min: scalePoint(&r.vertCopyLeft.Min, 1, scaleY),
			Max: scalePoint(&r.vertCopyLeft.Max, 1, scaleY),
		},
		horizCopyRight: image.Rectangle{
			Min: scalePoint(&r.horizCopyRight.Min, scaleX, 1),
			Max: scalePoint(&r.horizCopyRight.Max, scaleX, 1),
		},
		vertCopyRight: image.Rectangle{
			Min: scalePoint(&r.vertCopyRight.Min, 1, scaleY),
			Max: scalePoint(&r.vertCopyRight.Max, 1, scaleY),
		},
	}

	return &referenceBorderDimensions
}

// GetCornersOfReferenceBorder
// Determines the corners for the border, plus the horizontal and vertical slices it will fill in the blanks with
// NOTE: you must do this on the reference because you will muck up the aspect ratio if you do it AFTER a scale
func getCornersOfReferenceBorder(width, height int) *referenceBorderDimensions {
	xMiddle := int(math.Round(float64(width) / referenceBorderHalfDivisor))
	yMiddle := int(math.Round(float64(height) / referenceBorderHalfDivisor))

	return &referenceBorderDimensions{
		topLeft: image.Rectangle{
			Min: image.Point{X: 0, Y: 0},
			Max: image.Point{X: xMiddle - 1, Y: yMiddle - 1},
		},
		topRight: image.Rectangle{
			Min: image.Point{X: xMiddle + 1, Y: 0},
			Max: image.Point{X: width - 1, Y: yMiddle - 1},
		},
		bottomLeft: image.Rectangle{
			Min: image.Point{X: 0, Y: yMiddle + 1},
			Max: image.Point{X: xMiddle - 1, Y: height - 1},
		},
		bottomRight: image.Rectangle{
			Min: image.Point{X: xMiddle + 1, Y: yMiddle + 1},
			Max: image.Point{X: width - 1, Y: height - 1},
		},
		horizCopyLeft: image.Rectangle{
			Min: image.Point{X: 0, Y: yMiddle},
			Max: image.Point{X: width / referenceBorderHalfDivisor, Y: yMiddle + 1},
		},
		vertCopyLeft: image.Rectangle{
			Min: image.Point{X: xMiddle, Y: 0},
			Max: image.Point{X: xMiddle + 1, Y: height / referenceBorderHalfDivisor},
		},
		horizCopyRight: image.Rectangle{
			Min: image.Point{X: width / referenceBorderHalfDivisor, Y: yMiddle},
			Max: image.Point{X: width, Y: yMiddle + 1},
		},
		vertCopyRight: image.Rectangle{
			Min: image.Point{X: xMiddle, Y: height / referenceBorderHalfDivisor},
			Max: image.Point{X: xMiddle + 1, Y: height},
		},
	}
}
