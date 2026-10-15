package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// Border readouts use the same rasterized map-frame center, even when
// scaling the frame gives an odd number of pixels.
func mapFrameCenteredX(outer image.Rectangle, width int) int {
	return outer.Min.X + (outer.Dx()-width)/2
}

// Draw mirrored inward-pointing blue wedges with the same white perpendicular
// stroke as the outer border. Readout backings can have different sizes.
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

// Sample the actual frame band rather than assuming its source colour or scaled
// stroke. Top and bottom slice rounding can differ at fractional resolutions.
func mapBorderBandStyle(frame *ebiten.Image, band image.Rectangle) (int, color.RGBA) {
	const pixelBytes = 4
	center := band.Min.X + band.Dx()/2
	column, ok := frame.SubImage(image.Rect(center, band.Min.Y, center+1, band.Max.Y)).(*ebiten.Image)
	if !ok {
		panic("map border sample has unexpected image type")
	}
	pixels := make([]byte, column.Bounds().Dy()*pixelBytes)
	column.ReadPixels(pixels)
	stroke, run := 0, 0
	blue := ucolor.EGABlue.ToRGBA()
	for offset := 0; offset < len(pixels); offset += pixelBytes {
		pixel := color.RGBA{R: pixels[offset], G: pixels[offset+1], B: pixels[offset+2], A: pixels[offset+3]}
		if pixel == ucolor.EGAWhite.ToRGBA() {
			run++
			stroke = max(stroke, run)
		} else {
			run = 0
		}
		if pixel.B > pixel.R && pixel.B > pixel.G && pixel.A == ucolor.AlphaOpaque {
			blue = pixel
		}
	}

	return max(1, stroke), blue
}
