package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// Both border readouts use the same rasterized map-frame center, even when
// scaling the frame gives an odd number of pixels.
func mapFrameCenteredX(outer image.Rectangle, width int) int {
	return outer.Min.X + (outer.Dx()-width)/2
}

// Draw mirrored inward-pointing blue wedges with the same white perpendicular
// stroke as the outer border. The sky and wind backing can have different sizes.
func drawMapBorderBookends(backing *ebiten.Image, borderStroke int, borderBlue color.RGBA) {
	width, height := backing.Bounds().Dx(), backing.Bounds().Dy()
	capWidth := max(1, height/2)
	slope := float64(2*(capWidth-1)) / float64(max(1, height-1))
	edgeWidth := max(1, int(math.Round(float64(borderStroke)*math.Hypot(1, slope))))
	for row := range height {
		distance := math.Abs(float64(2*row - (height - 1)))
		reach := 1 + int(math.Round(float64(capWidth-1)*(1-distance/float64(max(1, height-1)))))
		vector.FillRect(backing, 0, float32(row), float32(reach), 1, ucolor.EGAWhite, false)
		vector.FillRect(backing, float32(width-reach), float32(row), float32(reach), 1, ucolor.EGAWhite, false)
		if interior := reach - edgeWidth; interior > 0 {
			vector.FillRect(backing, 0, float32(row), float32(interior), 1, borderBlue, false)
			vector.FillRect(backing, float32(width-interior), float32(row), float32(interior), 1, borderBlue, false)
		}
	}
}
