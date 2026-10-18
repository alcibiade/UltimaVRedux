// Package display owns cross-platform window requests and render geometry.
package display

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

const (
	// WindowFitFraction reserves portable headroom for decorations without
	// claiming to know the platform's work area. Maximize delegates that to the OS.
	WindowFitFraction   = 0.9
	maximumRenderWidth  = 3840
	maximumRenderHeight = 2160
)

// ValidScale normalizes unavailable/invalid monitor metrics during startup.
func ValidScale(scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 1
	}

	return scale
}

// WindowPixels converts integer window units to rounded output pixels.
func WindowPixels(width, height int, scale float64) (int, int) {
	scale = ValidScale(scale)

	return int(math.Round(float64(width) * scale)), int(math.Round(float64(height) * scale))
}

// WindowUnits quantizes a pixel target to the nearest representable window size.
func WindowUnits(width, height int, scale float64) (int, int) {
	scale = ValidScale(scale)

	return max(1, int(math.Round(float64(width)/scale))), max(1, int(math.Round(float64(height)/scale)))
}

// WindowMinimum keeps a 720p output where space permits, while allowing small
// logical desktops to use a smaller container with an internally readable canvas.
func WindowMinimum(monitorWidth, monitorHeight int, scale float64) (int, int) {
	width, height := WindowUnits(config.MinimumWindowWidth, config.MinimumWindowHeight, scale)
	if monitorWidth > 0 && monitorHeight > 0 {
		width = min(width, int(float64(monitorWidth)*WindowFitFraction))
		height = min(height, int(float64(monitorHeight)*WindowFitFraction))
	}

	return max(config.MinimumWindowUnitsWidth, width), max(config.MinimumWindowUnitsHeight, height)
}

// FitWindowSize retains saved logical units where possible and otherwise fits
// their aspect within conservative monitor bounds. It never rewrites their units.
func FitWindowSize(width, height, monitorWidth, monitorHeight int, scale float64) (int, int) {
	minWidth, minHeight := WindowMinimum(monitorWidth, monitorHeight, scale)
	width, height = max(width, minWidth), max(height, minHeight)
	if monitorWidth <= 0 || monitorHeight <= 0 {
		return width, height
	}
	limitWidth := max(minWidth, int(float64(monitorWidth)*WindowFitFraction))
	limitHeight := max(minHeight, int(float64(monitorHeight)*WindowFitFraction))
	fit := min(1, float64(limitWidth)/float64(width), float64(limitHeight)/float64(height))

	return max(minWidth, int(float64(width)*fit)), max(minHeight, int(float64(height)*fit))
}

// RestoreWindowSize preserves a captured custom size, including the space
// reserved only by presets. Fit only when a changed desktop/minimum requires it.
func RestoreWindowSize(width, height, monitorWidth, monitorHeight int, scale float64) (int, int) {
	minWidth, minHeight := WindowMinimum(monitorWidth, monitorHeight, scale)
	tooLarge := monitorWidth > 0 && monitorHeight > 0 && (width > monitorWidth || height > monitorHeight)
	if width >= minWidth && height >= minHeight && !tooLarge {
		return width, height
	}

	return FitWindowSize(width, height, monitorWidth, monitorHeight, scale)
}

// RenderSize fits the drawable pixel density into a 16:9 canvas. The canvas
// remains at least 720p for measured UI layout, and at most 4K for bounded buffers.
func RenderSize(outsideWidth, outsideHeight, scale float64) (int, int) {
	scale = ValidScale(scale)
	width := outsideWidth * scale
	height := outsideHeight * scale
	if width <= 0 || height <= 0 || math.IsNaN(width) || math.IsNaN(height) {
		return config.MinimumWindowWidth, config.MinimumWindowHeight
	}
	fitWidth := min(width, height*aspectWidth/aspectHeight)
	fitWidth = min(maximumRenderWidth, max(config.MinimumWindowWidth, fitWidth))
	fitHeight := min(maximumRenderHeight, fitWidth*aspectHeight/aspectWidth)

	return int(math.Floor(fitWidth)), int(math.Floor(fitHeight))
}
