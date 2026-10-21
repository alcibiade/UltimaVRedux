package visual

import (
	"github.com/bradhannah/UltimaVRedux/internal/display"
)

const (
	// Native game map dimensions based on visible tiles
	// From gamescene.go: xTilesVisibleOnGameScreen = 19, yTilesVisibleOnGameScreen = 13
	// From sprites/spritesheet.go: TileSize = 16
	nativeMapWidth  = 16 * 19 // 304 pixels
	nativeMapHeight = 16 * 13 // 208 pixels

	// Game area screen percentages (from gameScreenPercents in gamescene_draw.go)
	// StartPercentX: 0.015, EndPercentX: 0.75 → width = 0.735
	// StartPercentY: 0.02, EndPercentY: 0.98 → height = 0.96
	gameAreaWidthPercent  = 0.735 // 73.5% of screen width
	gameAreaHeightPercent = 0.96  // 96% of screen height

	axisCount = 2.0 // X and Y scales averaged into one uniform factor

	// Safety limits for mask dimensions
	minMaskSize = 8   // Prevent too coarse dissolve
	maxMaskSize = 512 // Prevent excessive memory usage
)

// CalculateDissolveScaleFactor determines the scale factor for MessageDialog dissolve masks
// based on the current screen resolution.
//
// The calculation matches the scaling used for map tiles:
//   - Native map is 304×208 pixels (19×16 tiles, 13×16 tiles)
//   - Scaled to fill game area (73.5% width, 96% height of screen)
//   - Returns the scale factor: (targetScreenSize / nativeMapSize)
//
// For a 1280×800 screen:
//   - Game area: 940×768 pixels
//   - Scale X: 940/304 ≈ 3.09
//   - Scale Y: 768/208 ≈ 3.69
//   - Average: ~3.4×
//
// MessageDialog masks should be created at 1/scaleFactor size for memory efficiency.
func CalculateDissolveScaleFactor() float64 {
	dm := display.GetManager()
	screenWidth, screenHeight := dm.GetCurrentSize()

	// Calculate game area dimensions
	gameAreaWidth := float64(screenWidth) * gameAreaWidthPercent
	gameAreaHeight := float64(screenHeight) * gameAreaHeightPercent

	// Calculate scale factors for X and Y
	scaleX := gameAreaWidth / float64(nativeMapWidth)
	scaleY := gameAreaHeight / float64(nativeMapHeight)

	// Use average of X and Y scales for uniform mask scaling
	// (This matches how tiles are scaled to fit the screen)
	averageScale := (scaleX + scaleY) / axisCount

	// Ensure minimum scale of 1.0 (never create masks larger than target)
	if averageScale < 1.0 {
		return 1.0
	}

	return averageScale
}

// CalculateMaskSizeForDialog determines the optimal mask dimensions for a dialog
// based on the current screen scale factor.
//
// Example:
//
//	Dialog size: 400×300 pixels (full display resolution)
//	Scale factor: 4.0× (high-res display)
//	Mask size: 100×75 pixels (1/4 scale - much more efficient!)
//
// Returns:
//
//	maskWidth, maskHeight - dimensions for the dissolve effect mask
func CalculateMaskSizeForDialog(dialogWidth, dialogHeight int) (int, int) {
	scaleFactor := CalculateDissolveScaleFactor()

	// Calculate native mask size (inverse of scale)
	maskWidth := int(float64(dialogWidth) / scaleFactor)
	maskHeight := int(float64(dialogHeight) / scaleFactor)

	// Apply safety limits
	if maskWidth < minMaskSize {
		maskWidth = minMaskSize
	}
	if maskHeight < minMaskSize {
		maskHeight = minMaskSize
	}

	if maskWidth > maxMaskSize {
		maskWidth = maxMaskSize
	}
	if maskHeight > maxMaskSize {
		maskHeight = maxMaskSize
	}

	return maskWidth, maskHeight
}
