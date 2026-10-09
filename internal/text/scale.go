package text

import (
	"github.com/bradhannah/UltimaVRedux/internal/display"
)

const (
	unscaledWindowWidth       = 1920
	unscaledWindowHeight      = 1080
	unscaledDefaultPointScale = 20
)

func GetScaleFactorByWindowSize(dm *display.DisplayManager) float64 {
	w, _ := dm.GetCurrentSize()

	return getScaleFactorByWidth(w)
}

func GetScaledNumberToResolution(dm *display.DisplayManager, defaultFontPoint float64) float64 {
	return GetScaleFactorByWindowSize(dm) * defaultFontPoint
}

func getScaleFactorByWidth(width int) float64 {
	return float64(max(1, width)) / float64(unscaledWindowWidth)
}

// GetScaledNumberToWidth scales a standard UI value for a logical render width.
func GetScaledNumberToWidth(width int, defaultValue float64) float64 {
	return getScaleFactorByWidth(width) * defaultValue
}
