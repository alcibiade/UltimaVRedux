package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const sleepOverlayPulsePeriodMs int64 = 1800

const sleepOverlayAlpha = 0.58

func (g *GameScene) drawSleepOverlay(
	screen, sprite *ebiten.Image,
	baseOptions *ebiten.DrawImageOptions,
	elapsedMs int64,
) {
	mask := sprites.GetSpriteMask(sprite)
	if mask == nil {
		return
	}
	overlayOptions := *baseOptions
	overlayOptions.ColorScale.Reset()
	overlayOptions.ColorScale.ScaleWithColor(sleepPulseColor(elapsedMs))
	overlayOptions.ColorScale.ScaleAlpha(sleepOverlayAlpha)
	screen.DrawImage(mask, &overlayOptions)
}

func sleepPulseColor(elapsedMs int64) color.RGBA {
	phase := float64(elapsedMs%sleepOverlayPulsePeriodMs) / float64(sleepOverlayPulsePeriodMs)
	amount := (math.Sin(phase*math.Pi*2) + 1) / 2
	dark := ucolor.EGAMagenta.ToRGBA()
	light := ucolor.EGALightMagenta.ToRGBA()

	return color.RGBA{
		R: interpolateColorChannel(dark.R, light.R, amount),
		G: interpolateColorChannel(dark.G, light.G, amount),
		B: interpolateColorChannel(dark.B, light.B, amount),
		A: ucolor.AlphaOpaque,
	}
}
