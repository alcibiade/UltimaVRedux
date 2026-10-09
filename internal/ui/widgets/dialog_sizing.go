package widgets

import (
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

// DialogPlacement defines the screen-relative positioning of a dialog
type DialogPlacement struct {
	StartPercentX float64
	EndPercentX   float64
	StartPercentY float64
	EndPercentY   float64
}

// DialogAutoSizingParams contains parameters for auto-sizing calculations
// This is used by both MessageDialog and InputBox to calculate dimensions
// based on content requirements and font metrics
type DialogAutoSizingParams struct {
	CenterX           float64 // Center point X coordinate (0.0-1.0)
	CenterY           float64 // Center point Y coordinate (0.0-1.0)
	TextLines         int     // Number of content text lines
	CharsPerLine      int     // Maximum characters per line
	PromptLines       int     // Number of prompt lines (0 for InputBox, 1+ for MessageDialog)
	FontPoint         float64 // Font size in points
	LineSpacing       float64 // Line spacing in points
	HorizontalPadding float64 // Horizontal padding as percentage of screen width (default: 0.04)
	VerticalPadding   float64 // Vertical padding as percentage of screen height (default: 0.06)
}

// NewDialogAutoSizingParams creates a DialogAutoSizingParams with default padding values
func NewDialogAutoSizingParams(centerX, centerY float64, contentLines, charsPerLine, promptLines int, fontPoint, lineSpacing float64) DialogAutoSizingParams {
	return DialogAutoSizingParams{
		CenterX:           centerX,
		CenterY:           centerY,
		TextLines:         contentLines,
		CharsPerLine:      charsPerLine,
		PromptLines:       promptLines,
		FontPoint:         fontPoint,
		LineSpacing:       lineSpacing,
		HorizontalPadding: 0.04, // Default 4% horizontal padding
		VerticalPadding:   0.06, // Default 6% vertical padding
	}
}

// CalculateDialogAutoSizeDimensions calculates dialog dimensions based on content requirements
// and font metrics. This shared function is used by both MessageDialog and InputBox to ensure
// consistent sizing behavior across all dialog types.
//
// The algorithm:
// 1. Calculates required width from chars per line + character width + horizontal padding
// 2. Calculates required height from total lines + line spacing + vertical padding
// 3. Centers the dialog at the specified center point
// 4. Returns screen-relative percentages for placement
func CalculateDialogAutoSizeDimensions(params DialogAutoSizingParams) DialogPlacement {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	// Calculate font metrics based on current screen resolution
	scaledFontSize := text.GetScaledNumberToResolution(dm, params.FontPoint)
	scaledLineSpacing := text.GetScaledNumberToResolution(dm, params.LineSpacing)

	// Measure actual character width using representative character "M"
	// This provides accurate sizing for the Ultima V font instead of estimation
	font := text.NewUltimaFont(scaledFontSize)
	charWidth, _ := etext.Measure("M", font.TextFace, 0)

	// Width calculation: chars * char_width + horizontal_padding
	requiredWidthPixels := (float64(params.CharsPerLine) * charWidth) + (float64(screenWidth) * params.HorizontalPadding)
	widthPercent := requiredWidthPixels / float64(screenWidth)

	// Height calculation: total_lines * line_spacing + vertical_padding
	// TextLines now includes prompt lines (not additive)
	totalLines := params.TextLines
	requiredHeightPixels := (float64(totalLines) * scaledLineSpacing) + (float64(screenHeight) * params.VerticalPadding)
	heightPercent := requiredHeightPixels / float64(screenHeight)

	// Center the dialog at the specified center point
	halfWidth := widthPercent / 2.0
	halfHeight := heightPercent / 2.0

	return DialogPlacement{
		StartPercentX: params.CenterX - halfWidth,
		EndPercentX:   params.CenterX + halfWidth,
		StartPercentY: params.CenterY - halfHeight,
		EndPercentY:   params.CenterY + halfHeight,
	}
}
