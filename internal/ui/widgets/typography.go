package widgets

import (
	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

// StandardFontSize provides semantic font sizes for consistent typography.
// These match the established sizes already in use across the widget system.
type StandardFontSize float64

const (
	// FontSizeXSmall is used for compact UI elements like Slider.
	FontSizeXSmall StandardFontSize = 12

	// FontSizeSmall is used for dense lists and compact elements.
	FontSizeSmall StandardFontSize = 14

	// FontSizeMedium is used for compact character-focused elements.
	FontSizeMedium StandardFontSize = 16

	// FontSizeHUD is for small fixed main-screen indicators such as provisions and wind.
	FontSizeHUD StandardFontSize = 18

	// FontSizeCompact is for dense tables measured not to fit at body size, such as Ready's gear.
	FontSizeCompact StandardFontSize = 20

	// FontSizeXLarge is used for modal dialog titles and headers.
	FontSizeXLarge StandardFontSize = 22

	// FontSizeBody is for readable dialog text: rows, prompts, conversation, footers and section labels.
	FontSizeBody StandardFontSize = 23

	// FontSizeTitle is for the one name at the top of each window.
	FontSizeTitle StandardFontSize = 26
)

// Line spacing multipliers for specific use cases
const (
	// MainScreenLineSpacingMultiplier is the line spacing ratio for mainscreen summaries
	// (character_summary, provision_summary). Calculated from original values: 35/22 ≈ 1.59
	// This provides extra spacing for readability in the compact main game display.
	MainScreenLineSpacingMultiplier = 1.59
)

// FontConfig holds typography settings for widget text rendering.
type FontConfig struct {
	Size                  StandardFontSize
	LineSpacing           float64 // Absolute line spacing (deprecated - use LineSpacingMultiplier)
	LineSpacingMultiplier float64 // Multiplier for line spacing (1.0 = same as font size, 1.3 = 30% extra)
	MaxLines              int
	MaxChars              int
}

// GetScaledFont returns a resolution-scaled UltimaFont.
// Use this when you only need the font without an Output.
func GetScaledFont(size StandardFontSize) *text.UltimaFont {
	dm := display.GetManager()

	return text.NewUltimaFont(text.GetScaledNumberToResolution(dm, float64(size)))
}

// GetScaledFontWithOutput creates both font and output in one call.
// This is the primary helper that replaces the 15+ duplicated initialization patterns.
//
// Example usage:
//
//	font, output := GetScaledFontWithOutput(FontConfig{
//	    Size:     FontSizeSmall,
//	    MaxLines: 10,
//	    MaxChars: 50,
//	})
func GetScaledFontWithOutput(config FontConfig) (*text.UltimaFont, *text.Output) {
	dm := display.GetManager()

	return newScaledFontWithOutput(
		config,
		text.GetScaledNumberToResolution(dm, float64(config.Size)),
		func(value float64) float64 { return text.GetScaledNumberToResolution(dm, value) },
	)
}

// GetScaledFontWithOutputForWidth builds semantic typography for a logical render width.
func GetScaledFontWithOutputForWidth(config FontConfig, width int) (*text.UltimaFont, *text.Output) {
	return newScaledFontWithOutput(
		config,
		text.GetScaledNumberToWidth(width, float64(config.Size)),
		func(value float64) float64 { return text.GetScaledNumberToWidth(width, value) },
	)
}

func newScaledFontWithOutput(
	config FontConfig,
	scaledSize float64,
	scale func(float64) float64,
) (*text.UltimaFont, *text.Output) {
	font := text.NewUltimaFont(scaledSize)

	// Calculate line spacing: prefer multiplier, fallback to absolute, default to 1:1
	var lineSpacing float64
	switch {
	case config.LineSpacingMultiplier > 0:
		// Use multiplier-based spacing (preferred - scales correctly)
		lineSpacing = float64(config.Size) * config.LineSpacingMultiplier
	case config.LineSpacing > 0:
		// Use absolute spacing (deprecated but supported for compatibility)
		lineSpacing = config.LineSpacing
	default:
		// Default 1:1 ratio
		lineSpacing = float64(config.Size)
	}

	scaledLineSpacing := scale(lineSpacing)

	output := text.NewOutput(font, scaledLineSpacing, config.MaxLines, config.MaxChars)
	output.SetFont(font, scaledLineSpacing)

	return font, output
}

// GetScaledLineHeight returns the resolution-scaled line height for a font size.
// Useful when you need to calculate text positioning without creating an Output.
func GetScaledLineHeight(size StandardFontSize) float64 {
	dm := display.GetManager()

	return text.GetScaledNumberToResolution(dm, float64(size))
}
