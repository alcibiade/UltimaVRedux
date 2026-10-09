package widgets

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

var _ Widget = &Border{}

type Border struct {
	border            *ebiten.Image
	borderDrawOptions *ebiten.DrawImageOptions

	background            *ebiten.Image
	backgroundDrawOptions *ebiten.DrawImageOptions

	bordersWithScaling int

	interiorColor               color.Color
	interiorAlpha               float32
	interiorRect                *image.Rectangle
	placement                   sprites.PercentBasedPlacement
	logicalWidth, logicalHeight int
}

func (b *Border) Draw(screen *ebiten.Image) {
	b.DrawBackground(screen)
	b.DrawBorder(screen)
}

func (b *Border) DrawBackground(screen *ebiten.Image) {
	if b.background == nil {
		b.initializeInterior()
		b.fillInternalRect()
	}
	screen.DrawImage(b.background, b.backgroundDrawOptions)
}

func (b *Border) DrawBorder(screen *ebiten.Image) {
	if b.border == nil {
		b.initializeBorders(b.placement)
	}
	screen.DrawImage(b.border, b.borderDrawOptions)
}

func (b *Border) Update() {
}

func NewBorder(percentBasedPlacement sprites.PercentBasedPlacement, bordersWithScaling int, interiorColor color.Color) *Border {
	b := &Border{}
	b.bordersWithScaling = bordersWithScaling

	b.interiorColor = interiorColor
	b.interiorAlpha = 0.85

	b.initializeGeometry(percentBasedPlacement)

	return b
}

// NewOpaqueBorder creates a border with a fully opaque background (no transparency)
func NewOpaqueBorder(percentBasedPlacement sprites.PercentBasedPlacement, bordersWithScaling int, interiorColor color.Color) *Border {
	b := &Border{}
	b.bordersWithScaling = bordersWithScaling

	b.interiorColor = interiorColor
	b.interiorAlpha = 1.0 // Fully opaque

	b.initializeGeometry(percentBasedPlacement)

	return b
}

// func (b *Border) SetInteriorColor(color color.Color) {
// 	b.interiorColor = color
// }

func (b *Border) initializeBorders(percentBasedPlacement sprites.PercentBasedPlacement) {
	mainBorder := sprites.NewBorderSprites()
	b.border, b.borderDrawOptions = mainBorder.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSpriteForSize(
		b.bordersWithScaling, percentBasedPlacement, b.logicalWidth, b.logicalHeight)
}

func (b *Border) initializeGeometry(placement sprites.PercentBasedPlacement) {
	b.placement = placement
	b.logicalWidth, b.logicalHeight = display.GetManager().GetCurrentSize()
	b.interiorRect = sprites.GetRectangleFromPercents(placement)
	b.borderDrawOptions = &ebiten.DrawImageOptions{}
	b.borderDrawOptions.GeoM.Translate(float64(b.logicalWidth)*placement.StartPercentX,
		float64(b.logicalHeight)*placement.StartPercentY)
}

func (b *Border) initializeInterior() {
	b.backgroundDrawOptions = &ebiten.DrawImageOptions{}
	*b.backgroundDrawOptions = *b.borderDrawOptions
	b.background = ebiten.NewImageWithOptions(*b.interiorRect, &ebiten.NewImageOptions{Unmanaged: true})
}

// Dispose releases this border's private background, never its shared frame.
// Pure layout users allocate neither; drawing can recreate the background later.
func (b *Border) Dispose() {
	if b.background != nil {
		b.background.Deallocate()
		b.background = nil
	}
}

func (b *Border) fillInternalRect() {
	xDiff := float32(b.interiorRect.Dx()) * LayoutSpacingSmall
	yDiff := float32(b.interiorRect.Dy()) * LayoutSpacingSmall
	vector.FillRect(b.background,
		float32(b.interiorRect.Min.X)+xDiff,
		float32(b.interiorRect.Min.Y)+yDiff,
		float32(b.interiorRect.Dx())-xDiff*2,
		float32(b.interiorRect.Dy())-yDiff*2,
		b.interiorColor,
		// u_color.Black,
		false)

	b.backgroundDrawOptions.ColorScale.ScaleAlpha(b.interiorAlpha)
}

// GetBounds returns the interior rectangle of the border in screen pixel coordinates.
// This is useful for positioning content relative to the border's actual boundaries.
func (b *Border) GetBounds() image.Rectangle {
	return *b.interiorRect
}

// GetInteriorBounds returns the actual drawable interior area, accounting for the border thickness.
// The border graphic takes up approximately 3% on each side (corners and edges).
// Use this when drawing content that must stay inside the visible border area.
func (b *Border) GetInteriorBounds() image.Rectangle {
	// The border graphic itself takes up space - use LayoutSpacingMedium (3%) inset to ensure content
	// is fully inside the visible border area (not just the filled background)
	xDiff := int(float32(b.interiorRect.Dx()) * LayoutSpacingMedium)
	yDiff := int(float32(b.interiorRect.Dy()) * LayoutSpacingMedium)

	return image.Rectangle{
		Min: image.Point{
			X: b.interiorRect.Min.X + xDiff,
			Y: b.interiorRect.Min.Y + yDiff,
		},
		Max: image.Point{
			X: b.interiorRect.Max.X - xDiff,
			Y: b.interiorRect.Max.Y - yDiff,
		},
	}
}
