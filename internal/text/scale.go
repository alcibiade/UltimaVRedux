package text

import (
	"github.com/bradhannah/UltimaVRedux/internal/display"
)

const (
	unscaledWindowWidth = 1920
)

func getScaleFactorByWindowSize(dm *display.DisplayManager) float64 {
	w, _ := dm.GetCurrentSize()

	return getScaleFactorByWidth(w)
}

// GetScaledNumberToResolution scales a default font point or spacing to the current window width.
func GetScaledNumberToResolution(dm *display.DisplayManager, defaultFontPoint float64) float64 {
	return getScaleFactorByWindowSize(dm) * defaultFontPoint
}

func getScaleFactorByWidth(width int) float64 {
	return float64(max(1, width)) / float64(unscaledWindowWidth)
}

// GetScaledNumberToWidth scales a standard UI value for a logical render width.
func GetScaledNumberToWidth(width int, defaultValue float64) float64 {
	return getScaleFactorByWidth(width) * defaultValue
}
