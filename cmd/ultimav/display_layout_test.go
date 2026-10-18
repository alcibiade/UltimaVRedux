package main

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/display"
)

// logicalGameBounds makes existing layout fixtures use the production policy
// at an explicit 1x scale, independent of the native test host's monitor.
func logicalGameBounds(width, height int) image.Rectangle {
	renderWidth, renderHeight := display.RenderSize(float64(width), float64(height), 1)
	return image.Rect(0, 0, renderWidth, renderHeight)
}
