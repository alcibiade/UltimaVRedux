package text

import (
	"image"
	"math"

	beradtext "github.com/bradhannah/berad-sdk/modules/berad-text"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/pkg/color"
)

// IntroChoices defines the adapted Redux title-menu order.
//
// REDUX DEVIATION: The original menu exposes six choices (ref:INTRO-0001). Redux adds save,
// import, data-path, and Configuration entries to its nine-choice title menu.
//
//nolint:gochecknoglobals // Immutable menu configuration shared by rendering and dispatch.
var IntroChoices = []string{
	"Journey Onward",
	"Select Save Game",
	"Import from Legacy",
	"Set Data Directory",
	"Create New Character",
	"Configure",
	"Ultima V Introduction",
	"Acknowledgments",
	"Return to the View",
}

// Intro menu layout constants
const (
	introMenuTextStartY      = 0.63
	introMenuTextEndY        = 0.96
	introMenuFrameStartX     = 0.02
	introMenuFrameEndX       = 0.98
	introMenuFrameStartY     = 0.595
	introMenuFrameEndY       = 0.995
	introMenuFrameSafeInsetY = 0.032
	introBaseFontHeight      = 0.035
	introTextCoverage        = 0.75
	introFontReferenceHeight = 0.30
	introFontReferenceRows   = 8
	introMenuMaxChoices      = 9
	introHighlightStartX     = 0.2
	introHighlightEndX       = 1 - introHighlightStartX
	introExtraYHeightPercent = 0.003
	introFontMetricScale     = 0.85
	introLogicalWidth        = 320
	introLogicalHeight       = 200
	introHighlightRadius     = 2
)

// IntroMenuFramePlacement returns the title-menu frame bounds.
func IntroMenuFramePlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: introMenuFrameStartX,
		EndPercentX:   introMenuFrameEndX,
		StartPercentY: introMenuFrameStartY,
		EndPercentY:   introMenuFrameEndY,
	}
}

// GetIntroFontSize calculates the font point size needed to fill the target height
// at 1:1 scale (no geometric stretching). This ensures crisp text rendering.
func GetIntroFontSize() float64 {
	dm := display.GetManager()
	_, screenHeight := dm.GetCurrentSize()

	targetPixelHeight := float64(screenHeight) * introMenuFontHeightPercent(len(IntroChoices))

	// Font point size roughly equals pixel height for this font
	// Slight adjustment factor based on font metrics (ascent + descent vs point size)
	return targetPixelHeight * introFontMetricScale
}

func introMenuFontHeightPercent(choiceCount int) float64 {
	if choiceCount <= 0 {
		return introBaseFontHeight
	}
	if choiceCount > introMenuMaxChoices {
		panic("intro menu exceeds its nine-choice layout capacity")
	}
	fontRows := min(choiceCount, introFontReferenceRows)

	return min(introBaseFontHeight, introFontReferenceHeight*introTextCoverage/float64(fontRows))
}

func introMenuChoicePositions(screenHeight int, fontHeight float64, choiceCount int) []float64 {
	if choiceCount <= 0 {
		return nil
	}
	menuTop := float64(screenHeight) * introMenuTextStartY
	menuBottom := float64(screenHeight) * introMenuTextEndY
	choiceStep := 0.0
	if choiceCount > 1 {
		choiceStep = (menuBottom - menuTop - fontHeight) / float64(choiceCount-1)
	}
	positions := make([]float64, choiceCount)
	for index := range positions {
		positions[index] = menuTop + choiceStep*float64(index)
	}

	return positions
}

// DrawIntroChoices renders the intro menu with selected and disabled choice states.
func (f *UltimaFont) DrawIntroChoices(screen *ebiten.Image, nSelection int, enabledChoices []bool) {
	// Render at 1:1 scale (no geometric stretching) for crisp text
	// Font was created at the exact size needed to fill target height
	screenWidth := screen.Bounds().Dx()
	screenHeight := screen.Bounds().Dy()

	// Use font's natural line spacing scaled appropriately
	fontHeight := f.TextFace.Metrics().HAscent + f.TextFace.Metrics().HDescent
	choicePositions := introMenuChoicePositions(screenHeight, fontHeight, len(IntroChoices))

	for index, choice := range IntroChoices {
		yPos := choicePositions[index]

		enabled := index >= len(enabledChoices) || enabledChoices[index]
		bIsSelected := enabled && index == nSelection

		// Draw selection highlight background
		if bIsSelected {
			highlightRect := image.Rect(
				int(float64(screenWidth)*introHighlightStartX),
				int(yPos-float64(screenHeight)*introExtraYHeightPercent),
				int(float64(screenWidth)*introHighlightEndX),
				int(math.Ceil(yPos+fontHeight+float64(screenHeight)*introExtraYHeightPercent)),
			)
			for _, band := range introChoiceHighlightBands(
				highlightRect,
				introClassicPixelUnit(screenWidth, screenHeight),
			) {
				vector.FillRect(
					screen,
					float32(band.Min.X),
					float32(band.Min.Y),
					float32(band.Dx()),
					float32(band.Dy()),
					color.White,
					false,
				)
			}
		}

		layout := f.Layout(choice)
		options := &ebiten.DrawImageOptions{}
		options.GeoM.Translate((float64(screenWidth)-layout.Bounds().Width())/centerDivisor, yPos)
		ink := color.EGAWhite
		switch {
		case bIsSelected:
			ink = color.EGABlack
		case !enabled:
			ink = color.EGADarkGray
		}
		// The title formerly drew directly, so preserve full glyph ink here.
		drawLayoutWithClip(screen, layout, options, beradtext.Palette{"": ink}, nil)
	}
}

func introClassicPixelUnit(width, height int) int {
	scaleX := float64(max(1, width)) / introLogicalWidth
	scaleY := float64(max(1, height)) / introLogicalHeight

	return max(1, int(math.Round(min(scaleX, scaleY))))
}

func introChoiceHighlightBands(rect image.Rectangle, pixelUnit int) []image.Rectangle {
	pixelUnit = max(1, pixelUnit)
	radius := pixelUnit * introHighlightRadius
	if rect.Dx() <= radius*2 || rect.Dy() <= radius*2 {
		return []image.Rectangle{rect}
	}

	return []image.Rectangle{
		image.Rect(rect.Min.X+radius, rect.Min.Y, rect.Max.X-radius, rect.Min.Y+pixelUnit),
		image.Rect(rect.Min.X+pixelUnit, rect.Min.Y+pixelUnit, rect.Max.X-pixelUnit, rect.Max.Y-pixelUnit),
		image.Rect(rect.Min.X+radius, rect.Max.Y-pixelUnit, rect.Max.X-radius, rect.Max.Y),
		image.Rect(rect.Min.X, rect.Min.Y+radius, rect.Max.X, rect.Max.Y-radius),
	}
}
