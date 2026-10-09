package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const poisonPulsePeriodMs int64 = 1400

const (
	poisonOverlayAlpha = 0.72
	poisonPulseDivisor = 2.0
)

func activePlayerNeedsPoisonOverlay(gameState *game_state.GameState) bool {
	activePlayer := gameState.GetActivePlayer()

	return activePlayer != nil && activePlayer.IsPoisoned()
}

func (g *GameScene) drawPoisonOverlay(
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
	overlayOptions.ColorScale.ScaleWithColor(poisonPulseColor(elapsedMs))
	overlayOptions.ColorScale.ScaleAlpha(poisonOverlayAlpha)
	screen.DrawImage(mask, &overlayOptions)
}

func poisonPulseColor(elapsedMs int64) color.RGBA {
	phase := float64(elapsedMs%poisonPulsePeriodMs) / float64(poisonPulsePeriodMs)
	amount := (math.Sin(phase*math.Pi*poisonPulseDivisor) + 1) / poisonPulseDivisor
	dark := ucolor.EGAGreen.ToRGBA()
	light := ucolor.EGALightGreen.ToRGBA()

	return color.RGBA{
		R: interpolateColorChannel(dark.R, light.R, amount),
		G: interpolateColorChannel(dark.G, light.G, amount),
		B: interpolateColorChannel(dark.B, light.B, amount),
		A: ucolor.AlphaOpaque,
	}
}

func interpolateColorChannel(from, to uint8, amount float64) uint8 {
	return uint8(math.Round(float64(from) + (float64(to)-float64(from))*amount))
}
