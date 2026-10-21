package main

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestWindDisplayResizeRebuildsLayoutAndPreservesVisibility(t *testing.T) {
	setTalkHistoryTestResolution(t, 1280, 720)

	windDisplay := NewWindDisplay()
	assert.True(t, windDisplay.IsVisible())
	require.NotNil(t, windDisplay.font)
	require.NotNil(t, windDisplay.output)
	t.Cleanup(func() { windDisplay.background.Deallocate() })
	windDisplay.SetVisible(false)
	windDisplay.ensureText(environment.North)
	oldOutput := windDisplay.output
	oldTextBounds := windDisplay.textBounds
	require.NotNil(t, windDisplay.background)
	oldBackground := windDisplay.background
	windDisplay.ensureText(environment.North)
	assert.Same(t, oldBackground, windDisplay.background, "unchanged wind text should reuse its background")

	setTalkHistoryTestResolution(t, 1920, 1080)
	windDisplay.Update()
	windDisplay.ensureText(environment.North)

	assert.False(t, windDisplay.IsVisible())
	assert.NotSame(t, oldOutput, windDisplay.output)
	assert.NotEqual(t, oldTextBounds, windDisplay.textBounds)
	assert.Equal(t, "North Winds", windDisplay.lastText)
	assert.NotNil(t, windDisplay.background)
	windDisplay.SetVisible(true)
	assert.True(t, windDisplay.IsVisible())
}

func TestWindDisplayGraphicEndCaps(t *testing.T) {
	w := NewWindDisplay()
	t.Cleanup(func() { w.background.Deallocate() })
	for _, direction := range []environment.WindDirection{environment.Calm, environment.North, environment.South, environment.East, environment.West} {
		w.ensureText(direction)
		assert.Equal(t, direction.GetUIString(), w.lastText, "arrows must not be part of the text")
		bounds := w.background.Bounds()
		pixels := make([]byte, bounds.Dx()*bounds.Dy()*4)
		w.background.ReadPixels(pixels)
		blue, white := 0, 0
		for y := range bounds.Dy() {
			for x := range bounds.Dx() / 2 {
				i := (y*bounds.Dx() + x) * 4
				mirror := (y*bounds.Dx() + bounds.Dx() - 1 - x) * 4
				require.Equal(t, pixels[i:i+4], pixels[mirror:mirror+4], "end caps must mirror exactly")
				pixel := color.RGBA{R: pixels[i], G: pixels[i+1], B: pixels[i+2], A: pixels[i+3]}
				if pixel == w.borderBlue {
					blue++
				}
				if pixel == color.RGBAModel.Convert(ucolor.EGAWhite) {
					white++
				}
			}
		}
		assert.Positive(t, blue, "filled blue interior")
		assert.Positive(t, white, "white sloped edge")
		assert.Equal(t, color.RGBAModel.Convert(ucolor.EGABlack), w.background.At(bounds.Dx()/2, bounds.Dy()/2))
		cached := w.background
		w.ensureText(direction)
		assert.Same(t, cached, w.background)
	}
}

// ensureText draws the label for a wind direction; layout tests use it without a screen.
func (w *WindDisplay) ensureText(windDirection environment.WindDirection) {
	w.ensureLabel(windDirection.GetUIString())
}
